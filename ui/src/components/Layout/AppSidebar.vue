<script setup>
const props = defineProps({
  activePage: {
    type: String,
    default: '',
  },
})

const pages = [
  { name: 'Slugs', path: '/slugs' },
  { name: 'Licenses', path: '/licenses' },
  { name: 'API Keys', path: '/api-keys' },
  { name: 'Webhooks', path: '/webhooks' },
  { name: 'Offline Licenses', path: '/offline-licenses' },
]

const isActive = (pageName) => {
  return props.activePage === pageName
}
</script>

<template>
  <aside class="flex w-64 flex-col bg-[#171b26]">
    <nav class="flex-1 px-4 py-6 space-y-2">
      <router-link
        v-for="page in pages"
        :key="page.path"
        :to="page.path"
        :class="[
          'flex items-center gap-3 px-4 py-3 rounded-lg transition-colors',
          isActive(page.name)
            ? 'bg-[#1c1f2a] text-white font-medium'
            : 'text-[#8f98ad] hover:bg-[#1c1f2a] hover:text-[#dfe2f1]'
        ]"
      >
        <span class="text-sm">{{ page.name }}</span>
      </router-link>
    </nav>

    <div class="p-4">
      <div class="rounded-lg bg-[#1c1f2a] p-4">
        <h3 class="mb-2 text-sm font-medium text-white">About</h3>
        <p class="mb-2 text-xs text-[#8f98ad]">
          Simple License Server
        </p>
        <p class="text-xs text-[#8f98ad]">
          A dead simple license server hosted in Docker.
        </p>
      </div>
    </div>
  </aside>
</template>
