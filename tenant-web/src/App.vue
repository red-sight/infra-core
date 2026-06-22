<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { UserManager, WebStorageStateStore, type User } from 'oidc-client-ts'
import { config } from './config'

type Tenant = { id: string; external_id: string; slug: string; name: string; is_master: boolean }
type Status = 'loading' | 'unknown' | 'error' | 'ready'

const status = ref<Status>('loading')
const tenant = ref<Tenant | null>(null)
const errorMsg = ref('')
const userName = ref('')
const isAuthenticated = ref(false)
const orgToken = ref<'idle' | 'ok' | 'denied'>('idle')

const host = location.host

// makeManager builds an org-scoped OIDC client. The org id scope binds login to
// this organization (and applies its branding); the project-roles + project-aud
// scopes get the user's roles asserted and set the audience KrakenD validates.
function makeManager(externalId: string) {
  const scope = [
    'openid',
    'profile',
    'email',
    'offline_access',
    'urn:zitadel:iam:org:projects:roles',
    `urn:zitadel:iam:org:project:id:${config.projectId}:aud`,
    `urn:zitadel:iam:org:id:${externalId}`,
  ].join(' ')
  return new UserManager({
    authority: config.issuer,
    client_id: config.clientId,
    redirect_uri: `${location.origin}/callback`,
    post_logout_redirect_uri: location.origin,
    response_type: 'code',
    scope,
    userStore: new WebStorageStateStore({ store: window.localStorage }),
  })
}

let manager: UserManager | null = null

async function resolveTenant() {
  try {
    const res = await fetch(
      `${config.apiBaseUrl}/service-core/v1/tenant/by-host?host=${encodeURIComponent(host)}`,
    )
    if (res.status === 404) {
      status.value = 'unknown'
      return
    }
    if (!res.ok) throw new Error(`resolve failed (${res.status})`)
    tenant.value = (await res.json()) as Tenant
    status.value = 'ready'
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : String(e)
    status.value = 'error'
  }
}

// rolesFromToken decodes the access token's flat `roles` claim (set by the Zitadel
// action). A member of this org has a grant → roles; a non-member has none.
function rolesFromToken(u: User): string[] {
  try {
    const p = JSON.parse(atob(u.access_token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return Array.isArray(p.roles) ? p.roles : []
  } catch {
    return []
  }
}

onMounted(async () => {
  // The host identifies the org on both initial load and the OIDC callback.
  await resolveTenant()
  if (status.value !== 'ready' || !tenant.value) return

  manager = makeManager(tenant.value.external_id)

  if (location.pathname === '/callback') {
    try {
      await manager.signinCallback()
    } catch (e) {
      errorMsg.value = e instanceof Error ? e.message : String(e)
    }
    window.history.replaceState({}, '', '/')
  }

  const user = await manager.getUser()
  if (!user || user.expired) {
    await manager.signinRedirect()
    return
  }
  isAuthenticated.value = true
  userName.value =
    (user.profile.name as string) ?? (user.profile.preferred_username as string) ?? user.profile.sub
  orgToken.value = rolesFromToken(user).length > 0 ? 'ok' : 'denied'
})

async function logout() {
  await manager?.signoutRedirect()
}
</script>

<template>
  <main class="wrap">
    <div v-if="status === 'loading'" class="card">Loading…</div>

    <div v-else-if="status === 'unknown'" class="card">
      <h1>Unknown tenant</h1>
      <p>No organization is registered for <code>{{ host }}</code>.</p>
    </div>

    <div v-else-if="status === 'error'" class="card">
      <h1>Something went wrong</h1>
      <p class="note">{{ errorMsg }}</p>
    </div>

    <div v-else-if="!isAuthenticated" class="card">Redirecting to sign in…</div>

    <div v-else class="card">
      <h1>Welcome to {{ tenant?.name }}</h1>
      <p>Signed in as <strong>{{ userName }}</strong> · <code>{{ tenant?.is_master ? 'master (apex)' : tenant?.slug }}</code></p>
      <p v-if="orgToken === 'ok'" class="ok">Organization token acquired ✓</p>
      <p v-else-if="orgToken === 'denied'" class="note">
        No organization access yet — you are not a member of this organization.
        (Membership / invite is a separate API.)
      </p>
      <button @click="logout">Sign out</button>
    </div>
  </main>
</template>

<style>
:root {
  font-family: system-ui, -apple-system, sans-serif;
}
body {
  margin: 0;
}
.wrap {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: #0b0b0f;
  color: #e7e7ea;
}
.card {
  background: #16161c;
  border: 1px solid #2a2a33;
  border-radius: 12px;
  padding: 32px 36px;
  max-width: 460px;
}
h1 {
  margin: 0 0 8px;
  font-size: 20px;
}
code {
  background: #23232c;
  padding: 1px 6px;
  border-radius: 5px;
}
.ok {
  color: #4ade80;
}
.note {
  color: #fbbf24;
  font-size: 14px;
}
button {
  margin-top: 16px;
  padding: 8px 14px;
  border-radius: 8px;
  border: 1px solid #2a2a33;
  background: #23232c;
  color: #e7e7ea;
  cursor: pointer;
}
</style>
