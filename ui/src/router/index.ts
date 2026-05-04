import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/licenses',
    },
    {
      path: '/slugs',
      name: 'slugs',
      component: () => import('@/views/SlugsView.vue'),
      meta: { title: 'Slugs' },
    },
    {
      path: '/licenses',
      name: 'licenses',
      component: () => import('@/views/LicensesView.vue'),
      meta: { title: 'Licenses' },
    },
    {
      path: '/api-keys',
      name: 'api-keys',
      component: () => import('@/views/ApiKeysView.vue'),
      meta: { title: 'API Keys' },
    },
    {
      path: '/webhooks',
      name: 'webhooks',
      component: () => import('@/views/WebhooksView.vue'),
      meta: { title: 'Webhooks' },
    },
    {
      path: '/offline-licenses',
      name: 'offline-licenses',
      component: () => import('@/views/OfflineLicensesView.vue'),
      meta: { title: 'Offline Licenses' },
    },
  ],
})

router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  if (!authStore.apiKey) {
    authStore.loadApiKey()
  }

  if (to.meta.requiresAuth && !authStore.apiKey) {
    next({ name: 'licenses' })
    return
  }

  document.title = to.meta.title ? `${to.meta.title} - Simple License Server` : 'Simple License Server'
  next()
})

export default router
