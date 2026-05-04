<script setup>
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const props = defineProps({
  activePage: {
    type: String,
    default: '',
  },
})

const authStore = useAuthStore()
const router = useRouter()

const pages = [
  { name: 'Licenses', path: '/licenses' },
  { name: 'Slugs', path: '/slugs' },
  { name: 'API Keys', path: '/api-keys' },
  { name: 'Webhooks', path: '/webhooks' },
  { name: 'Offline Licenses', path: '/offline-licenses' },
]

const isActive = (pageName) => props.activePage === pageName

const logout = () => {
  authStore.clearApiKey()
  router.push('/licenses')
}

const openGenerateDrawer = () => {
  window.dispatchEvent(new CustomEvent('open-generate-license'))
}
</script>

<template>
  <header class="sticky top-0 z-40 bg-[#0a0e18] shadow-[inset_0_-1px_rgba(65,71,81,0.22)]">
    <div class="mx-auto flex min-h-14 w-full max-w-[88rem] items-center gap-5 px-4 sm:px-6 lg:px-8">
      <div class="flex min-w-0 items-center">
        <router-link to="/licenses" class="shrink-0 text-base font-extrabold tracking-tight text-white sm:text-lg">
          LicenseManager
        </router-link>
      </div>

      <nav class="flex min-w-0 flex-1 items-center gap-5 overflow-x-auto text-xs font-semibold md:gap-7">
        <router-link
          v-for="page in pages"
          :key="page.path"
          :to="page.path"
          :class="[
            'relative whitespace-nowrap py-5 transition-colors',
            isActive(page.name)
              ? 'text-white after:absolute after:bottom-0 after:left-0 after:h-px after:w-full after:bg-[#a4c9ff]'
              : 'text-[#8f98ad] hover:text-[#dfe2f1]'
          ]"
        >
          {{ page.name }}
        </router-link>
      </nav>

      <button class="primary-action hidden shrink-0 px-4 py-2 text-xs sm:inline-flex" @click="openGenerateDrawer">
        + Generate Key
      </button>

      <button class="secondary-action shrink-0 px-3 py-2 metadata-label text-[0.65rem]" @click="logout">
        Logout
      </button>
    </div>
  </header>
</template>
