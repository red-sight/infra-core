#!/usr/bin/env node
// Headless Logto initialization. Runs once on first start, idempotent on repeat runs.
// Required env vars: LOGTO_ADMIN_ENDPOINT, LOGTO_ENDPOINT, DB_URL,
//                    ADMIN_USERNAME, ADMIN_PASSWORD, API_RESOURCE_INDICATOR

import pg from 'pg';
import fs from 'fs';
import yaml from 'js-yaml';

const {
  LOGTO_ADMIN_ENDPOINT = 'http://logto:3002',
  LOGTO_ENDPOINT       = 'http://logto:3001',
  DB_URL,
  ADMIN_USERNAME,
  ADMIN_PASSWORD,
  API_RESOURCE_INDICATOR,
} = process.env;

if (!DB_URL || !ADMIN_USERNAME || !ADMIN_PASSWORD || !API_RESOURCE_INDICATOR) {
  console.error('Missing required env vars: DB_URL, ADMIN_USERNAME, ADMIN_PASSWORD, API_RESOURCE_INDICATOR');
  process.exit(1);
}

// Load declarative config; substitute ${VAR} from environment
const rawConfig = fs.readFileSync('/app/logto.config.yaml', 'utf8')
  .replace(/\$\{([^}]+)\}/g, (_, k) => process.env[k] ?? '');
const config = yaml.load(rawConfig);

const db = new pg.Client({ connectionString: DB_URL });

async function getAppSecret(appId) {
  const { rows } = await db.query('SELECT secret FROM applications WHERE id = $1', [appId]);
  if (!rows.length) throw new Error(`Application ${appId} not found`);
  return rows[0].secret;
}

async function getToken(appId, secret, resource) {
  const endpoint = appId === 'm-admin' || appId === 'm-default'
    ? `${LOGTO_ADMIN_ENDPOINT}/oidc/token`
    : `${LOGTO_ENDPOINT}/oidc/token`;

  const res = await fetch(endpoint, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Authorization': 'Basic ' + Buffer.from(`${appId}:${secret}`).toString('base64'),
    },
    body: new URLSearchParams({ grant_type: 'client_credentials', resource, scope: 'all' }),
  });
  const data = await res.json();
  if (!data.access_token) throw new Error(`Token error: ${JSON.stringify(data)}`);
  return data.access_token;
}

