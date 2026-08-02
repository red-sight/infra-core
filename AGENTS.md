# Infra — Agent Context

Read this before working on anything in this repository.

## What this project is

A self-assembling Docker infrastructure suite. Candidate microservices join by adding Docker labels and a healthcheck — Infra discovers them, reads their OpenAPI docs, and registers their endpoints with the API gateway automatically. No manual gateway config.

Core components: Traefik (routing), Zitadel (OIDC identity provider) + our forked Login UI v2 container (`zitadel-login/`), KrakenD (API gateway), Postgres, Redis, the Registrator (the custom Go service that wires everything together), and zitadel-init (a one-shot Node.js init container).

## Documentation

Three docs must be kept up to date as the project evolves:

- `README.md` — user-facing: getting started, routing, services, authorization model
- `registrator/README.md` — implementer-facing: Registrator internals, env vars, KrakenD config generation, service contract
- `zitadel-login/README.md` — the forked Login UI: what we changed, how to build it, how to re-sync an upstream release
- `AGENTS.md` — agent-facing: key decisions, constraints, open items

**Rule:** when a feature changes how the system works, update the relevant doc(s) before committing. Do not leave docs describing unimplemented designs or outdated behavior.

## Key decisions and why

**Go for the Registrator.** The owner is learning Go and this is an intentional challenge project. Also: resource-constrained MVP servers — Go binary is ~15MB, no runtime, ~20MB RAM at runtime vs Node's 150MB+ image and 50–100MB baseline RAM. Do not suggest rewriting it in another language.

**Three-file Compose pattern:**
- `docker-compose.yml` — base (shared between all environments)
- `docker-compose.override.yml` — local dev only, auto-loaded by `docker compose up`. Contains `depends_on`, debug ports, Traefik insecure dashboard
- `docker-compose.swarm.yml` — Swarm/prod only, explicit. Contains `deploy` blocks

Swarm ignores `depends_on` and `build`. Compose ignores `deploy`. The split is intentional — do not merge them.

**Local dev command:** `docker compose up`
**Prod deploy command:** `docker stack deploy -c docker-compose.yml -c docker-compose.swarm.yml infra`

**Traefik for all routing including frontend.** Frontend services use Traefik's native Docker labels directly — no Registrator involvement. The Registrator only handles API microservices (OpenAPI-declared backends).

**KrakenD config is fully generated.** `config/krakend/krakend.json` has an empty `endpoints` array. The Registrator generates it — never edit endpoints manually. How the generated config reaches a running KrakenD differs by environment (see below).

**One generator, two reload modes — different last mile only.** Discovery and config generation are identical everywhere; only what happens to the generated config differs, selected by `INFRA_REGISTRATOR_RELOAD_MODE` (default inferred: compose→`auto`, swarm→`artifact`). No separate reloader service/component.
- **`auto` (local dev):** Registrator watches Docker health events + poll loop, regenerates `krakend.json`, validates it (`krakend check`), writes it atomically, and restarts the local KrakenD container. Restart is fine locally — no production traffic to drop, and a new API appearing instantly is the desired inner-loop behavior. This is the "dev KrakenD": a container the Registrator may freely restart.
- **`artifact` (prod):** the Registrator runs as a **one-shot deploy job**, not a watcher. After a deploy converges it scans labeled services once, fetches specs, generates the config, runs `krakend check`, and delivers it to KrakenD as an immutable Swarm **config object** (rolling update, `order: start-first`). It never reloads prod KrakenD autonomously and never watches runtime events — *less astonishing*: a new API only appears as the tail of an explicit deploy.

**Deploy-time config source of truth.** Single stack/repo: CI runs `docker stack deploy --detach=false` (waits for convergence), then the one-shot job. Independent repos: each service deploys into the shared overlay network with labels and, after its *own* rollout converges, triggers the same one-shot job (CI-to-CI, not an HTTP poke of a running service); the job scans all labeled services live, so it picks up the new service plus the rest as they currently are. No pipeline needs global knowledge and nobody waits for "all services" — each waits only for itself. Convergence-waiting is CI's job, not the binary.

**Idempotency via hashes — hashes are not versions.** Two hashes, neither is a version: (1) a per-service *contract hash* (over `paths`/operations/schemas/`x-infra-*`) is a change detector that feeds the version gate below; (2) a *full-render hash* (the whole config including resolved roles) is the idempotency key — an unchanged render produces no new config object, so KrakenD is not redeployed. The last-delivered `{service: {version, hash}}` manifest is stored as labels on the live config object — the cluster is its own ledger, no external store.

