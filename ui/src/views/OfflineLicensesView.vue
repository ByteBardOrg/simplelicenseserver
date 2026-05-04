<script setup>
import { computed, onMounted, ref } from 'vue'
import { offlineAPI } from '@/stores/api'

const signingKeys = ref([])
const publicKeys = ref([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const name = ref('')
const copiedPublicKeys = ref(false)

const activeKey = computed(() => signingKeys.value.find((key) => key.status === 'active') || null)
const exportText = computed(() => {
  if (publicKeys.value.length === 0) return ''
  return publicKeys.value.map((key) => `# kid=${key.kid} status=${key.status}\n${key.public_key_pem}`).join('\n')
})

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const [keysRes, publicRes] = await Promise.all([
      offlineAPI.signingKeys(),
      offlineAPI.publicKeys(),
    ])
    signingKeys.value = keysRes.data.signing_keys || []
    publicKeys.value = publicRes.data.signing_keys || []
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to load offline signing keys'
  } finally {
    loading.value = false
  }
}

const createKey = async () => {
  saving.value = true
  error.value = ''
  try {
    await offlineAPI.createSigningKey(name.value.trim())
    name.value = ''
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to create signing key'
  } finally {
    saving.value = false
  }
}

const activateKey = async (id) => {
  error.value = ''
  try {
    await offlineAPI.activateSigningKey(id)
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to activate signing key'
  }
}

const retireKey = async (id) => {
  if (!confirm('Retire this signing key? Existing offline tokens using it will fail once clients remove the public key.')) {
    return
  }

  error.value = ''
  try {
    await offlineAPI.retireSigningKey(id)
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to retire signing key'
  }
}

const copyPublicKeys = async () => {
  try {
    await navigator.clipboard.writeText(exportText.value)
    copiedPublicKeys.value = true
    window.setTimeout(() => {
      copiedPublicKeys.value = false
    }, 1400)
  } catch {
    error.value = 'Failed to copy public keys'
  }
}

const formatDate = (value) => {
  if (!value) return 'Never'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Never'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date)
}

const statusClass = (status) => {
  if (status === 'active') return 'bg-[#4ae176]/10 text-[#4ae176]'
  if (status === 'retired') return 'bg-[#ff7a8a]/10 text-[#ff7a8a]'
  return 'bg-[#a4c9ff]/10 text-[#a4c9ff]'
}

onMounted(load)
</script>

<template>
  <div class="grid gap-5 xl:grid-cols-[24rem_minmax(0,1fr)]">
    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <p class="metadata-label text-xs font-bold text-[#8f98ad]">Offline License Signing</p>
      <h1 class="mt-2 text-xl font-black tracking-tight text-white">Create signing key.</h1>
      <p class="mt-2 text-sm leading-6 text-[#8f98ad]">Generated Ed25519 keys sign offline JWTs for slugs that have offline mode enabled.</p>

      <form class="mt-5 space-y-4" @submit.prevent="createKey">
        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Key Name</span>
          <input v-model="name" placeholder="desktop-app-2026" class="field-control text-sm" />
        </label>
        <button :disabled="saving" class="primary-action px-4 py-2 text-sm disabled:opacity-60">
          {{ saving ? 'Creating...' : 'Create Key' }}
        </button>
      </form>

      <p v-if="error" class="mt-4 text-sm text-[#ff7a8a]">{{ error }}</p>

      <div class="mt-6 rounded-md bg-[#1c1f2a] p-4">
        <p class="metadata-label text-[0.65rem] font-bold text-[#8f98ad]">Active signer</p>
        <p class="mt-2 text-sm font-semibold text-white">{{ activeKey ? activeKey.name : 'No active key' }}</p>
        <p class="mt-1 break-all font-mono text-xs text-[#8f98ad]">{{ activeKey ? activeKey.kid : 'Create and activate a key before tokens can be issued.' }}</p>
      </div>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Signing keys.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">Only one key signs new tokens. Verify-only keys remain available for rollover.</p>
        </div>
        <button class="secondary-action px-3 py-2 text-xs" @click="load">Refresh</button>
      </div>

      <p v-if="loading" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">Loading signing keys...</p>
      <p v-else-if="signingKeys.length === 0" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">No signing keys found.</p>

      <div v-else class="mt-4 overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="metadata-label text-xs text-[#8f98ad]">
            <tr>
              <th class="px-3 py-3 font-bold">Name</th>
              <th class="px-3 py-3 font-bold">KID</th>
              <th class="px-3 py-3 font-bold">Status</th>
              <th class="px-3 py-3 font-bold">Created</th>
              <th class="px-3 py-3 text-right font-bold">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="key in signingKeys" :key="key.id" class="group">
              <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 font-semibold text-[#dfe2f1] group-hover:bg-[#262a35]">{{ key.name }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]"><code class="break-all text-xs text-[#a4c9ff]">{{ key.kid }}</code></td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="statusClass(key.status)">{{ key.status }}</span>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ formatDate(key.created_at) }}</td>
              <td class="rounded-r-md bg-[#1c1f2a] px-3 py-3 text-right group-hover:bg-[#262a35]">
                <button class="metadata-label text-[0.65rem] font-bold text-[#a4c9ff] transition hover:text-white disabled:text-[#8f98ad] disabled:opacity-50" :disabled="key.status === 'active' || key.status === 'retired'" @click="activateKey(key.id)">Activate</button>
                <button class="ml-4 metadata-label text-[0.65rem] font-bold text-[#ff7a8a] transition hover:text-white disabled:text-[#8f98ad] disabled:opacity-50" :disabled="key.status === 'retired'" @click="retireKey(key.id)">Retire</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5 xl:col-span-2">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Public key export.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">Ship active and verify-only public keys with client applications that perform offline validation.</p>
        </div>
        <button :disabled="!exportText" class="secondary-action px-3 py-2 text-xs disabled:opacity-50" @click="copyPublicKeys">
          {{ copiedPublicKeys ? 'Copied' : 'Copy Public Keys' }}
        </button>
      </div>

      <pre class="code-block mt-4 max-h-[28rem] overflow-auto rounded-md p-4 text-xs text-[#dfe2f1]">{{ exportText || 'No public keys available.' }}</pre>
    </section>
  </div>
</template>
