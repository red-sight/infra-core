import { createApp } from 'vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
// Tailwind layers first, then the design system (tokens + component styles) so
// it owns the cascade.
import './assets/index.css'
import './assets/theme.css'
import './assets/components.css'
import './assets/shell.css'
import App from './App.vue'
import router from './router'

// Auth is OIDC via the useAuth composable (oidc-client-ts); no app-level plugin.
const app = createApp(App)

app.use(router)
app.use(VueQueryPlugin)

app.mount('#app')
