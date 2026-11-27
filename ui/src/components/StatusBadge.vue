<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: { type: String, required: true },
  label: { type: String, default: '' },
})

// Maps every status the API can return onto the four semantic tones from DESIGN.md.
const tones = {
  active: 'success',
  enabled: 'success',
  delivered: 'success',
  inactive: 'neutral',
  verify_only: 'neutral',
  sending: 'neutral',
  pending: 'warning',
  expired: 'warning',
  revoked: 'critical',
  retired: 'critical',
  failed: 'critical',
  disabled: 'muted',
  archived: 'muted',
}

const toneClasses = {
  success: 'bg-success/10 text-success',
  neutral: 'bg-primary/10 text-primary',
  warning: 'bg-warning/[0.12] text-warning',
  critical: 'bg-critical/[0.12] text-critical',
  muted: 'bg-ink-muted/10 text-ink-muted',
}

const tone = computed(() => tones[props.status] || 'neutral')
const text = computed(() => props.label || props.status.replace(/_/g, ' '))
</script>

<template>
  <span
    class="inline-flex items-center gap-1.5 whitespace-nowrap rounded-full px-2 py-0.5 font-label text-[0.65rem] font-semibold uppercase tracking-[0.08em]"
    :class="toneClasses[tone]"
  >
    <span class="h-1.5 w-1.5 rounded-full bg-current" :class="tone === 'success' ? 'shadow-[0_0_8px_currentColor]' : ''"></span>
    {{ text }}
  </span>
</template>
