<script setup>
import { onMounted, reactive, ref } from 'vue'
import { Plus, RefreshCw } from 'lucide-vue-next'
import StatusBadge from '@/components/StatusBadge.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import UiDrawer from '@/components/UiDrawer.vue'
import { webhooksAPI } from '@/stores/api'
import { toast, toastError } from '@/lib/ui'
import { errorMessage, formatRelative } from '@/lib/format'

const supportedEvents = [
  'license.generated',
  'license.activated',
  'license.deactivated',
  'license.validated',
  'license.validation_failed',
  'license.revoked',
  'license.updated',
]

const webhooks = ref([])
const deliveries = ref([])
const loading = ref(false)
const loadError = ref('')
const deliveriesLoading = ref(false)

const drawerOpen = ref(false)
const editing = ref(null)
const saving = ref(false)
const formError = ref('')
const form = reactive({ name: '', url: '', enabled: true, events: [] })

const eventLabel = (event) => event.replace('license.', '').replace(/_/g, ' ')

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await webhooksAPI.list()
    webhooks.value = res.data.webhooks || []
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load webhooks')
  } finally {
    loading.value = false
  }
}

const loadDeliveries = async () => {
  deliveriesLoading.value = true
  try {
    const res = await webhooksAPI.deliveries(20)
    deliveries.value = res.data.deliveries || []
  } catch (err) {
    toastError(errorMessage(err, 'Failed to load deliveries'))
  } finally {
    deliveriesLoading.value = false
  }
}

const openCreate = () => {
  Object.assign(form, { name: '', url: '', enabled: true, events: ['license.generated'] })
  editing.value = null
  formError.value = ''
  drawerOpen.value = true
}

const openEdit = (hook) => {
  Object.assign(form, { name: hook.name, url: hook.url, enabled: hook.enabled, events: [...hook.events] })
  editing.value = hook
  formError.value = ''
  drawerOpen.value = true
}

const toggleEvent = (event) => {
  const index = form.events.indexOf(event)
  if (index === -1) form.events.push(event)
  else form.events.splice(index, 1)
}

const save = async () => {
  formError.value = ''
  if (form.events.length === 0) {
    formError.value = 'Pick at least one event'
    return
  }
  saving.value = true
  try {
    const payload = { name: form.name.trim(), url: form.url.trim(), events: form.events, enabled: form.enabled }
    if (editing.value) await webhooksAPI.update(editing.value.id, payload)
    else await webhooksAPI.create(payload)
    toast(editing.value ? 'Webhook saved' : 'Webhook added')
    drawerOpen.value = false
    await load()
  } catch (err) {
    formError.value = errorMessage(err, 'Could not save webhook')
  } finally {
    saving.value = false
  }
}

const remove = async () => {
  if (!editing.value) return
  try {
    await webhooksAPI.delete(editing.value.id)
    toast('Webhook deleted')
    drawerOpen.value = false
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not delete webhook'))
  }
}

onMounted(() => {
  load()
  loadDeliveries()
})
</script>

