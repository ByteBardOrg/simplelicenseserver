<script setup>
const model = defineModel({ type: Boolean, default: false })

defineProps({
  label: { type: String, required: true },
  hint: { type: String, default: '' },
})
</script>

<template>
  <label class="flex cursor-pointer items-start justify-between gap-4 rounded-lg bg-surface-lowest/50 px-3.5 py-3 shadow-ghost">
    <span>
      <span class="block text-sm font-medium text-ink">{{ label }}</span>
      <span v-if="hint" class="mt-0.5 block text-xs leading-5 text-ink-faint">{{ hint }}</span>
    </span>
    <input v-model="model" type="checkbox" class="peer sr-only" />
    <span
      class="relative mt-0.5 h-5 w-9 shrink-0 rounded-full bg-surface-highest transition-colors peer-checked:bg-primary-container peer-focus-visible:ring-2 peer-focus-visible:ring-primary/60 after:absolute after:left-0.5 after:top-0.5 after:h-4 after:w-4 after:rounded-full after:bg-ink after:shadow after:transition-transform peer-checked:after:translate-x-4 peer-checked:after:bg-white"
      aria-hidden="true"
    ></span>
  </label>
</template>
