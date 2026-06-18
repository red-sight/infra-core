// Runtime config, mirroring the admin app:
// - Dev (Docker): docker-entrypoint.sh reads /run/infra/tenant-app.json → VITE_* env.
// - Prod (Nginx): docker-entrypoint-prod.sh writes /config.js → window.__INFRA_TENANT_CONFIG__.
// - Fallback to VITE_* for plain `npm run dev` outside Docker.

declare global {
  interface Window {
    __INFRA_TENANT_CONFIG__?: {
      logtoAppId: string
      logtoEndpoint: string
      logtoApiResource: string
    }
  }
}

export const config = {
  logtoAppId: window.__INFRA_TENANT_CONFIG__?.logtoAppId ?? import.meta.env.VITE_LOGTO_APP_ID ?? '',
  logtoEndpoint: window.__INFRA_TENANT_CONFIG__?.logtoEndpoint ?? import.meta.env.VITE_LOGTO_ENDPOINT ?? '',
  logtoApiResource:
    window.__INFRA_TENANT_CONFIG__?.logtoApiResource ?? import.meta.env.VITE_LOGTO_API_RESOURCE ?? '',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
}

export {}
