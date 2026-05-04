<script setup>
import { onMounted, reactive, ref } from 'vue'
import { webhooksAPI } from '@/stores/api'

const supportedEvents = [
  'license.generated',
  'license.activated',
  'license.deactivated',
  'license.validated',
  'license.validation_failed',
  'license.revoked',
]

const webhooks = ref([])
const deliveries = ref([])
const loading = ref(false)
const deliveriesLoading = ref(false)
const saving = ref(false)
const editingId = ref(0)
const error = ref('')
const deliveriesError = ref('')

const form = reactive({
  name: '',
  url: '',
  enabled: true,
  events: ['license.generated'],
})

const resetForm = () => {
  form.name = ''
  form.url = ''
  form.enabled = true
  form.events = ['license.generated']
  editingId.value = 0
}

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await webhooksAPI.list()
    webhooks.value = res.data.webhooks || []
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to load webhooks'
  } finally {
    loading.value = false
  }
}

const loadDeliveries = async () => {
  deliveriesLoading.value = true
  deliveriesError.value = ''
  try {
    const res = await webhooksAPI.deliveries(25)
    deliveries.value = res.data.deliveries || []
  } catch (err) {
    deliveriesError.value = err?.response?.data?.error || 'Failed to load webhook run log'
  } finally {
    deliveriesLoading.value = false
  }
}

const submit = async () => {
  saving.value = true
  error.value = ''
  try {
    const payload = {
      name: form.name.trim(),
      url: form.url.trim(),
      events: form.events,
      enabled: form.enabled,
    }

    if (editingId.value) {
      await webhooksAPI.update(editingId.value, payload)
    } else {
      await webhooksAPI.create(payload)
    }

    resetForm()
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to save webhook'
  } finally {
    saving.value = false
  }
}

const editWebhook = (item) => {
  editingId.value = item.id
  form.name = item.name
  form.url = item.url
  form.enabled = item.enabled
  form.events = [...item.events]
}

const removeWebhook = async (id) => {
  if (!confirm('Delete webhook endpoint?')) {
    return
  }
  error.value = ''
  try {
    await webhooksAPI.delete(id)
    await load()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to delete webhook'
  }
}

const eventLabel = (event) => event.replace('license.', '')

