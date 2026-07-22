<template>
  <section class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.8fr)_minmax(16rem,0.9fr)_minmax(18rem,1fr)]">
    <article class="dashboard-panel overflow-hidden" :aria-label="t('dashboard.announcements.title')">
      <PanelHeader icon="bell" :title="t('dashboard.announcements.title')">
        <span class="rounded-full border border-gray-200 px-2 py-0.5 text-[11px] text-gray-500 dark:border-dark-600 dark:text-dark-400">
          {{ t('dashboard.announcements.latest', { count: visibleAnnouncements.length }) }}
        </span>
      </PanelHeader>

      <div v-if="loadingAnnouncements" class="flex min-h-[420px] items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <div v-else-if="visibleAnnouncements.length === 0" class="flex min-h-[420px] flex-col items-center justify-center px-6 text-center">
        <Icon name="bell" size="lg" class="text-gray-300 dark:text-dark-600" />
        <p class="mt-3 text-sm text-gray-400 dark:text-dark-500">{{ t('dashboard.announcements.empty') }}</p>
      </div>
      <ol v-else class="announcement-timeline max-h-[460px] overflow-y-auto px-5 py-5 md:px-6">
        <li
          v-for="(announcement, index) in visibleAnnouncements"
          :key="announcement.id"
          class="announcement-item"
          :class="index % 2 === 0 ? 'announcement-item-right' : 'announcement-item-left'"
        >
          <span class="announcement-dot" :class="announcement.read_at ? 'bg-gray-300 dark:bg-dark-500' : 'bg-emerald-500'" />
          <div class="min-w-0">
            <p class="line-clamp-2 text-sm font-medium leading-5 text-gray-700 dark:text-gray-200">
              {{ announcement.title || plainAnnouncementContent(announcement.content) }}
            </p>
            <p class="mt-1 text-[11px] text-gray-400 dark:text-dark-500">
              {{ relativeTime(announcement.created_at) }}
            </p>
          </div>
        </li>
      </ol>
    </article>

    <article class="dashboard-panel overflow-hidden" :aria-label="t('dashboard.faq.title')">
      <PanelHeader icon="questionCircle" :title="t('dashboard.faq.title')" />
      <div class="divide-y divide-gray-100 px-4 py-2 dark:divide-dark-700">
        <div v-for="item in faqItems" :key="item.id">
          <button
            type="button"
            class="flex w-full items-center justify-between gap-3 px-2 py-4 text-left text-sm font-semibold text-gray-800 transition-colors hover:text-indigo-600 dark:text-gray-100 dark:hover:text-indigo-300"
            :aria-expanded="activeFaq === item.id"
            @click="activeFaq = activeFaq === item.id ? null : item.id"
          >
            <span>{{ t(item.questionKey) }}</span>
            <Icon :name="activeFaq === item.id ? 'x' : 'plus'" size="xs" class="shrink-0 text-gray-500 dark:text-dark-400" />
          </button>
          <Transition name="faq-answer">
            <p
              v-if="activeFaq === item.id"
              class="px-2 pb-4 text-sm leading-6 text-gray-500 dark:text-dark-400"
            >
              {{ t(item.answerKey) }}
            </p>
          </Transition>
        </div>
      </div>
    </article>

    <article class="dashboard-panel overflow-hidden" :aria-label="t('dashboard.availability.title')">
      <PanelHeader icon="clock" :title="t('dashboard.availability.title')" />

      <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
        <label class="relative block">
          <Icon name="search" size="sm" class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
          <input
            v-model="monitorSearch"
            type="search"
            :aria-label="t('dashboard.availability.search')"
            class="h-9 w-full rounded-xl border-0 bg-gray-100/80 pl-9 pr-3 text-sm text-gray-700 outline-none ring-1 ring-transparent transition focus:bg-white focus:ring-indigo-300 dark:bg-dark-900/55 dark:text-gray-200 dark:focus:bg-dark-900 dark:focus:ring-indigo-700"
            :placeholder="t('dashboard.availability.search')"
          />
        </label>
      </div>

      <div v-if="loadingMonitors" class="flex min-h-[360px] items-center justify-center">
        <LoadingSpinner size="md" />
      </div>
      <div v-else-if="!monitorEnabled || filteredMonitors.length === 0" class="flex min-h-[360px] flex-col items-center justify-center px-6 text-center">
        <Icon name="server" size="lg" class="text-gray-300 dark:text-dark-600" />
        <p class="mt-3 text-sm text-gray-400 dark:text-dark-500">
          {{ monitorEnabled ? t('dashboard.availability.empty') : t('dashboard.availability.disabled') }}
        </p>
      </div>
      <div v-else class="max-h-[382px] divide-y divide-gray-100 overflow-y-auto px-4 dark:divide-dark-700">
        <div v-for="monitor in filteredMonitors" :key="monitor.id" class="py-3.5">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex min-w-0 items-center gap-2">
                <span class="h-2 w-2 shrink-0 rounded-full" :class="monitorDotClass(monitor.primary_status)" />
                <p class="truncate text-sm font-semibold text-gray-800 dark:text-gray-100" :title="monitor.primary_model">
                  {{ monitor.primary_model || monitor.name }}
                </p>
              </div>
              <div class="mt-1.5 flex flex-wrap items-center gap-1.5 pl-4">
                <span class="rounded-md bg-emerald-50 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-950/45 dark:text-emerald-300">
                  {{ monitor.group_name || monitor.provider }}
                </span>
                <span class="text-[10px] text-gray-400 dark:text-dark-500">{{ monitor.name }}</span>
              </div>
            </div>
            <span class="shrink-0 text-xs text-gray-400 dark:text-dark-500">
              {{ formatAvailability(monitor.availability_7d) }}%
            </span>
          </div>
          <div class="mt-2 pl-4">
            <div class="flex items-center gap-2 text-[11px] text-gray-400 dark:text-dark-500">
              <span>{{ statusLabel(monitor.primary_status) }}</span>
              <span v-if="monitor.primary_latency_ms !== null">· {{ monitor.primary_latency_ms }}ms</span>
            </div>
            <div class="mt-1.5 h-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
              <span
                class="block h-full rounded-full transition-[width] duration-300"
                :class="monitorBarClass(monitor.primary_status)"
                :style="{ width: `${Math.min(100, Math.max(0, monitor.availability_7d || 0))}%` }"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-center gap-3 border-t border-gray-100 px-4 py-3 text-[10px] text-gray-500 dark:border-dark-700 dark:text-dark-400">
        <LegendDot color="bg-red-500" :label="t('dashboard.availability.failed')" />
        <LegendDot color="bg-emerald-500" :label="t('dashboard.availability.operational')" />
        <LegendDot color="bg-amber-500" :label="t('dashboard.availability.degraded')" />
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { UserAnnouncement } from '@/types'
import type { MonitorStatus, UserMonitorView } from '@/api/channelMonitor'

