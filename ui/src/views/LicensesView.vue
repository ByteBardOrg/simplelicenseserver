<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { licensesAPI, slugAPI } from '@/stores/api'
import { ArrowDown, ArrowUp, Ban, Check, ChevronsUpDown, Clock3, Files, Search } from 'lucide-vue-next'

const result = ref('')
const error = ref('')
const listError = ref('')
const loadingLicenses = ref(false)
const licenses = ref([])
const showGenerateDrawer = ref(false)
const selectedLicense = ref(null)
const openActionMenu = ref(null)
const actionMenuLicense = ref(null)
const actionMenuPosition = reactive({ top: 0, left: 0 })
const copiedLicenseKey = ref('')
const loadingSlugs = ref(false)
const slugs = ref([])
const slugError = ref('')

const filters = reactive({
  q: '',
  status: '',
})

const sort = reactive({
  key: 'created_at',
  direction: 'desc',
})

const sortableHeaders = [
  { key: 'license_key', label: 'License Key' },
  { key: 'status', label: 'Status' },
  { key: 'slug', label: 'Slug' },
  { key: 'seats', label: 'Seats' },
  { key: 'created_at', label: 'Created' },
]

const statusSortOrder = {
  active: 1,
  inactive: 2,
  expired: 3,
  revoked: 4,
}

const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0,
  total_pages: 0,
})

const counts = reactive({
  total: 0,
  active: 0,
  inactive: 0,
  revoked: 0,
  expired: 0,
})

const generateForm = reactive({
  slug: 'default',
  metadata: '{"email":"user@example.com"}',
})

const revokeForm = reactive({
  license_key: '',
})

const selectedSlug = computed(() => slugs.value.find((slug) => slug.name === generateForm.slug) || null)
const sortedLicenses = computed(() => {
  return [...licenses.value].sort((left, right) => {
    const leftValue = sortValue(left, sort.key)
    const rightValue = sortValue(right, sort.key)
    const comparison = typeof leftValue === 'number' && typeof rightValue === 'number'
      ? leftValue - rightValue
      : String(leftValue).localeCompare(String(rightValue))

    return sort.direction === 'asc' ? comparison : -comparison
  })
})

const sortValue = (license, key) => {
  if (key === 'status') return statusSortOrder[license.status] || 99
  if (key === 'seats') return (Number(license.active_seats || 0) * 100000) + Number(license.max_activations || 0)
  if (key === 'created_at') return license.created_at ? new Date(license.created_at).getTime() : 0
  return license[key] || ''
}

const parseJSON = (raw) => {
  const trimmed = raw.trim()
  if (!trimmed) {
    return {}
  }
  return JSON.parse(trimmed)
}

const showResponse = (payload) => {
  result.value = JSON.stringify(payload, null, 2)
}

const clearMessages = () => {
  error.value = ''
  result.value = ''
}

const loadLicenses = async () => {
  loadingLicenses.value = true
  listError.value = ''
  try {
    const res = await licensesAPI.list({
      page: pagination.page,
      page_size: pagination.page_size,
      q: filters.q.trim() || undefined,
      status: filters.status || undefined,
    })
    licenses.value = res.data.licenses || []
    Object.assign(pagination, res.data.pagination || {})
    Object.assign(counts, res.data.counts || {})
  } catch (err) {
    listError.value = err?.response?.data?.error || 'Failed to load licenses'
  } finally {
    loadingLicenses.value = false
  }
}

const loadSlugs = async () => {
  loadingSlugs.value = true
  slugError.value = ''
  try {
    const res = await slugAPI.list()
    slugs.value = res.data.slugs || []
    if (slugs.value.length > 0 && !slugs.value.some((slug) => slug.name === generateForm.slug)) {
      const defaultSlug = slugs.value.find((slug) => slug.is_default) || slugs.value[0]
      generateForm.slug = defaultSlug.name
    }
  } catch (err) {
    slugError.value = err?.response?.data?.error || 'Failed to load slugs'
  } finally {
    loadingSlugs.value = false
  }
}

