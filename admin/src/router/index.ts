import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/callback',
      component: () => import('../pages/CallbackPage.vue'),
    },
    {
      path: '/',
      component: () => import('../components/layout/AppLayout.vue'),
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('../pages/DashboardPage.vue'),
        },
      ],
    },
  ],
})

export default router
