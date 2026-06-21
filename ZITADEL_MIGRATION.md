# RFC: Migrate identity provider Logto → Zitadel

Status: Draft · Branch: `zitadel` · Date: 2026-06-21

## Objective

Replace Logto with Zitadel as the identity provider to get **realm-style per-organization
isolation** (a person registers separately per org with a different password), per-org
white-labeling (branding + email sender), and a per-org OIDC application + frontend domain —
at MVP-viable cost (single self-hosted instance, single issuer).

## Context

- Logto OSS is single-tenant; its Organizations are a **shared user pool** — it cannot give
  per-org credentials. Realm isolation on Logto needs one instance per org (Node, heavy) —
  rejected as too expensive for an MVP.
- Zitadel: one Go binary + Postgres, **one instance / one issuer**, users scoped per
  organization (realm-like isolation natively). Org context rides in the JWT claim
  `urn:zitadel:iam:org:id` → the gateway stays single-JWKS. Per-org private labeling +
  custom texts on the hosted login (triggered by the org scope); per-org OIDC apps with their
  own `redirectURIs` + `additionalOrigins`.
- The current design already anticipated a provider swap: orgs are linked by a
  provider-neutral `external_id` and all Logto specifics live behind `internal/logto`. The
  org schema and the core→provider outbox flow are reused unchanged.

### Decisions already made

- **Login page domain: shared `auth.<base>`** (single issuer). Per-org branding via the
  `urn:zitadel:iam:org:id:{id}` scope. This is parity with today (Logto also used a single
  `auth.<base>` login endpoint) plus per-org branding Logto could not do. A per-org *login*
  domain would require Zitadel virtual-instances (multi-issuer) — deliberately avoided.
- **Default org from env** (extend `EnsureMaster`): one org on the base domain — the non-SaaS
  default. Nothing extra is provisioned when SaaS isn't needed.
- **Token in `Authorization` header, not cookies** → gateway `allow_credentials:false`.

## Approach (phased — each phase independently verifiable)

### Phase 0 — Spike & API shapes (de-risk before code)
Stand up Zitadel + Postgres locally and confirm the concrete shapes the rest depends on:
- Bootstrap mechanism: Management API script (mirrors today's `init.js`) vs the Zitadel
  Terraform provider. Recommend a **bootstrap script** for consistency with the current
  pattern; Zitadel's own `FirstInstance` setup creates the initial machine user + PAT/key
  that the script then uses. (No manual console — same principle as `init.js`.)
- **Role model mapping** (biggest conceptual change): today endpoints declare `x-infra-scopes`
  → registrator resolves scopes→roles via Logto → KrakenD checks `roles`. Zitadel uses
  **project roles + grants**, surfaced in the JWT as `urn:zitadel:iam:org:project:roles`
  (requires "assert roles on authentication"). Decide how `x-infra-scopes` maps to Zitadel
  project roles and how the registrator resolves them.
- RS256 JWKS endpoint; org-scope branding; per-org app `additionalOrigins`/`redirectURIs`;
  **notification HTTP provider** payload (fields + org context) for the mail-service.
- Output: a short addendum to this RFC pinning the API calls + claim names.

### Phase 1 — Infra / compose
- Add `zitadel` (image + Postgres DB via `scripts/postgres/init.sh`) and a one-shot
  `zitadel-init`/bootstrap; remove `logto` + `logto-init`. Traefik route `auth.<base>` →
  zitadel. Keep the three-file compose split. Machine creds (PAT/key) delivered via the
  shared `infra_init_data` volume (mirrors the current M2M file; Docker secret in Swarm).

### Phase 2 — Bootstrap config (`scripts/zitadel/`, replaces `scripts/logto/`)
Reproducibly create: the project + roles (from `Infra API` scopes), the **default org**
(env), OIDC apps (admin SPA + default tenant SPA), machine users for registrator and
service-core management, the **notification HTTP provider → mail-service**, and default
branding/texts. Write per-app config + machine creds to `infra_init_data`.

### Phase 3 — service-core
- `internal/logto` → `internal/zitadel` implementing the existing `OrgProvisioner` /
  `Directory` contracts against the Zitadel Management API: create org, create per-org OIDC
  app (redirectURIs + additionalOrigins for the org domain), set per-org branding, list
  members + their roles. Wire in `main.go`. **Schema unchanged** (`external_id` reused).
- `EnsureMaster` → seed the env-driven default org. The redirect-URI reconcile becomes a
  per-org app/origin reconcile.
- Owner-creation flow (the original task) on Zitadel: create org-scoped user + grant
  `org_owner` → Zitadel notification → HTTP provider → mail-service (org context known from
  the single instance; no Logto `link`-smuggling needed).

### Phase 4 — registrator + KrakenD
- JWT validator **ES384 → RS256**, JWKS URL → Zitadel; claim mapping (`urn:zitadel:iam:org:id`
  → `x-organization-id`; project-roles claim → `x-user-roles`).
- `internal/logto` (scope→role fetch) → Zitadel project roles per the Phase 0 mapping.
- **Per-org CORS:** generate each org's custom domain into KrakenD `allow_origins` (org
  domains read from core). The wildcard `*.<base>` no longer covers custom domains.