<template>
  <section>
    <div class="flex items-start justify-between gap-4">
      <div>
        <h2 class="text-lg font-semibold tracking-tight text-ink-strong">Webhooks</h2>
        <p class="mt-1 text-sm leading-6 text-ink-muted">Get a POST request when something happens to a license.</p>
      </div>
      <button class="btn-secondary shrink-0" @click="openCreate">
        <Plus class="h-4 w-4" :stroke-width="2.2" /> Add endpoint
      </button>
    </div>

    <div class="card mt-5 p-2">
      <p v-if="loadError" class="px-4 py-8 text-center text-sm text-critical">{{ loadError }}</p>
      <p v-else-if="!loading && webhooks.length === 0" class="px-4 py-10 text-center text-sm text-ink-muted">No endpoints yet.</p>
      <ul v-else>
        <li
          v-for="(hook, index) in webhooks"
          :key="hook.id"
          class="flex cursor-pointer items-center gap-4 rounded-lg px-4 py-3.5 transition hover:bg-surface-container/80"
          :class="index > 0 ? 'shadow-[inset_0_1px_0_rgba(65,71,81,0.16)]' : ''"
          @click="openEdit(hook)"
        >
          <span class="h-2 w-2 shrink-0 rounded-full" :class="hook.enabled ? 'bg-success shadow-[0_0_8px_#4ae176]' : 'bg-ink-faint'"></span>
          <div class="min-w-0 flex-1">
            <p class="font-medium text-ink">{{ hook.name }}</p>
            <p class="mt-0.5 truncate font-mono text-xs text-ink-muted">{{ hook.url }}</p>
          </div>
          <p class="hidden shrink-0 text-xs text-ink-faint sm:block">
            {{ hook.events.length === supportedEvents.length ? 'All events' : `${hook.events.length} event${hook.events.length === 1 ? '' : 's'}` }}
          </p>
        </li>
      </ul>
    </div>

    <div class="mt-10 flex items-center justify-between">
      <h3 class="eyebrow">Recent deliveries</h3>
      <button class="btn-icon" :disabled="deliveriesLoading" aria-label="Refresh deliveries" @click="loadDeliveries">
        <RefreshCw class="h-3.5 w-3.5" :class="deliveriesLoading ? 'animate-spin' : ''" :stroke-width="2" />
      </button>
    </div>

    <div class="card mt-3 p-2">
      <p v-if="deliveries.length === 0" class="px-4 py-8 text-center text-sm text-ink-muted">Nothing delivered yet.</p>
      <div v-else class="overflow-x-auto">
        <table class="data-table">
          <tbody>
            <tr v-for="delivery in deliveries" :key="delivery.id">
              <td class="w-full min-w-[14rem]">
                <p class="text-ink">
                  <span class="font-medium capitalize">{{ eventLabel(delivery.event_type) }}</span>
                  <span class="text-ink-faint"> → {{ delivery.endpoint_name }}</span>
                </p>
                <p v-if="delivery.last_error" class="mt-0.5 max-w-md truncate font-mono text-xs text-critical/80" :title="delivery.last_error">
                  {{ delivery.last_error }}
                </p>
              </td>
              <td><StatusBadge :status="delivery.status" /></td>
              <td class="whitespace-nowrap font-label text-xs tabular-nums text-ink-muted">
                <span v-if="delivery.last_response_status">HTTP {{ delivery.last_response_status }}</span>
                <span v-if="delivery.last_response_status && delivery.attempts > 1" class="text-ink-faint"> · </span>
                <span v-if="delivery.attempts > 1" class="text-ink-faint">{{ delivery.attempts }} tries</span>
              </td>
              <td class="whitespace-nowrap text-right text-xs text-ink-muted">{{ formatRelative(delivery.updated_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <UiDrawer :open="drawerOpen" :eyebrow="editing ? 'Edit endpoint' : 'New endpoint'" :title="editing ? editing.name : 'Add a webhook'" @close="drawerOpen = false">
      <form id="webhook-form" class="space-y-5" @submit.prevent="save">
        <label class="block">
          <span class="field-label">Name</span>
          <input v-model="form.name" required placeholder="Billing system" class="field" />
        </label>
        <label class="block">
          <span class="field-label">URL</span>
          <input v-model="form.url" required type="url" placeholder="https://example.com/hooks/licenses" class="field font-mono text-xs" />
        </label>

        <div>
          <span class="field-label">Events</span>
          <div class="flex flex-wrap gap-1.5">
            <button
              v-for="event in supportedEvents"
              :key="event"
              type="button"
              class="rounded-md px-2.5 py-1.5 text-xs font-medium capitalize transition"
              :class="form.events.includes(event) ? 'bg-primary/15 text-primary shadow-[inset_0_0_0_1px_rgba(164,201,255,0.35)]' : 'bg-surface-container text-ink-muted shadow-ghost hover:text-ink'"
              @click="toggleEvent(event)"
            >
              {{ eventLabel(event) }}
            </button>
          </div>
        </div>

        <ToggleSwitch v-model="form.enabled" label="Enabled" hint="Paused endpoints keep their settings but receive nothing." />

        <p v-if="formError" class="rounded-lg bg-critical/10 px-3 py-2 text-sm text-critical">{{ formError }}</p>
      </form>

      <template #footer>
        <ConfirmButton v-if="editing" label="Delete" confirm-label="Confirm delete" @confirm="remove" />
        <button class="btn-ghost ml-auto" @click="drawerOpen = false">Cancel</button>
        <button class="btn-primary" type="submit" form="webhook-form" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save changes' : 'Add endpoint' }}
        </button>
      </template>
    </UiDrawer>
  </section>
</template>
