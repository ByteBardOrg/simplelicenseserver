<script setup>
import { useRoute, useRouter } from 'vue-router'
import { LogOut, Plus } from 'lucide-vue-next'
import BrandMark from '@/components/BrandMark.vue'
import { useAuthStore } from '@/stores/auth'
import { openIssueDrawer } from '@/lib/ui'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const pages = [
  { name: 'Licenses', path: '/licenses' },
  { name: 'Slugs', path: '/slugs' },
  { name: 'Settings', path: '/settings' },
]

const isActive = (path) => route.path === path || route.path.startsWith(`${path}/`)

const logout = () => {
  authStore.clearApiKey()
  router.push('/licenses')
}
</script>

<template>
  <header class="glass sticky top-0 z-40 shadow-[inset_0_-1px_rgba(65,71,81,0.25)]">
    <div class="mx-auto flex h-16 w-full max-w-6xl items-center gap-2 px-3 sm:gap-6 sm:px-6">
      <router-link to="/licenses" class="flex shrink-0 items-center gap-2.5">
        <BrandMark size="h-7 w-7" />
        <span class="hidden text-[0.95rem] font-semibold tracking-tight text-ink-strong md:inline">Simple License Server</span>
      </router-link>

      <nav class="flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto sm:gap-1">
        <router-link
          v-for="page in pages"
          :key="page.path"
          :to="page.path"
          class="whitespace-nowrap rounded-lg px-2.5 py-1.5 text-sm font-medium transition-colors sm:px-3"
          :class="isActive(page.path) ? 'bg-surface-container text-ink-strong shadow-ghost' : 'text-ink-muted hover:text-ink'"
        >
          {{ page.name }}
        </router-link>
      </nav>

      <button class="btn-primary shrink-0 px-3 py-1.5" @click="openIssueDrawer">
        <Plus class="h-4 w-4" :stroke-width="2.4" />
        <span class="hidden sm:inline">New license</span>
      </button>

      <button class="btn-icon shrink-0" title="Sign out" aria-label="Sign out" @click="logout">
        <LogOut class="h-4 w-4" :stroke-width="1.8" />
      </button>
    </div>
  </header>
</template>
