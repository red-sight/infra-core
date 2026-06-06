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
  const data = text ? JSON.parse(text) : {};
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

// Configure Custom JWT access token claims from config.
async function applyJWT(token, jwtConfig) {
  if (!jwtConfig?.access_token?.include_roles) return;

  const script = `const getCustomJwtClaims = async ({ token, context, environmentVariables, api }) => {
  return {
    roles: (context.user?.roles ?? []).map(r => typeof r === "string" ? r : r.name)
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
  await applyJWT(defaultToken, config.jwt);

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

  // Write Registrator M2M credentials to shared volume
  fs.mkdirSync('/run/infra', { recursive: true });
  fs.writeFileSync('/run/infra/registrator-m2m.json', JSON.stringify({
    clientId: 'm-default',
    clientSecret: defaultSecret,
    tokenEndpoint: LOGTO_ADMIN_ENDPOINT,
    apiEndpoint: LOGTO_ENDPOINT,
  }));
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
