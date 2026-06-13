# Registrator

The Go service that wires Infra together. It watches Docker for healthy API services, generates KrakenD configuration, and aggregates OpenAPI specs into a unified Swagger UI.

## Internal packages

| Package | Responsibility |
|---|---|
| `registrar` | Service discovery, registry state |
| `gateway` | KrakenD config generation and reload |
| `logto` | Logto Management API client — fetches scope→role mappings |
| `openapi` | Spec fetching, aggregation, Swagger hosting |

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `INFRA_MODE` | auto-detect | `compose` or `swarm` — overrides auto-detection |
| `INFRA_REGISTRATOR_RELOAD_MODE` | inferred | `auto` or `artifact`. Default inferred from `INFRA_MODE`: compose→`auto`, swarm→`artifact`. `auto` restarts the local KrakenD container on change (dev); `artifact` regenerates only and never restarts KrakenD autonomously — applying is a gated deploy step (prod) |
| `INFRA_HTTP_PROTOCOL` | `http` | `http` or `https` |
| `INFRA_HTTP_BASE_DOMAIN` | `app.localhost` | base domain |
| `INFRA_HTTP_OIDC_SUBDOMAIN` | `auth` | OIDC subdomain prefix |
| `INFRA_API_ROUTE` | `api` | API gateway path prefix |
| `INFRA_LOGTO_API_RESOURCE_ID` | — | Logto API Resource indicator (required) |
| `INFRA_REGISTRATOR_RELOAD_DELAY` | `5s` | debounce delay before KrakenD reload |
| `INFRA_REGISTRATOR_RECONCILE_INTERVAL` | `30s` | reconcile loop interval — re-scans all healthy labeled containers and re-registers any that were falsely removed due to stale Docker events |
| `INFRA_KRAKEND_CONFIG_PATH` | `/etc/krakend/krakend.json` | path to the KrakenD config file (inside the registrator container) |
| `INFRA_REGISTRATOR_VALIDATE` | `true` | run `krakend check` on each candidate config before promoting it; set `false` to skip validation |
| `INFRA_KRAKEND_IMAGE` | `krakend:latest` | image used for the one-shot `krakend-check` validation container — pin to match the running KrakenD in prod |

## Service discovery

### EnvironmentAdapter

The only part that differs between Compose and Swarm. Everything downstream consumes `ServiceEvent`s regardless of source.

```go
type EnvironmentAdapter interface {
    WatchServices() <-chan ServiceEvent
    Mode() string // "compose" | "swarm"
}

type ServiceEvent struct {
    Name    string
    Port    int
    Healthy bool
    Removed bool
}
```

Backend address is `http://<name>:<port>` in both modes — Docker DNS resolves identically on bridge and overlay networks.

### ComposeAdapter

Subscribes to Docker `container` events. Emits a `ServiceEvent` on `health_status: healthy` (Healthy: true) and `health_status: unhealthy` / `die` / `stop` (Healthy: false, or Removed: true).

In addition to event-driven updates, `ComposeAdapter` runs a reconcile loop on every `INFRA_REGISTRATOR_RECONCILE_INTERVAL` tick. It re-scans all healthy labeled containers and re-emits healthy events for them, recovering any services that were falsely unregistered by stale stop/die events from `--force-recreate`.

### SwarmAdapter

Subscribes to Docker `service` task events. Emits a `ServiceEvent` when a task transitions to state `running` (Healthy: true) or leaves it (Healthy: false / Removed: true).

### Auto-detection

`GET /info` on the Docker socket. If `.Swarm.LocalNodeState == "active"`, use `SwarmAdapter`. Otherwise use `ComposeAdapter`. Overridden by `INFRA_MODE`.

## Debounce and reload

The debounce timer resets on every new `ServiceEvent`. When it fires (after `INFRA_REGISTRATOR_RELOAD_DELAY`):

1. Fetch OpenAPI spec from each healthy registered service
2. Query Logto Management API for current scope→role mappings
3. Generate the new config and, unless `INFRA_REGISTRATOR_VALIDATE=false`, validate it with `krakend check` in a one-shot `krakend-check` container before promoting it over `config/krakend/krakend.json`. The candidate is written to a temp file in the config dir, validated there, and atomically renamed in only if valid (and only if the rendered config changed). An invalid candidate is discarded — the live config is left untouched and KrakenD is not touched. The `krakend-check` container is removed on success and left for inspection on failure.
4. Deliver, per `INFRA_REGISTRATOR_RELOAD_MODE`:
   - `auto` (dev): restart the local KrakenD container so it re-reads the config.
   - `artifact` (prod): do nothing — KrakenD is never restarted autonomously; applying the regenerated config is a gated deploy step.

Generation is identical in both modes; only step 4 differs. Steps 1–4 are serialized by a mutex so the debounce timer and the poll ticker never generate concurrently.

> The prod `artifact` path — running as a one-shot deploy job, delivering the config as an immutable Swarm config object, and the per-service `info.version` gate — is being built incrementally. The current `artifact` reloader only enforces the "no autonomous reload" guarantee.

## RBAC — scope→role resolution

Before generating each KrakenD config, the Registrator queries the Logto Management API via the `logto` package to build a scope→role map. Both user roles and organization roles are fetched and merged into a single map.

When an OpenAPI operation declares `x-infra-scopes: [read:items, write:items]`, the Registrator resolves which roles hold those scopes and writes the resulting role list into the KrakenD `auth/validator` `roles` field. KrakenD then validates the flat `roles` claim in the JWT against that list.

