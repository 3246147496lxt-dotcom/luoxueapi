<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Activity, Code2, FileClock, Settings } from 'lucide-vue-next'
import type { Component } from 'vue'
import type { DesktopSnapshot, ViewName } from '@/types'

defineProps<{ activeView: ViewName; snapshot: DesktopSnapshot }>()
const emit = defineEmits<{ navigate: [view: ViewName] }>()
const { t } = useI18n()

const items: Array<{ id: ViewName; icon: Component }> = [
  { id: 'overview', icon: Activity },
  { id: 'codex', icon: Code2 },
  { id: 'requests', icon: FileClock },
  { id: 'settings', icon: Settings }
]
</script>

<template>
  <aside class="sidebar">
    <div class="traffic-lights" data-tauri-drag-region aria-hidden="true">
    </div>

    <div class="sidebar-brand" data-tauri-drag-region>
      <span class="brand-mark"><img class="brand-mark__image" src="/logo.png" alt="" /></span>
      <span class="sidebar-brand__text">
        <strong>落雪API</strong>
        <small>Desktop</small>
      </span>
    </div>

    <nav class="sidebar-nav" :aria-label="t('common.primaryNavigation')">
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="sidebar-nav__item"
        :class="{ 'sidebar-nav__item--active': activeView === item.id }"
        :aria-current="activeView === item.id ? 'page' : undefined"
        @click="emit('navigate', item.id)"
      >
        <component :is="item.icon" :size="18" stroke-width="2" />
        <span>{{ t(`nav.${item.id}`) }}</span>
      </button>
    </nav>

    <div class="sidebar-account">
      <span class="avatar">{{ snapshot.accountEmail?.slice(0, 1).toUpperCase() }}</span>
      <span class="sidebar-account__copy">
        <strong>{{ snapshot.accountEmail }}</strong>
        <small>{{ snapshot.deviceName }}</small>
      </span>
      <span class="account-state" :title="t('common.connected')" />
    </div>
  </aside>
</template>
