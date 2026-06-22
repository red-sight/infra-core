// Runtime config, mirroring the admin app:
// - Dev (Docker): docker-entrypoint.sh reads /run/infra/tenant-app.json → VITE_* env.
// - Prod (Nginx): docker-entrypoint-prod.sh writes /config.js → window.__INFRA_TENANT_CONFIG__.
// - Fallback to VITE_* for plain `npm run dev` outside Docker.

declare global {
  interface Window {
    __INFRA_TENANT_CONFIG__?: {
      issuer: string
      clientId: string
      projectId: string
    }
  }
}

export const config = {
  issuer: window.__INFRA_TENANT_CONFIG__?.issuer ?? import.meta.env.VITE_OIDC_ISSUER ?? '',
  clientId: window.__INFRA_TENANT_CONFIG__?.clientId ?? import.meta.env.VITE_OIDC_CLIENT_ID ?? '',
  projectId: window.__INFRA_TENANT_CONFIG__?.projectId ?? import.meta.env.VITE_OIDC_PROJECT_ID ?? '',
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? '',
}

export {}