### Phase 5 — frontends (admin + tenant-web)
- Swap `@logto/*` SDK → generic OIDC (e.g. `oidc-client-ts`), Auth Code + PKCE, pass the org
  scope. Read issuer/clientId/origin from the per-app config files written by the bootstrap.
- tenant-web on the org's own domain (Traefik router + TLS per domain); admin on `admin.<base>`.

### Phase 6 — docs & cleanup
- Rewrite **AGENTS.md** (ES384→RS256 note, org/identity model, shared-Tenant-app → per-org
  app, default org, login-domain decision), **README.md**, **registrator/README.md**.
- Remove `scripts/logto/`, `service-core/internal/logto`, `registrator/internal/logto`.
- Update `.env.example` / `admin/.env.example`. Reconcile `ARCHITECTURE_REVIEW.md` items that
  no longer apply.

## Trade-offs

- **Migration cost** is real but **contained** by the `external_id` + provider-package
  boundary; the org schema and outbox flow survive. Cheapest to do now (branch `logto`, MVP,
  no production).
- **Login-page domain** is shared `auth.<base>`, not per-org (accepted; parity with today).
- **Role model** is the riskiest change (scopes→roles vs project-roles+grants) — Phase 0 must
  pin it before Phase 4.
- **Password-hashing CPU spikes** on a small single server — acceptable for MVP, size CPU in prod.

## Open questions

1. Bootstrap via Management-API script (recommended) or Terraform provider?
2. ~~Exact `x-infra-scopes` → Zitadel project-role mapping~~ — RESOLVED (Phase 0): map to
   project `roleKey`s; flatten the object roles claim to a `roles` array + `organization_id`
   via a Zitadel Action so the gateway logic is unchanged.
3. ~~Notification HTTP provider payload / org context~~ — RESOLVED (Phase 0): org context rides
   in `templateData.url` (`orgID=`), recipient in `contextInfo`; webhook is signed
   (`signingKey`). Remaining sub-item: pin the `Executions.DenyList` override format so Zitadel
   can reach the internal mail-service.
4. Custom-domain provisioning automation (Traefik dynamic router + ACME TLS per org) — owned
   by service-core or the registrator?

## Phase 0 addendum — spike findings (verified live)

Ran against a local spike (`ghcr.io/zitadel/zitadel:v4.15.2` + `postgres:17-alpine`, isolated
compose). Critical assumptions confirmed against a real instance:

