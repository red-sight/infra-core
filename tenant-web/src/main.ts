import { createApp } from 'vue'
import { createLogto, UserScope } from '@logto/vue'
import App from './App.vue'
import { config } from './config'

const app = createApp(App)

app.use(createLogto, {
  endpoint: config.logtoEndpoint,
  appId: config.logtoAppId,
  // Organizations scope is required to request organization-scoped tokens.
  scopes: [UserScope.Email, UserScope.Profile, UserScope.Organizations],
  resources: [config.logtoApiResource],
})

app.mount('#app')
