import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/licenses' },
    {
      path: '/licenses',
      name: 'licenses',
      component: () => import('@/views/LicensesView.vue'),
      meta: { title: 'Licenses' },
    },
    {
      path: '/slugs',
      name: 'slugs',
      component: () => import('@/views/SlugsView.vue'),
      meta: { title: 'Slugs' },
    },
    {
      path: '/settings',
      component: () => import('@/views/SettingsView.vue'),
      children: [
        { path: '', redirect: '/settings/api-keys' },
        {
          path: 'api-keys',
          name: 'api-keys',
          component: () => import('@/views/settings/ApiKeysPanel.vue'),
          meta: { title: 'API keys' },
        },
        {
          path: 'webhooks',
          name: 'webhooks',
          component: () => import('@/views/settings/WebhooksPanel.vue'),
          meta: { title: 'Webhooks' },
        },
        {
          path: 'offline',
          name: 'offline',
          component: () => import('@/views/settings/SigningKeysPanel.vue'),
          meta: { title: 'Offline signing' },
        },
      ],
    },
    // Old top-level pages, kept so existing bookmarks still work.
    { path: '/api-keys', redirect: '/settings/api-keys' },
    { path: '/webhooks', redirect: '/settings/webhooks' },
    { path: '/offline-licenses', redirect: '/settings/offline' },
    { path: '/:pathMatch(.*)*', redirect: '/licenses' },
  ],
})

router.afterEach((to) => {
  const title = to.meta.title as string | undefined
  document.title = title ? `${title} · Simple License Server` : 'Simple License Server'
})

export default router
