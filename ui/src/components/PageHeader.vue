<script setup>
defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
})
</script>

<template>
  <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight text-ink-strong sm:text-[1.75rem]">{{ title }}</h1>
      <p v-if="description" class="mt-1.5 max-w-xl text-sm leading-6 text-ink-muted">{{ description }}</p>
    </div>
    <div v-if="$slots.default" class="flex shrink-0 items-center gap-2">
      <slot />
    </div>
  </div>
</template>