const applyFilters = () => {
  pagination.page = 1
  loadLicenses()
}

const setSort = (key) => {
  if (sort.key === key) {
    sort.direction = sort.direction === 'asc' ? 'desc' : 'asc'
    return
  }

  sort.key = key
  sort.direction = key === 'created_at' ? 'desc' : 'asc'
}

const changePage = (direction) => {
  const nextPage = pagination.page + direction
  if (nextPage < 1 || (pagination.total_pages && nextPage > pagination.total_pages)) {
    return
  }
  pagination.page = nextPage
  loadLicenses()
}

const doGenerate = async () => {
  clearMessages()
  try {
    const res = await licensesAPI.generate({ slug: generateForm.slug.trim(), metadata: parseJSON(generateForm.metadata) })
    showResponse(res.data)
    closeGenerateDrawer()
    await loadLicenses()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Generate failed'
  }
}

const doRevoke = async () => {
  clearMessages()
  try {
    const res = await licensesAPI.revoke({ license_key: revokeForm.license_key.trim() })
    showResponse(res.data)
    await loadLicenses()
  } catch (err) {
    error.value = err?.response?.data?.error || 'Revoke failed'
  }
}

const revokeListedLicense = async (licenseKey) => {
  closeActionMenu()
  revokeForm.license_key = licenseKey
  await doRevoke()
  closeDetails()
}

const openDetails = (license) => {
  closeActionMenu()
  selectedLicense.value = license
}

const closeDetails = () => {
  selectedLicense.value = null
}

const closeGenerateDrawer = () => {
  showGenerateDrawer.value = false
}

const openGenerateDrawer = () => {
  showGenerateDrawer.value = true
  if (slugs.value.length === 0) {
    loadSlugs()
  }
}

const closeActionMenu = () => {
  openActionMenu.value = null
  actionMenuLicense.value = null
}

const toggleActionMenu = (license, event) => {
  if (openActionMenu.value === license.id) {
    closeActionMenu()
    return
  }

  const rect = event.currentTarget.getBoundingClientRect()
  const menuWidth = 128
  const menuHeight = 92
  const margin = 8
  actionMenuPosition.left = Math.max(margin, Math.min(rect.right - menuWidth, window.innerWidth - menuWidth - margin))
  actionMenuPosition.top = rect.bottom + menuHeight > window.innerHeight
    ? Math.max(margin, rect.top - menuHeight - margin)
    : rect.bottom + 6
  actionMenuLicense.value = license
  openActionMenu.value = license.id
}

const copyLicenseKey = async (licenseKey) => {
  try {
    await navigator.clipboard.writeText(licenseKey)
    copiedLicenseKey.value = licenseKey
    window.setTimeout(() => {
      if (copiedLicenseKey.value === licenseKey) {
        copiedLicenseKey.value = ''
      }
    }, 1400)
  } catch {
    error.value = 'Failed to copy license key'
  }
}

const formatMetadata = (metadata) => {
  return JSON.stringify(metadata || {}, null, 2)
}

const formatDate = (value) => {
  if (!value) return 'Never'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(new Date(value))
}

