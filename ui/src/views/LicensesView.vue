<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight, KeyRound, Plus, Search } from 'lucide-vue-next'
import PageHeader from '@/components/PageHeader.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import CopyButton from '@/components/CopyButton.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import UiDrawer from '@/components/UiDrawer.vue'
import { licensesAPI } from '@/stores/api'
import { openIssueDrawer, toast, toastError, ui } from '@/lib/ui'
import { errorMessage, formatDate, formatLifetime, formatRelative, hasKeys, prettyJSON } from '@/lib/format'

const PAGE_SIZE = 15

const licenses = ref([])
const loading = ref(false)
const loadError = ref('')
const selected = ref(null)
const revoking = ref(false)

const query = ref('')
const status = ref('')
const page = ref(1)
const pagination = reactive({ total: 0, total_pages: 0 })
const counts = reactive({ total: 0, active: 0, inactive: 0, expired: 0, revoked: 0 })

const filters = computed(() => [
  { value: '', label: 'All', count: counts.total },
  { value: 'active', label: 'Active', count: counts.active },
  { value: 'inactive', label: 'Inactive', count: counts.inactive },
  { value: 'expired', label: 'Expired', count: counts.expired },
  { value: 'revoked', label: 'Revoked', count: counts.revoked },
])

const rangeLabel = computed(() => {
  if (pagination.total === 0) return '0 results'
  const start = (page.value - 1) * PAGE_SIZE + 1
  const end = start + licenses.value.length - 1
  return `${start}–${end} of ${pagination.total.toLocaleString()}`
})

const isFiltered = computed(() => Boolean(query.value.trim() || status.value))

const load = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await licensesAPI.list({
      page: page.value,
      page_size: PAGE_SIZE,
      q: query.value.trim() || undefined,
      status: status.value || undefined,
    })
    licenses.value = res.data.licenses || []
    Object.assign(pagination, res.data.pagination || {})
    Object.assign(counts, res.data.counts || {})
    if (selected.value) {
      selected.value = licenses.value.find((license) => license.id === selected.value.id) || selected.value
    }
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load licenses')
  } finally {
    loading.value = false
  }
}

const setStatus = (value) => {
  status.value = value
  page.value = 1
  load()
}

let searchTimer = 0
watch(query, () => {
  window.clearTimeout(searchTimer)
  searchTimer = window.setTimeout(() => {
    page.value = 1
    load()
  }, 250)
})

const goToPage = (next) => {
  if (next < 1 || next > pagination.total_pages) return
  page.value = next
  load()
}

const revoke = async () => {
  if (!selected.value) return
  revoking.value = true
  try {
    await licensesAPI.revoke({ license_key: selected.value.license_key })
    toast('License revoked')
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not revoke license'))
  } finally {
    revoking.value = false
  }
}

const seatPercent = (license) => {
  const max = Number(license.max_activations || 0)
  if (max <= 0) return 0
  return Math.min(100, (Number(license.active_seats || 0) / max) * 100)
}