- **Bootstrap → PAT file (confirmed).** `start-from-init --masterkey <32 chars>` +
  `ZITADEL_FIRSTINSTANCE_*` writes a machine-user **PAT (a bearer JWT)** to
  `ZITADEL_FIRSTINSTANCE_PATPATH`. This is the `zitadel-init` mechanism — mirrors today's M2M
  file on `infra_init_data`. Gotcha: the target dir must be writable by Zitadel's **non-root**
  user (first run failed with `permission denied` on a root-owned named volume → pre-create the
  path / fix ownership, don't rely on a fresh named volume).
- **Single issuer + RS256 (confirmed).** Discovery issuer = `ExternalDomain:ExternalPort`;
  JWKS at **`/oauth/v2/keys`**, keys are **RS256** RSA (`id_token_signing_alg` offers
  EdDSA/RS256/…/ES384, default RS256). Gateway: ES384 → RS256, JWKS URL → `<auth>/oauth/v2/keys`.
  Note: the container **always listens on 8080**; `ExternalPort` only shapes the issuer URL —
  so Traefik routes `auth.<base>` → `zitadel:8080` and we set EXTERNALDOMAIN/PORT to the public host.
- **Management API shape (confirmed).** Base `/management/v1`, `Authorization: Bearer <PAT>`,
  org scoping via the **`x-zitadel-orgid`** header. Verified: `orgs/me`, create org
  (`POST /orgs`), create project (`POST /projects`), create project role
  (`POST /projects/{id}/roles` — `roleKey`/`displayName`/`group`).
- **Realm isolation (confirmed).** A second org is created cleanly; its project/roles/app are
  `resourceOwner`-scoped to that org. This is the per-org isolation primitive.
- **Role model = project roles + roles claim (confirmed live).** Roles are project roles
  (`roleKey`), asserted via the project's `projectRoleAssertion=true`. Captured in a live
  token (machine user, client_credentials, JWT access token) the claim is:
  `urn:zitadel:iam:org:project:<PROJECT_ID>:roles` with value
  `{ "<roleKey>": { "<orgId>": "<orgDomain>" } }` — i.e. an **object keyed by role**, not a
  flat array, and the **org id is embedded as the inner key** (so org context rides in the
  roles claim). Project id is fixed/known (our single "Infra API" project) → the claim key is
  deterministic. **Implication for Phase 4:** KrakenD expects a flat roles **array**; mirror
  today's `init.js getCustomJwtClaims` with a **Zitadel Action (pre-access-token)** that
  flattens to a `roles: [...]` array + an `organization_id` claim. Then the gateway/registrator
  role logic and claim propagation (`x-user-roles`, `x-organization-id`) stay essentially as-is —
  only `alg` (RS256), JWKS URL, and the source claim setup change. Mapping: endpoint
  `x-infra-scopes` → Zitadel project `roleKey`s.
- **Per-org OIDC app + CORS (confirmed).** `POST /management/v1/projects/{id}/apps/oidc`
  accepts `redirectUris`, `postLogoutRedirectUris`, **`additionalOrigins`**, `appType:
  OIDC_APP_TYPE_USER_AGENT`, `authMethodType: OIDC_AUTH_METHOD_TYPE_NONE` (public/PKCE) →
  returns `appId` + `clientId`. Per-org app + browser→Zitadel CORS handled here.
- **Per-org branding (confirmed).** `GET /management/v1/policies/label` with `x-zitadel-orgid`
  → 200; private labeling is per-org.
- **Notification HTTP provider (path + payload from docs; live send SSRF-blocked).** REST
  path confirmed: **`POST /admin/v1/email/http`** body `{endpoint, description?}`; the response
  carries a **`signingKey`** → Zitadel **signs** the webhook (mail-service must verify it).
  Payload Zitadel POSTs (per docs): `contextInfo` (`eventType` e.g.
  `user.human.initialization.code.added`, `recipientEmailAddress`) + `templateData`
  (`subject`, `url` — **with `orgID=` embedded** — plus styling) + `args` (`code`, `userName`,
  names, …). So **org context arrives via the `url`** (direct analog of the earlier Logto
  `link`-param plan) and the recipient is present → mail-service can route per-org template +
  transport. **Live capture was blocked** by Zitadel's SSRF guard:
  `Errors.NotificationProvider.Blocked.Endpoint` — notification endpoints resolving to
  localhost/private ranges are denied (`Executions.DenyList`). **Migration requirement:** in
  the real stack configure `Executions.DenyList`/allow so Zitadel can reach the internal
  `mail-service` host. (Single-env `ZITADEL_EXECUTIONS_DENYLIST` override didn't take in the
  spike — pin the exact list-override format in Phase 1/3.)

### Phase 1 addendum — Login UI v2 is a separate container (verified live)

Zitadel v4 moved the hosted login out of the core: the core only **redirects** OIDC auth
to `/ui/v2/login`; the login pages are served by a separate **`ghcr.io/zitadel/zitadel-login`**
container (Next.js). Without it, the browser login 404s (the core has no v1 login form to fall
back to in this build, and there is no instance feature flag to revert). Added to the stack:
- `zitadel-login` service, routed `Host(auth.<base>) && PathPrefix(/ui/v2/login)` (priority
  above the core router), env `ZITADEL_API_URL` = issuer host (via the Traefik alias, so the
  Host matches) + `ZITADEL_SERVICE_USER_TOKEN_FILE`.
- zitadel-init now provisions a **`login-client`** machine user with the instance role
  **`IAM_LOGIN_CLIENT`** and writes its PAT to `/run/infra/login-client.pat`. That file is
  **0644** (not 0600) because the login container runs non-root and must read it — dev-only;
  prod should deliver it via a Docker secret.
- The login image is **version-locked to the core** (`:v4.15.2`).
Verified end-to-end with a real browser (Playwright): loginname → password → OIDC callback →
authenticated Management Console. This same Login v2 flow is what the app SPAs use in Phase 5.

### Phase 4 addendum — gateway + flatten action (verified live, except interactive e2e)

- Registrator is **decoupled from the IdP API**: it no longer fetches scope→role from
  the provider. The JWT audience is the **Zitadel project id**, read from
  `/run/infra/zitadel-platform.json`; the scope→role mapping is gateway authorization
  policy in `registrator/internal/roles` (built-in default mirroring the old
  logto.config.yaml roles, overridable via `INFRA_GATEWAY_ROLE_SCOPES`). `internal/logto`
  is now dead (removed in Phase 6).
- Generated KrakenD validator verified live: `alg: RS256`, `jwk_url:
  <auth>/oauth/v2/keys`, `audience: [<projectId>]`; scope→role applied correctly
  (delete:items → [admin,org_owner]; *:organizations → [admin]); `krakend check` passes.
- **Flatten action** (`scripts/zitadel/init.js` `ensureFlattenAction`): created + wired to
  the Complement Token flow (type 2) triggers 4 (pre-userinfo) + 5 (pre-access-token),
  idempotently. It maps the object roles claim → flat `roles` array + `organization_id`.
- **Open / to verify with Phase 5:** the flat claims only appear in **interactive user
  tokens** — `client_credentials` (machine) tokens do NOT run the Complement Token flow,
  so the flatten output and the full KrakenD propagation (`x-user-roles` /
  `x-organization-id`) must be confirmed once a real login mints a user token (Phase 5).
- **Per-org CORS** unchanged (admin origin + `*.<base>` tenant wildcard). Custom-domain
  CORS generation stays deferred until custom domains exist (open question 4).

### Phase 3 addendum — cross-org role model (verified live)

Provisioning + role assignment were validated end-to-end on a clean-slate bring-up
(full `down -v` → fresh up; the whole Phase 1–3 chain bootstraps with no manual steps,
service-core seeds the default org into Zitadel automatically). Key model finding:

- **Roles are cross-org and need TWO grants.** The shared "Infra API" project lives in
  the platform org; a user in a *tenant* org gets roles only if (a) the project is
  **granted to that org** (a Zitadel *project grant*, `grantedRoleKeys`), and (b) the
  user has a **user grant** (`projectId` + `projectGrantId` + `roleKeys`). A direct
  user grant without the project grant asserts **no** roles.
  - service-core now does (a) in `provisionOrg` via `EnsureRoleAccess` (project grant of
    org_owner/org_user — never platform `admin` — to every org, idempotent). Verified:
    each provisioned org gets an ACTIVE grant with `grantedRoleKeys:[org_owner,org_user]`.
  - The owner flow will do (b) (Phase 3b / with mail-service).
- **The token must request scope `urn:zitadel:iam:org:projects:roles`** (plural) for roles
  to appear — `projectRoleAssertion=true` alone is not enough for the assertion. → the
  SPAs must request it (Phase 5).
- **Roles claim key is deterministic:** `urn:zitadel:iam:org:project:<projectId>:roles`
  with value `{roleKey:{orgId:orgDomain}}`; `projectId` is in `/run/infra/zitadel-platform.json`
  (Phase 4 flatten Action + KrakenD use it).

Net: every load-bearing assumption holds — single-issuer **RS256**, per-org isolation +
branding + OIDC app + `additionalOrigins` CORS, PAT-file bootstrap, and the **roles claim**
captured live (object-shaped → needs a flattening Action). Two concrete carry-overs:
a **flattening Zitadel Action** (`roles`+`organization_id`) before Phase 4, and the
**`Executions.DenyList` allow** for the internal mail-service before the Phase 3 email piece.