const formatDateTime = (value) => {
  if (!value) return 'Never'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Never'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

const formatExpirationPolicy = (slug) => {
  if (!slug) return 'Select a slug'
  if (slug.expiration_type === 'duration') return `${slug.expiration_days} days after generation`
  if (slug.expiration_type === 'fixed_date') return formatDate(slug.fixed_expires_at)
  return 'Never expires'
}

const formatOfflineLifetime = (hours) => {
  const value = Number(hours || 0)
  if (value <= 0) return 'Not set'
  if (value % 24 === 0) return `${value / 24}d`
  return `${value}h`
}

const statusClass = (status) => {
  if (status === 'active') return 'bg-[#4ae176]/10 text-[#4ae176]'
  if (status === 'expired') return 'bg-[#f2b36d]/10 text-[#f2b36d]'
  if (status === 'revoked') return 'bg-[#ff7a8a]/10 text-[#ff7a8a]'
  return 'bg-[#a4c9ff]/10 text-[#a4c9ff]'
}

onMounted(() => {
  loadLicenses()
  loadSlugs()
  window.addEventListener('open-generate-license', openGenerateDrawer)
})

onUnmounted(() => {
  window.removeEventListener('open-generate-license', openGenerateDrawer)
})
</script>

<template>
  <div class="space-y-5">
      <section class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <article class="rounded-lg bg-[#171b26] p-4 ghost-border">
          <div class="flex items-center gap-3">
            <span class="flex h-7 w-7 items-center justify-center rounded-md bg-[#a4c9ff]/10 text-[#a4c9ff]">
              <Files class="h-4 w-4" :stroke-width="1.8" />
            </span>
            <p class="metadata-label text-xs font-bold text-[#dfe2f1]">Total</p>
          </div>
          <p class="mt-4 text-3xl font-black text-white">{{ counts.total.toLocaleString() }}</p>
          <p class="mt-1 text-xs text-[#8f98ad]">Every generated key</p>
        </article>

        <article class="rounded-lg bg-[#171b26] p-4 ghost-border">
          <div class="flex items-center gap-3">
            <span class="flex h-7 w-7 items-center justify-center rounded-md bg-[#4ae176]/10 text-[#4ae176]">
              <Check class="h-4 w-4" :stroke-width="2" />
            </span>
            <p class="metadata-label text-xs font-bold text-[#dfe2f1]">Active</p>
          </div>
          <p class="mt-4 text-3xl font-black text-white">{{ counts.active.toLocaleString() }}</p>
          <p class="mt-1 text-xs text-[#4ae176]">Currently valid</p>
        </article>

        <article class="rounded-lg bg-[#171b26] p-4 ghost-border">
          <div class="flex items-center gap-3">
            <span class="flex h-7 w-7 items-center justify-center rounded-md bg-[#f2b36d]/10 text-[#f2b36d]">
              <Clock3 class="h-4 w-4" :stroke-width="1.8" />
            </span>
            <p class="metadata-label text-xs font-bold text-[#dfe2f1]">Expired</p>
          </div>
          <p class="mt-4 text-3xl font-black text-white">{{ counts.expired.toLocaleString() }}</p>
          <p class="mt-1 text-xs text-[#f2b36d]">Past expiration</p>
        </article>

        <article class="rounded-lg bg-[#171b26] p-4 ghost-border">
          <div class="flex items-center gap-3">
            <span class="flex h-7 w-7 items-center justify-center rounded-md bg-[#ff7a8a]/10 text-[#ff7a8a]">
              <Ban class="h-4 w-4" :stroke-width="1.8" />
            </span>
            <p class="metadata-label text-xs font-bold text-[#dfe2f1]">Revoked</p>
          </div>
          <p class="mt-4 text-3xl font-black text-white">{{ counts.revoked.toLocaleString() }}</p>
          <p class="mt-1 text-xs text-[#ff7a8a]">Manually invalidated</p>
        </article>
      </section>

      <section class="rounded-lg bg-[#171b26] p-4 sm:p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 class="text-xl font-black tracking-tight text-white">Registry: Issued keys.</h1>
            <p v-if="listError" class="mt-2 text-sm text-[#ff7a8a]">{{ listError }}</p>
          </div>

          <div class="metadata-label text-xs text-[#8f98ad]">
            Showing {{ licenses.length }} of {{ pagination.total.toLocaleString() }} results
          </div>
        </div>

        <form class="mt-5 grid gap-3 sm:grid-cols-[minmax(0,18rem)_10rem_auto] sm:items-center" @submit.prevent="applyFilters">
          <label class="relative block">
            <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[#8f98ad]">⌕</span>
            <input v-model="filters.q" placeholder="Search" class="field-control field-control-with-icon text-sm" />
          </label>
          <select v-model="filters.status" class="field-control text-sm">
            <option value="">All statuses</option>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
            <option value="expired">Expired</option>
            <option value="revoked">Revoked</option>
          </select>
          <button class="secondary-action flex h-9 w-9 items-center justify-center p-0" aria-label="Apply search filters">
            <Search class="h-4 w-4" :stroke-width="1.8" />
          </button>
        </form>

        <div class="mt-4 overflow-x-auto">
          <table class="min-w-full text-left text-sm">
            <thead class="metadata-label text-xs text-[#8f98ad]">
              <tr>
                <th v-for="header in sortableHeaders" :key="header.key" class="px-3 py-3 font-bold">
                  <button type="button" class="sortable-header" @click="setSort(header.key)">
                    <span>{{ header.label }}</span>
                    <ArrowUp v-if="sort.key === header.key && sort.direction === 'asc'" class="h-3.5 w-3.5" :stroke-width="2" />
                    <ArrowDown v-else-if="sort.key === header.key && sort.direction === 'desc'" class="h-3.5 w-3.5" :stroke-width="2" />
                    <ChevronsUpDown v-else class="h-3.5 w-3.5 opacity-45" :stroke-width="1.8" />
                  </button>
                </th>
                <th class="px-3 py-3 text-right font-bold">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loadingLicenses">
                <td colspan="6" class="rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-[#8f98ad]">Loading licenses...</td>
              </tr>
              <tr v-else-if="licenses.length === 0">
                <td colspan="6" class="rounded-md bg-[#1c1f2a] px-3 py-8 text-center text-[#8f98ad]">No licenses found.</td>
              </tr>
              <template v-else>
                <tr v-for="license in sortedLicenses" :key="license.id" class="group">
                  <td class="rounded-l-md bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                    <button
                      class="font-mono text-xs text-[#dfe2f1] transition hover:text-[#a4c9ff]"
                      :title="copiedLicenseKey === license.license_key ? 'Copied' : 'Copy license key'"
                      @click="copyLicenseKey(license.license_key)"
                    >
                      {{ license.license_key }}
                    </button>
                  </td>
                  <td class="bg-[#1c1f2a] px-3 py-3 group-hover:bg-[#262a35]">
                    <span class="inline-flex items-center rounded-md px-2 py-1 metadata-label text-[0.65rem] font-bold" :class="statusClass(license.status)">
                      {{ license.status }}
                    </span>
                  </td>
                  <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ license.slug }}</td>
                  <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ license.active_seats }}/{{ license.max_activations }}</td>
                  <td class="bg-[#1c1f2a] px-3 py-3 text-[#dfe2f1] group-hover:bg-[#262a35]">{{ formatDate(license.created_at) }}</td>
                <td class="relative rounded-r-md bg-[#1c1f2a] px-3 py-3 text-right group-hover:bg-[#262a35]">
                  <button
                    class="rounded-md bg-[#262a35] px-2.5 py-1 text-[#8f98ad] transition hover:text-white"
                    @click="toggleActionMenu(license, $event)"
                  >
                    ...
                  </button>
                </td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <div class="mt-4 flex items-center justify-end gap-4 text-[#8f98ad]">
          <button class="hover:text-[#a4c9ff] disabled:text-[#8f98ad] disabled:opacity-50" :disabled="pagination.page <= 1" @click="changePage(-1)">‹</button>
          <button class="hover:text-[#a4c9ff] disabled:text-[#8f98ad] disabled:opacity-50" :disabled="pagination.total_pages === 0 || pagination.page >= pagination.total_pages" @click="changePage(1)">›</button>
        </div>
      </section>
    <section v-if="error || result" class="rounded-lg bg-[#171b26] p-4 sm:p-5">
      <h2 class="text-base font-black text-white">Last operation.</h2>
      <p v-if="error" class="mt-4 text-sm text-[#ff7a8a]">{{ error }}</p>
      <pre v-if="result" class="code-block mt-4 max-h-80 overflow-auto rounded-md p-4 text-xs text-[#dfe2f1]">{{ result }}</pre>
    </section>

    <template v-if="actionMenuLicense">
      <button
        class="fixed inset-0 z-[998] cursor-default bg-transparent"
        aria-label="Close actions menu"
        @click="closeActionMenu"
      ></button>
      <div
        class="fixed z-[999] w-32 rounded-md bg-[#171b26] p-1 text-left shadow-[inset_0_0_0_1px_rgba(65,71,81,0.22)]"
        :style="{ top: `${actionMenuPosition.top}px`, left: `${actionMenuPosition.left}px` }"
      >
        <button class="block w-full rounded px-3 py-2 text-left text-xs font-semibold text-[#dfe2f1] hover:bg-[#1c1f2a]" @click="openDetails(actionMenuLicense)">
          Details
        </button>
        <button
          class="block w-full rounded px-3 py-2 text-left text-xs font-semibold text-[#ff7a8a] hover:bg-[#262a35] disabled:text-[#8f98ad] disabled:opacity-50 disabled:hover:bg-transparent"
          :disabled="actionMenuLicense.status === 'revoked'"
          @click="revokeListedLicense(actionMenuLicense.license_key)"
        >
          {{ actionMenuLicense.status === 'revoked' ? 'Revoked' : 'Revoke' }}
        </button>
      </div>
    </template>

    <div v-if="showGenerateDrawer" class="fixed inset-0 z-50 bg-[#0f131d]/78" @click.self="closeGenerateDrawer">
      <aside id="generate-key" class="ml-auto flex h-full w-full max-w-sm flex-col bg-[#171b26] shadow-[inset_1px_0_rgba(65,71,81,0.22)]">
        <div class="flex items-start justify-between px-5 py-5 shadow-[inset_0_-1px_rgba(65,71,81,0.22)]">
          <div>
            <h2 class="text-base font-black text-white">Generate new license key.</h2>
            <p class="mt-1 text-xs text-[#8f98ad]">Uses your verified management API key.</p>
          </div>
          <button class="text-[#8f98ad] transition hover:text-white" @click="closeGenerateDrawer">✕</button>
        </div>

        <div class="flex-1 overflow-y-auto p-5">
          <div class="space-y-4">
            <label class="block text-sm text-[#c6ccdc]">
              <span class="mb-2 block text-xs font-semibold text-white">Slug</span>
              <select v-model="generateForm.slug" class="field-control text-sm" :disabled="loadingSlugs || slugs.length === 0">
                <option v-if="loadingSlugs" value="">Loading slugs...</option>
                <option v-for="slug in slugs" :key="slug.id" :value="slug.name">
                  {{ slug.name }}{{ slug.is_default ? ' (default)' : '' }}
                </option>
              </select>
            </label>

            <p v-if="slugError" class="text-sm text-[#ff7a8a]">{{ slugError }}</p>

            <section class="rounded-md bg-[#1c1f2a] p-3">
              <p class="metadata-label text-[0.65rem] font-bold text-[#8f98ad]">Selected Slug Settings</p>
              <div v-if="selectedSlug" class="mt-3 grid gap-3 text-sm">
                <div class="flex items-center justify-between gap-4">
                  <span class="text-[#8f98ad]">Max activations</span>
                  <span class="font-semibold text-white">{{ selectedSlug.max_activations }}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-[#8f98ad]">Expiration</span>
                  <span class="text-right font-semibold text-white">{{ formatExpirationPolicy(selectedSlug) }}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-[#8f98ad]">Offline JWTs</span>
                  <span class="font-semibold text-white">{{ selectedSlug.offline_enabled ? 'Enabled' : 'Disabled' }}</span>
                </div>
                <div v-if="selectedSlug.offline_enabled" class="flex items-center justify-between gap-4">
                  <span class="text-[#8f98ad]">Token TTL</span>
                  <span class="font-semibold text-white">{{ formatOfflineLifetime(selectedSlug.offline_token_lifetime_hours) }}</span>
                </div>
                <div class="flex items-center justify-between gap-4">
                  <span class="text-[#8f98ad]">Slug type</span>
                  <span class="font-semibold text-white">{{ selectedSlug.is_default ? 'Default' : 'Custom' }}</span>
                </div>
              </div>
              <p v-else class="mt-3 text-sm text-[#8f98ad]">No slug selected.</p>
            </section>

            <label class="block text-sm text-[#c6ccdc]">
              <span class="mb-2 block text-xs font-semibold text-white">Metadata JSON</span>
              <textarea v-model="generateForm.metadata" rows="7" class="field-control font-mono text-xs"></textarea>
            </label>
          </div>
        </div>

        <div class="p-5 shadow-[inset_0_1px_rgba(65,71,81,0.22)]">
          <button class="primary-action w-full px-5 py-3 text-sm disabled:opacity-60" :disabled="loadingSlugs || !generateForm.slug" @click="doGenerate">Issue License</button>
        </div>
      </aside>
    </div>

    <div v-if="selectedLicense" class="fixed inset-0 z-50 flex items-center justify-center bg-[#0f131d]/78 p-4" @click.self="closeDetails">
      <section class="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-[#171b26] p-5 shadow-[inset_0_0_0_1px_rgba(65,71,81,0.22)]">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="metadata-label text-xs font-bold text-[#8f98ad]">License Details</p>
            <h2 class="mt-2 break-all text-xl font-black text-white">{{ selectedLicense.license_key }}</h2>
          </div>
          <button class="text-[#8f98ad] transition hover:text-white" @click="closeDetails">✕</button>
        </div>

        <div class="mt-5 grid gap-3 sm:grid-cols-2">
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Status</p>
            <span class="mt-2 inline-flex rounded-md px-2 py-1 metadata-label text-xs font-bold" :class="statusClass(selectedLicense.status)">{{ selectedLicense.status }}</span>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Slug</p>
            <p class="mt-2 text-sm text-white">{{ selectedLicense.slug }}</p>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Seats</p>
            <p class="mt-2 text-sm text-white">{{ selectedLicense.active_seats }} / {{ selectedLicense.max_activations }}</p>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Created</p>
            <p class="mt-2 text-sm text-white">{{ formatDate(selectedLicense.created_at) }}</p>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Last Validated</p>
            <p class="mt-2 text-sm text-white">{{ formatDateTime(selectedLicense.last_validated_at) }}</p>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Expires</p>
            <p class="mt-2 text-sm text-white">{{ formatDate(selectedLicense.expires_at) }}</p>
          </div>
          <div class="rounded-md bg-[#1c1f2a] p-3">
            <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Revoked</p>
            <p class="mt-2 text-sm text-white">{{ formatDate(selectedLicense.revoked_at) }}</p>
          </div>
        </div>

        <div class="mt-3 rounded-md bg-[#1c1f2a] p-3">
          <p class="metadata-label text-[0.65rem] text-[#8f98ad]">Metadata</p>
          <pre class="code-block mt-3 overflow-x-auto rounded-md p-4 text-xs text-[#dfe2f1]">{{ formatMetadata(selectedLicense.metadata) }}</pre>
        </div>

        <div class="mt-5 flex justify-end gap-3">
          <button class="secondary-action px-4 py-2 text-sm" @click="closeDetails">Close</button>
          <button
            class="rounded-md bg-[#1c1f2a] px-4 py-2 text-sm font-bold text-[#ff7a8a] transition hover:bg-[#262a35] disabled:text-[#8f98ad] disabled:opacity-50"
            :disabled="selectedLicense.status === 'revoked'"
            @click="revokeListedLicense(selectedLicense.license_key)"
          >
            {{ selectedLicense.status === 'revoked' ? 'Revoked' : 'Revoke License' }}
          </button>
        </div>
      </section>
    </div>
  </div>
</template>
