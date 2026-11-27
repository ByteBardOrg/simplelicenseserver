<script setup>
import { ref } from 'vue'
import { Check, Copy } from 'lucide-vue-next'
import { toastError } from '@/lib/ui'

const props = defineProps({
  value: { type: String, required: true },
  label: { type: String, default: '' },
})

const copied = ref(false)

const copy = async () => {
  try {
    await navigator.clipboard.writeText(props.value)
    copied.value = true
    window.setTimeout(() => {
      copied.value = false
    }, 1400)
  } catch {
    toastError('Could not copy to clipboard')
  }
}
</script>

<template>
  <button
    type="button"
    :class="label ? 'btn-secondary' : 'btn-icon'"
    :title="copied ? 'Copied' : 'Copy'"
    :aria-label="label || 'Copy'"
    @click.stop="copy"
  >
    <Check v-if="copied" class="h-4 w-4 text-success" :stroke-width="2.2" />
    <Copy v-else class="h-4 w-4" :stroke-width="1.8" />
    <span v-if="label">{{ copied ? 'Copied' : label }}</span>
  </button>
</template>
