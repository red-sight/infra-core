# RFC: Cookie-based auth via a BFF (kill token-in-localStorage)

Status: Draft · Branch: `zitadel` · Date: 2026-06-22

## Objective

Stop storing OIDC tokens in the browser. Today `admin` and `tenant-web` keep the full
`User` object (access + id + refresh token) in `localStorage` via `oidc-client-ts`
(`WebStorageStateStore`) — readable by any XSS. Move to a **Backend-for-Frontend (BFF)**:
tokens live server-side, the browser gets only an opaque, `httpOnly` session cookie. The
frontend never sees a token.

## Context

Current flow (`tenant-web/src/App.vue`, `admin/src/composables/useAuth.ts`,
`admin/src/lib/api.ts`):

```
SPA (<slug>.<base>) ──[Bearer from localStorage]──▶ Traefik ──▶ KrakenD (validates JWT) ──▶ services
```

- KrakenD is the trust boundary. JWT validator is **generated** by the registrator
  (`registrator/internal/gateway/gateway.go` ~L282–314): RS256, JWKS at
  `…/oauth/v2/keys`, `audience` = Zitadel project id, propagates `sub→x-user-id`,
  `roles→x-user-roles`, `organization_id→x-organization-id`. Go services trust those
  headers and do **no** JWT parsing themselves.
- Zitadel is one instance / one issuer → **JWKS + issuer are instance-level**. Per-org
  OIDC apps live under the same `Infra API` project → `audience` stays constant. **The
  KrakenD validator does not change regardless of how many per-org clients exist.**
- Per-org isolation is a hard requirement: each org gets **its own OIDC client**
  (own creds/domain/branding). This is why an off-the-shelf static proxy
  (oauth2-proxy / a Traefik OIDC plugin) was rejected — none can select the OIDC
  client by `Host` dynamically for orgs provisioned at runtime.
- Redis is already in the stack but **unused** — it becomes the session store.
- This **supersedes** the `ZITADEL_MIGRATION.md` decision "*Token in Authorization
  header, not cookies → gateway `allow_credentials:false`*". With a same-origin BFF the
  browser↔BFF leg is cookie-based; the BFF↔KrakenD leg stays Bearer.

Target flow:

```
SPA (<slug>.<base>) ──[httpOnly cookie]──▶ Traefik ──▶ auth-bff ──[Bearer]──▶ KrakenD (validates JWT) ──▶ services
                                                            │
                                                            └─ session in Redis (sess:<id> → tokens)
```

## Decisions already made

- **Session store: Redis.** Opaque 256-bit session id in the cookie; tokens in Redis.
  Revocable, small cookie, clean refresh-token rotation. (Rejected: stateless encrypted
  cookie — not revocable, grows, breaks rotation idempotency.)
- **Keep JWT validation in KrakenD.** The BFF only converts cookie→Bearer and reverse-
  proxies `/api/*`. Minimal blast radius; registrator/KrakenD config untouched.
- **Per-org OIDC apps become confidential** `OIDC_APP_TYPE_WEB` + PKCE, auth method
  secret (today they are public `USER_AGENT`). The code-exchange now happens server-side,
  so a client secret is both possible and correct.
