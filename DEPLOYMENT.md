# RFC: Deployment schemes (single-org & SaaS)

Status: Draft · Branch: `zitadel` · Date: 2026-08-02

## Objective

One codebase, **one deployment model (SaaS-shaped)**, two usage profiles — no mode flag:

1. **Single-org** — the SaaS stack run with exactly one **main (master) organization**:
   the master org is the product, its owner comes from config, admins simply don't add
   more orgs. Initialized entirely from passed configuration: first migrations, seed the
   org + its owner, serve routes/settings, bring up **both** the user app and the admin.
   Turnkey: `up` and it works.
2. **SaaS** — same stack, admins add more organizations from the panel (each with its own
   owner). Each org can get its **own custom domains** (app + login) with per-org branding and
   security policies, on **one shared Zitadel instance** (org-level isolation) — not
   instance-per-org. See the SaaS section.

The two are the same code path — the only difference is whether org #2..N are ever created.
There is deliberately **no `single|saas` mode flag**.

## Context — current readiness (verified)

Ready and production-grade:
- **Migrations**: goose, versioned reversible SQL (`service-core/migrations/0000*.sql`),
  embedded, run blocking on startup (`service-core/main.go:41`), first-boot clean. No AutoMigrate.
- **Config init**: env + `scripts/zitadel/zitadel.config.yaml`, applied idempotently by `zitadel-init`.
- **Master/apex org**: `EnsureMaster` (`organization.go:587`) seeds the single empty-slug
  org from `INFRA_DEFAULT_ORG_NAME`, served at the apex; `GET /tenant/by-host` resolves apex → master.
- **Config-driven routes**: Registrator generates KrakenD config from Docker labels
  (`infra.*`) + each service's OpenAPI. A new microservice is added by labels + healthcheck
  + spec, no gateway editing. Per-route RBAC via `x-infra-scopes` in the service spec.
- **User app (tenant-web)** works at the apex; **frontends** configured at runtime from
  `admin-app.json` / `tenant-app.json` descriptors on the shared volume.

"One domain" is confirmed to mean **one base domain with subdomains** (`admin.`, `auth.`) —
so admin needing `admin.<base>` is not a gap.

Gaps for a turnkey single-org deploy:
- **No owner/first human user** for any org (seeded master or panel-created) — the one real
  blocker (see Approach P1).
- Config hygiene + a dev-friendly domain scheme (drop the `app.` prefix).

The SaaS-shaped machinery (org-create API, tenant wildcard router) stays live in both profiles —
that's intended, not a gap.

## Domain scheme

`INFRA_HTTP_BASE_DOMAIN` is the single knob. Layout per mode:

- **Single-org**: apex `<base>` = user app (master org); `admin.<base>` = admin; `auth.<base>` = Zitadel.
- **Local dev**: set `INFRA_HTTP_BASE_DOMAIN=localhost` → apex `localhost`, `admin.localhost`,
  `auth.localhost`; drop the current `app.localhost` default. `*.localhost` resolves to loopback
  in browsers, no hosts file.
- **SaaS**: tenants as subdomains `tenant1.<base>`, … ; plus optional per-org **custom domains**
  (own app + login host + issuer) — see the SaaS section.

Edge cases to verify when base becomes `localhost` (or any short base):
- The tenant wildcard `HostRegexp(^[a-z0-9-]+\.<base>$)` also matches `admin.<base>`/`auth.<base>`;
  today exact-Host routers win by priority — confirm this still holds and priorities are explicit.
- **Reserved slugs**: `admin`, `auth`, `traefik` must stay rejected as tenant slugs (slug validation
  already reserves words — verify the set covers the subdomains).
- Cookie/redirect origins and CORS derive from `<base>` — confirm nothing assumes a 3-label domain.

## Approach — work items for turnkey single-org (ordered)

**P1 — Owner provisioning (the blocker) — IMPLEMENTED.** A human-owner create was added to
`internal/zitadel` (`EnsureOwner`, `EnsureRoleAccess` now takes role keys and widens grants) and
called from `provisionOrg`/`ReconcileOrgs`, so **every** org gets an owner on creation. Two sources,
one code path:
- **Master org** (single-org): owner from config `INFRA_DEFAULT_ORG_OWNER_EMAIL` /
  `INFRA_DEFAULT_ORG_OWNER_PASSWORD` (+ optional `INFRA_DEFAULT_ORG_OWNER_NAME`, default derived
  from the email). Presence of `INFRA_DEFAULT_ORG_OWNER_EMAIL` is the trigger to initialize the org.
