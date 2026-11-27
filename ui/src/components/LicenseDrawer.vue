<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { Download, KeyRound, MonitorSmartphone, Pencil, Plus, X } from 'lucide-vue-next'
import UiDrawer from '@/components/UiDrawer.vue'
import StatusBadge from '@/components/StatusBadge.vue'
import CopyButton from '@/components/CopyButton.vue'
import ConfirmButton from '@/components/ConfirmButton.vue'
import { licensesAPI, runtimeAPI } from '@/stores/api'
import { notifyLicensesChanged, toast, toastError } from '@/lib/ui'
import {
  endOfLocalDay,
  errorMessage,
  formatDate,
  formatLifetime,
  formatRelative,
  hasKeys,
  jwtExpiry,
  prettyJSON,
  reasonLabel,
  toDateInputValue,
} from '@/lib/format'

// Shows one license: its limits (editable), the devices using it, and revoke / restore.
const props = defineProps({
  licenseId: { type: String, default: '' },
})

const emit = defineEmits(['close'])

const license = ref(null)
const loading = ref(false)
const loadError = ref('')
const busy = ref(false)
const editing = ref(false)
const form = reactive({ max_activations: 1, never_expires: true, expires_on: '' })

const isRevoked = computed(() => license.value?.status === 'revoked')
const isUsable = computed(() => license.value && !['revoked', 'expired'].includes(license.value.status))

// Binding a device by hand: for on-prem installs that never call the server, the admin activates the
// customer's machine id here and sends them the offline JWT.
const adding = ref(false)
const deviceForm = reactive({ fingerprint: '', name: '' })
const issued = ref(null) // { label, token, expiresAt }

const load = async () => {
  if (!props.licenseId) return
  loading.value = true
  loadError.value = ''
  try {
    const res = await licensesAPI.get(props.licenseId)
    license.value = res.data
  } catch (err) {
    loadError.value = errorMessage(err, 'Failed to load license')
  } finally {
    loading.value = false
  }
}

const startEditing = () => {
  form.max_activations = license.value.max_activations
  form.never_expires = !license.value.expires_at
  form.expires_on = toDateInputValue(license.value.expires_at)
  editing.value = true
}

// Every change goes through one PATCH; the response is the updated license.
const update = async (payload, message) => {
  busy.value = true
  try {
    const res = await licensesAPI.update(props.licenseId, payload)
    license.value = res.data
    notifyLicensesChanged()
    toast(message)
    return true
  } catch (err) {
    toastError(errorMessage(err, 'Could not update license'))
    return false
  } finally {
    busy.value = false
  }
}

const saveLimits = async () => {
  const payload = { max_activations: Number(form.max_activations) }
  if (form.never_expires) {
    payload.expires_at = null
  } else {
    const date = endOfLocalDay(form.expires_on)
    if (!date) {
      toastError('Pick an expiration date')
      return
    }
    payload.expires_at = date.toISOString()
  }
  if (await update(payload, 'License updated')) editing.value = false
}

const removeDevice = async (activation) => {
  busy.value = true
  try {
    const res = await licensesAPI.removeActivation(props.licenseId, activation.id)
    license.value = res.data
    notifyLicensesChanged()
    toast('Device removed, seat freed')
  } catch (err) {
    toastError(errorMessage(err, 'Could not remove device'))
  } finally {
    busy.value = false
  }
}

const showToken = (label, token) => {
  issued.value = { label, token, expiresAt: jwtExpiry(token) }
}

const addDevice = async () => {
  const fingerprint = deviceForm.fingerprint.trim()
  const name = deviceForm.name.trim()
  busy.value = true
  try {
    const res = await runtimeAPI.activate({
      license_key: license.value.license_key,
      fingerprint,
      metadata: name ? { name } : {},
    })
    if (!res.data.valid) {
      toastError(reasonLabel(res.data.reason))
      return
    }
    adding.value = false
    deviceForm.fingerprint = ''
    deviceForm.name = ''
    if (res.data.token) showToken(name || fingerprint, res.data.token)
    else toast('Device added')
    notifyLicensesChanged()
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not add device'))
  } finally {
    busy.value = false
  }
}

const issueToken = async (activation) => {
  busy.value = true
  try {
    const res = await runtimeAPI.validate({ license_key: license.value.license_key, fingerprint: activation.fingerprint })
    if (!res.data.valid) {
      toastError(reasonLabel(res.data.reason))
    } else if (!res.data.token) {
      toastError('No token: check the slug has offline tokens on and a signing key is active')
    } else {
      showToken(deviceName(activation) || activation.fingerprint, res.data.token)
    }
    await load()
  } catch (err) {
    toastError(errorMessage(err, 'Could not issue token'))
  } finally {
    busy.value = false
  }
}

