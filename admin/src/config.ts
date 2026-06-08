// Runtime config is injected two ways:
// - Dev (Docker): docker-entrypoint.sh reads /run/infra/admin-app.json and exports VITE_* env vars before Vite starts.
// - Prod (Nginx): docker-entrypoint-prod.sh generates /config.js which sets window.__INFRA_ADMIN_CONFIG__.
// Fallback to VITE_* env vars for running outside Docker (local npm run dev with .env file).

declare global {
  interface Window {
    __INFRA_ADMIN_CONFIG__?: {
      logtoAppId: string
      logtoEndpoint: string
      logtoApiResource: string
    }
  }
}

export const config = {
  logtoAppId: window.__INFRA_ADMIN_CONFIG__?.logtoAppId ?? import.meta.env.VITE_LOGTO_APP_ID ?? '',
  logtoEndpoint: window.__INFRA_ADMIN_CONFIG__?.logtoEndpoint ?? import.meta.env.VITE_LOGTO_ENDPOINT ?? '',
  logtoApiResource: window.__INFRA_ADMIN_CONFIG__?.logtoApiResource ?? import.meta.env.VITE_LOGTO_API_RESOURCE ?? '',
  redirectUri: import.meta.env.VITE_REDIRECT_URI ?? '',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
}
