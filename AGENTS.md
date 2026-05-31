# Infra — Agent Context

Read this before working on anything in this repository.

## What this project is

A self-assembling Docker infrastructure suite. Candidate microservices join by adding Docker labels and a healthcheck — Infra discovers them, reads their OpenAPI docs, and registers their endpoints with the API gateway automatically. No manual gateway config.

Core components: Traefik (routing), Logto (OIDC), KrakenD (API gateway), Postgres, Redis, and the Registrator (the custom Go service that wires everything together).

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

**`disable_jwk_security`** must be `true` for local HTTP, `false` for prod HTTPS. The Registrator sets this based on `INFRA_HTTP_PROTOCOL`.

**Logto API Resource is required.** Without a configured API Resource in Logto, issued JWTs have no `aud` claim and KrakenD rejects every token. This must be created during headless init.

## Service images (pinned)

| Service | Image |
|---|---|
| Traefik | `traefik:v3.7` |
| Logto | `svhd/logto:latest` |
| KrakenD | `krakend:latest` |
| Postgres | `postgres:17-alpine` |
| Redis | `redis:7-alpine` |

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

A Go service planned in `registrator/`. See [`registrator/README.md`](registrator/README.md) for the full design.

Quick reference — candidate service labels:

| Label | Required | Default | Description |
|---|---|---|---|
| `infra.enabled` | yes | — | must be `true` to be discovered |
| `infra.name` | yes | — | service identifier |
| `infra.openapi-route` | no | `openapi` | path where the OpenAPI spec is served |
| `infra.auth.protected` | no | `true` | default auth requirement for all endpoints |

## Postgres conventions

Each service gets its own database. `scripts/postgres/init.sh` creates them on first start via `docker-entrypoint-initdb.d`. Add a `CREATE DATABASE` statement here for each new core service that needs one.

In Swarm, Postgres is pinned to a labeled node (`node.labels.infra.postgres == true`) so its data volume survives service rescheduling.

## Open / unresolved

- **Logto startup command**: the `entrypoint` in `docker-compose.yml` that seeds the DB and starts the node process needs validation against the actual image internals.
- **Postgres init env vars**: `scripts/postgres/init.sh` references `INFRA_PG_LOGTO_DB` — confirm this env var is available inside `docker-entrypoint-initdb.d` at runtime.
