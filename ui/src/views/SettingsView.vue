<script setup>
import { KeyRound, ShieldCheck, Webhook } from 'lucide-vue-next'
import PageHeader from '@/components/PageHeader.vue'

const sections = [
  { path: '/settings/api-keys', label: 'API keys', icon: KeyRound },
  { path: '/settings/webhooks', label: 'Webhooks', icon: Webhook },
  { path: '/settings/offline', label: 'Offline signing', icon: ShieldCheck },
]
</script>

<template>
  <div>
    <PageHeader title="Settings" description="Things you set up once and rarely touch again." />

    <div class="mt-8 grid gap-8 md:grid-cols-[12rem_minmax(0,1fr)]">
      <nav class="flex gap-1 overflow-x-auto md:flex-col">
        <router-link
          v-for="section in sections"
          :key="section.path"
          :to="section.path"
          class="flex items-center gap-2.5 whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium transition"
          active-class="bg-surface-container text-ink-strong shadow-ghost"
          :class="$route.path === section.path ? '' : 'text-ink-muted hover:text-ink'"
        >
          <component :is="section.icon" class="h-4 w-4" :stroke-width="1.8" />
          {{ section.label }}
        </router-link>
      </nav>

      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </div>
  </div>
</template>
