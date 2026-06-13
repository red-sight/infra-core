import { createRouter, createWebHistory } from 'vue-router'

// Route names equal screen ids so the shell can derive the active nav link.
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
        { path: '', name: 'overview', component: () => import('../pages/OverviewPage.vue') },
        { path: 'organizations', name: 'organizations', component: () => import('../pages/OrganizationsPage.vue') },
        { path: 'users', name: 'users', component: () => import('../pages/UsersPage.vue') },
        { path: 'roles', name: 'roles', component: () => import('../pages/RolesPage.vue') },
        { path: 'billing', name: 'billing', component: () => import('../pages/BillingPage.vue') },
        { path: 'audit', name: 'audit', component: () => import('../pages/AuditPage.vue') },
        { path: 'flags', name: 'flags', component: () => import('../pages/FlagsPage.vue') },
      ],
    },
  ],
})

export default router
