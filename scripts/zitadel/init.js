// Zitadel bootstrap (replaces scripts/logto/init.js).
//
// Reproducibly provisions the instance from zitadel.config.yaml on every
// `compose up`, idempotently (the container reruns each start). Mirrors the old
// logto-init contract: writes per-app configs and machine-user credentials to the
// shared /run/infra volume for the frontends and backend services to consume.
//
// Deferred to later phases (intentionally, see plan ZITADEL_MIGRATION.md):
//   - Pre-access-token Action flattening the object roles claim into a `roles`
//     array + `organization_id` — coupled to KrakenD claim consumption (Phase 4).
//   - HTTP email provider → mail-service — needs the not-yet-built mail-service,
//     and Zitadel's Executions.DenyList must allow that internal host (Phase 3+).

import { readFile, writeFile, mkdir, chmod } from 'node:fs/promises';
import { existsSync, readFileSync } from 'node:fs';
import yaml from 'js-yaml';

const API = process.env.ZITADEL_API_ENDPOINT ?? 'http://zitadel:8080';
const ISSUER = process.env.ZITADEL_ISSUER ?? API;
const PAT_FILE = process.env.ZITADEL_PAT_FILE ?? '/machinekey/admin-sa.pat';
const RUN_DIR = '/run/infra';

const PROTO = process.env.INFRA_HTTP_PROTOCOL ?? 'http';
const BASE = process.env.INFRA_HTTP_BASE_DOMAIN ?? 'app.localhost';
const ADMIN_SUB = process.env.INFRA_ADMIN_SUBDOMAIN ?? 'admin';

let PAT;

async function waitReady(timeoutMs = 120_000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(`${API}/.well-known/openid-configuration`);
      if (r.ok) return;
    } catch {
      // not up yet
    }
    await new Promise((res) => setTimeout(res, 3000));
  }
  throw new Error('zitadel not ready within timeout');
}