- **Panel-created orgs** (SaaS): owner from the create request payload (admin enters the owner email).

Grant `org_owner` **and** the platform `admin` role (the single-org owner also runs the admin panel).
Idempotent (skip if a human member already exists). Email/invite delivery deferred (unbuilt
mail-service, AGENTS.md:177) — until then, the master-org owner password is config-supplied, so no
delivery is needed; the SaaS temp-password path is surfaced to the admin.

**P2 — Domain scheme.** Make the base cleanly configurable, support `localhost` for dev, fix the
`app.` default, and verify the routing edge cases above.

**P3 — Config hygiene.** Add `INFRA_ADMIN_SUBDOMAIN`, `INFRA_PG_CORE_DB`,
`INFRA_DEFAULT_ORG_OWNER_*` and the `INFRA_SMTP_*` groundwork to `.env.example`; reconcile the
`INFRA_DEFAULT_ORG_NAME` default (`Master` vs `My Organization`);
add a `tenant-web` `image:` to `docker-compose.swarm.yml`; move secrets to Docker secrets (already
tracked as remaining).

## SaaS (scheme #2) — single instance + custom domains

**Model decided:** keep the current **one-instance, org-per-org** architecture and give each org its
own domains as **Zitadel instance custom domains** — not instance-per-org. Verified this covers what
the topology needs:
- per-org **login domain + OIDC issuer** — Zitadel serves the issuer, authorize endpoint and login
  UI per custom domain (each custom domain is a distinct issuer);
- per-org **branding** (label policy);
- per-org **security policies** — Management API custom policies: password complexity, lockout, and
  login (MFA/2FA enforcement, passwordless, allowed factors, external IdP, self-registration);
- **data isolation** via the `organization_id` claim (org managers are scoped to their own org).

Instance-per-org (virtual instances, System API) is **deferred** — it only adds hard infrastructural
separation (per-org data store, no cross-org `IAM_OWNER`, compliance/data-residency), accepted to
forgo for now. Genuinely instance-only and thus shared across orgs: the **domain policy** (loginname
uniqueness scope), **security settings** (iframe/embedding origins), OIDC token lifetimes, and the
cross-org super-admin (`IAM_OWNER`).

**Domain topology (example)** — platform on `saas.com`, a customer org on `org.com`:

| Surface | Host | OIDC issuer |
|---|---|---|
| Platform admin | `admin.saas.com` | `auth.saas.com` |
| Org admin panel | `admin.org.com` | `auth-admin.saas.com` |
| Org user app | `org.com` | `auth.org.com` |

All are custom domains on the one instance; which domain an app authenticates against is a config +
branding choice, gated by role (`admin` / `org_owner` vs `org_user`).

**Work items (post single-org):**
1. **Per-org custom domains** — register the org's domains (e.g. `org.com`, `auth.org.com`,
   `admin.org.com`) as instance domains via the System/Admin API in service-core's provisioning;
   Traefik route + **TLS per customer domain** (ACME with the customer's DNS delegation, or
   customer-supplied certs). This is the main operational cost.
2. **Per-org policy provisioning** — extend service-core to set the org's login/password/lockout
   policies (Management API custom policies) alongside the existing flatten action + redirect URI.
3. **Multi-issuer gateway** — KrakenD must validate tokens from multiple issuers (`auth.org.com`, …)
   now that the issuer is per-domain; today it validates a single issuer.

## Trade-offs

- Env owner + temp password is a stopgap vs a proper invite/mail flow — acceptable to make
  single-org turnkey; revisit when the mail-service lands.
- A mode flag adds one branch; kept minimal (two toggles) to avoid a forked codebase.

## Resolved decisions

- **Master-org owner source:** dedicated env owner `INFRA_DEFAULT_ORG_OWNER_*` (not the Zitadel
  console admin). Owner gets `org_owner` + platform `admin`.
- **SaaS isolation model:** one shared Zitadel instance + instance custom domains (org-level),
  not instance-per-org. Instance-per-org deferred to a hard-isolation/compliance need.

## Open decisions

- **Temp-password delivery (pre mail-service):** for panel-created (SaaS) org owners — log it, write
  it to the init volume, or return it from the create API for the admin to relay. (Master org uses a
  config-supplied password, so this only affects the SaaS path.)
- **Per-org policy defaults:** which login/password/lockout policy an org gets on creation (inherit
  instance default vs a stricter SaaS baseline).
- **Customer-domain TLS:** ACME with delegated DNS vs customer-supplied certs.
