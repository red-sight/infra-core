// Runtime config is injected two ways:
// - Dev (Docker): docker-entrypoint.sh reads /run/infra/admin-app.json and exports VITE_* env vars before Vite starts.
// - Prod (Nginx): docker-entrypoint-prod.sh generates /config.js which sets window.__INFRA_ADMIN_CONFIG__.
// Fallback to VITE_* env vars for running outside Docker (local npm run dev with .env file).

declare global {
  interface Window {
    __INFRA_ADMIN_CONFIG__?: {
      issuer: string
      clientId: string
      projectId: string
    }
  }
}

export const config = {
  // OIDC issuer (Zitadel), e.g. http://auth.app.localhost
  issuer: window.__INFRA_ADMIN_CONFIG__?.issuer ?? import.meta.env.VITE_OIDC_ISSUER ?? '',
  // The Admin SPA's OIDC client id.
  clientId: window.__INFRA_ADMIN_CONFIG__?.clientId ?? import.meta.env.VITE_OIDC_CLIENT_ID ?? '',
  // The Infra API project id — requested as a token audience so KrakenD accepts the token.
  projectId: window.__INFRA_ADMIN_CONFIG__?.projectId ?? import.meta.env.VITE_OIDC_PROJECT_ID ?? '',
  redirectUri: import.meta.env.VITE_REDIRECT_URI ?? '',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
}