async function api(baseUrl, token, method, path, body) {
  const res = await fetch(`${baseUrl}/api${path}`, {
    method,
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const text = await res.text();
  let data = {};
  try {
    data = text ? JSON.parse(text) : {};
  } catch {
    // non-JSON response — leave data as {}
  }
  // 404 = not found (ok for GET checks), 422 = already exists / already assigned (idempotent)
  if (!res.ok && res.status !== 404 && res.status !== 422) {
    throw new Error(`${method} ${path} → ${res.status}: ${text}`);
  }
  return { status: res.status, data };
}

// Create resources and scopes declared in config.
// Returns scopeIndex: map of scope name → scope ID (across all resources).
async function applyResources(token, resources) {
  const { data: existing } = await api(LOGTO_ENDPOINT, token, 'GET', '/resources?page_size=50');
  const scopeIndex = {};

  for (const res of resources ?? []) {
    let resource = Array.isArray(existing) && existing.find(r => r.indicator === res.indicator);
    if (!resource) {
      const { data } = await api(LOGTO_ENDPOINT, token, 'POST', '/resources', {
        name: res.name,
        indicator: res.indicator,
      });
      resource = data;
      console.log(`Created resource: ${res.name}`);
    } else {
      console.log(`Resource exists: ${res.name}`);
    }

    const { data: existingScopes } = await api(LOGTO_ENDPOINT, token, 'GET', `/resources/${resource.id}/scopes?page_size=50`);
    for (const scope of res.scopes ?? []) {
      let s = Array.isArray(existingScopes) && existingScopes.find(e => e.name === scope.name);
      if (!s) {
        const { data } = await api(LOGTO_ENDPOINT, token, 'POST', `/resources/${resource.id}/scopes`, {
          name: scope.name,
          description: scope.description ?? '',
        });
        s = data;
        console.log(`  Created scope: ${scope.name}`);
      } else {
        console.log(`  Scope exists: ${scope.name}`);
      }
      scopeIndex[scope.name] = s.id;
    }
  }

  return scopeIndex;
}

// Create roles and assign their scopes declared in config.
async function applyRoles(token, roles, scopeIndex) {
  const { data: existingRoles } = await api(LOGTO_ENDPOINT, token, 'GET', '/roles?type=User&page_size=50');

  for (const role of roles ?? []) {
    let r = Array.isArray(existingRoles) && existingRoles.find(e => e.name === role.name);
    if (!r) {
      const { data } = await api(LOGTO_ENDPOINT, token, 'POST', '/roles', {
        name: role.name,
        description: role.description ?? '',
        type: 'User',
      });
      r = data;
      console.log(`Created role: ${role.name}`);
    } else {
      console.log(`Role exists: ${role.name}`);
    }

    if (!role.scopes?.length) continue;

    const { data: currentScopes } = await api(LOGTO_ENDPOINT, token, 'GET', `/roles/${r.id}/scopes?page_size=50`);
    const currentIds = new Set((Array.isArray(currentScopes) ? currentScopes : []).map(s => s.id));

    const toAssign = role.scopes
      .map(name => scopeIndex[name])
      .filter(id => id && !currentIds.has(id));

    if (toAssign.length) {
      await api(LOGTO_ENDPOINT, token, 'POST', `/roles/${r.id}/scopes`, { scopeIds: toAssign });
      const names = role.scopes.filter(n => toAssign.includes(scopeIndex[n]));
      console.log(`  Assigned scopes to ${role.name}: ${names.join(', ')}`);
    } else {
      console.log(`  Scopes up-to-date for ${role.name}`);
    }
  }
}

// Create organization roles and assign resource scopes.
async function applyOrganizationRoles(token, orgRoles, scopeIndex) {
  const { data: existing } = await api(LOGTO_ENDPOINT, token, 'GET', '/organization-roles?page_size=50');

  for (const role of orgRoles ?? []) {
    let r = Array.isArray(existing) && existing.find(e => e.name === role.name);
    if (!r) {
      const { data } = await api(LOGTO_ENDPOINT, token, 'POST', '/organization-roles', {
        name: role.name,
        description: role.description ?? '',
      });
      r = data;
      console.log(`Created org role: ${role.name}`);
    } else {
      console.log(`Org role exists: ${role.name}`);
    }

    if (!role.scopes?.length) continue;

    const { data: currentScopes } = await api(LOGTO_ENDPOINT, token, 'GET', `/organization-roles/${r.id}/resource-scopes?page_size=50`);
    const currentIds = new Set((Array.isArray(currentScopes) ? currentScopes : []).map(s => s.id));

    const toAssign = role.scopes
      .map(name => scopeIndex[name])
      .filter(id => id && !currentIds.has(id));

    if (toAssign.length) {
      await api(LOGTO_ENDPOINT, token, 'POST', `/organization-roles/${r.id}/resource-scopes`, { scopeIds: toAssign });
      const names = role.scopes.filter(n => toAssign.includes(scopeIndex[n]));
      console.log(`  Assigned resource scopes to org role ${role.name}: ${names.join(', ')}`);
    } else {
      console.log(`  Resource scopes up-to-date for org role ${role.name}`);
    }
  }
}

// Create SPA/Native applications declared in config.
// Returns a map of application name → application ID.
async function applyApplications(token, applications) {
  const { data: existing } = await api(LOGTO_ENDPOINT, token, 'GET', '/applications?page_size=50');
  const result = {};

  for (const app of applications ?? []) {
    const found = Array.isArray(existing) && existing.find(e => e.name === app.name && e.type === app.type);

    if (!found) {
      const { data } = await api(LOGTO_ENDPOINT, token, 'POST', '/applications', {
        name: app.name,
        type: app.type,
        oidcClientMetadata: {
          redirectUris: app.redirectUris ?? [],
          postLogoutRedirectUris: app.postLogoutRedirectUris ?? [],
        },
      });
      result[app.name] = data.id;
      console.log(`Created application: ${app.name} (${data.id})`);
    } else {
      await api(LOGTO_ENDPOINT, token, 'PATCH', `/applications/${found.id}`, {
        oidcClientMetadata: {
          redirectUris: app.redirectUris ?? [],
          postLogoutRedirectUris: app.postLogoutRedirectUris ?? [],
        },
      });
      result[app.name] = found.id;
      console.log(`Application exists: ${app.name} (${found.id})`);
    }
  }

  return result;
}

// Configure Custom JWT access token claims from config.
// Merges user roles and organization roles into a single flat "roles" array.
// Includes organization_id when the token is org-scoped.
async function applyJWT(token, jwtConfig) {
  if (!jwtConfig?.access_token?.include_roles) return;

  const script = `const getCustomJwtClaims = async ({ token, context, environmentVariables, api }) => {
  const userRoles = (context.user?.roles ?? []).map(r => typeof r === "string" ? r : r.name);

  const orgId = context.organization?.id ?? null;
  const orgRoles = orgId
    ? (context.user?.organizationRoles?.[orgId] ?? []).map(r => typeof r === "string" ? r : r.name)
    : [];

  return {
    roles: [...new Set([...userRoles, ...orgRoles])],
    ...(orgId ? { organization_id: orgId } : {}),
  };
};`;

  await api(LOGTO_ENDPOINT, token, 'PUT', '/configs/jwt-customizer/access-token', { script });
  console.log('Custom JWT claims configured.');
}

async function main() {
  await db.connect();

  const adminSecret   = await getAppSecret('m-admin');
  const defaultSecret = await getAppSecret('m-default');

  const adminToken   = await getToken('m-admin',   adminSecret,   'https://admin.logto.app/api');
  const defaultToken = await getToken('m-default', defaultSecret, 'https://default.logto.app/api');

  // Apply declarative config
  const scopeIndex = await applyResources(defaultToken, config.resources);
  await applyRoles(defaultToken, config.roles, scopeIndex);
  await applyOrganizationRoles(defaultToken, config.organization_roles, scopeIndex);
  await applyJWT(defaultToken, config.jwt);
  const appIds = await applyApplications(defaultToken, config.applications);

  // Write per-app configs to shared volume for runtime consumption
  const adminAppId = appIds['Admin'];
  if (adminAppId) {
    fs.writeFileSync('/run/infra/admin-app.json', JSON.stringify({
      appId: adminAppId,
      endpoint: LOGTO_ENDPOINT,
      apiResource: API_RESOURCE_INDICATOR,
    }));
    console.log(`Admin app config written (appId: ${adminAppId}).`);
  }

  // --- Admin user (default tenant — app users) ---
  const { data: defaultUsers } = await api(LOGTO_ENDPOINT, defaultToken, 'GET', '/users?page_size=50');
  const existingDefaultUser = Array.isArray(defaultUsers) && defaultUsers.find(u => u.username === ADMIN_USERNAME);

  let defaultUserId;
  if (existingDefaultUser) {
    defaultUserId = existingDefaultUser.id;
    console.log(`App admin user "${ADMIN_USERNAME}" already exists (${defaultUserId}), updating password.`);
  } else {
    const { data: newUser } = await api(LOGTO_ENDPOINT, defaultToken, 'POST', '/users', {
      username: ADMIN_USERNAME,
      password: ADMIN_PASSWORD,
      name: 'Admin',
    });
    defaultUserId = newUser.id;
    console.log(`Created app admin user "${ADMIN_USERNAME}" (${defaultUserId}).`);
  }

  await api(LOGTO_ENDPOINT, defaultToken, 'PATCH', `/users/${defaultUserId}/password`, { password: ADMIN_PASSWORD });
  console.log('App admin password set.');

  // Assign the admin role in the default tenant
  const { data: defaultRoles } = await api(LOGTO_ENDPOINT, defaultToken, 'GET', '/roles?type=User&page_size=50');
  const defaultAdminRole = Array.isArray(defaultRoles) && defaultRoles.find(r => r.name === 'admin');
  if (defaultAdminRole) {
    await api(LOGTO_ENDPOINT, defaultToken, 'POST', `/users/${defaultUserId}/roles`, { roleIds: [defaultAdminRole.id] });
    console.log('App admin role assigned (or already assigned).');
  }

  // --- Admin user (admin tenant) ---
  const { data: existingUsers } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/users?page_size=50');
  const existingUser = Array.isArray(existingUsers) && existingUsers.find(u => u.username === ADMIN_USERNAME);

  let userId;
  if (existingUser) {
    userId = existingUser.id;
    console.log(`Admin user "${ADMIN_USERNAME}" already exists (${userId}), updating password.`);
  } else {
    const { data: newUser } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'POST', '/users', {
      username: ADMIN_USERNAME,
      password: ADMIN_PASSWORD,
      name: 'Admin',
    });
    userId = newUser.id;
    console.log(`Created admin user "${ADMIN_USERNAME}" (${userId}).`);
  }

  await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'PATCH', `/users/${userId}/password`, { password: ADMIN_PASSWORD });
  console.log('Admin password set.');

  const { data: adminRoles } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/roles?type=User');
  const adminRole = Array.isArray(adminRoles) && adminRoles.find(r => r.name === 'default:admin');
  if (adminRole) {
    await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'POST', `/users/${userId}/roles`, { roleIds: [adminRole.id] });
    console.log('Admin role assigned (or already assigned).');
  }

  // Add admin user to t-default org
  const { data: members } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/organizations/t-default/users');
  const isMember = Array.isArray(members) && members.some(m => m.id === userId);
  if (!isMember) {
    await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'POST', '/organizations/t-default/users', { userIds: [userId] });
    await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'POST', `/organizations/t-default/users/${userId}/roles`, { organizationRoleIds: ['admin'] });
    console.log('Admin user added to t-default organization with admin role.');
  } else {
    console.log('Admin user already in t-default organization.');
  }

  // Write Registrator M2M credentials to shared volume.
  // The file holds a plaintext clientSecret, so restrict it to the owner (0600).
  // The directory is 0711 (traversable but not listable by others) so the non-root
  // admin container can read its public config (admin-app.json, 0644) by known name,
  // while the M2M secret stays owner-only. The registrator runs as root regardless.
  // For Swarm, prefer a Docker secret over this shared volume.
  fs.mkdirSync('/run/infra', { recursive: true, mode: 0o711 });
  fs.writeFileSync('/run/infra/registrator-m2m.json', JSON.stringify({
    clientId: 'm-default',
    clientSecret: defaultSecret,
    tokenEndpoint: LOGTO_ADMIN_ENDPOINT,
    apiEndpoint: LOGTO_ENDPOINT,
  }), { mode: 0o600 });
  // mode in mkdir/writeFile only applies on creation; logto-init reruns on every
  // compose up, so chmod explicitly to enforce perms on a pre-existing dir/file.
  fs.chmodSync('/run/infra', 0o711);
  fs.chmodSync('/run/infra/registrator-m2m.json', 0o600);
  console.log('Registrator M2M credentials written to /run/infra/registrator-m2m.json.');

  // Mark onboarding complete
  const { data: consoleCfg } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/configs/admin-console');
  if (!consoleCfg.signInExperienceCustomized || !consoleCfg.organizationCreated) {
    await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'PATCH', '/configs/admin-console', {
      signInExperienceCustomized: true,
      organizationCreated: true,
    });
    console.log('Onboarding marked complete.');
  } else {
    console.log('Onboarding already complete.');
  }

  console.log('Logto init done.');
  await db.end();
}

main().catch(async err => {
  console.error('Init failed:', err.message);
  await db.end().catch(() => {});
  process.exit(1);
});
