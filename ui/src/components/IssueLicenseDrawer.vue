<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { CircleCheck, Clock3, KeyRound, Users, WifiOff } from 'lucide-vue-next'
import UiDrawer from '@/components/UiDrawer.vue'
import CopyButton from '@/components/CopyButton.vue'
import { licensesAPI, slugAPI } from '@/stores/api'
import { closeIssueDrawer, notifyLicensesChanged, ui } from '@/lib/ui'
import { errorMessage, formatExpiration, formatLifetime, formatSeats, hasKeys, parseJSONObject } from '@/lib/format'

const slugs = ref([])
const loadingSlugs = ref(false)
const saving = ref(false)
const error = ref('')
const issuedKey = ref('')

const form = reactive({
  slug: '',
  metadata: '',
})

const selectedSlug = computed(() => slugs.value.find((slug) => slug.name === form.slug) || null)

const loadSlugs = async () => {
  loadingSlugs.value = true
  try {
    const res = await slugAPI.list()
    slugs.value = res.data.slugs || []
    if (!slugs.value.some((slug) => slug.name === form.slug)) {
      form.slug = (slugs.value.find((slug) => slug.is_default) || slugs.value[0])?.name || ''
    }
  } catch (err) {
    error.value = errorMessage(err, 'Failed to load slugs')
  } finally {
    loadingSlugs.value = false
  }
}

const reset = () => {
  issuedKey.value = ''
  error.value = ''
  form.metadata = ''
}

const issue = async () => {
  error.value = ''
  saving.value = true
  try {
    const res = await licensesAPI.generate({
      slug: form.slug,
      metadata: parseJSONObject(form.metadata, 'Metadata'),
    })
    issuedKey.value = res.data.license_key
    notifyLicensesChanged()
  } catch (err) {
    error.value = errorMessage(err, 'Could not issue license')
  } finally {
    saving.value = false
  }
}

watch(
  () => ui.issueDrawerOpen,
  (open) => {
    if (!open) return
    reset()
    loadSlugs()
  },
)
</script>

<template>
  <UiDrawer :open="ui.issueDrawerOpen" title="New license" @close="closeIssueDrawer">
    <div v-if="issuedKey" class="flex flex-col items-center pt-8 text-center">
      <span class="flex h-12 w-12 items-center justify-center rounded-full bg-success/10 text-success shadow-[0_0_32px_-4px_rgba(74,225,118,0.5)]">
        <CircleCheck class="h-6 w-6" :stroke-width="2" />
      </span>
      <h3 class="mt-5 text-lg font-semibold text-ink-strong">License issued</h3>
      <p class="mt-1 text-sm text-ink-muted">Send this key to your customer.</p>

      <div class="mt-6 w-full rounded-xl bg-surface-lowest p-4 shadow-ghost">
        <p class="break-all font-mono text-[0.95rem] font-medium tracking-wide text-primary">{{ issuedKey }}</p>
      </div>
      <CopyButton class="mt-3" :value="issuedKey" label="Copy key" />
    </div>

    <form v-else id="issue-license-form" class="space-y-5" @submit.prevent="issue">
      <label class="block">
        <span class="field-label">Slug</span>
        <select v-model="form.slug" class="field" :disabled="loadingSlugs || slugs.length === 0" required>
          <option v-if="loadingSlugs && slugs.length === 0" value="">Loading…</option>
          <option v-for="slug in slugs" :key="slug.id" :value="slug.name">
            {{ slug.name }}{{ slug.is_default && slug.name !== 'default' ? ' (default)' : '' }}
          </option>
        </select>
      </label>

      <div v-if="selectedSlug" class="flex flex-wrap gap-1.5">
        <span class="inline-flex items-center gap-1.5 rounded-md bg-surface-container px-2 py-1 text-xs text-ink-soft">
          <Users class="h-3.5 w-3.5 text-ink-muted" :stroke-width="1.8" /> {{ formatSeats(selectedSlug.max_activations) }}
        </span>
        <span class="inline-flex items-center gap-1.5 rounded-md bg-surface-container px-2 py-1 text-xs text-ink-soft">
          <Clock3 class="h-3.5 w-3.5 text-ink-muted" :stroke-width="1.8" /> {{ formatExpiration(selectedSlug) }}
        </span>
        <span v-if="selectedSlug.offline_enabled" class="inline-flex items-center gap-1.5 rounded-md bg-surface-container px-2 py-1 text-xs text-ink-soft">
          <WifiOff class="h-3.5 w-3.5 text-ink-muted" :stroke-width="1.8" /> Offline tokens · {{ formatLifetime(selectedSlug.offline_token_lifetime_hours) }}
        </span>
        <span v-if="hasKeys(selectedSlug.attributes)" class="inline-flex items-center gap-1.5 rounded-md bg-surface-container px-2 py-1 text-xs text-ink-soft">
          <KeyRound class="h-3.5 w-3.5 text-ink-muted" :stroke-width="1.8" /> {{ Object.keys(selectedSlug.attributes).length }} attributes
        </span>
      </div>

      <label class="block">
        <span class="field-label">Metadata <span class="text-ink-faint">· optional</span></span>
        <textarea
          v-model="form.metadata"
          rows="5"
          spellcheck="false"
          class="field font-mono text-xs leading-relaxed"
          placeholder='{ "email": "customer@example.com" }'
        ></textarea>
        <span class="field-hint">A JSON object stored with the license, such as who bought it.</span>
      </label>

      <p v-if="error" class="rounded-lg bg-critical/10 px-3 py-2 text-sm text-critical">{{ error }}</p>
    </form>

    <template #footer>
      <template v-if="issuedKey">
        <button class="btn-ghost" @click="reset">Issue another</button>
        <button class="btn-primary ml-auto" @click="closeIssueDrawer">Done</button>
      </template>
      <template v-else>
        <button class="btn-ghost" @click="closeIssueDrawer">Cancel</button>
        <button class="btn-primary ml-auto" type="submit" form="issue-license-form" :disabled="saving || !form.slug">
          {{ saving ? 'Issuing…' : 'Issue license' }}
        </button>
      </template>
    </template>
  </UiDrawer>
</template>