const formatDateTime = (value) => {
  if (!value) return 'Not set'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Not set'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

const deliveryStatusClass = (status) => {
  if (status === 'delivered') return 'bg-[#4ae176]/10 text-[#4ae176]'
  if (status === 'failed') return 'bg-[#ff7a8a]/10 text-[#ff7a8a]'
  if (status === 'sending') return 'bg-[#a4c9ff]/10 text-[#a4c9ff]'
  return 'bg-[#f2b36d]/10 text-[#f2b36d]'
}

onMounted(() => {
  load()
  loadDeliveries()
})
</script>

<template>
  <div class="grid gap-5 xl:grid-cols-[24rem_minmax(0,1fr)]">
    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <p class="metadata-label text-xs font-bold text-[#8f98ad]">Webhook Endpoint</p>
      <h1 class="mt-2 text-xl font-black tracking-tight text-white">{{ editingId ? 'Edit webhook.' : 'Create webhook.' }}</h1>
      <p v-if="error" class="mt-4 text-sm text-[#ff7a8a]">{{ error }}</p>

      <form class="mt-5 space-y-4" @submit.prevent="submit">
        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Name</span>
          <input v-model="form.name" required placeholder="Name" class="field-control text-sm" />
        </label>
        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">URL</span>
          <input v-model="form.url" required placeholder="https://example.com/hooks" class="field-control text-sm" />
        </label>

        <label class="inline-flex items-center gap-2 text-sm text-[#c6ccdc]">
          <input v-model="form.enabled" type="checkbox" class="accent-[#a4c9ff]" />
          <span>Enabled</span>
        </label>

        <div>
          <p class="mb-2 text-xs font-semibold text-white">Events</p>
          <div class="grid grid-cols-1 gap-2">
            <label v-for="event in supportedEvents" :key="event" class="inline-flex items-center gap-2 rounded-md bg-[#1c1f2a] px-3 py-2 text-sm text-[#c6ccdc]">
              <input v-model="form.events" type="checkbox" :value="event" class="accent-[#a4c9ff]" />
              <span>{{ event }}</span>
            </label>
          </div>
        </div>

        <div class="flex gap-2 pt-1">
          <button :disabled="saving" class="primary-action px-4 py-2 text-sm disabled:opacity-60">
            {{ editingId ? 'Update Webhook' : 'Create Webhook' }}
          </button>
          <button v-if="editingId" type="button" class="secondary-action px-4 py-2 text-sm" @click="resetForm">Cancel</button>
        </div>
      </form>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Configured webhooks.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">{{ webhooks.length }} endpoints configured</p>
        </div>
        <button class="secondary-action px-3 py-2 text-xs" @click="load">Refresh</button>
      </div>

      <p v-if="loading" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">Loading webhooks...</p>
      <p v-else-if="webhooks.length === 0" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">No webhooks found.</p>

      <div v-else class="mt-4 overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="metadata-label text-xs text-[#8f98ad]">
            <tr>
              <th class="px-3 py-3 font-bold">Name</th>
              <th class="px-3 py-3 font-bold">URL</th>
              <th class="px-3 py-3 font-bold">Events</th>
              <th class="px-3 py-3 font-bold">Status</th>
              <th class="px-3 py-3 text-right font-bold">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="hook in webhooks" :key="hook.id" class="group">
              <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 font-semibold text-[#dfe2f1] group-hover:bg-[#262a35]">{{ hook.name }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]"><code class="text-xs text-[#a4c9ff]">{{ hook.url }}</code></td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <div class="flex flex-wrap gap-1.5">
                  <span v-for="event in hook.events" :key="event" class="rounded-md bg-[#a4c9ff]/10 px-2 py-1 text-xs text-[#a4c9ff]">{{ eventLabel(event) }}</span>
                </div>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="hook.enabled ? 'bg-[#4ae176]/10 text-[#4ae176]' : 'bg-[#8f98ad]/10 text-[#8f98ad]'">
                  {{ hook.enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td class="rounded-r-md bg-[#1c1f2a] px-3 py-3 text-right group-hover:bg-[#262a35]">
                <button class="metadata-label text-[0.65rem] font-bold text-[#a4c9ff] transition hover:text-white" @click="editWebhook(hook)">Edit</button>
                <button class="ml-4 metadata-label text-[0.65rem] font-bold text-[#ff7a8a] transition hover:text-white" @click="removeWebhook(hook.id)">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5 xl:col-span-2">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Webhook run log.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">Recent delivery attempts for enabled webhook endpoints</p>
        </div>
        <button class="secondary-action px-3 py-2 text-xs" @click="loadDeliveries">Refresh Log</button>
      </div>

      <p v-if="deliveriesError" class="mt-4 text-sm text-[#ff7a8a]">{{ deliveriesError }}</p>
      <p v-if="deliveriesLoading" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">Loading webhook run log...</p>
      <p v-else-if="deliveries.length === 0" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">No webhook deliveries recorded yet.</p>

      <div v-else class="mt-4 overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="metadata-label text-xs text-[#8f98ad]">
            <tr>
              <th class="px-3 py-3 font-bold">Delivery</th>
              <th class="px-3 py-3 font-bold">Endpoint</th>
              <th class="px-3 py-3 font-bold">Event</th>
              <th class="px-3 py-3 font-bold">Status</th>
              <th class="px-3 py-3 font-bold">Attempts</th>
              <th class="px-3 py-3 font-bold">HTTP</th>
              <th class="px-3 py-3 font-bold">Updated</th>
              <th class="px-3 py-3 font-bold">Error</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="delivery in deliveries" :key="delivery.id" class="group">
              <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 font-semibold text-[#dfe2f1] group-hover:bg-[#262a35]">#{{ delivery.id }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <p class="font-semibold text-[#dfe2f1]">{{ delivery.endpoint_name }}</p>
                <code class="mt-1 block max-w-[18rem] truncate text-xs text-[#8f98ad]">{{ delivery.endpoint_url }}</code>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="rounded-md bg-[#a4c9ff]/10 px-2 py-1 text-xs text-[#a4c9ff]">{{ eventLabel(delivery.event_type) }}</span>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="deliveryStatusClass(delivery.status)">
                  {{ delivery.status }}
                </span>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ delivery.attempts }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ delivery.last_response_status || 'none' }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ formatDateTime(delivery.updated_at) }}</td>
              <td class="rounded-r-md bg-[#1c1f2a] px-3 py-3 text-[#8f98ad] group-hover:bg-[#262a35]">
                <span class="block max-w-[22rem] truncate" :title="delivery.last_error || ''">{{ delivery.last_error || 'none' }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