watch(() => ui.licensesVersion, load)
onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Licenses" description="Every key you have issued. Click a row for details or to revoke it." />

    <div class="mt-8 flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex gap-1 overflow-x-auto rounded-xl bg-surface-low/80 p-1 shadow-ghost">
        <button
          v-for="filter in filters"
          :key="filter.value"
          class="flex items-center gap-2 whitespace-nowrap rounded-lg px-3 py-1.5 text-sm font-medium transition"
          :class="status === filter.value ? 'bg-surface-high text-ink-strong shadow-ghost' : 'text-ink-muted hover:text-ink'"
          @click="setStatus(filter.value)"
        >
          {{ filter.label }}
          <span
            class="rounded-md px-1.5 font-label text-[0.7rem] tabular-nums"
            :class="status === filter.value ? 'bg-primary/15 text-primary' : 'text-ink-faint'"
          >{{ filter.count.toLocaleString() }}</span>
        </button>
      </div>

      <label class="relative block lg:w-72">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-faint" :stroke-width="1.8" />
        <input v-model="query" type="search" placeholder="Search key, slug or metadata" class="field pl-9" />
      </label>
    </div>

    <section class="card mt-4 p-2">
      <p v-if="loadError" class="px-4 py-10 text-center text-sm text-critical">{{ loadError }}</p>

      <div v-else-if="!loading && licenses.length === 0" class="flex flex-col items-center px-4 py-16 text-center">
        <span class="flex h-11 w-11 items-center justify-center rounded-xl bg-surface-container text-ink-muted shadow-ghost">
          <KeyRound class="h-5 w-5" :stroke-width="1.8" />
        </span>
        <template v-if="isFiltered">
          <p class="mt-4 font-medium text-ink">No matching licenses</p>
          <p class="mt-1 text-sm text-ink-muted">Try a different search or status.</p>
        </template>
        <template v-else>
          <p class="mt-4 font-medium text-ink">No licenses yet</p>
          <p class="mt-1 text-sm text-ink-muted">Issue your first key to get started.</p>
          <button class="btn-primary mt-5" @click="openIssueDrawer">
            <Plus class="h-4 w-4" :stroke-width="2.4" /> New license
          </button>
        </template>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="data-table" :class="loading ? 'opacity-60 transition-opacity' : ''">
          <thead>
            <tr>
              <th>License key</th>
              <th>Status</th>
              <th class="hidden sm:table-cell">Slug</th>
              <th class="hidden md:table-cell">Seats</th>
              <th class="hidden text-right md:table-cell">Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="license in licenses" :key="license.id" class="is-clickable group" @click="selected = license">
              <td class="w-full max-w-0 sm:min-w-[15rem]">
                <div class="flex items-center gap-1">
                  <span class="truncate font-mono text-[0.8rem] text-ink">{{ license.license_key }}</span>
                  <CopyButton :value="license.license_key" class="h-7 w-7 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100" />
                </div>
              </td>
              <td><StatusBadge :status="license.status" /></td>
              <td class="hidden whitespace-nowrap text-ink-soft sm:table-cell">{{ license.slug }}</td>
              <td class="hidden whitespace-nowrap md:table-cell">
                <div class="flex items-center gap-2.5">
                  <span class="h-1 w-12 overflow-hidden rounded-full bg-surface-highest">
                    <span class="block h-full rounded-full bg-primary" :style="{ width: `${seatPercent(license)}%` }"></span>
                  </span>
                  <span class="font-label text-xs tabular-nums text-ink-soft">{{ license.active_seats }}/{{ license.max_activations }}</span>
                </div>
              </td>
              <td class="hidden whitespace-nowrap text-right text-ink-muted md:table-cell" :title="formatDate(license.created_at)">{{ formatRelative(license.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="pagination.total > 0" class="flex items-center justify-between px-4 pb-1 pt-3">
        <p class="font-label text-xs tabular-nums text-ink-faint">{{ rangeLabel }}</p>
        <div class="flex items-center gap-1">
          <button class="btn-icon" :disabled="page <= 1" aria-label="Previous page" @click="goToPage(page - 1)">
            <ChevronLeft class="h-4 w-4" :stroke-width="2" />
          </button>
          <button class="btn-icon" :disabled="page >= pagination.total_pages" aria-label="Next page" @click="goToPage(page + 1)">
            <ChevronRight class="h-4 w-4" :stroke-width="2" />
          </button>
        </div>
      </div>
    </section>

    <UiDrawer :open="Boolean(selected)" title="License" @close="selected = null">
      <template v-if="selected">
        <div class="rounded-xl bg-surface-lowest p-4 shadow-ghost">
          <div class="flex items-start justify-between gap-3">
            <p class="break-all font-mono text-[0.9rem] font-medium leading-relaxed text-primary">{{ selected.license_key }}</p>
            <CopyButton :value="selected.license_key" class="-mr-1 -mt-1" />
          </div>
          <div class="mt-3"><StatusBadge :status="selected.status" /></div>
        </div>

        <dl class="mt-6 grid grid-cols-2 gap-x-6 gap-y-5 text-sm">
          <div>
            <dt class="eyebrow">Slug</dt>
            <dd class="mt-1 text-ink">{{ selected.slug }}</dd>
          </div>
          <div>
            <dt class="eyebrow">Seats in use</dt>
            <dd class="mt-1 tabular-nums text-ink">{{ selected.active_seats }} of {{ selected.max_activations }}</dd>
          </div>
          <div>
            <dt class="eyebrow">Created</dt>
            <dd class="mt-1 text-ink">{{ formatDate(selected.created_at) }}</dd>
          </div>
          <div>
            <dt class="eyebrow">Expires</dt>
            <dd class="mt-1 text-ink">{{ formatDate(selected.expires_at, 'Never') }}</dd>
          </div>
          <div>
            <dt class="eyebrow">Last validated</dt>
            <dd class="mt-1 text-ink">{{ formatRelative(selected.last_validated_at) }}</dd>
          </div>
          <div v-if="selected.revoked_at">
            <dt class="eyebrow">Revoked</dt>
            <dd class="mt-1 text-critical">{{ formatDate(selected.revoked_at) }}</dd>
          </div>
          <div v-else-if="selected.offline_enabled">
            <dt class="eyebrow">Offline tokens</dt>
            <dd class="mt-1 text-ink">{{ formatLifetime(selected.offline_token_lifetime_hours) }}</dd>
          </div>
        </dl>

        <div v-if="hasKeys(selected.metadata)" class="mt-7">
          <p class="eyebrow">Metadata</p>
          <pre class="code-block mt-2">{{ prettyJSON(selected.metadata) }}</pre>
        </div>

        <div v-if="hasKeys(selected.attributes)" class="mt-6">
          <p class="eyebrow">Attributes</p>
          <pre class="code-block mt-2">{{ prettyJSON(selected.attributes) }}</pre>
        </div>
      </template>

      <template #footer>
        <button class="btn-ghost" @click="selected = null">Close</button>
        <ConfirmButton
          v-if="selected && selected.status !== 'revoked'"
          class="ml-auto"
          label="Revoke license"
          confirm-label="Confirm revoke"
          :disabled="revoking"
          @confirm="revoke"
        />
      </template>
    </UiDrawer>
  </div>
</template>
