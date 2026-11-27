<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight } from 'lucide-vue-next'
import AppHeader from '@/components/Layout/AppHeader.vue'
import BrandMark from '@/components/BrandMark.vue'
import IssueLicenseDrawer from '@/components/IssueLicenseDrawer.vue'
import ToastHost from '@/components/ToastHost.vue'
import { useAuthStore } from '@/stores/auth'
import { licensesAPI } from '@/stores/api'

const router = useRouter()
const authStore = useAuthStore()

const managementKey = ref('')
const loginError = ref('')
const loginLoading = ref(false)
const authChecked = ref(false)

const isAuthenticated = computed(() => authChecked.value && Boolean(authStore.apiKey))

const verifyManagementKey = async (key) => {
  localStorage.setItem('management_api_key', key)
  try {
    await licensesAPI.list({ page: 1, page_size: 1 })
    authStore.setApiKey(key)
    return true
  } catch {
    authStore.clearApiKey()
    return false
  }
}

const login = async () => {
  const key = managementKey.value.trim()
  loginError.value = ''
  if (!key) {
    loginError.value = 'Enter your management API key.'
    return
  }

  loginLoading.value = true
  try {
    if (!(await verifyManagementKey(key))) {
      loginError.value = 'That key was not accepted.'
      return
    }
    managementKey.value = ''
    router.push('/licenses')
  } finally {
    loginLoading.value = false
  }
}

onMounted(async () => {
  const stored = localStorage.getItem('management_api_key') || localStorage.getItem('api_key')
  if (stored) {
    await verifyManagementKey(stored)
  }
  authChecked.value = true
})
</script>

<template>
  <div class="min-h-screen">
    <main v-if="!authChecked" class="flex min-h-screen items-center justify-center">
      <BrandMark size="h-10 w-10" class="animate-pulse" />
    </main>

    <main v-else-if="!isAuthenticated" class="relative flex min-h-screen items-center justify-center overflow-hidden px-5 py-12">
      <div class="pointer-events-none absolute inset-0" style="background: radial-gradient(36rem 26rem at 50% 32%, rgba(96, 165, 250, 0.13), transparent 70%)"></div>

      <section class="relative w-full max-w-sm">
        <div class="flex flex-col items-center text-center">
          <BrandMark size="h-12 w-12" />
          <h1 class="mt-6 text-[1.75rem] font-semibold tracking-tight text-ink-strong">Simple License Server</h1>
          <p class="mt-2 text-sm text-ink-muted">Issue keys. Revoke keys. Validate keys.</p>
        </div>

        <form class="card mt-8 space-y-4 p-6" @submit.prevent="login">
          <label class="block">
            <span class="field-label">Management API key</span>
            <input
              v-model="managementKey"
              type="password"
              autocomplete="current-password"
              autofocus
              class="field font-mono"
              placeholder="management_key_…"
            />
          </label>
          <p v-if="loginError" class="text-sm text-critical">{{ loginError }}</p>
          <button class="btn-primary w-full py-2.5" :disabled="loginLoading">
            {{ loginLoading ? 'Checking…' : 'Sign in' }}
            <ArrowRight v-if="!loginLoading" class="h-4 w-4" :stroke-width="2.2" />
          </button>
        </form>

        <p class="mt-6 text-center text-xs text-ink-faint">
          Set with <code class="font-mono text-ink-muted">MANAGEMENT_API_KEYS</code> on the server.
        </p>
      </section>
    </main>

    <template v-else>
      <AppHeader />
      <main class="mx-auto w-full max-w-6xl px-4 pb-16 pt-8 sm:px-6 sm:pt-10">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
      <IssueLicenseDrawer />
    </template>

    <ToastHost />
  </div>
</template>