This means:
- API specs declare intent (required scopes).
- Role assignments live in Logto (`logto.config.yaml` / Management API).
- KrakenD enforces a flat role list — it never sees scope names.

## KrakenD config generation

`config/krakend/krakend.json` always has an empty `endpoints` array at rest. The Registrator owns it entirely — never edit endpoints manually.

### Gateway route format

```
/api/{service-name}/{version}/{service-api-route}
```

`{version}` is taken from the service's OpenAPI `info.version` field, defaulting to `v1` if absent or non-semver.

**Example:** service `orders` with `info.version: v2` exposing `GET /orders` →
KrakenD endpoint `GET /api/orders/v2/orders`, backend `http://orders:3000/orders`.

### Auth/validator — global fields

Applied to every protected endpoint, derived from env:

| KrakenD field | Value |
|---|---|
| `alg` | `ES384` (Logto signs with EC P-384) |
| `jwk_url` | `{INFRA_HTTP_PROTOCOL}://{INFRA_HTTP_OIDC_SUBDOMAIN}.{INFRA_HTTP_BASE_DOMAIN}/oidc/jwks` |
| `disable_jwk_security` | `true` if `INFRA_HTTP_PROTOCOL=http`, else `false` |
| `audience` | `INFRA_LOGTO_API_RESOURCE_ID` |

### OpenAPI extension labels

Declared per-operation in the service's OpenAPI spec:

| Extension | Type | Default | Description |
|---|---|---|---|
| `x-infra-protected` | `bool` | inherits `infra.auth.protected` Docker label | `false` = public endpoint, no `auth/validator` block generated |
| `x-infra-scopes` | `string[]` | — | required scopes; Registrator resolves these to roles via Logto and sets them in KrakenD `roles` |
| `x-infra-scopes-matcher` | `"all"\|"any"` | `"any"` | scope matching logic used when resolving which roles qualify — `"any"` (OR) means roles that hold at least one listed scope; `"all"` (AND) means roles that hold every listed scope |

A protected endpoint with no scopes declared (`x-infra-protected: true`, no `x-infra-scopes`) accepts any valid JWT.

### Claim propagation

Applied to every protected endpoint via KrakenD `propagate_claims` and `input_headers`:

| JWT claim | Forwarded header |
|---|---|
| `sub` | `x-user-id` |
| `roles` | `x-user-roles` |
| `organization_id` | `x-organization-id` |
| `Authorization` | `Authorization` (raw JWT passthrough via `input_headers`) |

## OpenAPI aggregation and Swagger

On each debounce fire, the Registrator fetches `http://<name>:<port>/<openapi-route>` from every healthy service and writes an aggregated spec to `config/openapi/openapi.json`. This file is bind-mounted into both the Registrator and the Swagger UI container — no volume seeding or container restarts needed when the spec updates. The Registrator itself also exposes `GET /openapi.json` for programmatic access (CI, tooling).

No UI is embedded in the Registrator binary.

### Path rewriting

Each service's raw paths are prefixed in the aggregated spec to reflect the gateway route:

```
/items  →  /api/{service-name}/{version}/items
```

The aggregated spec sets `servers: [{url: "http://{INFRA_HTTP_BASE_DOMAIN}"}]` so Swagger Try It Out sends requests through the gateway.

### Security fields in aggregated spec

Services declare auth requirements via `x-infra-*` extensions. The Registrator reads these and generates standard OpenAPI `security` fields in the aggregated spec so Swagger UI renders lock icons and scope requirements.

Input (service raw spec):
```yaml
post /items:
  x-infra-protected: true
  x-infra-scopes: ['write:items']
get /items:
  x-infra-protected: false
```

Output (aggregated spec):
```yaml
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT (Logto)
paths:
  /api/service-a/v1/items:
    post:
      security:
        - BearerAuth: ['write:items']
    get:
      security: []    # explicit empty overrides any global default
```

Services do not need `@ApiBearerAuth()` for the aggregated Swagger — the Registrator generates it. Adding it in the service is optional and only affects the service's own local Swagger at `/docs`.

## Candidate service contract

A microservice joins Infra by:

1. Connecting to the `infra` Docker network
2. Declaring Docker labels:

| Label | Required | Default | Description |
|---|---|---|---|
| `infra.enabled` | yes | — | must be `true` to be discovered |
| `infra.name` | yes | — | service identifier, used in gateway routes |
| `infra.port` | yes | — | internal container port the service listens on |
| `infra.openapi-route` | no | `openapi` | path where the OpenAPI spec is served |
| `infra.auth.protected` | no | `true` | default auth requirement for all endpoints |

3. Exposing a Docker healthcheck
4. Serving an OpenAPI 3.x spec at `/{openapi-route}`

## Open items

- **LocalAdapter (no-Docker dev mode):** `LocalAdapter` implementing `EnvironmentAdapter`, accepting REST registrations with heartbeat TTL. Deferred post-MVP.
- **KrakenD zero-downtime in Swarm:** two replicas + blue/green via Traefik. Out of scope for MVP; single replica with `start-first` is sufficient.
- **`depends_on` gap:** `logto-init` does not declare `depends_on: logto` — startup ordering relies on `restart: on-failure` retries. This is intentional for now but means init may attempt Logto's API before it is ready and will need several retries on a cold start.
