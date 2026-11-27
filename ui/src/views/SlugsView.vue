<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { Layers, Plus } from 'lucide-vue-next'
import PageHeader from '@/components/PageHeader.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import ToggleSwitch from '@/components/ToggleSwitch.vue'
import UiDrawer from '@/components/UiDrawer.vue'
import { slugAPI } from '@/stores/api'
import { toast, toastError } from '@/lib/ui'
import { errorMessage, formatExpiration, formatLifetime, hasKeys, parseJSONObject, prettyJSON } from '@/lib/format'

const slugs = ref([])
const loading = ref(false)
const loadError = ref('')
const showArchived = ref(false)

const drawerOpen = ref(false)
const editing = ref(null)
const saving = ref(false)
const formError = ref('')

const emptyForm = () => ({
  name: '',
  max_activations: 1,
  expiration_type: 'forever',
  expiration_days: 365,
  fixed_expires_at: '',
  offline_enabled: false,
  offline_token_lifetime_hours: 24,
  attributes: '',
})

const form = reactive(emptyForm())

const expirationOptions = [
  { value: 'forever', label: 'Never' },
  { value: 'duration', label: 'After a period' },
  { value: 'fixed_date', label: 'On a date' },
]

const archivedCount = computed(() => slugs.value.filter((slug) => slug.deleted_at).length)

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await slugAPI.list({ include_archived: showArchived.value })
    slugs.value = res.data.slugs || []
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load slugs')
  } finally {
    loading.value = false
  }
}

// Dates are entered as a calendar day and expire at the end of that day, local time.
const toDateInputValue = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 10)
}

const endOfLocalDay = (value) => {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null
  const [, year, month, day] = match.map(Number)
  const date = new Date(year, month - 1, day, 23, 59, 59)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) return null
  return date
}

const openCreate = () => {
  Object.assign(form, emptyForm())
  editing.value = null
  formError.value = ''
  drawerOpen.value = true
}

const openEdit = (slug) => {
  if (slug.deleted_at) return
  Object.assign(form, {
    name: slug.name,
    max_activations: slug.max_activations,
    expiration_type: slug.expiration_type,
    expiration_days: slug.expiration_days ?? 365,
    fixed_expires_at: toDateInputValue(slug.fixed_expires_at),
    offline_enabled: Boolean(slug.offline_enabled),
    offline_token_lifetime_hours: slug.offline_token_lifetime_hours || 24,
    attributes: hasKeys(slug.attributes) ? prettyJSON(slug.attributes) : '',
  })
  editing.value = slug
  formError.value = ''
  drawerOpen.value = true
}

const buildPayload = () => {
  const payload = {
    name: form.name.trim(),
    max_activations: Number(form.max_activations),
    expiration_type: form.expiration_type,
    offline_enabled: form.offline_enabled,
    offline_token_lifetime_hours: Number(form.offline_token_lifetime_hours),
    attributes: parseJSONObject(form.attributes, 'Attributes'),
  }

  if (form.expiration_type === 'duration') {
    payload.expiration_days = Number(form.expiration_days)
  }

  if (form.expiration_type === 'fixed_date') {
    const date = endOfLocalDay(form.fixed_expires_at || '')
    if (!date) throw new Error('Pick an expiration date')
    payload.fixed_expires_at = date.toISOString()
  }

  return payload
}

const save = async () => {
  formError.value = ''
  saving.value = true
  try {
    const payload = buildPayload()
    if (editing.value) {
      await slugAPI.update(editing.value.name, payload)
      toast(`Saved ${payload.name}`)
    } else {
      await slugAPI.create(payload)
      toast(`Created ${payload.name}`)
    }
    drawerOpen.value = false
    await load()
  } catch (err) {
    formError.value = errorMessage(err, 'Could not save slug')
  } finally {
    saving.value = false
  }
}

const remove = async () => {
  if (!editing.value) return
  const name = editing.value.name
  try {
    await slugAPI.delete(name)
    toast(`Archived ${name}`)
    drawerOpen.value = false
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not delete slug'))
  }
}