- **Cookie:** `__Host-session` — `httpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, **no
  `Domain`** → host-only. Per-org isolation for free: a cookie on `acme.<base>` is never
  sent to `beta.<base>`. (`SameSite=Lax` is required so the top-level callback redirect
  carries it; in local `http` dev drop `Secure`/`__Host-` prefix.)
- **Rollout: `tenant-web` first** (pilots per-org isolation), then `admin`.

## Approach (phased — each phase independently verifiable)

### Phase 0 — Zitadel app shape + spike
- Change the **default tenant app** in `scripts/zitadel/zitadel.config.yaml` /
  `scripts/zitadel/init.js` from `OIDC_APP_TYPE_USER_AGENT` to `OIDC_APP_TYPE_WEB`
  (auth method secret + PKCE). Capture the returned `client_id` + `client_secret`.
- Confirm by hand: auth-code+PKCE exchange against the per-org client, refresh-token
  grant (scope `offline_access`), and `end_session` endpoint shape.
- Confirm the issued **access token** carries `audience` = project id and the flattened
  `roles` / `organization_id` claims (the existing Zitadel Action) so KrakenD is happy.
- Output: pin the exact token-endpoint + secret-delivery shape for Phase 1/4.

### Phase 1 — `auth-bff` service skeleton + infra
- New Go service `auth-bff/` (Chi; reuse the service-core layout). Config strictly via
  env (issuer, base domain, Redis URL, KrakenD upstream, cookie name/flags, per-org
  client lookup source). **No secrets in code/logs.**
- Redis session client: `sess:<id>` → `{sub, org_id, roles, access_token, refresh_token,
  id_token, access_exp, refresh_exp}`; TTL = refresh lifetime, sliding.
- Compose wiring across the three files (`docker-compose.yml`,
  `docker-compose.override.yml`, `docker-compose.swarm.yml`): add `auth-bff`, depends_on
  Redis + Zitadel.
- **Traefik route:** `HostRegexp(^[a-z0-9-]+\.<base>$) && PathPrefix(/api)` → `auth-bff`
  instead of KrakenD. The BFF forwards non-auth `/api/*` to KrakenD internally.
- Verify: health endpoint reachable through Traefik; Redis round-trip.

### Phase 2 — OIDC endpoints on the BFF
Org resolved by `Host` on every request (reuse the existing `GET /tenant/by-host`
resolution path), then its `client_id/secret` looked up (Phase 4 source).
- `GET /api/auth/login` — generate PKCE verifier + `state` + `nonce` (stash in Redis,
  short TTL), 302 to the org client's Zitadel authorize URL with the org scope.
- `GET /api/auth/callback` — validate `state`, exchange `code`+verifier for tokens,
  create the Redis session, set the `__Host-session` cookie, 302 back into the SPA.
- `GET /api/auth/me` — return `{sub, org_id, roles, …}` from the id-token claims for UI
  rendering. **Never returns a token.**
- `POST /api/auth/logout` — delete the Redis session, clear the cookie, optional Zitadel
  `end_session`.
- Verify: full login→cookie→me→logout cycle with curl against one org.

### Phase 3 — reverse-proxy `/api/*` with token injection
- For every non-auth `/api/*`: load session by cookie → if `access_exp` passed, refresh
  via the org client's refresh-token grant (rotate + persist) → inject
  `Authorization: Bearer <access_token>` → proxy to KrakenD. No cookie → 401.
- Concurrency-safe refresh (per-session lock in Redis) so parallel requests don't double-
  rotate. Tie into the reload-race concerns already noted in `ARCHITECTURE_REVIEW.md`.
- Verify: an authenticated `/api/*` call succeeds end-to-end; a forced-expired access
  token transparently refreshes; revoking the Redis session → 401.

### Phase 4 — service-core per-org client provisioning
- When an org is created (the existing Zitadel `OrgProvisioner` / outbox path in
  service-core), additionally **create a confidential `WEB` OIDC app** for the org:
  redirect URI `https://<slug>.<base>/api/auth/callback`, the org's origins, PKCE.
- Persist `client_id` + `client_secret` where the BFF reads them — **secrets manager /
  encrypted column, env-injected**, never plaintext config, never committed. Define the
  lookup contract the BFF uses in Phase 2.
- `EnsureMaster` / default-org seed creates the same confidential app for the base domain.
- Verify: provisioning a fresh org yields a working login on its subdomain end-to-end.

### Phase 5 — frontends (tenant-web, then admin)
- Remove `oidc-client-ts` + the `localStorage` `WebStorageStateStore`. No tokens client-
  side at all.
- Auth state from `GET /api/auth/me`; login = redirect to `/api/auth/login`; logout =
  `POST /api/auth/logout`. API calls go to same-origin `/api/*` with
  `credentials: 'include'`; drop the manual `Authorization` header in
  `admin/src/lib/api.ts` (and the tenant-web equivalent).
- Verify: clean login/refresh/logout in the browser (user-checked, no Playwright).

### Phase 6 — CORS / credentials / hardening
- Same-origin `/api` per subdomain → CORS largely moot, but set
  `allow_credentials:true` and exact `allow_origins` (no `*`) wherever credentials cross.
- CSRF: `SameSite=Lax` + `state` covers the OIDC leg; add a double-submit/Origin check on
  state-changing `/api/*` if any non-`SameSite`-safe method is exposed.
- Cookie/security audit: `__Host-` prefix in prod, `Secure`, no token logging, Redis
  session TTL + idle expiry, refresh-token rotation on every use.

### Phase 7 — docs & cleanup
- Update `AGENTS.md`, `README.md`, `ZITADEL_MIGRATION.md` (flip the
  Authorization-vs-cookie decision), and `ARCHITECTURE_REVIEW.md` follow-ups.

## Trade-offs

- **+** No tokens in the browser → XSS can't exfiltrate them; sessions are revocable;
  per-org cookie isolation is structural (host-only). **−** A new stateful service on the
  hot path (Redis dependency, an extra hop) and per-org client-secret provisioning &
  storage to get right.
- **Custom BFF over oauth2-proxy/Traefik plugin:** required for dynamic per-`Host` client
  selection; cost is ~one small Go service we own and must maintain.
- **Keep KrakenD validation vs fold into BFF:** keeping it means two hops but zero change
  to the trust boundary and the registrator/KrakenD generator.

## Open questions

- Secret storage backend for per-org `client_secret` (Docker secret vs Vault/SM vs
  encrypted Postgres column) — decide in Phase 4.
- One BFF instance for all orgs (dynamic per-`Host` lookup) — confirmed direction; revisit
  only if per-org isolation later needs process separation.
- `admin` (single platform app) — confidential `WEB` like the tenant apps, or leave until
  after the tenant-web pilot? (Plan assumes after.)
- Session lifetime / idle-timeout policy and whether to support "remember me".
- Logout breadth: local session only vs Zitadel `end_session` (single logout) — UX call.
