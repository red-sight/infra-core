<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useLogto } from '@logto/vue'
import { config } from './config'

type Tenant = { id: string; slug: string; name: string }
type Status = 'loading' | 'unknown' | 'error' | 'ready'

const logto = useLogto()
const { isAuthenticated, isLoading, signIn, signOut, fetchUserInfo, getOrganizationToken } = logto
// handleSignInCallback exists at runtime but isn't in the typed surface of useLogto.
const handleSignInCallback = (
  logto as unknown as { handleSignInCallback: (url: string) => Promise<void> }
).handleSignInCallback

const status = ref<Status>('loading')
const tenant = ref<Tenant | null>(null)
const errorMsg = ref('')
const userName = ref('')
const orgToken = ref<'idle' | 'ok' | 'denied'>('idle')

// The subdomain label is the tenant slug (e.g. acme.app.localhost → "acme").
const slug = location.hostname.split('.')[0]
const redirectUri = `${location.origin}/callback`

async function resolveTenant() {
  try {
    const res = await fetch(`${config.apiBaseUrl}/service-core/v1/tenant/by-slug/${slug}`)
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

onMounted(async () => {
  if (location.pathname === '/callback') {
    try {
      await handleSignInCallback(location.href)
    } catch (e) {
      errorMsg.value = e instanceof Error ? e.message : String(e)
    }
    window.history.replaceState({}, '', '/')
  }
  await resolveTenant()
})

// Once the tenant is resolved and Logto is ready: sign in if needed, else load
// the profile and try to acquire an organization-scoped token.
watch(
  [isLoading, isAuthenticated, status],
  async () => {
    if (isLoading.value || status.value !== 'ready' || !tenant.value) return
    if (!isAuthenticated.value) {
      await signIn(redirectUri)
      return
    }
    const info = await fetchUserInfo()
    userName.value = info?.name ?? info?.username ?? info?.sub ?? 'user'
    try {
      const token = await getOrganizationToken(tenant.value.id)
      orgToken.value = token ? 'ok' : 'denied'
    } catch {
      orgToken.value = 'denied'
    }
  },
  { immediate: true },
)

function logout() {
  signOut(location.origin)
}
</script>

<template>
  <main class="wrap">
    <div v-if="status === 'loading'" class="card">Loading…</div>

    <div v-else-if="status === 'unknown'" class="card">
      <h1>Unknown tenant</h1>
      <p>No organization is registered for <code>{{ slug }}</code>.</p>
    </div>

    <div v-else-if="status === 'error'" class="card">
      <h1>Something went wrong</h1>
      <p class="note">{{ errorMsg }}</p>
    </div>

    <div v-else-if="!isAuthenticated" class="card">Redirecting to sign in…</div>

    <div v-else class="card">
      <h1>Welcome to {{ tenant?.name }}</h1>
      <p>Signed in as <strong>{{ userName }}</strong> · tenant <code>{{ tenant?.slug }}</code></p>
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
