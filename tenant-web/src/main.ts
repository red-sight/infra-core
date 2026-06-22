import { createApp } from 'vue'
import App from './App.vue'

// Auth is OIDC (oidc-client-ts) configured per-organization inside App.vue once the
// tenant is resolved by host; no app-level plugin.
createApp(App).mount('#app')