const downloadToken = () => {
  const blob = new Blob([`${issued.value.token}\n`], { type: 'application/jwt' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'license.jwt'
  link.click()
  URL.revokeObjectURL(url)
}

const deviceName = (activation) => {
  const name = activation.metadata?.name
  return typeof name === 'string' && name.trim() ? name : null
}

watch(
  () => props.licenseId,
  (key) => {
    editing.value = false
    adding.value = false
    issued.value = null
    license.value = null
    if (key) load()
  },
  { immediate: true },
)
</script>

<template>
  <UiDrawer :open="Boolean(licenseId)" title="License" @close="emit('close')">
    <p v-if="loadError" class="rounded-lg bg-critical/10 px-3 py-2 text-sm text-critical">{{ loadError }}</p>
    <div v-else-if="loading && !license" class="space-y-3">
      <div class="h-24 animate-pulse rounded-xl bg-surface-container"></div>
      <div class="h-40 animate-pulse rounded-xl bg-surface-container/60"></div>
    </div>

    <template v-else-if="license">
      <div class="rounded-xl bg-surface-lowest p-4 shadow-ghost">
        <div class="flex items-start justify-between gap-3">
          <p class="break-all font-mono text-[0.9rem] font-medium leading-relaxed text-primary">{{ license.license_key }}</p>
          <CopyButton :value="license.license_key" class="-mr-1 -mt-1" />
        </div>
        <div class="mt-3"><StatusBadge :status="license.status" /></div>
      </div>

      <!-- Limits: read view -->
      <section v-if="!editing" class="mt-6">
        <div class="flex items-center justify-between">
          <p class="eyebrow">Limits</p>
          <button v-if="!isRevoked" class="btn-ghost -mr-2 px-2 py-1 text-xs" @click="startEditing">
            <Pencil class="h-3.5 w-3.5" :stroke-width="2" /> Edit
          </button>
        </div>
        <dl class="mt-3 grid grid-cols-2 gap-x-6 gap-y-5 text-sm">
          <div>
            <dt class="text-xs text-ink-muted">Seats in use</dt>
            <dd class="mt-1 tabular-nums text-ink">{{ license.active_seats }} of {{ license.max_activations }}</dd>
          </div>
          <div>
            <dt class="text-xs text-ink-muted">Expires</dt>
            <dd class="mt-1 text-ink">{{ formatDate(license.expires_at, 'Never') }}</dd>
          </div>
          <div>
            <dt class="text-xs text-ink-muted">Slug</dt>
            <dd class="mt-1 text-ink">
              {{ license.slug }}
              <span v-if="license.product" class="mt-0.5 block truncate font-mono text-xs text-ink-muted" :title="license.product">{{ license.product }}</span>
            </dd>
          </div>
          <div>
            <dt class="text-xs text-ink-muted">Created</dt>
            <dd class="mt-1 text-ink">{{ formatDate(license.created_at) }}</dd>
          </div>
          <div v-if="license.revoked_at">
            <dt class="text-xs text-ink-muted">Revoked</dt>
            <dd class="mt-1 text-critical">{{ formatDate(license.revoked_at) }}</dd>
          </div>
          <div v-if="license.offline_enabled">
            <dt class="text-xs text-ink-muted">Offline tokens</dt>
            <dd class="mt-1 text-ink">{{ formatLifetime(license.offline_token_lifetime_hours) }}</dd>
          </div>
        </dl>
      </section>

      <!-- Limits: edit view -->
      <form v-else id="license-limits-form" class="mt-6 space-y-4 rounded-xl bg-surface-container/60 p-4 shadow-ghost" @submit.prevent="saveLimits">
        <p class="eyebrow">Edit limits</p>
        <label class="block">
          <span class="field-label">Seats</span>
          <input v-model="form.max_activations" type="number" min="1" required class="field w-28 tabular-nums" />
          <span v-if="Number(form.max_activations) < license.active_seats" class="field-hint text-warning">
            {{ license.active_seats }} devices are active. They keep working, but no new ones can activate until some are removed.
          </span>
        </label>
        <div>
          <span class="field-label">Expires</span>
          <label class="flex items-center gap-2 text-sm text-ink-soft">
            <input v-model="form.never_expires" type="checkbox" class="accent-[#a4c9ff]" /> Never
          </label>
          <input v-if="!form.never_expires" v-model="form.expires_on" type="date" required class="field mt-2" />
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn-ghost" @click="editing = false">Cancel</button>
          <button class="btn-primary" :disabled="busy">{{ busy ? 'Saving…' : 'Save' }}</button>
        </div>
      </form>

      <section class="mt-8">
        <div class="flex items-center justify-between">
          <p class="eyebrow">Devices</p>
          <button v-if="isUsable && !adding" class="btn-ghost -mr-2 px-2 py-1 text-xs" @click="adding = true">
            <Plus class="h-3.5 w-3.5" :stroke-width="2.2" /> Add device
          </button>
        </div>

        <form v-if="adding" class="mt-3 space-y-3 rounded-xl bg-surface-container/60 p-4 shadow-ghost" @submit.prevent="addDevice">
          <label class="block">
            <span class="field-label">Fingerprint</span>
            <input v-model="deviceForm.fingerprint" required autofocus placeholder="The customer's machine id" class="field font-mono text-xs" />
          </label>
          <label class="block">
            <span class="field-label">Name <span class="text-ink-faint">· optional</span></span>
            <input v-model="deviceForm.name" placeholder="Acme Corp production server" class="field" />
          </label>
          <p v-if="license.offline_enabled" class="text-xs leading-5 text-ink-faint">
            You'll get an offline token valid for {{ formatLifetime(license.offline_token_lifetime_hours) }}{{ license.expires_at ? ', or until the license expires if sooner' : '' }}.
          </p>
          <div class="flex justify-end gap-2">
            <button type="button" class="btn-ghost" @click="adding = false">Cancel</button>
            <button class="btn-primary" :disabled="busy">{{ busy ? 'Adding…' : 'Add device' }}</button>
          </div>
        </form>

        <div v-if="issued" class="mt-3 rounded-xl bg-success/[0.06] p-4 shadow-[inset_0_0_0_1px_rgba(74,225,118,0.22)]">
          <div class="flex items-start gap-3">
            <KeyRound class="mt-0.5 h-4 w-4 shrink-0 text-success" :stroke-width="2" />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium text-ink">Offline token for {{ issued.label }}</p>
              <p class="mt-0.5 text-xs text-ink-muted">
                {{ issued.expiresAt ? `Valid until ${formatDate(issued.expiresAt)}` : 'Signed offline license' }} · verify it with your public key
              </p>
            </div>
            <button class="text-ink-faint transition hover:text-ink" aria-label="Dismiss" @click="issued = null">
              <X class="h-4 w-4" :stroke-width="2" />
            </button>
          </div>
          <pre class="code-block mt-3 max-h-32 whitespace-pre-wrap break-all text-[0.7rem]">{{ issued.token }}</pre>
          <div class="mt-3 flex justify-end gap-2">
            <button class="btn-secondary px-3 py-1.5 text-xs" @click="downloadToken">
              <Download class="h-3.5 w-3.5" :stroke-width="2" /> Download
            </button>
            <CopyButton :value="issued.token" label="Copy token" class="px-3 py-1.5 text-xs" />
          </div>
        </div>

        <p v-if="license.activations.length === 0 && !adding" class="mt-3 rounded-lg bg-surface-container/50 px-3 py-4 text-center text-sm text-ink-muted">
          Not activated on any device yet.
        </p>
        <ul v-else class="mt-3 space-y-1.5">
          <li
            v-for="activation in license.activations"
            :key="activation.id"
            class="flex items-center gap-3 rounded-lg bg-surface-container/60 px-3 py-2.5 shadow-ghost"
          >
            <MonitorSmartphone class="h-4 w-4 shrink-0 text-ink-muted" :stroke-width="1.8" />
            <div class="min-w-0 flex-1">
              <p v-if="deviceName(activation)" class="truncate text-sm text-ink">{{ deviceName(activation) }}</p>
              <p class="truncate font-mono text-xs" :class="deviceName(activation) ? 'text-ink-faint' : 'text-ink'" :title="activation.fingerprint">
                {{ activation.fingerprint }}
              </p>
              <p class="mt-0.5 text-xs text-ink-faint">
                Activated {{ formatRelative(activation.activated_at) }} · seen {{ formatRelative(activation.last_validated_at, 'never') }}
              </p>
            </div>
            <button
              v-if="license.offline_enabled && isUsable"
              class="btn-secondary px-2.5 py-1 text-xs"
              :disabled="busy"
              title="Issue a fresh offline token for this device"
              @click="issueToken(activation)"
            >
              Token
            </button>
            <ConfirmButton size="sm" label="Remove" confirm-label="Confirm" :disabled="busy" @confirm="removeDevice(activation)" />
          </li>
        </ul>
      </section>

      <div v-if="hasKeys(license.metadata)" class="mt-8">
        <p class="eyebrow">Metadata</p>
        <pre class="code-block mt-2">{{ prettyJSON(license.metadata) }}</pre>
      </div>

      <div v-if="hasKeys(license.attributes)" class="mt-6">
        <p class="eyebrow">Attributes</p>
        <pre class="code-block mt-2">{{ prettyJSON(license.attributes) }}</pre>
      </div>
    </template>

    <template #footer>
      <button class="btn-ghost" @click="emit('close')">Close</button>
      <template v-if="license">
        <button v-if="isRevoked" class="btn-secondary ml-auto" :disabled="busy" @click="update({ revoked: false }, 'License restored')">
          Restore license
        </button>
        <ConfirmButton
          v-else
          class="ml-auto"
          label="Revoke license"
          confirm-label="Confirm revoke"
          :disabled="busy"
          @confirm="update({ revoked: true }, 'License revoked')"
        />
      </template>
    </template>
  </UiDrawer>
</template>