**Per-service API version gate — the only versioning in automation.** Each service's API version lives in its spec `info.version` (semver). On delivery: if a service's contract hash changed but `info.version` did not increase, the delivery is **rejected** ("bump your version"). The gate enforces *a* bump on any contract change; it does NOT classify breaking vs non-breaking (that needs a per-operation diff, deliberately out of scope), so semver discipline beyond "bump on change" is on the developer. Services serve all live API versions (`/v1/...`, `/v2/...`) from one spec; the Registrator aggregates paths as-is, no per-version backend routing. The infra-wide API version (the aggregated spec's `info.version`) is a **manual** marker bumped only on platform releases — automation never touches it.

**A/B testing is a separate layer, not a gateway-config concern.** Percentage/canary splits live in Traefik (weighted services); product experiments by cohort live in the services themselves, using the propagated `x-user-id` / `x-user-roles` / `x-organization-id`. Valid as long as A/B variants are contract-compatible (same spec); divergent contracts must be expressed as distinct version paths instead.

**Replica load balancing is Swarm's job, not the Registrator's.** Services on the overlay network get a VIP; `http://<name>:<port>` resolves to it and IPVS balances connections across healthy tasks. Replica scaling and task health never touch the KrakenD config — `SwarmAdapter` deliberately watches only `service create/update/remove` events, not tasks. Caveat: VIP balancing is per-connection (L4) and KrakenD keeps backend keep-alives, so traffic can stick to a replica; acceptable at current scale.

**KrakenD has no OIDC auto-discovery.** The JWKS URL must be explicit. The Registrator derives it from `INFRA_HTTP_OIDC_SUBDOMAIN` + `INFRA_HTTP_BASE_DOMAIN` + `/oidc/jwks`.

**KrakenD uses `alg: RS256`.** Zitadel signs tokens with RS256 and publishes its JWKS at `/oauth/v2/keys` (derived from `INFRA_HTTP_OIDC_SUBDOMAIN` + `INFRA_HTTP_BASE_DOMAIN`). The Registrator sets these on the JWT validator.

**`disable_jwk_security`** must be `true` for local HTTP, `false` for prod HTTPS. The Registrator sets this based on `INFRA_HTTP_PROTOCOL`.

**JWT audience = the Zitadel project id.** Tokens must carry the "Infra API" project id in `aud` or KrakenD rejects them. The SPA requests it via the scope `urn:zitadel:iam:org:project:id:<projectId>:aud`; the Registrator reads the project id from `/run/infra/zitadel-platform.json` and sets it as the validator audience.

**All initial Zitadel config via `scripts/zitadel/init.js`.** Never configure Zitadel manually via the console for anything the bootstrap manages (project/roles/apps/machine users/flatten action/admin role) — `zitadel-init` runs on every `compose up` and reconciles idempotently. Structural declarations (project, roles, app keys, machine users) live in `scripts/zitadel/zitadel.config.yaml`. Runtime changes not covered by the bootstrap may use the Management API directly.

**The Login UI v2 is a fork we build, not the upstream image.** Login v2 is a separate Next.js app in Zitadel v4, so it is the only part of the auth surface whose appearance we can own beyond Zitadel's private labeling (colors/logo). `zitadel-login/` vendors upstream `apps/login` at tag `v4.15.2`, re-skinned to admin's tokens (zinc palette, Geist, `--radius*`); the login flow and its security surface are untouched. It mirrors upstream's monorepo layout (`apps/login` + `packages/zitadel-client` + `packages/zitadel-proto` + `proto/`) so re-syncing is a copy, not a re-patch — the npm `@zitadel/client`/`@zitadel/proto` are unusable (last published from the archived `zitadel/typescript`, a year behind v4). Proto codegen is hermetic: `proto-deps/` vendors the third-party protos and our root `buf.yaml` replaces upstream's BSR-backed `proto/buf.yaml`, so no Buf Schema Registry access is needed to build. `INFRA_ZITADEL_LOGIN_IMAGE` selects the image and can be pointed back at `ghcr.io/zitadel/zitadel-login:v4.15.2` — the runtime contract is unchanged. Keep the fork's tag, the core image and that fallback in lockstep on upgrades.

**The login's colors come from the Zitadel label policy, not from the fork.** Private labeling overrides the login's compiled-in defaults for every field it sets, and Zitadel's stock instance policy ships its own blue — so re-skinning the fork alone changes nothing on screen. The effective palette is the `branding:` block in `scripts/zitadel/zitadel.config.yaml`, applied by `zitadel-init` (`ensureBranding`) as an instance label policy; the constants in the fork's `helpers/colors.ts` mirror it as the fallback for orgs with no policy. **Change both together.** Two Zitadel quirks the bootstrap handles: label policy edits land in a preview and need `POST /admin/v1/policies/label/_activate`, and a no-op `PUT` is rejected with 400 ("has not been changed") — so the desired state is diffed against the active policy first, treating an absent boolean as `false` (proto3 omits it).

**Roles reach the access token only with three things set**: the OIDC app's `accessTokenRoleAssertion: true` (else roles go only to id_token/userinfo); the SPA requesting `urn:zitadel:iam:org:projects:roles`; and the **flatten Action** (Complement Token flow) that turns Zitadel's object roles claim `urn:zitadel:iam:org:project:<id>:roles` into a flat `roles` array + `organization_id` — KrakenD can't read the object form. The Action's JS function name must equal the action name; grants are read from `ctx.v1.user.grants`. **Zitadel Actions are per-organization**: the bootstrap installs the flatten Action only in the platform org, so `service-core` installs a per-org copy in each tenant org it provisions (`EnsureRoleFlattenAction`). The Go action script in `internal/zitadel` and `FLATTEN_SCRIPT` in `scripts/zitadel/init.js` must stay in sync.

**`infra_init_data` volume.** Shared between `zitadel-init`, `zitadel-login`, `registrator`, `service-core`, `admin` and `tenant-web`. `zitadel-init` writes machine-user PATs to `/run/infra/registrator-m2m.json` and `/run/infra/service-core-m2m.json` (`{token, apiEndpoint, issuer}`), the Login UI v2 PAT to `login-client.pat` (mode 0644 — the login container is non-root), per-app SPA configs `admin-app.json` / `tenant-app.json` (`{issuer, clientId, projectId, appId, orgId}` — read by the frontends, and `tenant-app.json` also by service-core to append per-org redirect URIs), and `zitadel-platform.json` (`{issuer, projectId, orgId}` — read by the Registrator for the JWT audience). The Zitadel FirstInstance admin PAT (`admin-sa.pat`) is on a separate `zitadel_machinekey` volume. In Swarm the readers are pinned to the same node; moving creds to Docker secrets would remove that pin.

**`service-core` is the source of truth for organizations; Zitadel is a follower.** The goal is to manage everything through our own APIs. An organization is created in our Postgres first (authoritative), then provisioned in Zitadel (a Zitadel *organization*). Zitadel holds the org because membership and the `organization_id` JWT claim depend on it — but core owns it. service-core uses a dedicated `service-core` machine user (PAT) for the Management API so its credentials rotate independently. The provider-neutral link column is `external_id` — Zitadel specifics live behind `internal/zitadel`, so swapping providers changes only that package, not the schema. (The earlier `internal/logto` predecessor proved this: the swap touched only that package + the contracts held.)

**Core→Zitadel sync uses a transactional outbox, never periodic reconciliation.** `POST /admin/organizations` writes the org row and an `outbox_events` row in one DB transaction (core is always consistent). A background worker in service-core (`internal/outbox`) delivers pending events to Zitadel with retries and exponential backoff, filling `external_id` on success. Provisioning is idempotent via Zitadel's **unique org name** (a re-create 409 adopts the existing org by name) — Zitadel v4 removed the v1 org-metadata endpoints, so there is no coreId stamp. Provisioning also grants the org the shared Infra API **project grant** (org_owner/org_user) so its members can hold roles, installs the per-org **flatten Action** (`EnsureRoleFlattenAction` — tenant orgs each need their own, see above), and registers the tenant redirect URIs. A startup `ReconcileOrgs` pass re-asserts the per-org flatten Action + redirect URIs for every existing org (idempotent; heals state an identity-provider re-init resets). "Synced" is derived from `external_id IS NOT NULL`. Failed deliveries are visible as `outbox_events.status = 'failed'`.

**Tenant frontends: one shared SPA, subdomain per organization.** Each organization is reached at `<slug>.<base-domain>` (e.g. `acme.app.localhost`), served by a single multi-tenant frontend (`tenant-web`) — not an app per org. Traefik routes the wildcard `HostRegexp(^[a-z0-9-]+\.<base>$)` to it at a lower priority than the exact-host routers (admin/auth/…), and the org `slug` (DNS-safe, validated, reserved-word-checked) is the subdomain label. Before login, `tenant-web` resolves its org via the public `GET /tenant/by-host?host=<hostname>` (unauthenticated, `x-infra-protected: false`) — service-core maps the host to an org — then signs in and requests an organization-scoped token.

**Master organization — the apex-domain tenant.** Exactly one org may have an **empty slug**; it is the *master* organization (the product owner's org, for non-SaaS or owner use), served on the apex domain `<base>` itself rather than a subdomain. `UNIQUE(slug)` guarantees at most one (the single empty-string row). service-core **seeds it on startup** (`organization.EnsureMaster`, name from `INFRA_DEFAULT_ORG_NAME`, idempotent) via the normal outbox flow, so it is provisioned in Zitadel with the apex redirect URI (`<proto>://<base>/callback`). Traefik routes the apex `Host(<base>)` to `tenant-web` (low priority, so the `/api`, `/docs`, `/openapi.json` path routers still win). The apex is same-origin with the gateway, so no CORS is needed there. The master org is **not** created through the admin form — it is part of initialization.

**One shared Zitadel `Tenant` SPA application; per-org redirect URIs appended at runtime.** On org creation service-core's outbox appends `<proto>://<slug>.<base>/callback` to the shared `Tenant` app's `redirectUris` and `additionalOrigins` (the app is created by the bootstrap; its clientId/projectId are written to `/run/infra/tenant-app.json`, consumed by both `tenant-web` and service-core). Idempotent (PUT only on a real diff — Zitadel rejects no-op updates with 400 "No changes"). **Caveat:** the GET-modify-PUT of the shared app races under concurrent org creation across replicas — fine single-replica; at scale serialize or move to an app-per-org. `tenant-web` scopes login to its org with `urn:zitadel:iam:org:id:<external_id>` (the org's Zitadel id, returned by `/tenant/by-host`). Gateway CORS allows the wildcard tenant origin `<proto>://*.<base>` (`allow_credentials: false`); browser→Zitadel CORS is covered by the app's `additionalOrigins`.

**Data ownership boundary — the rule for any future identity entity (users, roles, membership).**
- **Zitadel owns authentication & authorization**: credentials, login, sessions/MFA, membership-for-auth, project roles/grants that mint JWT claims. Never duplicate authority for these.
- **Core DB owns domain data & relationships.** For an identity entity we must query/join/reference by FK, keep a *minimal projection* (`external_id` + a few denormalized fields), not a full copy.
- **One sync direction per concern** — never bidirectional on the same field (avoids split-brain). Domain-owned creation (org) is core→Zitadel via the outbox; auth facts we only consume (effective roles in the JWT) are read from the token, not stored as authority.
- **Add a projection only when a concrete need appears** (an FK, a rich query, a domain field), not preemptively. Consequence: users are **not** mirrored yet — nothing in core references a user beyond the propagated `x-user-id`.

## Service images (pinned)

| Service | Image |
|---|---|
| Traefik | `traefik:v3.7` |
| Zitadel | `ghcr.io/zitadel/zitadel:v4.15.2` |
| Zitadel Login UI v2 | built from `zitadel-login/` (fork of upstream `apps/login` @ `v4.15.2`) |
| KrakenD | `krakend:latest` |
| Postgres | `postgres:17-alpine` |
| Redis | `redis:7-alpine` |
| NATS | `nats:2.10-alpine` |

When suggesting image updates, pin to a specific version. `latest` is acceptable during development but not for production Swarm deployments.

## Default routes

| Route | Service | Port |
|---|---|---|
| `http://app.localhost` | frontend (user-configured) | — |
| `http://app.localhost/api` | KrakenD | 8080 |
| `http://app.localhost/docs` | Swagger UI | 8080 |
| `http://admin.app.localhost` | admin console (SPA) | 80/5173 |
| `http://<slug>.app.localhost` | tenant-web (per-org frontend, wildcard) | 80/5174 |
| `http://auth.app.localhost` | Zitadel (OIDC + console at `/ui/console`) | 8080 |
| `http://auth.app.localhost/ui/v2/login` | Zitadel Login UI v2 (separate container) | 3000 |
| `http://traefik.app.localhost` | Traefik dashboard (dev only) | — |

All derived from env vars: `INFRA_HTTP_PROTOCOL`, `INFRA_HTTP_BASE_DOMAIN`, `INFRA_HTTP_OIDC_SUBDOMAIN`, `INFRA_API_ROUTE`.

## Registrator

A Go service in `registrator/`. See [`registrator/README.md`](registrator/README.md) for the full design.

Internal packages:

| Package | Responsibility |
|---|---|
| `registrar` | Service discovery, registry state |
| `gateway` | KrakenD config generation and reload |
| `roles` | scope→role mapping (gateway authorization policy; built-in default, env-overridable) |
| `openapi` | Spec fetching, aggregation, Swagger hosting |

Quick reference — candidate service labels:

| Label | Required | Default | Description |
|---|---|---|---|
| `infra.enabled` | yes | — | must be `true` to be discovered |
| `infra.name` | yes | — | service identifier |
| `infra.port` | yes | — | internal container port |
| `infra.openapi-route` | no | `openapi` | path where the OpenAPI spec is served |
| `infra.auth.protected` | no | `true` | default auth requirement for all endpoints |

OpenAPI operation extensions:

| Extension | Default | Description |
|---|---|---|
| `x-infra-protected` | inherits `infra.auth.protected` | override auth per operation |
| `x-infra-scopes` | `[]` | required scopes; Registrator resolves these to roles via its `roles` policy (default mirrors the project roles; override with `INFRA_GATEWAY_ROLE_SCOPES`) and writes them to KrakenD `roles` |
| `x-infra-scopes-matcher` | `"any"` | `"any"` (OR) — roles holding at least one listed scope; `"all"` (AND) — roles holding every listed scope |

Claim propagation to backends:

| JWT claim | Forwarded header |
|---|---|
| `sub` | `x-user-id` |
| `roles` | `x-user-roles` |
| `organization_id` | `x-organization-id` |

## Postgres conventions

Each service gets its own database. `scripts/postgres/init.sh` creates them on first start via `docker-entrypoint-initdb.d`. Add a `CREATE DATABASE` statement here for each new core service that needs one.

In Swarm, Postgres is pinned to a labeled node (`node.labels.infra.postgres == true`) so its data volume survives service rescheduling.

## Known issues

**Stale Docker events cause false unregister after `--force-recreate`** — ~~workaround is `docker compose restart registrator`~~. **Fixed** by the reconcile loop (`INFRA_REGISTRATOR_RECONCILE_INTERVAL`, default `30s`): on each tick, `ComposeAdapter` re-scans all healthy labeled containers and emits healthy events for them, re-registering any that were falsely removed due to stale stop/die events.



**Swagger UI "Unknown Type: array,null"** — Huma generates OpenAPI 3.1 schemas where nullable types use the JSON Schema array syntax (`"type": ["array", "null"]`). The Registrator's aggregator declares the combined spec as `openapi: 3.0.0` but copies Huma schemas verbatim. Swagger UI interprets the spec as 3.0, where `nullable: true` is expected, and renders the field as "Unknown Type: array,null". The API itself is unaffected. Fix options (in preference order):
1. Avoid nullable types in response structs — use zero values instead of pointers where semantically valid.
2. Configure Huma to emit 3.0: `huma.DefaultConfig(...)` accepts an `OpenAPIVersion` field — with `"3.0.3"` Huma emits `nullable: true` instead.
3. Add a schema post-processing step in the Registrator aggregator to rewrite `type: [X, null]` → `type: X, nullable: true`.

## Open / unresolved

- **LocalAdapter (no-Docker dev mode):** `LocalAdapter` for `EnvironmentAdapter` with REST-based registration and heartbeat TTL. Deferred post-MVP.
- **Swarm prod TLS and secrets still manual:** the registrator side is built and the stack is wired. Registrator: `INFRA_REGISTRATOR_RELOAD_MODE` (auto/artifact), `auto` restart with `krakend check` + atomic promote, `artifact` delivery (render → per-service `info.version` gate → immutable config object `krakend-config-<hash>` → `UpdateServiceConfig`), `-once`/`-dry-run`. Verified incl. live single-node swarm tests. Stack (`docker-compose.swarm.yml`): one-shot `zitadel-init`/`zitadel-prepare` (restart `none`), `zitadel`/`zitadel-login` with images + resource limits, all build-only services given images, `order:start-first`, and KrakenD config delivery as a config object — KrakenD boots on the `krakend_bootstrap` config object (the `./config/krakend` bind is dev-only, in the override), then the registrator rolls it onto generated objects. **Remaining (need real infra/decisions):** TLS on :443 (ACME resolver + domain + persisted storage; documented in the swarm file header), Docker secrets for PG password / Zitadel masterkey + admin password (`*_FILE`), and the CI scripting (`stack deploy --detach=false` → `registrator -once`; multi-repo CI-to-CI trigger). The creds files stay on `infra_init_data`, so `zitadel-init`, `zitadel-login`, `registrator` and `service-core` are pinned to the same (manager) node.
- **Owner flow (credential delivery) is pending a mail-service.** Creating an org owner = create a Zitadel user in the org + grant `org_owner` + invite. The invitation/credential delivery needs a mail-service (HTTP email provider → our relay; Zitadel's `Executions.DenyList` must allow that internal host). Not built yet.
- **`startup ordering`:** `zitadel-init` waits for Zitadel readiness in-script (polls discovery) and uses `restart: on-failure`; `zitadel-login` and the frontends `depends_on: zitadel-init` completing.
