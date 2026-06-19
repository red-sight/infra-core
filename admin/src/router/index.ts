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
        {
          path: 'organizations/:id',
          component: () => import('../pages/OrganizationDetailPage.vue'),
          // Detail routes aren't nav screens; map them back to the Organizations
          // section so the sidebar highlight and topbar title resolve (route.meta
          // merges onto children).
          meta: { section: 'organizations' },
          children: [
            // Tabs are nested routes so each is deep-linkable. More land per slice
            // (users, roles, settings); the bare path redirects to the first tab.
            { path: '', name: 'organization-detail', redirect: (to) => ({ name: 'organization-overview', params: to.params }) },
            { path: 'overview', name: 'organization-overview', component: () => import('../pages/org/OverviewTab.vue') },
            { path: 'users', name: 'organization-users', component: () => import('../pages/org/UsersTab.vue') },
          ],
        },
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
