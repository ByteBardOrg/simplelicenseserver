<script setup>
import { computed, onMounted, ref } from 'vue'
import { ChevronRight, ShieldAlert, ShieldCheck } from 'lucide-vue-next'
import StatusBadge from '@/components/StatusBadge.vue'
import CopyButton from '@/components/CopyButton.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import { offlineAPI } from '@/stores/api'
import { toast, toastError } from '@/lib/ui'
import { errorMessage, formatDate } from '@/lib/format'

const signingKeys = ref([])
const publicKeys = ref([])
const loading = ref(false)
const loadError = ref('')
const name = ref('')
const creating = ref(false)

const activeKey = computed(() => signingKeys.value.find((key) => key.status === 'active') || null)
const exportText = computed(() =>
  publicKeys.value.map((key) => `# kid=${key.kid} status=${key.status}\n${key.public_key_pem}`).join('\n'),
)

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const [keysRes, publicRes] = await Promise.all([offlineAPI.signingKeys(), offlineAPI.publicKeys()])
    signingKeys.value = keysRes.data.signing_keys || []
    publicKeys.value = publicRes.data.signing_keys || []
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load signing keys')
  } finally {
    loading.value = false
  }
}

const create = async () => {
  creating.value = true
  try {
    await offlineAPI.createSigningKey(name.value.trim())
    name.value = ''
    toast('Signing key created')
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not create signing key'))
  } finally {
    creating.value = false
  }
}

const activate = async (key) => {
  try {
    await offlineAPI.activateSigningKey(key.id)
    toast(`${key.name} now signs new tokens`)
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not activate signing key'))
  }
}

const retire = async (key) => {
  try {
    await offlineAPI.retireSigningKey(key.id)
    toast(`Retired ${key.name}`)
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not retire signing key'))
  }
}

onMounted(load)
</script>

<template>
  <section>
    <h2 class="text-lg font-semibold tracking-tight text-ink-strong">Offline signing</h2>
    <p class="mt-1 text-sm leading-6 text-ink-muted">
      Slugs with offline tokens turned on get an Ed25519-signed JWT on activation. Ship the public keys with your app so
      it can verify tokens without a network connection.
    </p>

    <div
      v-if="!loading && !loadError"
      class="mt-5 flex items-center gap-4 rounded-xl p-4"
      :class="activeKey ? 'bg-success/[0.06] shadow-[inset_0_0_0_1px_rgba(74,225,118,0.22)]' : 'bg-warning/[0.07] shadow-[inset_0_0_0_1px_rgba(242,179,109,0.25)]'"
    >
      <span
        class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
        :class="activeKey ? 'bg-success/10 text-success' : 'bg-warning/10 text-warning'"
      >
        <ShieldCheck v-if="activeKey" class="h-5 w-5" :stroke-width="1.8" />
        <ShieldAlert v-else class="h-5 w-5" :stroke-width="1.8" />
      </span>
      <div class="min-w-0 flex-1">
        <template v-if="activeKey">
          <p class="text-sm font-medium text-ink">Signing with {{ activeKey.name }}</p>
          <p class="mt-0.5 truncate font-mono text-xs text-ink-muted">{{ activeKey.kid }}</p>
        </template>
        <template v-else>
          <p class="text-sm font-medium text-ink">No active signing key</p>
          <p class="mt-0.5 text-xs text-ink-muted">Offline tokens can't be issued until you create and activate one.</p>
        </template>
      </div>
      <CopyButton v-if="exportText" :value="exportText" label="Copy public keys" class="shrink-0" />
    </div>

    <form class="mt-5 flex gap-2" @submit.prevent="create">
      <input v-model="name" placeholder="Key name, e.g. desktop-2026" class="field" />
      <button class="btn-primary shrink-0" :disabled="creating">{{ creating ? 'Creating…' : 'Create key' }}</button>
    </form>

    <div class="card mt-5 p-2">
      <p v-if="loadError" class="px-4 py-8 text-center text-sm text-critical">{{ loadError }}</p>
      <p v-else-if="!loading && signingKeys.length === 0" class="px-4 py-10 text-center text-sm text-ink-muted">No signing keys yet.</p>
      <div v-else class="overflow-x-auto">
        <table class="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Created</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="key in signingKeys" :key="key.id" :class="key.status === 'retired' ? 'opacity-55' : ''">
              <td class="w-full min-w-[14rem]">
                <p class="font-medium text-ink">{{ key.name }}</p>
                <p class="mt-0.5 truncate font-mono text-xs text-ink-faint">{{ key.kid }}</p>
              </td>
              <td><StatusBadge :status="key.status" /></td>
              <td class="whitespace-nowrap text-ink-muted">{{ formatDate(key.created_at) }}</td>
              <td class="whitespace-nowrap text-right">
                <div v-if="key.status !== 'retired'" class="flex justify-end gap-1.5">
                  <button v-if="key.status !== 'active'" class="btn-secondary px-2.5 py-1 text-xs" @click="activate(key)">Make active</button>
                  <ConfirmButton size="sm" label="Retire" confirm-label="Confirm" @confirm="retire(key)" />
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <details v-if="exportText" class="group mt-6">
      <summary class="eyebrow flex cursor-pointer select-none list-none items-center gap-1 transition hover:text-ink [&::-webkit-details-marker]:hidden">
        <ChevronRight class="h-3.5 w-3.5 transition-transform group-open:rotate-90" :stroke-width="2.2" /> Public keys (PEM)
      </summary>
      <pre class="code-block mt-3 max-h-80">{{ exportText }}</pre>
    </details>
  </section>
</template>
