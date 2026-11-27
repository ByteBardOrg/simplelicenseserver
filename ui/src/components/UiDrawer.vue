<script setup>
import { onUnmounted, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: '' },
  eyebrow: { type: String, default: '' },
  width: { type: String, default: 'max-w-md' },
})

const emit = defineEmits(['close'])

const onKeydown = (event) => {
  if (event.key === 'Escape') emit('close')
}

watch(
  () => props.open,
  (open) => {
    if (open) window.addEventListener('keydown', onKeydown)
    else window.removeEventListener('keydown', onKeydown)
  },
  { immediate: true },
)

onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div v-if="open" class="fixed inset-0 z-50 bg-surface-lowest/60 backdrop-blur-[3px]" @click="emit('close')"></div>
    </Transition>

    <Transition
      enter-active-class="transition duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
      leave-active-class="transition duration-150 ease-in"
      enter-from-class="translate-x-8 opacity-0"
      leave-to-class="translate-x-8 opacity-0"
    >
      <aside
        v-if="open"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        class="fixed inset-y-0 right-0 z-50 flex w-full flex-col bg-surface-low shadow-float sm:inset-y-3 sm:right-3 sm:rounded-2xl"
        :class="width"
      >
        <header class="flex items-start justify-between gap-4 px-6 pb-4 pt-6">
          <div class="min-w-0">
            <p v-if="eyebrow" class="eyebrow">{{ eyebrow }}</p>
            <h2 class="mt-1 text-lg font-semibold tracking-tight text-ink-strong">
              <slot name="title">{{ title }}</slot>
            </h2>
          </div>
          <button class="btn-icon -mr-2 -mt-1" aria-label="Close" @click="emit('close')">
            <X class="h-4 w-4" :stroke-width="2" />
          </button>
        </header>

        <div class="flex-1 overflow-y-auto px-6 pb-6">
          <slot />
        </div>

        <footer v-if="$slots.footer" class="flex items-center gap-2 bg-surface-container/50 px-6 py-4 sm:rounded-b-2xl">
          <slot name="footer" />
        </footer>
      </aside>
    </Transition>
  </Teleport>
</template>
