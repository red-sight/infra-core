import { createApp } from 'vue'
import { createLogto, UserScope } from '@logto/vue'
import { VueQueryPlugin } from '@tanstack/vue-query'
import './assets/index.css'
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
