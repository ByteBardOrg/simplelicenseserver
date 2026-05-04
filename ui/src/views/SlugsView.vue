<script setup>
import { onMounted, reactive, ref } from 'vue'
import { slugAPI } from '@/stores/api'

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const slugs = ref([])
const editing = ref('')
const showArchived = ref(false)

const form = reactive({
  name: '',
  max_activations: 1,
  expiration_type: 'forever',
  expiration_days: '',
  fixed_expires_at: '',
  offline_enabled: false,
  offline_token_lifetime_hours: 24,
})

const resetForm = () => {
  form.name = ''
  form.max_activations = 1
  form.expiration_type = 'forever'
  form.expiration_days = ''
  form.fixed_expires_at = ''
  form.offline_enabled = false
  form.offline_token_lifetime_hours = 24
  editing.value = ''
}

const parseLocalExpirationDate = (value) => {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/)
  if (!match) return null

  const [, year, month, day] = match.map(Number)
  const date = new Date(year, month - 1, day, 23, 59, 59)
  if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
    return null
  }

  return date
}

const toDateInputValue = (value) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  const offsetDate = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return offsetDate.toISOString().slice(0, 10)
}

const formatDate = (value) => {
  if (!value) return 'Not set'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Not set'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date)
}

const buildPayload = () => {
  const payload = {
    name: form.name.trim(),
    max_activations: Number(form.max_activations),
    expiration_type: form.expiration_type,
    offline_enabled: form.offline_enabled,
    offline_token_lifetime_hours: Number(form.offline_token_lifetime_hours),
  }

  if (form.expiration_type === 'duration' && form.expiration_days !== '') {
    payload.expiration_days = Number(form.expiration_days)
  }

  if (form.expiration_type === 'fixed_date') {
    if (!form.fixed_expires_at) {
      throw new Error('Enter a fixed expiration date')
    }
    const fixedDate = parseLocalExpirationDate(form.fixed_expires_at)
    if (!fixedDate) {
      throw new Error('Enter a valid fixed expiration date')
    }
    payload.fixed_expires_at = fixedDate.toISOString()
  }

  return payload
}

const loadSlugs = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await slugAPI.list({ include_archived: showArchived.value })
    slugs.value = res.data.slugs || []
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to load slugs'
  } finally {
    loading.value = false
  }
}

const submit = async () => {
  saving.value = true
  error.value = ''
  try {
    if (editing.value) {
      await slugAPI.update(editing.value, buildPayload())
    } else {
      await slugAPI.create(buildPayload())
    }
    resetForm()
    await loadSlugs()
  } catch (err) {
    error.value = err?.response?.data?.error || err?.message || 'Failed to save slug'
  } finally {
    saving.value = false
  }
}

const editSlug = (slug) => {
  if (slug.deleted_at) {
    return
  }
  editing.value = slug.name
  form.name = slug.name
  form.max_activations = slug.max_activations
  form.expiration_type = slug.expiration_type
  form.expiration_days = slug.expiration_days ?? ''
  form.fixed_expires_at = toDateInputValue(slug.fixed_expires_at)
  form.offline_enabled = Boolean(slug.offline_enabled)
  form.offline_token_lifetime_hours = slug.offline_token_lifetime_hours || 24
}

const removeSlug = async (slug) => {
  if (slug.deleted_at) {
    return
  }
  if (!confirm(`Delete slug "${slug.name}"?`)) {
    return
  }

  error.value = ''
  try {
    await slugAPI.delete(slug.name)
    await loadSlugs()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Failed to delete slug'
  }
}

const formatExpiration = (slug) => {
  if (slug.expiration_type === 'duration') return `${slug.expiration_days} days`
  if (slug.expiration_type === 'fixed_date') return formatDate(slug.fixed_expires_at)
  return 'Never expires'
}

const formatOfflineLifetime = (hours) => {
  const value = Number(hours || 0)
  if (value <= 0) return 'Not set'
  if (value % 24 === 0) return `${value / 24}d`
  return `${value}h`
}

onMounted(loadSlugs)
</script>

