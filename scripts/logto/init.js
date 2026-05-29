#!/usr/bin/env node
// Headless Logto initialization. Runs once on first start, idempotent on repeat runs.
// Required env vars: LOGTO_ADMIN_ENDPOINT, LOGTO_ENDPOINT, DB_URL,
//                    ADMIN_USERNAME, ADMIN_PASSWORD, API_RESOURCE_INDICATOR

import pg from 'pg';

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
  if (!res.ok && res.status !== 404) throw new Error(`${method} ${path} → ${res.status}: ${text}`);
  return { status: res.status, data };
}

async function main() {
  await db.connect();

  const adminSecret   = await getAppSecret('m-admin');
  const defaultSecret = await getAppSecret('m-default');

  const adminToken   = await getToken('m-admin',   adminSecret,   'https://admin.logto.app/api');
  const defaultToken = await getToken('m-default', defaultSecret, 'https://default.logto.app/api');

  // --- Admin user (admin tenant, port 3002) ---
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

  // Assign admin role
  const { data: roles } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/roles?type=User');
  const adminRole = Array.isArray(roles) && roles.find(r => r.name === 'default:admin');
  if (adminRole) {
    const assign = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'POST', `/users/${userId}/roles`, { roleIds: [adminRole.id] });
    if (assign.status === 200 || assign.status === 201 || assign.status === 422) {
      console.log('Admin role assigned (or already assigned).');
    }
  }

  // --- API Resource (default tenant, port 3001) ---
  const { data: resources } = await api(LOGTO_ENDPOINT, defaultToken, 'GET', '/resources');
  const existingResource = Array.isArray(resources) && resources.find(r => r.indicator === API_RESOURCE_INDICATOR);

  if (existingResource) {
    console.log(`API resource "${API_RESOURCE_INDICATOR}" already exists.`);
  } else {
    await api(LOGTO_ENDPOINT, defaultToken, 'POST', '/resources', {
      name: 'Infra API',
      indicator: API_RESOURCE_INDICATOR,
    });
    console.log(`Created API resource "${API_RESOURCE_INDICATOR}".`);
  }

  // --- Mark onboarding complete (admin tenant, port 3002) ---
  const { data: consoleCfg } = await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'GET', '/configs/admin-console');
  if (!consoleCfg.signInExperienceCustomized) {
    await api(LOGTO_ADMIN_ENDPOINT, adminToken, 'PATCH', '/configs/admin-console', {
      signInExperienceCustomized: true,
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
