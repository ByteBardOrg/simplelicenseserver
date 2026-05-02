<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppHeader from '@/components/Layout/AppHeader.vue'
import { useAuthStore } from '@/stores/auth'
import { licensesAPI } from '@/stores/api'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const managementKey = ref('')
const loginError = ref('')
const loginLoading = ref(false)
const authChecked = ref(false)
const activePage = computed(() => route.meta.title)

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
    loginError.value = 'Management API key is required.'
    return
  }

  loginLoading.value = true
  try {
    const valid = await verifyManagementKey(key)
    if (!valid) {
      loginError.value = 'Invalid management API key.'
      return
    }

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
  <div class="min-h-screen bg-[#0f131d] text-[#dfe2f1]">
    <main v-if="!authChecked" class="flex min-h-screen items-center justify-center px-5 py-12">
      <p class="metadata-label text-sm font-bold text-[#8f98ad]">Checking Management Key...</p>
    </main>

    <main v-else-if="!isAuthenticated" class="flex min-h-screen items-center justify-center px-5 py-12">
      <section class="grid w-full max-w-5xl overflow-hidden rounded-lg bg-[#171b26] lg:grid-cols-[1.1fr_0.9fr]">
        <div class="flex min-h-[28rem] flex-col justify-between bg-[#0a0e18] p-8 sm:p-10">
          <div>
            <p class="metadata-label text-xs font-bold text-[#8f98ad]">Dead Simple License Server</p>
            <h1 class="mt-5 max-w-xl text-4xl font-black tracking-tight text-white sm:text-5xl">
              Issue keys. Revoke keys. Validate keys.
            </h1>
            <p class="mt-6 max-w-lg text-lg leading-8 text-[#b8c0d4]">
              A compact management console for the few operations that matter. No customer objects, no user model, no ceremony.
            </p>
          </div>

          <pre class="code-block mt-10 overflow-x-auto rounded-md p-5 text-sm text-[#4ae176]">POST /generate
POST /revoke
POST /validate</pre>
        </div>

        <div class="flex items-center p-8 sm:p-10">
          <form class="w-full space-y-5" @submit.prevent="login">
            <div>
              <p class="metadata-label text-xs font-bold text-[#8f98ad]">Management Console</p>
              <h2 class="mt-3 text-2xl font-extrabold text-white">Enter API key</h2>
            </div>

            <label class="block text-sm text-[#c6ccdc]">
              <span class="metadata-label mb-2 block text-xs text-[#8f98ad]">Management API Key</span>
              <input
                v-model="managementKey"
                type="password"
                autofocus
                class="field-control"
                placeholder="management_key..."
              />
            </label>
            <p v-if="loginError" class="text-sm text-[#ffb4b4]">{{ loginError }}</p>
            <button class="primary-action w-full px-5 py-3 disabled:opacity-60" :disabled="loginLoading">
              {{ loginLoading ? 'Checking Key...' : 'Enter Console' }}
            </button>
          </form>
        </div>
      </section>
    </main>

    <template v-else>
      <AppHeader :active-page="activePage" />
      <main class="mx-auto w-full max-w-[88rem] px-4 py-6 sm:px-6 lg:px-8">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </template>
  </div>
</template>