const props = defineProps<{
  announcements: UserAnnouncement[]
  monitors: UserMonitorView[]
  loadingAnnouncements: boolean
  loadingMonitors: boolean
  monitorEnabled: boolean
}>()

const { t, locale } = useI18n()
const activeFaq = ref<string | null>(null)
const monitorSearch = ref('')

const visibleAnnouncements = computed(() => props.announcements.slice(0, 8))
const faqItems = ['models', 'group', 'endpoint', 'billing', 'client'].map((id) => ({
  id,
  questionKey: `home.faq.items.${id}.question`,
  answerKey: `home.faq.items.${id}.answer`,
}))

const filteredMonitors = computed(() => {
  const query = monitorSearch.value.trim().toLowerCase()
  if (!query) return props.monitors
  return props.monitors.filter((monitor) => [
    monitor.name,
    monitor.primary_model,
    monitor.group_name,
    monitor.provider,
  ].some((value) => value?.toLowerCase().includes(query)))
})

type PanelIconName = 'bell' | 'questionCircle' | 'clock'

const PanelHeader = defineComponent({
  name: 'DashboardPanelHeader',
  props: {
    icon: { type: String as PropType<PanelIconName>, required: true },
    title: { type: String, required: true },
  },
  setup(componentProps, { slots }) {
    return () => h('div', {
      class: 'flex min-h-[61px] items-center justify-between gap-3 border-b border-gray-100 px-5 py-3 text-sm font-medium text-gray-800 dark:border-dark-700 dark:text-gray-100',
    }, [
      h('div', { class: 'flex items-center gap-2' }, [
        h(Icon, { name: componentProps.icon, size: 'sm', strokeWidth: 1.8 }),
        h('span', componentProps.title),
      ]),
      slots.default?.(),
    ])
  },
})