// api performs an authenticated Management/Admin API call. orgId, when set, scopes
// the call to that organization via the x-zitadel-orgid header. okStatuses lets a
// caller treat an expected error (e.g. 409 already-exists) as success.
async function api(method, path, body, { orgId, okStatuses = [] } = {}) {
  const headers = { authorization: `Bearer ${PAT}` };
  if (body !== undefined) headers['content-type'] = 'application/json';
  if (orgId) headers['x-zitadel-orgid'] = orgId;
  const r = await fetch(`${API}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  const text = await r.text();
  if (!r.ok && !okStatuses.includes(r.status)) {
    throw new Error(`${method} ${path} -> ${r.status}: ${text}`);
  }
  return { status: r.status, data: text ? JSON.parse(text) : {} };
}

// resolve ${VAR} from env in the config file (mirrors logto.config.yaml handling).
function loadConfig() {
  const txt = readFileSync(`${process.cwd()}/zitadel.config.yaml`, 'utf8');
  return yaml.load(txt.replace(/\$\{(\w+)\}/g, (_, k) => process.env[k] ?? ''));
}

function originsFor(app) {
  // Symbolic origins ("admin") expand to <proto>://<sub>.<base>; full URLs pass through.
  return app.origins.map((o) => (o.includes('://') ? o : `${PROTO}://${o}.${BASE}`));
}

async function ensureProject(orgId, name) {
  const { data } = await api('POST', '/management/v1/projects/_search', {
    queries: [{ nameQuery: { name, method: 'TEXT_QUERY_METHOD_EQUALS' } }],
  }, { orgId });
  let id = (data.result ?? [])[0]?.id;
  if (!id) {
    const created = await api('POST', '/management/v1/projects', { name }, { orgId });
    id = created.data.id;
    console.log(`Created project "${name}" (${id}).`);
  } else {
    console.log(`Project "${name}" exists (${id}).`);
  }
  // Assert project roles into tokens (idempotent PUT).
  await api('PUT', `/management/v1/projects/${id}`, {
    name,
    projectRoleAssertion: true,
    projectRoleCheck: false,
    hasProjectCheck: false,
    privateLabelingSetting: 'PRIVATE_LABELING_SETTING_UNSPECIFIED',
  }, { orgId });
  return id;
}

async function ensureRoles(orgId, projectId, roles) {
  const { data } = await api('POST', `/management/v1/projects/${projectId}/roles/_search`, {}, { orgId });
  const existing = new Set((data.result ?? []).map((r) => r.key));
  for (const role of roles) {
    if (existing.has(role.key)) {
      console.log(`  Role exists: ${role.key}`);
      continue;
    }
    await api('POST', `/management/v1/projects/${projectId}/roles`, {
      roleKey: role.key, displayName: role.displayName, group: role.group ?? '',
    }, { orgId });
    console.log(`  Created role: ${role.key}`);
  }
}

async function ensureApp(orgId, projectId, app) {
  const origins = originsFor(app);
  const redirectUris = origins.map((o) => `${o}/callback`);
  const oidc = {
    name: app.name,
    redirectUris,
    postLogoutRedirectUris: origins,
    additionalOrigins: origins,
    responseTypes: ['OIDC_RESPONSE_TYPE_CODE'],
    grantTypes: ['OIDC_GRANT_TYPE_AUTHORIZATION_CODE'],
    appType: 'OIDC_APP_TYPE_USER_AGENT',
    authMethodType: 'OIDC_AUTH_METHOD_TYPE_NONE',
    accessTokenType: 'OIDC_TOKEN_TYPE_JWT',
    // Assert project roles into the ACCESS token (not just id_token/userinfo) — the
    // gateway validates the access token, so without this it carries no roles.
    accessTokenRoleAssertion: true,
    idTokenRoleAssertion: true,
    devMode: PROTO === 'http',
  };

  const { data } = await api('POST', `/management/v1/projects/${projectId}/apps/_search`, {}, { orgId });
  const found = (data.result ?? []).find((a) => a.name === app.name);

  let appId, clientId;
  if (found) {
    appId = found.id;
    const detail = await api('GET', `/management/v1/projects/${projectId}/apps/${appId}`, undefined, { orgId });
    const cur = detail.data.app?.oidcConfig ?? {};
    clientId = cur.clientId;
    // Keep base/local URIs in sync (per-org URIs are appended elsewhere at runtime,
    // so merge rather than clobber). Only PUT when something actually changed —
    // Zitadel rejects a no-op update with 400 "No changes".
    const merge = (a = [], b = []) => [...new Set([...a, ...b])];
    const next = {
      redirectUris: merge(cur.redirectUris, redirectUris),
      postLogoutRedirectUris: merge(cur.postLogoutRedirectUris, origins),
      additionalOrigins: merge(cur.additionalOrigins, origins),
    };
    const changed = (a = [], b = []) => a.length !== b.length;
    if (changed(cur.redirectUris, next.redirectUris) ||
        changed(cur.postLogoutRedirectUris, next.postLogoutRedirectUris) ||
        changed(cur.additionalOrigins, next.additionalOrigins) ||
        cur.accessTokenRoleAssertion !== true) {
      await api('PUT', `/management/v1/projects/${projectId}/apps/${appId}/oidc_config`, { ...oidc, ...next }, { orgId });
      console.log(`App "${app.name}" exists (${appId}), URIs reconciled.`);
    } else {
      console.log(`App "${app.name}" exists (${appId}), URIs up to date.`);
    }
  } else {
    const created = await api('POST', `/management/v1/projects/${projectId}/apps/oidc`, oidc, { orgId });
    appId = created.data.appId;
    clientId = created.data.clientId;
    console.log(`Created app "${app.name}" (${appId}).`);
  }

  await writeRun(`${app.key}-app.json`, { issuer: ISSUER, orgId, projectId, appId, clientId }, 0o644);
}

async function ensureMachine(orgId, m) {
  const file = `${RUN_DIR}/${m.file}`;
  const { data } = await api('POST', '/management/v1/users/_search', {
    queries: [{ userNameQuery: { userName: m.userName, method: 'TEXT_QUERY_METHOD_EQUALS' } }],
  }, { orgId });
  let userId = (data.result ?? [])[0]?.userId ?? (data.result ?? [])[0]?.id;

  if (userId && existsSync(file)) {
    console.log(`Machine "${m.userName}" exists (${userId}) and creds present — skip.`);
    return;
  }
  if (!userId) {
    const created = await api('POST', '/management/v1/users/machine', {
      userName: m.userName, name: m.name, description: 'infra service account',
      accessTokenType: 'ACCESS_TOKEN_TYPE_JWT',
    }, { orgId });
    userId = created.data.userId;
    console.log(`Created machine "${m.userName}" (${userId}).`);
  }

  // Grant instance-level management access. TODO(least-privilege): scope down from
  // IAM_OWNER to the minimum each service needs (registrator reads; service-core writes).
  await api('POST', '/admin/v1/members', { userId, roles: ['IAM_OWNER'] }, { okStatuses: [409] });

  // Personal access token — returned once, so (re)write the creds file now.
  const pat = await api('POST', `/management/v1/users/${userId}/pats`, {
    expirationDate: '2030-01-01T00:00:00Z',
  }, { orgId });
  await writeRun(m.file, {
    token: pat.data.token, userId, apiEndpoint: API, issuer: ISSUER,
  }, 0o600);
  console.log(`Wrote ${m.file} (PAT for ${m.userName}).`);
}

async function writeRun(name, obj, mode) {
  await writeRunRaw(name, JSON.stringify(obj), mode);
}

async function writeRunRaw(name, content, mode) {
  await mkdir(RUN_DIR, { recursive: true, mode: 0o711 });
  const path = `${RUN_DIR}/${name}`;
  await writeFile(path, content, { mode });
  await chmod(path, mode); // mode only applies on create; enforce on rerun
}

// FLATTEN_SCRIPT runs in Zitadel's "Complement Token" flow (pre-userinfo and
// pre-access-token). The raw project-roles claim Zitadel emits is an object
// (urn:zitadel:iam:org:project:<id>:roles = {role:{orgId:domain}}) which KrakenD
// can't read as a role array, so we flatten it to a `roles` string array and lift
// the org id into `organization_id` — the flat claims the gateway propagates as
// x-user-roles / x-organization-id. (Runs for interactive user tokens; the grants
// context is not populated for client_credentials machine tokens.)
// The function name MUST match the action name ("flattenRoles") — Zitadel invokes
// the function whose name equals the action's name. The context API (v4.x):
// ctx.v1.user.grants = { count, grants: [ { roles: [..], projectId, userResourceOwner } ] }.
// ctx.v1.org is empty in this flow, so the org id comes from the grant's
// userResourceOwner (the user's home org).
const FLATTEN_SCRIPT = `function flattenRoles(ctx, api) {
  var roles = [];
  var orgId = "";
  var ug = ctx.v1.user.grants;
  if (ug && ug.grants) {
    for (var i = 0; i < ug.grants.length; i++) {
      var g = ug.grants[i];
      if (g.roles) { for (var j = 0; j < g.roles.length; j++) { if (roles.indexOf(g.roles[j]) < 0) { roles.push(g.roles[j]); } } }
      if (g.userResourceOwner) { orgId = g.userResourceOwner; }
    }
  }
  api.v1.claims.setClaim("roles", roles);
  if (orgId) { api.v1.claims.setClaim("organization_id", orgId); }
}`;

// ensureFlattenAction creates the role-flattening action and wires it to the
// Complement Token flow (trigger 4 = pre-userinfo, 5 = pre-access-token). Idempotent.
async function ensureFlattenAction(orgId) {
  const name = 'flattenRoles';
  const body = { name, script: FLATTEN_SCRIPT, timeout: '10s', allowedToFail: true };
  const { data } = await api('POST', '/management/v1/actions/_search', {}, { orgId });
  let actionId = (data.result ?? []).find((a) => a.name === name)?.id;
  if (!actionId) {
    const created = await api('POST', '/management/v1/actions', body, { orgId });
    actionId = created.data.id;
    console.log(`Created action ${name} (${actionId}).`);
  } else {
    // Update the script in case it changed; tolerate the no-op 400.
    try {
      await api('PUT', `/management/v1/actions/${actionId}`, body, { orgId });
      console.log(`Action ${name} (${actionId}) updated.`);
    } catch (e) {
      if (!String(e.message).includes('No Changes')) throw e;
      console.log(`Action ${name} (${actionId}) unchanged.`);
    }
  }
  // Complement Token flow = type 2; triggers 4 (pre-userinfo) + 5 (pre-access-token).
  // Re-setting an unchanged trigger returns 400 "No Changes" — tolerate it (idempotent).
  for (const trigger of [4, 5]) {
    try {
      await api('POST', `/management/v1/flows/2/trigger/${trigger}`, { actionIds: [actionId] }, { orgId });
    } catch (e) {
      if (!String(e.message).includes('No Changes')) throw e;
    }
  }
  console.log('Flatten action wired to Complement Token triggers.');
}

// ensureAdminRole grants the platform admin user the `admin` project role so its
// tokens carry the admin role (needed to use the admin API/console). The admin user
// lives in the platform org alongside the project, so a direct user grant suffices
// (no project grant). Idempotent.
async function ensureAdminRole(orgId, projectId) {
  const username = process.env.INFRA_ZITADEL_ADMIN_USERNAME ?? 'admin';
  const { data } = await api('POST', '/management/v1/users/_search', {
    queries: [{ typeQuery: { type: 'TYPE_HUMAN' } }],
  }, { orgId });
  const admin = (data.result ?? []).find((u) =>
    (u.userName ?? '').startsWith(username + '@') || u.userName === username ||
    (u.loginNames ?? []).some((l) => l.startsWith(username + '@')));
  if (!admin) {
    console.log(`Admin user "${username}" not found in platform org — skip role grant.`);
    return;
  }
  const userId = admin.userId ?? admin.id;
  try {
    await api('POST', `/management/v1/users/${userId}/grants`, {
      projectId, roleKeys: ['admin'],
    }, { orgId });
    console.log(`Granted admin role to ${userId}.`);
  } catch (e) {
    if (String(e.message).includes('already') || String(e.message).includes('Already')) {
      console.log('Admin role already granted.');
    } else {
      throw e;
    }
  }
}

// ensureLoginClient provisions the service user the Login UI v2 container runs as.
// It needs the instance-level IAM_LOGIN_CLIENT role and a PAT; the login container
// reads the raw token from /run/infra/login-client.pat.
async function ensureLoginClient(orgId) {
  const userName = 'login-client';
  const file = `${RUN_DIR}/login-client.pat`;
  const { data } = await api('POST', '/management/v1/users/_search', {
    queries: [{ userNameQuery: { userName, method: 'TEXT_QUERY_METHOD_EQUALS' } }],
  }, { orgId });
  let userId = (data.result ?? [])[0]?.userId ?? (data.result ?? [])[0]?.id;

  if (userId && existsSync(file)) {
    console.log(`Login client exists (${userId}) and token present — skip.`);
    return;
  }
  if (!userId) {
    const created = await api('POST', '/management/v1/users/machine', {
      userName, name: 'Login Client', description: 'Zitadel Login UI v2 service user',
      accessTokenType: 'ACCESS_TOKEN_TYPE_JWT',
    }, { orgId });
    userId = created.data.userId;
    console.log(`Created login client (${userId}).`);
  }
  await api('POST', '/admin/v1/members', { userId, roles: ['IAM_LOGIN_CLIENT'] }, { okStatuses: [409] });
  const pat = await api('POST', `/management/v1/users/${userId}/pats`, {
    expirationDate: '2030-01-01T00:00:00Z',
  }, { orgId });
  // 0644: the zitadel-login container runs as a non-root user and must read this.
  // Dev-only tradeoff — in prod deliver this PAT via a Docker secret instead.
  await writeRunRaw('login-client.pat', pat.data.token, 0o644);
  console.log('Wrote login-client.pat (PAT for Login UI v2).');
}

async function main() {
  console.log(`zitadel-init: waiting for ${API} ...`);
  await waitReady();

  PAT = (await readFile(PAT_FILE, 'utf8')).trim();
  if (!PAT) throw new Error(`empty PAT at ${PAT_FILE}`);

  const cfg = loadConfig();
  const me = await api('GET', '/management/v1/orgs/me');
  const orgId = me.data.org.id;
  console.log(`Default (platform) org: ${orgId} (${me.data.org.name}).`);

  const projectId = await ensureProject(orgId, cfg.project.name);
  await ensureRoles(orgId, projectId, cfg.project.roles ?? []);
  for (const app of cfg.apps ?? []) await ensureApp(orgId, projectId, app);
  for (const m of cfg.machines ?? []) await ensureMachine(orgId, m);
  await ensureFlattenAction(orgId);
  await ensureAdminRole(orgId, projectId);
  await ensureLoginClient(orgId);

  // Record the platform project id for backend services (registrator needs it to
  // build the deterministic roles claim key urn:zitadel:iam:org:project:<id>:roles).
  await writeRun('zitadel-platform.json', { issuer: ISSUER, projectId, orgId }, 0o644);

  console.log('zitadel-init: done.');
}

main().catch((err) => {
  console.error('zitadel-init failed:', err.message);
  process.exit(1);
});
