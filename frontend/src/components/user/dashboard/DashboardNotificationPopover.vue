<template>
  <div ref="root" class="dashboard-notifications">
    <button
      type="button"
      class="dashboard-notifications__trigger"
      :aria-label="t('announcements.title')"
      :aria-expanded="open"
      aria-haspopup="dialog"
      @click="toggle"
    >
      <NotificationIcon aria-hidden="true" />
      <span v-if="unreadCount > 0" class="dashboard-notifications__dot" aria-hidden="true" />
    </button>

    <Transition name="dashboard-notifications-popover">
      <section v-if="open" class="dashboard-notifications__panel" role="dialog">
        <header class="dashboard-notifications__header">
          <h2>{{ t('dashboard.workspace.notifications') }}</h2>
          <button
            v-if="unreadCount > 0"
            type="button"
            :disabled="loading"
            @click="markAllRead"
          >
            {{ t('announcements.markAllRead') }}
          </button>
        </header>

        <div v-if="loading && announcements.length === 0" class="dashboard-notifications__state">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="announcements.length === 0" class="dashboard-notifications__state">
          {{ t('announcements.empty') }}
        </div>
        <div v-else class="dashboard-notifications__list">
          <button
            v-for="announcement in visibleAnnouncements"
            :key="announcement.id"
            type="button"
            class="dashboard-notifications__item"
            :class="{ 'dashboard-notifications__item--unread': !announcement.read_at }"
            @click="markRead(announcement.id)"
          >
            <span class="dashboard-notifications__item-head">
              <strong>{{ announcement.title }}</strong>
              <time :datetime="announcement.created_at">
                {{ formatRelativeTime(announcement.created_at) }}
              </time>
            </span>
            <span>{{ t('dashboard.workspace.systemAnnouncement') }}</span>
          </button>
        </div>

        <button
          v-if="announcements.length > collapsedCount"
          type="button"
          class="dashboard-notifications__view-all"
          @click="showAll = !showAll"
        >
          {{ showAll ? t('dashboard.workspace.collapseNotifications') : t('announcements.viewAll') }}
        </button>
      </section>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import NotificationIcon from '@/components/icons/NotificationIcon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { formatRelativeTime } from '@/utils/format'

const collapsedCount = 4
const { t } = useI18n()
const announcementStore = useAnnouncementStore()
const { announcements, loading } = storeToRefs(announcementStore)
const root = ref<HTMLElement | null>(null)
const open = ref(false)
const showAll = ref(false)

const unreadCount = computed(() => announcementStore.unreadCount)
const visibleAnnouncements = computed(() => (
  showAll.value ? announcements.value : announcements.value.slice(0, collapsedCount)
))

function toggle(): void {
  open.value = !open.value
  if (open.value) void announcementStore.fetchAnnouncements()
}

function markRead(id: number): void {
  void announcementStore.markAsRead(id)
}

async function markAllRead(): Promise<void> {
  try {
    await announcementStore.markAllAsRead()
  } catch (error) {
    console.warn('Failed to mark dashboard announcements as read:', error)
  }
}

function handlePointerDown(event: PointerEvent): void {
  if (!open.value || root.value?.contains(event.target as Node)) return
  open.value = false
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('pointerdown', handlePointerDown)
  document.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.dashboard-notifications {
  position: relative;
}

.dashboard-notifications__trigger {
  position: relative;
  display: inline-flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  color: var(--workspace-dashboard-text-muted);
  background: transparent;
  transition: color 140ms ease, background-color 140ms ease;
}

.dashboard-notifications__trigger :deep(svg) {
  width: 20px;
  height: 20px;
}

.dashboard-notifications__trigger:hover {
  color: var(--workspace-dashboard-notification-hover);
  background: transparent;
}

.dashboard-notifications__trigger:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-notifications__dot {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border: 1px solid var(--workspace-card-surface);
  border-radius: 999px;
  background: var(--lx-clay-danger);
}

.dashboard-notifications__panel {
  position: absolute;
  z-index: 50;
  top: calc(100% + 8px);
  right: 0;
  width: min(340px, calc(100vw - 32px));
  overflow: hidden;
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 16px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-notifications__header {
  display: flex;
  min-height: 54px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 var(--workspace-space-4);
  border-bottom: 1px solid var(--workspace-dashboard-divider);
}

.dashboard-notifications__header h2 {
  color: var(--workspace-dashboard-text-strong);
  font-size: var(--workspace-type-navigation-size);
  font-weight: 700;
}

.dashboard-notifications__header button,
.dashboard-notifications__view-all {
  border: 0;
  color: var(--workspace-dashboard-text-subtle);
  background: transparent;
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-notifications__header button:hover:not(:disabled),
.dashboard-notifications__view-all:hover {
  color: var(--workspace-dashboard-notification-hover);
}

.dashboard-notifications__header button:disabled {
  cursor: wait;
  opacity: 0.55;
}

.dashboard-notifications__list {
  max-height: 360px;
  overflow-y: auto;
}

.dashboard-notifications__item {
  display: block;
  width: 100%;
  min-height: 70px;
  padding: var(--workspace-space-4);
  border: 0;
  border-bottom: 1px solid var(--workspace-dashboard-divider);
  color: var(--workspace-dashboard-text-muted);
  background: transparent;
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: var(--workspace-type-secondary-weight);
  text-align: left;
  transition: background-color 140ms ease;
}

.dashboard-notifications__item:hover,
.dashboard-notifications__item--unread {
  background: var(--workspace-dashboard-hover);
}

.dashboard-notifications__item-head {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 5px;
}

.dashboard-notifications__item-head strong {
  overflow: hidden;
  color: var(--workspace-dashboard-text-strong);
  font-size: var(--workspace-type-secondary-size);
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-notifications__item-head time {
  flex: 0 0 auto;
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 2px);
  font-weight: var(--workspace-type-secondary-weight);
}

.dashboard-notifications__state {
  display: flex;
  min-height: 150px;
  align-items: center;
  justify-content: center;
  color: var(--workspace-dashboard-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.dashboard-notifications__view-all {
  width: 100%;
  min-height: 44px;
  border-top: 1px solid var(--workspace-dashboard-divider);
  font-weight: 700;
}

.dashboard-notifications-popover-enter-active,
.dashboard-notifications-popover-leave-active {
  transition: opacity 120ms ease, transform 120ms ease;
  transform-origin: top right;
}

.dashboard-notifications-popover-enter-from,
.dashboard-notifications-popover-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.98);
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-notifications__trigger,
  .dashboard-notifications__item,
  .dashboard-notifications-popover-enter-active,
  .dashboard-notifications-popover-leave-active {
    transition-duration: 0.01ms;
  }
}

:global(html.dark .dashboard-notifications .dashboard-notifications__panel) {
  box-shadow: 0 16px 40px rgb(0 0 0 / 0.28);
}
</style>