const LegendDot = defineComponent({
  name: 'DashboardLegendDot',
  props: {
    color: { type: String, required: true },
    label: { type: String, required: true },
  },
  setup(componentProps) {
    return () => h('span', { class: 'inline-flex items-center gap-1' }, [
      h('span', { class: `h-1.5 w-1.5 rounded-full ${componentProps.color}` }),
      h('span', componentProps.label),
    ])
  },
})

function relativeTime(value: string): string {
  const timestamp = new Date(value).getTime()
  if (!Number.isFinite(timestamp)) return value
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000))
  const formatter = new Intl.RelativeTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', { numeric: 'auto' })
  if (seconds < 60) return formatter.format(-seconds, 'second')
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return formatter.format(-minutes, 'minute')
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return formatter.format(-hours, 'hour')
  const days = Math.floor(hours / 24)
  if (days < 30) return formatter.format(-days, 'day')
  return formatter.format(-Math.floor(days / 30), 'month')
}

function plainAnnouncementContent(value: string): string {
  return value.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
}

function monitorDotClass(status: MonitorStatus): string {
  if (status === 'operational') return 'bg-emerald-500'
  if (status === 'degraded') return 'bg-amber-500'
  return 'bg-red-500'
}

function monitorBarClass(status: MonitorStatus): string {
  if (status === 'operational') return 'bg-emerald-500/80'
  if (status === 'degraded') return 'bg-amber-500/80'
  return 'bg-red-500/80'
}

function statusLabel(status: MonitorStatus): string {
  if (status === 'operational') return t('dashboard.availability.operational')
  if (status === 'degraded') return t('dashboard.availability.degraded')
  return t('dashboard.availability.failed')
}

function formatAvailability(value: number): string {
  if (!Number.isFinite(value)) return '0.00'
  return value.toFixed(2)
}
</script>

<style scoped>
.dashboard-panel {
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-form);
}

.announcement-timeline {
  position: relative;
}

.announcement-timeline::before {
  position: absolute;
  top: 1.25rem;
  bottom: 1.25rem;
  left: 50%;
  width: 1px;
  content: '';
  background: rgb(209 213 219);
}

.announcement-item {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 1.75rem minmax(0, 1fr);
  min-height: 4.75rem;
  align-items: start;
}

.announcement-item > div {
  grid-column: 3;
  padding: 0.1rem 0 1.2rem 0.25rem;
}

.announcement-item-left > div {
  grid-column: 1;
  padding-left: 0;
  padding-right: 0.25rem;
  text-align: right;
}

.announcement-dot {
  position: relative;
  z-index: 1;
  grid-column: 2;
  justify-self: center;
  width: 0.55rem;
  height: 0.55rem;
  margin-top: 0.3rem;
  border-radius: 9999px;
  box-shadow: 0 0 0 4px var(--lx-clay-surface);
}

.faq-answer-enter-active,
.faq-answer-leave-active {
  transition: opacity 140ms ease, transform 140ms ease;
}

.faq-answer-enter-from,
.faq-answer-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

:global(.dark) .announcement-timeline::before {
  background: rgb(71 85 105);
}

@media (max-width: 639px) {
  .announcement-timeline::before {
    left: 0.3rem;
  }

  .announcement-item,
  .announcement-item-left {
    grid-template-columns: 1rem minmax(0, 1fr);
  }

  .announcement-item > div,
  .announcement-item-left > div {
    grid-column: 2;
    padding: 0.1rem 0 1.2rem 0.5rem;
    text-align: left;
  }

  .announcement-dot {
    grid-column: 1;
  }
}
</style>