<template>
  <div class="grid gap-5 xl:grid-cols-[22rem_minmax(0,1fr)]">
    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <p class="metadata-label text-xs font-bold text-[#8f98ad]">Slug Template</p>
      <h1 class="mt-2 text-xl font-black tracking-tight text-white">{{ editing ? 'Edit slug.' : 'Create slug.' }}</h1>
      <p class="mt-2 text-sm leading-6 text-[#8f98ad]">Slug settings define generated license limits and expiration behavior.</p>
      <p v-if="error" class="mt-4 text-sm text-[#ff7a8a]">{{ error }}</p>

      <form class="mt-5 space-y-4" @submit.prevent="submit">
        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Name</span>
          <input v-model="form.name" required class="field-control text-sm" />
        </label>

        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Max Activations</span>
          <input v-model="form.max_activations" min="1" type="number" required class="field-control text-sm" />
        </label>

        <label class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Expiration Type</span>
          <select v-model="form.expiration_type" class="field-control text-sm">
            <option value="forever">forever</option>
            <option value="duration">duration</option>
            <option value="fixed_date">fixed_date</option>
          </select>
        </label>

        <label v-if="form.expiration_type === 'duration'" class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Expiration Days</span>
          <input v-model="form.expiration_days" min="1" type="number" class="field-control text-sm" />
        </label>

        <label v-if="form.expiration_type === 'fixed_date'" class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Fixed Expiration Date</span>
          <input v-model="form.fixed_expires_at" type="date" class="field-control text-sm" />
          <span class="mt-2 block text-xs text-[#8f98ad]">Expires automatically at 11:59:59 PM local time.</span>
        </label>

        <label class="flex items-start gap-3 rounded-md bg-[#1c1f2a] px-3 py-3 text-sm text-[#c6ccdc]">
          <input v-model="form.offline_enabled" type="checkbox" class="mt-1 accent-[#a4c9ff]" />
          <span>
            <span class="block text-xs font-semibold text-white">Offline JWTs</span>
            <span class="mt-1 block text-xs leading-5 text-[#8f98ad]">Allow activated seats for this slug to receive refreshed offline tokens.</span>
          </span>
        </label>

        <label v-if="form.offline_enabled" class="block text-sm text-[#c6ccdc]">
          <span class="mb-2 block text-xs font-semibold text-white">Offline Token Lifetime (hours)</span>
          <input v-model="form.offline_token_lifetime_hours" min="1" type="number" class="field-control text-sm" />
          <span class="mt-2 block text-xs text-[#8f98ad]">Default is 24 hours. License expiration still caps token expiration.</span>
        </label>

        <div class="flex gap-2 pt-1">
          <button :disabled="saving" class="primary-action px-4 py-2 text-sm disabled:opacity-60">
            {{ editing ? 'Update Slug' : 'Create Slug' }}
          </button>
          <button v-if="editing" type="button" class="secondary-action px-4 py-2 text-sm" @click="resetForm">Cancel</button>
        </div>
      </form>
    </section>

    <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 class="text-xl font-black tracking-tight text-white">Configured slugs.</h2>
          <p class="mt-1 text-sm text-[#8f98ad]">{{ slugs.length }} slug templates {{ showArchived ? 'loaded' : 'available' }}</p>
        </div>
        <div class="flex items-center gap-2">
          <label class="flex items-center gap-2 rounded-md bg-[#1c1f2a] px-3 py-2 text-xs text-[#c6ccdc]">
            <input v-model="showArchived" type="checkbox" class="accent-[#a4c9ff]" @change="loadSlugs" />
            Show archived
          </label>
          <button class="secondary-action px-3 py-2 text-xs" @click="loadSlugs">Refresh</button>
        </div>
      </div>

      <p v-if="loading" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">Loading slugs...</p>
      <p v-else-if="slugs.length === 0" class="mt-6 rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-sm text-[#8f98ad]">No slugs found.</p>

      <div v-else class="mt-4 overflow-x-auto">
        <table class="min-w-full text-left text-sm">
          <thead class="metadata-label text-xs text-[#8f98ad]">
            <tr>
              <th class="px-3 py-3 font-bold">Name</th>
              <th class="px-3 py-3 font-bold">Max</th>
              <th class="px-3 py-3 font-bold">Expiration</th>
              <th class="px-3 py-3 font-bold">Offline</th>
              <th class="px-3 py-3 font-bold">Token TTL</th>
              <th class="px-3 py-3 font-bold">Default</th>
              <th class="px-3 py-3 text-right font-bold">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="slug in slugs" :key="slug.id" class="group">
              <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 font-semibold group-hover:bg-[#262a35]" :class="slug.deleted_at ? 'text-[#8f98ad] line-through' : 'text-[#dfe2f1]'">
                {{ slug.name }}
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]" :class="slug.deleted_at ? 'text-[#8f98ad]' : 'text-[#dfe2f1]'">{{ slug.max_activations }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]" :class="slug.deleted_at ? 'text-[#8f98ad]' : 'text-[#dfe2f1]'">{{ formatExpiration(slug) }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="slug.offline_enabled ? 'bg-[#4ae176]/10 text-[#4ae176]' : 'bg-[#8f98ad]/10 text-[#8f98ad]'">
                  {{ slug.offline_enabled ? 'Enabled' : 'Disabled' }}
                </span>
              </td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]" :class="slug.deleted_at ? 'text-[#8f98ad]' : 'text-[#dfe2f1]'">{{ formatOfflineLifetime(slug.offline_token_lifetime_hours) }}</td>
              <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                <span class="inline-flex rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="slug.deleted_at ? 'bg-[#ffb347]/10 text-[#ffb347]' : (slug.is_default ? 'bg-[#4ae176]/10 text-[#4ae176]' : 'bg-[#a4c9ff]/10 text-[#a4c9ff]')">
                  {{ slug.deleted_at ? 'Archived' : (slug.is_default ? 'Default' : 'Custom') }}
                </span>
              </td>
              <td class="rounded-r-md bg-[#1c1f2a] px-3 py-3 text-right group-hover:bg-[#262a35]">
                <button class="metadata-label text-[0.65rem] font-bold text-[#a4c9ff] transition hover:text-white disabled:text-[#8f98ad] disabled:opacity-50" :disabled="Boolean(slug.deleted_at)" @click="editSlug(slug)">Edit</button>
                <button class="ml-4 metadata-label text-[0.65rem] font-bold text-[#ff7a8a] transition hover:text-white disabled:text-[#8f98ad] disabled:opacity-50" :disabled="slug.is_default || Boolean(slug.deleted_at)" @click="removeSlug(slug)">Delete</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>
