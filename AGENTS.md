# Infra — Agent Context

Read this before working on anything in this repository.

## What this project is

A self-assembling Docker infrastructure suite. Candidate microservices join by adding Docker labels and a healthcheck — Infra discovers them, reads their OpenAPI docs, and registers their endpoints with the API gateway automatically. No manual gateway config.

Core components: Traefik (routing), Logto (OIDC), KrakenD (API gateway), Postgres, Redis, the Registrator (the custom Go service that wires everything together), and logto-init (a one-shot Node.js init container).

## Documentation

Three docs must be kept up to date as the project evolves:

- `README.md` — user-facing: getting started, routing, services, authorization model
- `registrator/README.md` — implementer-facing: Registrator internals, env vars, KrakenD config generation, service contract
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

**KrakenD config is fully generated.** `config/krakend/krakend.json` has an empty `endpoints` array. The Registrator writes and reloads it — never edit endpoints manually.

**KrakenD has no OIDC auto-discovery.** The JWKS URL must be explicit. The Registrator derives it from `INFRA_HTTP_OIDC_SUBDOMAIN` + `INFRA_HTTP_BASE_DOMAIN` + `/oidc/jwks`.

**KrakenD uses `alg: ES384`.** Logto signs tokens with EC P-384. Do not use RS256 — it will reject all tokens.

**`disable_jwk_security`** must be `true` for local HTTP, `false` for prod HTTPS. The Registrator sets this based on `INFRA_HTTP_PROTOCOL`.

**Logto API Resource is required.** Without a configured API Resource in Logto, issued JWTs have no `aud` claim and KrakenD rejects every token. This is created by `logto-init` via `logto.config.yaml`.

**All initial Logto config via `scripts/logto/init.js`.** Never configure Logto manually via the admin UI for anything declared in `logto.config.yaml` — `logto-init` runs on every `compose up` and will overwrite manual changes. Runtime changes to things not covered by the config file may use the Management API directly.

**`infra_init_data` volume.** Shared between `logto-init` and `registrator`. `logto-init` writes M2M credentials to `/run/infra/registrator-m2m.json`; the Registrator reads them on startup.

## Service images (pinned)

| Service | Image |
|---|---|
| Traefik | `traefik:v3.7` |
| Logto | `svhd/logto:latest` |
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
| `http://auth.app.localhost` | Logto OIDC | 3001 |
| `http://auth-admin.app.localhost` | Logto admin console | 3002 |
| `http://traefik.app.localhost` | Traefik dashboard (dev only) | — |

All derived from env vars: `INFRA_HTTP_PROTOCOL`, `INFRA_HTTP_BASE_DOMAIN`, `INFRA_HTTP_OIDC_SUBDOMAIN`, `INFRA_API_ROUTE`.

## Registrator

A Go service in `registrator/`. See [`registrator/README.md`](registrator/README.md) for the full design.

Internal packages:

| Package | Responsibility |
|---|---|
| `registrar` | Service discovery, registry state |
| `gateway` | KrakenD config generation and reload |
| `logto` | Logto Management API client — fetches scope→role mappings |
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
| `x-infra-scopes` | `[]` | required scopes; Registrator resolves these to roles via Logto and writes them to KrakenD `roles` |
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
- **KrakenD zero-downtime in Swarm:** two replicas + blue/green via Traefik. Out of scope for MVP; single replica with `start-first` is sufficient.
- **`depends_on` gap:** `logto-init` does not declare `depends_on: logto` — startup ordering relies on `restart: on-failure` retries. Known issue; on a cold start `logto-init` will retry several times before Logto is ready.
