<script setup>
import { onMounted, ref } from 'vue'
import { TriangleAlert, X } from 'lucide-vue-next'
import StatusBadge from '@/components/StatusBadge.vue'
import CopyButton from '@/components/CopyButton.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import { apiKeysAPI } from '@/stores/api'
import { toast, toastError } from '@/lib/ui'
import { errorMessage, formatDate } from '@/lib/format'

const keys = ref([])
const loading = ref(false)
const loadError = ref('')
const name = ref('')
const creating = ref(false)
const newKey = ref('')

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await apiKeysAPI.list()
    keys.value = res.data.api_keys || []
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load API keys')
  } finally {
    loading.value = false
  }
}

const create = async () => {
  creating.value = true
  try {
    const res = await apiKeysAPI.create(name.value.trim())
    newKey.value = res.data.api_key || ''
    name.value = ''
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not create API key'))
  } finally {
    creating.value = false
  }
}

const revoke = async (key) => {
  try {
    await apiKeysAPI.revoke(key.id)
    toast(`Revoked ${key.name || 'key'}`)
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not revoke API key'))
  }
}

onMounted(load)
</script>

<template>
  <section>
    <h2 class="text-lg font-semibold tracking-tight text-ink-strong">Server API keys</h2>
    <p class="mt-1 text-sm leading-6 text-ink-muted">
      Your backend uses these to call <code class="font-mono text-xs text-ink-soft">/generate</code> and
      <code class="font-mono text-xs text-ink-soft">/revoke</code>, for example from a payment webhook.
    </p>

    <form class="mt-5 flex gap-2" @submit.prevent="create">
      <input v-model="name" placeholder="Key name, e.g. stripe-webhook" class="field" />
      <button class="btn-primary shrink-0" :disabled="creating">{{ creating ? 'Creating…' : 'Create key' }}</button>
    </form>

    <Transition
      enter-active-class="transition duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
      enter-from-class="-translate-y-1 opacity-0"
    >
      <div v-if="newKey" class="mt-4 rounded-xl bg-success/[0.06] p-4 shadow-[inset_0_0_0_1px_rgba(74,225,118,0.25)]">
        <div class="flex items-start gap-3">
          <TriangleAlert class="mt-0.5 h-4 w-4 shrink-0 text-warning" :stroke-width="2" />
          <p class="flex-1 text-sm text-ink">Copy this key now. It will not be shown again.</p>
          <button class="text-ink-faint transition hover:text-ink" aria-label="Dismiss" @click="newKey = ''">
            <X class="h-4 w-4" :stroke-width="2" />
          </button>
        </div>
        <div class="mt-3 flex items-center gap-2 rounded-lg bg-surface-lowest py-1.5 pl-3 pr-1.5 shadow-ghost">
          <code class="flex-1 break-all font-mono text-xs text-success">{{ newKey }}</code>
          <CopyButton :value="newKey" />
        </div>
      </div>
    </Transition>

    <div class="card mt-5 p-2">
      <p v-if="loadError" class="px-4 py-8 text-center text-sm text-critical">{{ loadError }}</p>
      <p v-else-if="!loading && keys.length === 0" class="px-4 py-10 text-center text-sm text-ink-muted">No server keys yet.</p>
      <div v-else class="overflow-x-auto">
        <table class="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Key</th>
              <th>Created</th>
              <th>Status</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="key in keys" :key="key.id" :class="key.revoked_at ? 'opacity-55' : ''">
              <td class="w-full font-medium text-ink">{{ key.name || 'Unnamed' }}</td>
              <td class="whitespace-nowrap font-mono text-xs text-ink-muted"><span class="text-ink-faint">••••</span>{{ key.hint }}</td>
              <td class="whitespace-nowrap text-ink-muted">{{ formatDate(key.created_at) }}</td>
              <td><StatusBadge :status="key.revoked_at ? 'revoked' : 'active'" /></td>
              <td class="text-right">
                <ConfirmButton v-if="!key.revoked_at" size="sm" label="Revoke" confirm-label="Confirm" @confirm="revoke(key)" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>
