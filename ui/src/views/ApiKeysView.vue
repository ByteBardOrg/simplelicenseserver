<script setup>
import { onMounted, ref } from 'vue'
import { apiKeysAPI } from '@/stores/api'

const keys = ref([])
const loading = ref(false)
const error = ref('')
const name = ref('')
const lastCreated = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await apiKeysAPI.list()
    keys.value = res.data.api_keys || []
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to load API keys'
  } finally {
    loading.value = false
  }
}

const createKey = async () => {
  error.value = ''
  lastCreated.value = ''
  try {
    const res = await apiKeysAPI.create(name.value.trim())
    lastCreated.value = res.data.api_key || ''
    name.value = ''
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to create API key'
  }
}

const revoke = async (id) => {
  error.value = ''
  try {
    await apiKeysAPI.revoke(id)
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to revoke API key'
  }
}

const formatDate = (value) => {
  if (!value) return 'Never'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(value))
}

onMounted(load)
</script>

<template>
  <div class="grid gap-5 xl:grid-cols-[22rem_minmax(0,1fr)]">
    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <p class="metadata-label text-xs font-bold text-[#8f98ad]">Provisioning Access</p>
      <h1 class="mt-2 text-xl font-black tracking-tight text-white">Create server API key.</h1>
      <p class="mt-2 text-sm leading-6 text-[#8f98ad]">Server keys authenticate runtime provisioning calls outside the management console.</p>

      <div class="mt-5 space-y-4">
        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Key Name</span>
          <input v-model="name" placeholder="optional" class="field-control text-sm" />
        </label>
        <button class="primary-action px-4 py-2 text-sm" @click="createKey">Create Key</button>
      </div>

      <p v-if="error" class="mt-4 text-sm text-[#ff7a8a]">{{ error }}</p>
      <div v-if="lastCreated" class="mt-5 rounded-md bg-[#1c1f2a] p-3">
        <p class="metadata-label text-[0.65rem] font-bold text-[#8f98ad]">New key, shown once</p>
        <code class="mt-3 block break-all text-xs text-[#4ae176]">{{ lastCreated }}</code>
      </div>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Existing server keys.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">{{ keys.length }} keys configured</p>
        </div>
        <button class="secondary-action px-3 py-2 text-xs" @click="load">Refresh</button>
      </div>

      <p v-if="loading" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">Loading keys...</p>
      <p v-else-if="keys.length === 0" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">No API keys found.</p>

      <div v-else class="mt-4 overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="metadata-label text-xs text-[#8f98ad]">
            <tr>
              <th class="px-3 py-3 font-bold">ID</th>
              <th class="px-3 py-3 font-bold">Name</th>
              <th class="px-3 py-3 font-bold">Hint</th>
              <th class="px-3 py-3 font-bold">Created</th>
              <th class="px-3 py-3 font-bold">Status</th>
              <th class="px-3 py-3 text-right font-bold">Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="key in keys" :key="key.id" class="group">
              <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ key.id }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ key.name || 'unnamed' }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]"><code class="text-xs text-[#a4c9ff]">{{ key.hint }}</code></td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ formatDate(key.created_at) }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="key.revoked_at ? 'bg-[#ff7a8a]/10 text-[#ff7a8a]' : 'bg-[#4ae176]/10 text-[#4ae176]'">
                  {{ key.revoked_at ? 'revoked' : 'active' }}
                </span>
              </td>
              <td class="rounded-r-md bg-[#1c1f2a] px-3 py-3 text-right group-hover:bg-[#262a35]">
                <button class="metadata-label text-[0.65rem] font-bold text-[#ff7a8a] transition hover:text-white disabled:text-[#8f98ad] disabled:opacity-50" :disabled="Boolean(key.revoked_at)" @click="revoke(key.id)">
                  {{ key.revoked_at ? 'Revoked' : 'Revoke' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
