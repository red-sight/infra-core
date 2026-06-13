import { createApp } from 'vue'
import { createLogto, UserScope } from '@logto/vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
// Tailwind layers first, then the design system (tokens + component styles) so
// it owns the cascade.
import './assets/index.css'
import './assets/theme.css'
import './assets/components.css'
import './assets/shell.css'
import App from './App.vue'
import router from './router'
import { config } from './config'

const app = createApp(App)

app.use(router)
app.use(VueQueryPlugin)
app.use(createLogto, {
  endpoint: config.logtoEndpoint,
  appId: config.logtoAppId,
  scopes: [UserScope.Email, UserScope.Profile, UserScope.Organizations],
  resources: [config.logtoApiResource],
})

app.mount('#app')
