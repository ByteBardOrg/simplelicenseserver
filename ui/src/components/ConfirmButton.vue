<script setup>
import { onUnmounted, ref } from 'vue'

// A destructive button that asks for a second click instead of a browser confirm() dialog.
const props = defineProps({
  label: { type: String, required: true },
  confirmLabel: { type: String, default: 'Click to confirm' },
  disabled: { type: Boolean, default: false },
  size: { type: String, default: 'md' },
})

const emit = defineEmits(['confirm'])

const armed = ref(false)
let timer = 0

const click = () => {
  if (props.disabled) return
  if (armed.value) {
    window.clearTimeout(timer)
    armed.value = false
    emit('confirm')
    return
  }
  armed.value = true
  timer = window.setTimeout(() => {
    armed.value = false
  }, 3000)
}

onUnmounted(() => window.clearTimeout(timer))
</script>

<template>
  <button
    type="button"
    :disabled="disabled"
    :class="[
      size === 'sm' ? 'btn px-2.5 py-1 text-xs' : 'btn',
      armed ? 'bg-critical text-surface hover:bg-critical' : 'bg-critical/10 text-critical hover:bg-critical/20',
    ]"
    @click.stop="click"
  >
    <slot />
    {{ armed ? confirmLabel : label }}
  </button>
</template>
