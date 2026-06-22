# Registrator

The Go service that wires Infra together. It watches Docker for healthy API services, generates KrakenD configuration, and aggregates OpenAPI specs into a unified Swagger UI.

## Internal packages

| Package | Responsibility |
|---|---|
| `registrar` | Service discovery, registry state |
| `gateway` | KrakenD config generation and reload |
| `roles` | scope→role mapping policy (built-in default, env-overridable) |
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
| `INFRA_ZITADEL_PLATFORM_FILE` | `/run/infra/zitadel-platform.json` | descriptor written by zitadel-init; the Registrator reads the project id from it for the JWT audience |
| `INFRA_GATEWAY_ROLE_SCOPES` | built-in | optional JSON `{role:[scope,…]}` overriding the default scope→role policy |
| `INFRA_REGISTRATOR_RELOAD_DELAY` | `5s` | debounce delay before KrakenD reload |
| `INFRA_REGISTRATOR_RECONCILE_INTERVAL` | `30s` | reconcile loop interval — re-scans all healthy labeled containers and re-registers any that were falsely removed due to stale Docker events |
| `INFRA_KRAKEND_CONFIG_PATH` | `/etc/krakend/krakend.json` | path to the KrakenD config file (inside the registrator container) |
| `INFRA_REGISTRATOR_VALIDATE` | `true` | run `krakend check` on each candidate config before promoting it; set `false` to skip validation |
| `INFRA_KRAKEND_IMAGE` | `krakend:latest` | image used for the one-shot `krakend-check` validation container — pin to match the running KrakenD in prod |
| `INFRA_KRAKEND_SERVICE` | `infra_krakend` | (artifact mode) name of the Swarm KrakenD service to roll onto a new config object |

## CLI flags

By default the registrator runs as a long-lived watcher. Two flags switch it to a single synchronous pass that exits:

| Flag | Behavior |
|---|---|
| `-once` | Scan once, generate + validate + deliver a single time, then exit. The prod path: a CI job runs this after the deploy converges. |
| `-dry-run` | Scan once, render the KrakenD config to **stdout**, and exit — no write, no validation, no apply. Logs go to stderr, so `registrator -dry-run > krakend.json` captures exactly what would be generated. Use it to preview a change locally before opening a PR. |

Neither flag starts the event watcher, poll loop, or HTTP server.

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

## Reload modes

A pass runs on each debounce fire (after `INFRA_REGISTRATOR_RELOAD_DELAY`), on the poll tick, and once per `-once` invocation. The two modes share spec fetching and scope→role resolution but diverge entirely at delivery, so each is its own runner (passes are serialized by a mutex).

**`auto` (dev):**
1. Fetch each service's OpenAPI spec; build the scope→role map from the `roles` policy.
2. Generate the new `krakend.json` and, unless `INFRA_REGISTRATOR_VALIDATE=false`, validate it with `krakend check` in a one-shot `krakend-check` container before promoting it: the candidate is written to a temp file in the config dir, validated there, and atomically renamed into place only if valid (and only if the render changed). An invalid candidate is discarded — the live config and KrakenD are untouched. The `krakend-check` container is removed on success, left for inspection on failure.
3. Restart the local KrakenD container so it re-reads the config. Cheap with no prod traffic.

**`artifact` (prod):** the registrator runs as a one-shot deploy job (`-once`) and never writes a local file or restarts a container.
1. Render the config bytes and a per-service digest (`info.version` + a contract hash over paths/schemas).
2. **Version gate:** read the last-delivered `{service: {version, hash}}` manifest from the `infra.manifest` label of the config object the KrakenD service currently mounts. If any service's contract hash changed without its `info.version` increasing, the delivery is **rejected** (bump your version). New services are not gated.
3. **Idempotency:** name the config object `krakend-config-<render-hash>`. If one with that name already exists, the render is unchanged — nothing to do.
4. Otherwise create the immutable Swarm config object (labelled `infra.managed=true` + the new manifest) and `UpdateServiceConfig` the KrakenD service onto it (`ForceUpdate`), so Swarm rolls it out with `order:start-first`.

> KrakenD config *validity* (`krakend check`) is enforced in `auto` mode and is expected to run as a CI step in `artifact` mode (`registrator -dry-run | krakend check`) before `-once`, since validating bytes inside a one-shot job has no shared config mount. The artifact runner enforces the version policy and idempotent delivery.

## RBAC — scope→role resolution

Before generating each KrakenD config, the Registrator builds a scope→role map from the `roles` package. Unlike the former Logto setup (which fetched scopes-on-roles from the IdP), Zitadel project roles carry no scopes, so "which roles satisfy which scope" is gateway authorization policy: a built-in default (mirroring the project roles), overridable with `INFRA_GATEWAY_ROLE_SCOPES` (JSON `{role:[scope,…]}`).

When an OpenAPI operation declares `x-infra-scopes: [read:items, write:items]`, the Registrator resolves which roles hold those scopes and writes the resulting role list into the KrakenD `auth/validator` `roles` field. KrakenD then validates the flat `roles` claim in the JWT against that list (the flat claim is produced by a Zitadel Action — see AGENTS.md).

This means:
- API specs declare intent (required scopes).
- Role *membership* lives in Zitadel (project roles + grants); the scope→role *policy* lives in the gateway (`roles` package).
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
| `alg` | `RS256` (Zitadel signs with RS256) |
| `jwk_url` | `{INFRA_HTTP_PROTOCOL}://{INFRA_HTTP_OIDC_SUBDOMAIN}.{INFRA_HTTP_BASE_DOMAIN}/oauth/v2/keys` |
| `disable_jwk_security` | `true` if `INFRA_HTTP_PROTOCOL=http`, else `false` |
| `audience` | the Zitadel project id (read from `/run/infra/zitadel-platform.json`) |

### OpenAPI extension labels

Declared per-operation in the service's OpenAPI spec:

| Extension | Type | Default | Description |
|---|---|---|---|
| `x-infra-protected` | `bool` | inherits `infra.auth.protected` Docker label | `false` = public endpoint, no `auth/validator` block generated |
| `x-infra-scopes` | `string[]` | — | required scopes; Registrator resolves these to roles via the `roles` policy and sets them in KrakenD `roles` |
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
      bearerFormat: JWT (Zitadel)
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
- **Startup ordering:** `zitadel-init` polls Zitadel readiness in-script and uses `restart: on-failure`; the frontends and `zitadel-login` wait on `zitadel-init` completing.