const toggleArchived = () => {
  showArchived.value = !showArchived.value
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Slugs" description="Templates for new licenses. A slug sets how many devices a key allows, when it expires, and what it unlocks.">
      <button class="btn-primary" @click="openCreate">
        <Plus class="h-4 w-4" :stroke-width="2.4" /> New slug
      </button>
    </PageHeader>

    <section class="card mt-8 p-2">
      <p v-if="loadError" class="px-4 py-10 text-center text-sm text-critical">{{ loadError }}</p>

      <div v-else-if="!loading && slugs.length === 0" class="flex flex-col items-center px-4 py-16 text-center">
        <span class="flex h-11 w-11 items-center justify-center rounded-xl bg-surface-container text-ink-muted shadow-ghost">
          <Layers class="h-5 w-5" :stroke-width="1.8" />
        </span>
        <p class="mt-4 font-medium text-ink">No slugs</p>
        <p class="mt-1 text-sm text-ink-muted">Create one to define seats and expiry for new licenses.</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="data-table" :class="loading ? 'opacity-60 transition-opacity' : ''">
          <thead>
            <tr>
              <th>Name</th>
              <th>Seats</th>
              <th>Expiry</th>
              <th>Offline tokens</th>
              <th class="text-right">Attributes</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="slug in slugs"
              :key="slug.id"
              :class="slug.deleted_at ? 'opacity-50' : 'is-clickable'"
              @click="openEdit(slug)"
            >
              <td class="w-full">
                <div class="flex items-center gap-2.5">
                  <span class="font-medium text-ink-strong" :class="slug.deleted_at ? 'line-through' : ''">{{ slug.name }}</span>
                  <StatusBadge v-if="slug.is_default && slug.name !== 'default'" status="active" label="default" />
                  <StatusBadge v-if="slug.deleted_at" status="archived" />
                </div>
              </td>
              <td class="whitespace-nowrap tabular-nums text-ink-soft">{{ slug.max_activations }}</td>
              <td class="whitespace-nowrap text-ink-soft">{{ formatExpiration(slug) }}</td>
              <td class="whitespace-nowrap text-ink-soft">
                <span v-if="slug.offline_enabled">{{ formatLifetime(slug.offline_token_lifetime_hours) }}</span>
                <span v-else class="text-ink-faint">Off</span>
              </td>
              <td class="whitespace-nowrap text-right tabular-nums text-ink-muted">
                {{ hasKeys(slug.attributes) ? Object.keys(slug.attributes).length : '—' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <div class="mt-3 flex justify-end">
      <button class="btn-ghost px-2.5 py-1 text-xs" @click="toggleArchived">
        {{ showArchived ? `Hide archived${archivedCount ? ` (${archivedCount})` : ''}` : 'Show archived' }}
      </button>
    </div>

    <UiDrawer :open="drawerOpen" :eyebrow="editing ? 'Edit slug' : 'New slug'" :title="editing ? editing.name : 'Create a slug'" @close="drawerOpen = false">
      <form id="slug-form" class="space-y-5" @submit.prevent="save">
        <div class="grid grid-cols-[minmax(0,1fr)_7rem] gap-3">
          <label class="block">
            <span class="field-label">Name</span>
            <input v-model="form.name" required placeholder="pro-yearly" class="field font-mono" />
          </label>
          <label class="block">
            <span class="field-label">Seats</span>
            <input v-model="form.max_activations" min="1" type="number" required class="field tabular-nums" />
          </label>
        </div>

        <div>
          <span class="field-label">Expires</span>
          <div class="grid grid-cols-3 gap-1 rounded-lg bg-surface-lowest/70 p-1 shadow-ghost">
            <button
              v-for="option in expirationOptions"
              :key="option.value"
              type="button"
              class="rounded-md px-2 py-1.5 text-xs font-medium transition"
              :class="form.expiration_type === option.value ? 'bg-surface-high text-ink-strong shadow-ghost' : 'text-ink-muted hover:text-ink'"
              @click="form.expiration_type = option.value"
            >
              {{ option.label }}
            </button>
          </div>

          <label v-if="form.expiration_type === 'duration'" class="mt-3 flex items-center gap-3">
            <input v-model="form.expiration_days" min="1" type="number" required class="field w-28 tabular-nums" />
            <span class="text-sm text-ink-muted">days after the license is issued</span>
          </label>

          <label v-if="form.expiration_type === 'fixed_date'" class="mt-3 block">
            <input v-model="form.fixed_expires_at" type="date" required class="field" />
            <span class="field-hint">Every license from this slug expires at the end of this day.</span>
          </label>
        </div>

        <div>
          <ToggleSwitch
            v-model="form.offline_enabled"
            label="Offline tokens"
            hint="Activations get a signed JWT the app can verify without calling the server."
          />
          <label v-if="form.offline_enabled" class="mt-3 flex items-center gap-3">
            <input v-model="form.offline_token_lifetime_hours" min="1" type="number" required class="field w-28 tabular-nums" />
            <span class="text-sm text-ink-muted">hours before a token must be refreshed</span>
          </label>
        </div>

        <label class="block">
          <span class="field-label">Attributes <span class="text-ink-faint">· optional</span></span>
          <textarea
            v-model="form.attributes"
            rows="9"
            spellcheck="false"
            class="field font-mono text-xs leading-relaxed"
            placeholder='{ "features": ["export", "sync"] }'
          ></textarea>
          <span class="field-hint">Copied onto each license when it is issued and included in offline tokens.</span>
        </label>

        <p v-if="formError" class="rounded-lg bg-critical/10 px-3 py-2 text-sm text-critical">{{ formError }}</p>
      </form>

      <template #footer>
        <ConfirmButton
          v-if="editing && !editing.is_default"
          label="Delete"
          confirm-label="Confirm delete"
          @confirm="remove"
        />
        <button class="btn-ghost ml-auto" @click="drawerOpen = false">Cancel</button>
        <button class="btn-primary" type="submit" form="slug-form" :disabled="saving">
          {{ saving ? 'Saving…' : editing ? 'Save changes' : 'Create slug' }}
        </button>
      </template>
    </UiDrawer>
  </div>
</template>
