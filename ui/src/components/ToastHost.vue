<script setup>
import { CircleAlert, CircleCheck, X } from 'lucide-vue-next'
import { dismissToast, ui } from '@/lib/ui'
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed bottom-5 left-1/2 z-[60] flex w-full max-w-sm -translate-x-1/2 flex-col items-center gap-2 px-4">
      <TransitionGroup
        enter-active-class="transition duration-300 ease-[cubic-bezier(0.16,1,0.3,1)]"
        leave-active-class="transition duration-150"
        enter-from-class="translate-y-2 opacity-0"
        leave-to-class="opacity-0"
      >
        <div
          v-for="item in ui.toasts"
          :key="item.id"
          class="glass pointer-events-auto flex w-full items-start gap-3 rounded-xl px-4 py-3 text-sm shadow-float"
          role="status"
        >
          <CircleCheck v-if="item.tone === 'success'" class="mt-0.5 h-4 w-4 shrink-0 text-success" :stroke-width="2" />
          <CircleAlert v-else class="mt-0.5 h-4 w-4 shrink-0 text-critical" :stroke-width="2" />
          <p class="flex-1 text-ink">{{ item.message }}</p>
          <button class="text-ink-faint transition hover:text-ink" aria-label="Dismiss" @click="dismissToast(item.id)">
            <X class="h-3.5 w-3.5" :stroke-width="2" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
