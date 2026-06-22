// Generic OIDC auth (Authorization Code + PKCE) against Zitadel, via oidc-client-ts.
// Replaces @logto/vue. The surface mirrors the bits the app used from useLogto()
// — isAuthenticated, isLoading, signIn, signOut, getAccessToken,
// handleSignInCallback, fetchUserInfo — so call sites stay unchanged.
import { computed, ref } from 'vue'
import { UserManager, WebStorageStateStore, type User } from 'oidc-client-ts'
import { config } from '@/config'

// Scopes:
// - openid/profile/email: standard identity.
// - urn:zitadel:iam:org:projects:roles: ask Zitadel to assert project roles into
//   the token (the flatten action then exposes them as a flat `roles` claim).
// - urn:zitadel:iam:org:project:id:<projectId>:aud: set the token audience to the
//   Infra API project so KrakenD's audience check passes.
const scope = [
  'openid',
  'profile',
  'email',
  'offline_access',
  'urn:zitadel:iam:org:projects:roles',
  `urn:zitadel:iam:org:project:id:${config.projectId}:aud`,
].join(' ')

const manager = new UserManager({
  authority: config.issuer,
  client_id: config.clientId,
  redirect_uri: config.redirectUri,
  post_logout_redirect_uri: window.location.origin,
  response_type: 'code',
  scope,
  userStore: new WebStorageStateStore({ store: window.localStorage }),
  automaticSilentRenew: true,
})

const user = ref<User | null>(null)
const isLoading = ref(true)
let started = false

// load resolves the persisted session once and subscribes to token changes.
function start() {
  if (started) return
  started = true
  manager.events.addUserLoaded((u) => (user.value = u))
  manager.events.addUserUnloaded(() => (user.value = null))
  manager
    .getUser()
    .then((u) => (user.value = u))
    .finally(() => (isLoading.value = false))
}

export function useAuth() {
  start()

  const isAuthenticated = computed(() => !!user.value && !user.value.expired)

  return {
    isAuthenticated,
    isLoading,
    signIn: () => manager.signinRedirect(),
    signOut: () => manager.signoutRedirect(),
    // resource arg kept for call-site compatibility; the audience is fixed via scope.
    getAccessToken: async (_resource?: string) => (await manager.getUser())?.access_token ?? '',
    handleSignInCallback: async () => {
      const u = await manager.signinCallback()
      if (u) user.value = u
    },
    fetchUserInfo: async () => (await manager.getUser())?.profile ?? null,
  }
}
