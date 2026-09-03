<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { opsAPI, type AlertEventsQuery } from '@/api/admin/ops'
import type { AlertEvent } from '../types'
import { formatDateTime } from '../utils/opsFormatters'

interface OpsAlertRelatedLogsPayload {
  alertId: number
  firedAt: string
  resolvedAt?: string | null
  platform?: string
  groupId?: number
  requestId?: string
  region?: string
  title?: string
  severity?: string
  status?: string
}

const props = withDefaults(defineProps<{
  enabled?: boolean
  platformFilter?: string
  groupIdFilter?: number | null
}>(), {
  enabled: true,
  platformFilter: '',
  groupIdFilter: null
})

const emit = defineEmits<{
  'view-related-logs': [payload: OpsAlertRelatedLogsPayload]
  'update:platform': [value: string]
  'update:group': [value: number | null]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const PAGE_SIZE = 10

const loading = ref(false)
const loadingMore = ref(false)
const events = ref<AlertEvent[]>([])
const hasMore = ref(true)
const groups = ref<Array<{ id: number; name: string; platform: string }>>([])

const selected = ref<AlertEvent | null>(null)
const detailLoading = ref(false)
const detailActionLoading = ref(false)
const historyLoading = ref(false)
const history = ref<AlertEvent[]>([])
const historyRange = ref('7d')
const historyRangeOptions = computed(() => [
  { value: '7d', label: t('admin.ops.timeRange.7d') },
  { value: '30d', label: t('admin.ops.timeRange.30d') }
])

const silenceDuration = ref('1h')
const silenceDurationOptions = computed(() => [
  { value: '1h', label: '1h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' }
])

const timeRange = ref('1h')
const timeRangeOptions = computed(() => [
  { value: '5m', label: '5m' },
  { value: '30m', label: '30m' },
  { value: '1h', label: '1h' },
  { value: '6h', label: '6h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' }
])

const severity = ref<string>('')
const severityOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'P0', label: 'P0' },
  { value: 'P1', label: 'P1' },
  { value: 'P2', label: 'P2' },
  { value: 'P3', label: 'P3' }
])

const status = ref<string>('')
const statusOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'firing', label: t('admin.ops.alertEvents.status.firing') },
  { value: 'resolved', label: t('admin.ops.alertEvents.status.resolved') },
  { value: 'manual_resolved', label: t('admin.ops.alertEvents.status.manualResolved') }
])

const emailSent = ref<string>('')
const emailSentOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'true', label: t('admin.ops.alertEvents.table.emailSent') },
  { value: 'false', label: t('admin.ops.alertEvents.table.emailIgnored') }
])

const platformOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' }
])

const groupOptions = computed(() => {
  const visible = props.platformFilter
    ? groups.value.filter((group) => group.platform === props.platformFilter)
    : groups.value
  return [
    { value: null, label: t('common.all') },
    ...visible.map((group) => ({ value: group.id, label: group.name }))
  ]
})

let listController: AbortController | null = null
let loadMoreController: AbortController | null = null
let historyController: AbortController | null = null
let listRequestSequence = 0
let selectionSequence = 0

function isCanceledRequest(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false
  const value = err as Record<string, unknown>
  return value.code === 'ERR_CANCELED' || value.name === 'AbortError'
}

function abortListRequests() {
  listController?.abort()
  loadMoreController?.abort()
  listController = null
  loadMoreController = null
}

function resetSelection() {
  selectionSequence += 1
  historyController?.abort()
  historyController = null
  selected.value = null
  detailLoading.value = false
  historyLoading.value = false
  history.value = []
}

function resetDisabledState() {
  listRequestSequence += 1
  abortListRequests()
  resetSelection()
  loading.value = false
  loadingMore.value = false
  events.value = []
  hasMore.value = false
}

function buildQuery(overrides: Partial<AlertEventsQuery> = {}): AlertEventsQuery {
  const query: AlertEventsQuery = {
    limit: PAGE_SIZE,
    time_range: timeRange.value
  }
  if (severity.value) query.severity = severity.value
  if (status.value) query.status = status.value
  if (emailSent.value === 'true') query.email_sent = true
  if (emailSent.value === 'false') query.email_sent = false
  if (props.platformFilter) query.platform = props.platformFilter
  if (props.groupIdFilter != null) query.group_id = props.groupIdFilter
  return { ...query, ...overrides }
}

function updatePlatform(value: string | number | boolean | null) {
  const nextPlatform = typeof value === 'string' ? value : ''
  const currentGroup = groups.value.find((group) => group.id === props.groupIdFilter)
  if (currentGroup && nextPlatform && currentGroup.platform !== nextPlatform) emit('update:group', null)
  emit('update:platform', nextPlatform)
}

function updateGroup(value: string | number | boolean | null) {
  if (value === null || value === '' || typeof value === 'boolean') {
    emit('update:group', null)
    return
  }
  const parsed = typeof value === 'number' ? value : Number.parseInt(value, 10)
  emit('update:group', Number.isFinite(parsed) && parsed > 0 ? parsed : null)
}

async function loadFirstPage() {
  if (!props.enabled) return

  listRequestSequence += 1
  const requestSequence = listRequestSequence
  abortListRequests()
  const controller = new AbortController()
  listController = controller
  loading.value = true
  loadingMore.value = false

  try {
    const data = await opsAPI.listAlertEvents(buildQuery(), { signal: controller.signal })
    if (controller.signal.aborted || requestSequence !== listRequestSequence || !props.enabled) return
    events.value = data
    hasMore.value = data.length === PAGE_SIZE
  } catch (err: any) {
    if (controller.signal.aborted || requestSequence !== listRequestSequence || isCanceledRequest(err)) return
    console.error('[OpsAlertEventsCard] Failed to load alert events', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.alertEvents.loadFailed'))
    events.value = []
    hasMore.value = false
  } finally {
    if (requestSequence === listRequestSequence) loading.value = false
    if (listController === controller) listController = null
  }
}

async function loadMore() {
  if (!props.enabled || loadingMore.value || loading.value || !hasMore.value) return
  const last = events.value[events.value.length - 1]
  if (!last) return

  const requestSequence = listRequestSequence
  loadMoreController?.abort()
  const controller = new AbortController()
  loadMoreController = controller
  loadingMore.value = true

  try {
    const data = await opsAPI.listAlertEvents(
      buildQuery({ before_fired_at: last.fired_at || last.created_at, before_id: last.id }),
      { signal: controller.signal }
    )
    if (controller.signal.aborted || requestSequence !== listRequestSequence || !props.enabled) return
    if (!data.length) {
      hasMore.value = false
      return
    }
    events.value = [...events.value, ...data]
    if (data.length < PAGE_SIZE) hasMore.value = false
  } catch (err) {
    if (controller.signal.aborted || requestSequence !== listRequestSequence || isCanceledRequest(err)) return
    console.error('[OpsAlertEventsCard] Failed to load more alert events', err)
    hasMore.value = false
  } finally {
    if (loadMoreController === controller) {
      loadMoreController = null
      loadingMore.value = false
    }
  }
}

function onScroll(event: Event) {
  const element = event.target as HTMLElement | null
  if (!element) return
  const nearBottom = element.scrollTop + element.clientHeight >= element.scrollHeight - 120
  if (nearBottom) void loadMore()
}

function getDimensionString(event: AlertEvent | null | undefined, key: string): string {
  const value = event?.dimensions?.[key]
  if (value == null) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

function getDimensionNumber(event: AlertEvent | null | undefined, key: string): number | undefined {
  const value = event?.dimensions?.[key]
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) return value
  if (typeof value !== 'string' || !value.trim()) return undefined
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined
}

function formatDurationMs(ms: number): string {
  const safe = Math.max(0, Math.floor(ms))
  const seconds = Math.floor(safe / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

function formatDurationLabel(event: AlertEvent): string {
  const firedAt = new Date(event.fired_at || event.created_at)
  if (Number.isNaN(firedAt.getTime())) return '-'
  const statusValue = String(event.status || '').trim().toLowerCase()

  if (event.resolved_at) {
    const resolvedAt = new Date(event.resolved_at)
    if (!Number.isNaN(resolvedAt.getTime())) {
      const prefix = statusValue === 'manual_resolved'
        ? t('admin.ops.alertEvents.status.manualResolved')
        : t('admin.ops.alertEvents.status.resolved')
      return `${prefix} ${formatDurationMs(resolvedAt.getTime() - firedAt.getTime())}`
    }
  }

  return `${t('admin.ops.alertEvents.status.firing')} ${formatDurationMs(Date.now() - firedAt.getTime())}`
}

function formatDimensionsSummary(event: AlertEvent): string {
  const parts: string[] = []
  const platform = getDimensionString(event, 'platform')
  if (platform) parts.push(`platform=${platform}`)
  const groupId = getDimensionNumber(event, 'group_id')
  if (groupId) parts.push(`group_id=${groupId}`)
  const region = getDimensionString(event, 'region')
  if (region) parts.push(`region=${region}`)
  return parts.length ? parts.join(' · ') : '-'
}

function formatMetric(event: AlertEvent): string {
  if (typeof event.metric_value !== 'number' || typeof event.threshold_value !== 'number') return '-'
  return `${event.metric_value.toFixed(2)} / ${event.threshold_value.toFixed(2)}`
}

async function loadHistory(event: AlertEvent, requestSequence = selectionSequence) {
  if (!props.enabled) return
  historyController?.abort()
  const controller = new AbortController()
  historyController = controller
  historyLoading.value = true

  try {
    const platform = getDimensionString(event, 'platform')
    const groupId = getDimensionNumber(event, 'group_id')
    const items = await opsAPI.listAlertEvents({
      limit: 20,
      time_range: historyRange.value,
      platform: platform || undefined,
      group_id: groupId,
      status: ''
    }, { signal: controller.signal })

    if (
      controller.signal.aborted ||
      requestSequence !== selectionSequence ||
      selected.value?.id !== event.id ||
      !props.enabled
    ) return

    history.value = items.filter((item) => {
      if (item.rule_id !== event.rule_id) return false
      if (getDimensionString(item, 'platform') !== platform) return false
      return (getDimensionNumber(item, 'group_id') ?? null) === (groupId ?? null)
    })
  } catch (err) {
    if (controller.signal.aborted || requestSequence !== selectionSequence || isCanceledRequest(err)) return
    console.error('[OpsAlertEventsCard] Failed to load alert history', err)
    history.value = []
  } finally {
    if (historyController === controller) historyController = null
    if (requestSequence === selectionSequence && selected.value?.id === event.id) {
      historyLoading.value = false
    }
  }
}

async function openDetail(row: AlertEvent) {
  if (!props.enabled) return

  selectionSequence += 1
  const requestSequence = selectionSequence
  historyController?.abort()
  historyController = null
  selected.value = row
  history.value = []
  detailLoading.value = true
  historyLoading.value = true
  let detail = row

  try {
    detail = await opsAPI.getAlertEvent(row.id)
    if (requestSequence !== selectionSequence || !props.enabled) return
    selected.value = detail
  } catch (err: any) {
    if (requestSequence !== selectionSequence) return
    console.error('[OpsAlertEventsCard] Failed to load alert detail', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.alertEvents.detail.loadFailed'))
  } finally {
    if (requestSequence === selectionSequence) detailLoading.value = false
  }

  if (requestSequence === selectionSequence && props.enabled) {
    await loadHistory(detail, requestSequence)
  }
}

function durationToUntilRFC3339(duration: string): string {
  const now = Date.now()
  if (duration === '1h') return new Date(now + 60 * 60 * 1000).toISOString()
  if (duration === '24h') return new Date(now + 24 * 60 * 60 * 1000).toISOString()
  if (duration === '7d') return new Date(now + 7 * 24 * 60 * 60 * 1000).toISOString()
  return new Date(now + 60 * 60 * 1000).toISOString()
}

const selectedPlatform = computed(() => getDimensionString(selected.value, 'platform'))
const canSilence = computed(() => (
  props.enabled &&
  !!selected.value &&
  !!selectedPlatform.value &&
  !detailLoading.value &&
  !detailActionLoading.value
))
const canManualResolve = computed(() => (
  props.enabled &&
  selected.value?.status === 'firing' &&
  !detailLoading.value &&
  !detailActionLoading.value
))
const canViewRelatedLogs = computed(() => props.enabled && !!selected.value && !detailLoading.value)

async function silenceAlert() {
  const event = selected.value
  if (!event || !canSilence.value) return

  detailActionLoading.value = true
  try {
    await opsAPI.createAlertSilence({
      rule_id: event.rule_id,
      platform: selectedPlatform.value,
      group_id: getDimensionNumber(event, 'group_id'),
      region: getDimensionString(event, 'region') || undefined,
      until: durationToUntilRFC3339(silenceDuration.value),
      reason: `silence from UI (${silenceDuration.value})`
    })
    appStore.showSuccess(t('admin.ops.alertEvents.detail.silenceSuccess'))
  } catch (err: any) {
    console.error('[OpsAlertEventsCard] Failed to silence alert', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.alertEvents.detail.silenceFailed'))
  } finally {
    detailActionLoading.value = false
  }
}

async function manualResolve() {
  const event = selected.value
  if (!event || !canManualResolve.value) return

  const eventId = event.id
  detailActionLoading.value = true
  try {
    await opsAPI.updateAlertEventStatus(eventId, 'manual_resolved')
    appStore.showSuccess(t('admin.ops.alertEvents.detail.manualResolvedSuccess'))

    const detail = await opsAPI.getAlertEvent(eventId)
    if (props.enabled && selected.value?.id === eventId) selected.value = detail
    await loadFirstPage()
    if (props.enabled && selected.value?.id === eventId) {
      await loadHistory(selected.value, selectionSequence)
    }
  } catch (err: any) {
    console.error('[OpsAlertEventsCard] Failed to resolve alert', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.alertEvents.detail.manualResolvedFailed'))
  } finally {
    detailActionLoading.value = false
  }
}

function viewRelatedLogs() {
  const event = selected.value
  if (!event || !canViewRelatedLogs.value) return

  emit('view-related-logs', {
    alertId: event.id,
    firedAt: event.fired_at || event.created_at,
    resolvedAt: event.resolved_at ?? null,
    platform: getDimensionString(event, 'platform') || undefined,
    groupId: getDimensionNumber(event, 'group_id'),
    requestId: getDimensionString(event, 'request_id') || undefined,
    region: getDimensionString(event, 'region') || undefined,
    title: event.title,
    severity: String(event.severity || '') || undefined,
    status: event.status
  })
}

function severityBadgeClass(value: string | undefined): string {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'p0' || normalized === 'critical') return 'ops-alert-badge--danger'
  if (normalized === 'p1' || normalized === 'warning') return 'ops-alert-badge--warning'
  if (normalized === 'p2' || normalized === 'info') return 'ops-alert-badge--accent'
  return 'ops-alert-badge--neutral'
}

function statusBadgeClass(value: string | undefined): string {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'firing') return 'ops-alert-status--firing'
  if (normalized === 'resolved') return 'ops-alert-status--resolved'
  if (normalized === 'manual_resolved') return 'ops-alert-status--manual'
  return 'ops-alert-status--neutral'
}

function formatStatusLabel(value: string | undefined): string {
  const normalized = String(value || '').trim().toLowerCase()
  if (!normalized) return '-'
  if (normalized === 'firing') return t('admin.ops.alertEvents.status.firing')
  if (normalized === 'resolved') return t('admin.ops.alertEvents.status.resolved')
  if (normalized === 'manual_resolved') return t('admin.ops.alertEvents.status.manualResolved')
  return normalized.toUpperCase()
}

const empty = computed(() => props.enabled && events.value.length === 0 && !loading.value)

onMounted(() => {
  void adminAPI.groups.getAll()
    .then((result) => {
      groups.value = result.map((group) => ({ id: group.id, name: group.name, platform: group.platform }))
    })
    .catch((error) => {
      console.error('[OpsAlertEventsCard] Failed to load groups', error)
      groups.value = []
    })
  if (props.enabled) void loadFirstPage()
})

onUnmounted(() => {
  listRequestSequence += 1
  abortListRequests()
  resetSelection()
})

watch([timeRange, severity, status, emailSent], () => {
  if (!props.enabled) return
  resetSelection()
  events.value = []
  hasMore.value = true
  void loadFirstPage()
})

watch(() => [props.platformFilter, props.groupIdFilter] as const, () => {
  if (!props.enabled) return
  resetSelection()
  events.value = []
  hasMore.value = true
  void loadFirstPage()
})

watch(historyRange, () => {
  if (props.enabled && selected.value) void loadHistory(selected.value, selectionSequence)
})

watch(() => props.enabled, (enabled, wasEnabled) => {
  if (!enabled) {
    resetDisabledState()
    return
  }
  if (!wasEnabled) {
    hasMore.value = true
    void loadFirstPage()
  }
})
</script>

<template>
  <section
    class="ops-alert-workbench"
    :aria-label="t('admin.ops.alertEvents.title')"
    data-testid="ops-alert-workbench"
  >
    <div class="ops-alert-layout">
      <aside
        class="ops-alert-rail"
        :aria-label="t('admin.ops.alertEvents.title')"
        data-testid="ops-alert-rail"
      >
        <header class="ops-alert-filter-header">
          <div class="flex items-center justify-between gap-3">
            <div class="flex min-w-0 items-center gap-1.5">
              <Icon name="filter" size="xs" aria-hidden="true" />
              <h3>{{ t('admin.ops.alertEvents.filterLabel') }}</h3>
            </div>
            <button
              type="button"
              class="ops-alert-refresh-button"
              :disabled="!enabled || loading"
              data-testid="ops-alert-refresh"
              @click="loadFirstPage"
            >
              {{ t('admin.ops.alertEvents.refreshQueue') }}
            </button>
          </div>
        </header>

        <div
          class="ops-alert-filters grid grid-cols-2 gap-2"
          data-testid="ops-alert-filters"
        >
          <div class="ops-alert-filter-field ops-alert-filter-field--wide">
            <span>{{ t('admin.ops.errorLog.platform') }}</span>
            <Select
              :model-value="platformFilter"
              :options="platformOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.errorLog.platform')"
              class="min-w-0"
              @update:model-value="updatePlatform"
            />
          </div>
          <div class="ops-alert-filter-field ops-alert-filter-field--wide">
            <span>{{ t('admin.ops.errorLog.group') }}</span>
            <Select
              :model-value="groupIdFilter"
              :options="groupOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.errorLog.group')"
              class="min-w-0"
              @update:model-value="updateGroup"
            />
          </div>
          <div class="ops-alert-filter-field">
            <span>{{ t('admin.ops.alertEvents.filterTime') }}</span>
            <Select
              :model-value="timeRange"
              :options="timeRangeOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.systemLogs.timeRange')"
              class="min-w-0"
              @change="timeRange = String($event || '1h')"
            />
          </div>
          <div class="ops-alert-filter-field">
            <span>{{ t('admin.ops.alertEvents.table.severity') }}</span>
            <Select
              :model-value="severity"
              :options="severityOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.alertEvents.table.severity')"
              class="min-w-0"
              @change="severity = String($event || '')"
            />
          </div>
          <div class="ops-alert-filter-field">
            <span>{{ t('admin.ops.alertEvents.table.status') }}</span>
            <Select
              :model-value="status"
              :options="statusOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.alertEvents.table.status')"
              class="min-w-0"
              @change="status = String($event || '')"
            />
          </div>
          <div class="ops-alert-filter-field">
            <span>{{ t('admin.ops.alertEvents.filterEmail') }}</span>
            <Select
              :model-value="emailSent"
              :options="emailSentOptions"
              :disabled="!enabled"
              :aria-label="t('admin.ops.alertEvents.filterEmail')"
              class="min-w-0"
              @change="emailSent = String($event || '')"
            />
          </div>
        </div>

        <div class="ops-alert-queue-heading">
          <h3>{{ t('admin.ops.alertEvents.title') }}</h3>
          <p>{{ t('admin.ops.alertEvents.description') }}</p>
        </div>

        <div
          class="ops-alert-queue-scroll"
          role="region"
          tabindex="0"
          :aria-label="t('admin.ops.alertEvents.title')"
          data-testid="ops-alert-queue-scroll"
          @scroll="onScroll"
        >
          <div
            v-if="!enabled"
            class="ops-alert-empty-state"
            role="status"
            data-testid="ops-alert-disabled"
          >
            <Icon name="ban" size="lg" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
            <p class="ops-alert-empty-title">{{ t('admin.ops.alertEvents.disabledTitle') }}</p>
            <p class="ops-alert-empty-copy">{{ t('admin.ops.alertEvents.disabledHint') }}</p>
          </div>

          <div
            v-else-if="loading"
            class="flex min-h-56 items-center justify-center gap-2 px-6 py-10 text-sm text-[var(--lx-clay-text-muted)]"
            role="status"
          >
            <span class="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
            {{ t('admin.ops.alertEvents.loading') }}
          </div>

          <div
            v-else-if="empty"
            class="ops-alert-empty-state"
            role="status"
          >
            <Icon name="bellOff" size="lg" class="text-[var(--lx-clay-text-muted)]" aria-hidden="true" />
            <p class="ops-alert-empty-title">{{ t('admin.ops.alertEvents.empty') }}</p>
            <p class="ops-alert-empty-copy">{{ t('admin.ops.alertEvents.emptyHint') }}</p>
          </div>

          <ul v-else class="divide-y divide-[var(--lx-clay-border)]" role="list">
            <li v-for="row in events" :key="row.id">
              <button
                type="button"
                class="ops-alert-row"
                :class="{ 'ops-alert-row--active': selected?.id === row.id }"
                :aria-current="selected?.id === row.id ? 'true' : undefined"
                :aria-labelledby="`ops-alert-title-${row.id}`"
                :aria-describedby="[
                  `ops-alert-status-${row.id}`,
                  row.description ? `ops-alert-description-${row.id}` : '',
                  `ops-alert-context-${row.id}`
                ].filter(Boolean).join(' ')"
                aria-controls="ops-alert-inspector"
                :data-alert-id="row.id"
                data-testid="ops-alert-detail-trigger"
                @click="openDetail(row)"
              >
                <span :id="`ops-alert-status-${row.id}`" class="flex items-center justify-between gap-3">
                  <span class="flex min-w-0 items-center gap-2">
                    <span class="rounded-full px-2 py-1 text-[10px] font-bold" :class="severityBadgeClass(String(row.severity || ''))">
                      {{ row.severity || '-' }}
                    </span>
                    <span class="inline-flex items-center rounded-full px-2 py-1 text-[10px] font-bold ring-1 ring-inset" :class="statusBadgeClass(row.status)">
                      {{ formatStatusLabel(row.status) }}
                    </span>
                  </span>
                  <span class="shrink-0 text-[11px] tabular-nums text-[var(--lx-clay-text-muted)]">
                    {{ formatDateTime(row.fired_at || row.created_at) }}
                  </span>
                </span>
                <span :id="`ops-alert-title-${row.id}`" class="mt-2 block truncate text-sm font-bold text-[var(--lx-clay-text)]">
                  {{ row.title || '-' }}
                </span>
                <span v-if="row.description" :id="`ops-alert-description-${row.id}`" class="mt-1 line-clamp-2 text-xs leading-5 text-[var(--lx-clay-text-secondary)]">
                  {{ row.description }}
                </span>
                <span :id="`ops-alert-context-${row.id}`" class="mt-2 flex items-center justify-between gap-3 text-[11px] text-[var(--lx-clay-text-muted)]">
                  <span class="min-w-0 truncate">{{ formatDimensionsSummary(row) }}</span>
                  <span class="shrink-0 tabular-nums">{{ formatDurationLabel(row) }}</span>
                </span>
              </button>
            </li>
          </ul>

          <div v-if="loadingMore" class="flex items-center justify-center gap-2 py-3 text-xs text-[var(--lx-clay-text-muted)]" role="status">
            <span class="h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
            {{ t('admin.ops.alertEvents.loading') }}
          </div>

          <button
            v-else-if="enabled && hasMore && events.length > 0"
            type="button"
            class="ops-alert-load-more"
            @click="loadMore"
          >
            {{ t('admin.ops.alertEvents.loadMore') }}
          </button>
        </div>
      </aside>

      <main class="ops-alert-canvas">
        <header class="ops-alert-page-heading">
          <div>
            <h2>{{ t('admin.ops.incidentsSectionTitle') }}</h2>
            <p>{{ t('admin.ops.incidentsSectionDescription') }}</p>
          </div>
          <span class="ops-alert-page-signal">
            <i aria-hidden="true"></i>
            {{ t('admin.ops.incidentsSectionStatus') }}
          </span>
        </header>

      <section
        id="ops-alert-inspector"
        class="ops-alert-inspector"
        :aria-label="t('admin.ops.alertEvents.detail.title')"
        data-testid="ops-alert-inspector"
      >
        <header class="ops-alert-action-bar">
          <div class="ops-alert-actions" data-testid="ops-alert-actions">
            <div class="ops-alert-actions__primary">
              <div class="ops-alert-silence-control">
                <span>
                  {{ t('admin.ops.alertEvents.detail.silenceDuration') }}
                </span>
                <Select
                  :model-value="silenceDuration"
                  :options="silenceDurationOptions"
                  :disabled="!canSilence"
                  :aria-label="t('admin.ops.alertEvents.detail.silence')"
                  class="ops-alert-silence-select w-[104px]"
                  @change="silenceDuration = String($event || '1h')"
                />
                <button
                  type="button"
                  class="ops-alert-inline-action"
                  :disabled="!canSilence"
                  data-testid="ops-alert-silence"
                  @click="silenceAlert"
                >
                  {{ t('common.apply') }}
                </button>
              </div>

              <button
                type="button"
                class="ops-alert-action-button"
                :disabled="!canManualResolve"
                data-testid="ops-alert-resolve"
                @click="manualResolve"
              >
                <Icon name="checkCircle" size="sm" aria-hidden="true" />
                {{ t('admin.ops.alertEvents.detail.manualResolve') }}
              </button>
            </div>

            <div class="ops-alert-actions__secondary">
              <a
                v-if="selected && !detailLoading"
                class="ops-alert-action-button"
                :href="`/admin/ops?open_alert_rules=1&alert_rule_id=${selected.rule_id}`"
                data-testid="ops-alert-view-rule"
              >
                <Icon name="externalLink" size="xs" aria-hidden="true" />
                {{ t('admin.ops.alertEvents.detail.viewRule') }}
              </a>
              <button v-else type="button" class="ops-alert-action-button" disabled>
                {{ t('admin.ops.alertEvents.detail.viewRule') }}
              </button>

              <button
                type="button"
                class="ops-alert-action-button"
                :disabled="!canViewRelatedLogs"
                data-testid="ops-alert-view-logs"
                @click="viewRelatedLogs"
              >
                <Icon name="externalLink" size="xs" aria-hidden="true" />
                {{ t('admin.ops.alertEvents.detail.viewLogs') }}
              </button>
            </div>
          </div>
        </header>

        <div
          v-if="detailLoading"
          class="flex min-h-80 items-center justify-center gap-2 px-6 py-12 text-sm text-[var(--lx-clay-text-muted)]"
          role="status"
        >
          <span class="h-4 w-4 animate-spin rounded-full border-2 border-current border-r-transparent" aria-hidden="true"></span>
          {{ t('admin.ops.alertEvents.detail.loading') }}
        </div>

        <div
          v-else-if="!selected"
          class="ops-alert-no-selection"
          data-testid="ops-alert-no-selection"
        >
          <div class="ops-alert-no-selection__icon">
            <Icon name="mousePointerClick" size="lg" aria-hidden="true" />
          </div>
          <p class="ops-alert-no-selection__title">{{ t('admin.ops.alertEvents.detail.emptyTitle') }}</p>
          <p class="ops-alert-no-selection__copy">{{ t('admin.ops.alertEvents.detail.emptyHint') }}</p>
          <p class="ops-alert-no-selection__tip">{{ t('admin.ops.alertEvents.detail.emptyTip') }}</p>
        </div>

        <div v-else class="space-y-6 px-5 py-5 sm:px-6 sm:py-6" data-testid="ops-alert-selected-detail">
          <section>
            <div class="flex flex-wrap items-center gap-2">
              <span class="rounded-full px-2 py-1 text-[10px] font-bold" :class="severityBadgeClass(String(selected.severity || ''))">
                {{ selected.severity || '-' }}
              </span>
              <span class="inline-flex items-center rounded-full px-2 py-1 text-[10px] font-bold ring-1 ring-inset" :class="statusBadgeClass(selected.status)">
                {{ formatStatusLabel(selected.status) }}
              </span>
              <span class="text-xs font-semibold text-[var(--lx-clay-text-muted)]">
                #{{ selected.rule_id }}
              </span>
            </div>
            <h4 class="mt-3 text-lg font-extrabold leading-7 text-[var(--lx-clay-text)]">
              {{ selected.title || '-' }}
            </h4>
            <p v-if="selected.description" class="mt-2 max-w-3xl whitespace-pre-wrap text-sm leading-6 text-[var(--lx-clay-text-secondary)]">
              {{ selected.description }}
            </p>
          </section>

          <dl class="grid grid-cols-1 border-y border-[var(--lx-clay-border)] sm:grid-cols-2 xl:grid-cols-3">
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.detail.firedAt') }}</dt>
              <dd>{{ formatDateTime(selected.fired_at || selected.created_at) }}</dd>
            </div>
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.detail.resolvedAt') }}</dt>
              <dd>{{ selected.resolved_at ? formatDateTime(selected.resolved_at) : '-' }}</dd>
            </div>
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.table.duration') }}</dt>
              <dd>{{ formatDurationLabel(selected) }}</dd>
            </div>
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.table.metric') }}</dt>
              <dd>{{ formatMetric(selected) }}</dd>
            </div>
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.table.email') }}</dt>
              <dd>{{ selected.email_sent ? t('admin.ops.alertEvents.table.emailSent') : t('admin.ops.alertEvents.table.emailIgnored') }}</dd>
            </div>
            <div class="ops-alert-fact">
              <dt>{{ t('admin.ops.alertEvents.detail.dimensions') }}</dt>
              <dd class="break-words">{{ formatDimensionsSummary(selected) }}</dd>
            </div>
          </dl>

          <section class="ops-alert-history--embedded" aria-labelledby="ops-alert-history-heading">
            <div class="flex flex-wrap items-end justify-between gap-3">
              <div>
                <h4 id="ops-alert-history-heading" class="text-sm font-extrabold text-[var(--lx-clay-text)]">
                  {{ t('admin.ops.alertEvents.detail.historyTitle') }}
                </h4>
                <p class="mt-1 text-xs leading-5 text-[var(--lx-clay-text-muted)]">
                  {{ t('admin.ops.alertEvents.detail.historyHint') }}
                </p>
              </div>
              <Select
                :model-value="historyRange"
                :options="historyRangeOptions"
                :disabled="historyLoading"
                :aria-label="t('admin.ops.alertEvents.detail.historyTitle')"
                class="w-[140px]"
                @change="historyRange = String($event || '7d')"
              />
            </div>

            <div class="mt-4 overflow-hidden rounded-xl border border-[var(--lx-clay-border)]">
              <div v-if="historyLoading" class="px-4 py-8 text-center text-xs text-[var(--lx-clay-text-muted)]" role="status">
                {{ t('admin.ops.alertEvents.detail.historyLoading') }}
              </div>
              <div v-else-if="history.length === 0" class="px-4 py-8 text-center text-xs text-[var(--lx-clay-text-muted)]" role="status">
                {{ t('admin.ops.alertEvents.detail.historyEmpty') }}
              </div>
              <div
                v-else
                class="overflow-auto"
                role="region"
                tabindex="0"
                :aria-label="t('admin.ops.alertEvents.detail.historyTitle')"
              >
                <table class="min-w-full divide-y divide-[var(--lx-clay-border)]">
                  <thead class="bg-[var(--lx-clay-recessed)]">
                    <tr>
                      <th class="px-3 py-2 text-left text-[11px] font-bold text-[var(--lx-clay-text-muted)]">{{ t('admin.ops.alertEvents.table.time') }}</th>
                      <th class="px-3 py-2 text-left text-[11px] font-bold text-[var(--lx-clay-text-muted)]">{{ t('admin.ops.alertEvents.table.status') }}</th>
                      <th class="px-3 py-2 text-left text-[11px] font-bold text-[var(--lx-clay-text-muted)]">{{ t('admin.ops.alertEvents.table.metric') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-[var(--lx-clay-border)]">
                    <tr v-for="item in history" :key="item.id">
                      <td class="whitespace-nowrap px-3 py-2 text-xs text-[var(--lx-clay-text-secondary)]">{{ formatDateTime(item.fired_at || item.created_at) }}</td>
                      <td class="px-3 py-2 text-xs">
                        <span class="inline-flex items-center rounded-full px-2 py-1 text-[10px] font-bold ring-1 ring-inset" :class="statusBadgeClass(item.status)">
                          {{ formatStatusLabel(item.status) }}
                        </span>
                      </td>
                      <td class="whitespace-nowrap px-3 py-2 text-xs text-[var(--lx-clay-text-secondary)]">{{ formatMetric(item) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>

      </section>

        <section v-if="$slots.evidence" class="ops-alert-evidence" data-testid="ops-alert-evidence">
          <h3>{{ t('admin.ops.alertEvents.globalContext') }}</h3>
          <slot name="evidence" :selected="selected" :loading="detailLoading" :enabled="enabled" />
        </section>

        <section class="ops-alert-history-card" aria-labelledby="ops-alert-history-heading-visible">
          <header class="ops-alert-history-card__header">
            <div>
              <h3 id="ops-alert-history-heading-visible">{{ t('admin.ops.alertEvents.detail.historySectionTitle') }}</h3>
              <p>{{ t('admin.ops.alertEvents.detail.historySectionHint') }}</p>
            </div>
            <div class="ops-alert-history-ranges" :aria-label="t('admin.ops.alertEvents.detail.historyTitle')">
              <button
                v-for="option in historyRangeOptions"
                :key="String(option.value)"
                type="button"
                :class="{ 'is-active': historyRange === option.value }"
                :disabled="historyLoading"
                @click="historyRange = String(option.value)"
              >
                {{ option.value }}
              </button>
            </div>
          </header>

          <div class="ops-alert-history-table-wrap" role="region" tabindex="0" :aria-label="t('admin.ops.alertEvents.detail.historyTitle')">
            <table>
              <thead>
                <tr>
                  <th>{{ t('admin.ops.alertEvents.table.time') }}</th>
                  <th>{{ t('admin.ops.alertEvents.table.status') }}</th>
                  <th>{{ t('admin.ops.alertEvents.table.metric').split('/')[0]?.trim() }}</th>
                  <th>{{ t('admin.ops.alertEvents.table.threshold') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="historyLoading">
                  <td colspan="4" class="ops-alert-history-empty">{{ t('admin.ops.alertEvents.detail.historyLoading') }}</td>
                </tr>
                <tr v-else-if="history.length === 0">
                  <td colspan="4" class="ops-alert-history-empty">
                    <Icon name="clock" size="lg" aria-hidden="true" />
                    <p>{{ t('admin.ops.alertEvents.detail.historyEmpty') }}</p>
                  </td>
                </tr>
                <template v-else>
                  <tr v-for="item in history" :key="item.id">
                    <td>{{ formatDateTime(item.fired_at || item.created_at) }}</td>
                    <td><span class="ops-alert-status-badge" :class="statusBadgeClass(item.status)">{{ formatStatusLabel(item.status) }}</span></td>
                    <td>{{ typeof item.metric_value === 'number' ? item.metric_value.toFixed(2) : '-' }}</td>
                    <td>{{ typeof item.threshold_value === 'number' ? item.threshold_value.toFixed(2) : '-' }}</td>
                  </tr>
                </template>
              </tbody>
            </table>
          </div>
        </section>
      </main>
    </div>
  </section>
</template>

<style scoped>
.ops-alert-workbench {
  --lx-clay-radius-ops: 14px;
  --lx-clay-recessed: #efebf5;
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
}

.ops-alert-layout {
  display: grid;
  min-width: 0;
  min-height: 640px;
  grid-template-columns: minmax(0, 1fr);
}

.ops-alert-rail {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  border-bottom: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.ops-alert-queue-scroll {
  min-height: 0;
  max-height: 480px;
  overflow: auto;
}

.ops-alert-filter-field {
  display: grid;
  min-width: 0;
  gap: 5px;
}

.ops-alert-filter-field > span {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 750;
}

.ops-alert-load-more {
  display: flex;
  width: calc(100% - 24px);
  min-height: 44px;
  align-items: center;
  justify-content: center;
  margin: 8px 12px 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-accent);
  background: var(--lx-clay-surface);
  font: inherit;
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
}

.ops-alert-load-more:hover {
  background: var(--lx-clay-accent-soft);
}

.ops-alert-load-more:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.ops-alert-row {
  display: block;
  width: 100%;
  min-height: 44px;
  padding: 14px 16px;
  border: 0;
  color: inherit;
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background-color 160ms cubic-bezier(0.16, 1, 0.3, 1);
}

.ops-alert-row:hover {
  background: var(--lx-clay-surface-soft);
}

.ops-alert-row--active {
  background: var(--lx-clay-accent-soft);
  box-shadow: inset 0 0 0 1px var(--lx-clay-border-strong);
}

.ops-alert-row:focus-visible {
  position: relative;
  z-index: 1;
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

.ops-alert-inspector {
  min-width: 0;
  background: var(--lx-clay-surface);
}

.ops-alert-fact {
  min-width: 0;
  padding: 16px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-alert-fact dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 800;
  line-height: 1.4;
}

.ops-alert-fact dd {
  margin-top: 6px;
  color: var(--lx-clay-text);
  font-size: 0.8125rem;
  font-weight: 650;
  line-height: 1.5;
}

@media (min-width: 640px) {
  .ops-alert-fact:nth-child(odd) {
    border-right: 1px solid var(--lx-clay-border);
  }
}

@media (min-width: 1024px) {
  .ops-alert-layout {
    grid-template-columns: 340px minmax(0, 1fr);
  }

  .ops-alert-rail {
    border-right: 1px solid var(--lx-clay-border);
    border-bottom: 0;
  }

  .ops-alert-queue-scroll {
    max-height: none;
  }
}

@media (min-width: 1280px) {
  .ops-alert-fact {
    border-right: 1px solid var(--lx-clay-border);
  }

  .ops-alert-fact:nth-child(3n) {
    border-right: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-alert-row,
  .animate-spin {
    animation: none !important;
    transition: none;
  }
}

@media (pointer: coarse), (max-width: 640px) {
  .ops-alert-inspector .btn,
  .ops-alert-rail .btn {
    min-height: 44px;
  }
}

/* Superdesign Option B — incident investigation flow */
.ops-alert-workbench {
  min-height: calc(100vh - 104px);
  height: calc(100vh - 104px);
  border: 0;
  border-radius: 0;
  background: #fcfcfc;
  font-family: var(--lx-clay-font-ui, "DM Sans", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif);
}

.ops-alert-layout {
  height: 100%;
  min-height: 0;
  grid-template-columns: 340px minmax(0, 1fr);
}

.ops-alert-rail {
  height: 100%;
  border-right: 1px solid var(--lx-clay-border);
  border-bottom: 0;
  background: #fff;
}

.ops-alert-filter-header {
  padding: 16px 16px 6px;
  color: var(--lx-clay-text-muted);
  background: #fcfcfc;
}

.ops-alert-filter-header h3 {
  margin: 0;
  font-size: 11px;
  font-weight: 800;
  line-height: 16px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ops-alert-refresh-button {
  border: 0;
  padding: 0;
  color: var(--lx-clay-accent);
  background: transparent;
  font: inherit;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}

.ops-alert-refresh-button:hover {
  text-decoration: underline;
}

.ops-alert-refresh-button:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ops-alert-filters {
  padding: 4px 16px 16px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fcfcfc;
}

.ops-alert-filter-field {
  position: relative;
  display: block;
  min-width: 0;
}

.ops-alert-filter-field--wide {
  grid-column: span 2 / span 2;
}

.ops-alert-filter-field > span {
  position: absolute;
  z-index: 1;
  top: 50%;
  left: 10px;
  max-width: 55%;
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  font-size: 12px;
  font-weight: 500;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
  pointer-events: none;
  transform: translateY(-50%);
}

.ops-alert-filter-field > span::after {
  content: ":";
}

.ops-alert-filter-field :deep(.select-trigger) {
  height: 32px;
  min-height: 32px;
  padding: 0 9px 0 66px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: #fff;
  box-shadow: none;
  font-size: 12px;
  font-weight: 500;
}

.ops-alert-filter-field--wide :deep(.select-trigger) {
  padding-left: 70px;
}

.ops-alert-filter-field :deep(.select-trigger:hover) {
  border-color: var(--lx-clay-text-muted);
}

.ops-alert-filter-field :deep(.select-trigger-open),
.ops-alert-filter-field :deep(.select-trigger:focus-visible) {
  border-color: var(--lx-clay-accent);
  outline: 2px solid color-mix(in srgb, var(--lx-clay-accent) 16%, transparent);
  outline-offset: 0;
  box-shadow: none;
}

.ops-alert-filter-field :deep(.select-icon) {
  width: 14px;
  color: var(--lx-clay-text-muted);
}

.ops-alert-queue-heading {
  position: sticky;
  z-index: 2;
  top: 0;
  padding: 12px 16px;
  border-bottom: 1px solid rgba(91, 80, 112, 0.06);
  background: #fff;
}

.ops-alert-queue-heading h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
}

.ops-alert-queue-heading p {
  margin: 0;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  line-height: 15px;
}

.ops-alert-queue-scroll {
  flex: 1;
  max-height: none;
  scrollbar-width: thin;
  scrollbar-color: var(--lx-clay-border) transparent;
}

.ops-alert-queue-scroll::-webkit-scrollbar {
  width: 4px;
}

.ops-alert-queue-scroll::-webkit-scrollbar-thumb {
  border-radius: 10px;
  background: var(--lx-clay-border);
}

.ops-alert-empty-state {
  display: flex;
  min-height: 230px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px;
  text-align: center;
  opacity: 0.6;
}

.ops-alert-empty-title {
  margin: 12px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
}

.ops-alert-empty-copy {
  margin: 4px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  line-height: 15px;
}

.ops-alert-row {
  padding: 13px 16px;
  transition: background-color 140ms ease, box-shadow 140ms ease;
}

.ops-alert-row:hover {
  background: #fcfcfc;
}

.ops-alert-row--active {
  background: var(--lx-clay-accent-soft);
  box-shadow: inset 3px 0 0 var(--lx-clay-accent);
}

.ops-alert-badge--danger,
.ops-alert-status--firing {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
  box-shadow: inset 0 0 0 1px rgba(194, 65, 91, 0.12);
}

.ops-alert-badge--warning {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.ops-alert-badge--accent {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-alert-badge--neutral,
.ops-alert-status--neutral,
.ops-alert-status--manual {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  box-shadow: inset 0 0 0 1px var(--lx-clay-border);
}

.ops-alert-status--resolved {
  color: var(--lx-clay-success);
  background: var(--lx-clay-success-soft);
  box-shadow: inset 0 0 0 1px rgba(4, 120, 87, 0.12);
}

.ops-alert-canvas {
  min-width: 0;
  height: 100%;
  overflow-y: auto;
  padding: 32px;
  background: #fcfcfc;
  scrollbar-width: thin;
  scrollbar-color: var(--lx-clay-border) transparent;
}

.ops-alert-canvas > * {
  width: 100%;
  max-width: 1152px;
  margin-right: auto;
  margin-left: auto;
}

.ops-alert-page-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  padding: 0 0 24px;
  border-bottom: 1px solid var(--lx-clay-border);
  margin-bottom: 24px;
}

.ops-alert-page-heading h2 {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display, "Nunito", "PingFang SC", system-ui, sans-serif);
  font-size: 24px;
  font-weight: 900;
  line-height: 32px;
  letter-spacing: -0.025em;
}

.ops-alert-page-heading p {
  margin: 4px 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 14px;
  line-height: 20px;
}

.ops-alert-page-signal {
  display: inline-flex;
  height: 32px;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  border: 1px solid rgba(180, 83, 9, 0.1);
  border-radius: 999px;
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
  font-size: 11px;
  font-weight: 700;
}

.ops-alert-page-signal i {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
}

.ops-alert-inspector {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops, 14px);
  margin-bottom: 24px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.ops-alert-action-bar {
  padding: 16px 24px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fcfcfc;
}

.ops-alert-actions,
.ops-alert-actions__primary,
.ops-alert-actions__secondary,
.ops-alert-silence-control {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.ops-alert-actions {
  justify-content: space-between;
}

.ops-alert-silence-control {
  min-height: 32px;
  padding: 4px 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  background: #fff;
}

.ops-alert-silence-control > span {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  font-weight: 700;
}

.ops-alert-silence-control:has(.select-trigger-disabled) {
  opacity: 0.5;
}

.ops-alert-silence-select :deep(.select-trigger) {
  height: 24px;
  min-height: 24px;
  padding: 0 6px;
  border: 0;
  border-radius: 4px;
  color: var(--lx-clay-text-muted);
  background: #f9fafb;
  box-shadow: none;
  font-size: 10px;
}

.ops-alert-inline-action,
.ops-alert-action-button {
  display: inline-flex;
  height: 32px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  color: var(--lx-clay-text-secondary);
  background: #fff;
  font: inherit;
  font-size: 10px;
  font-weight: 700;
  text-decoration: none;
  cursor: pointer;
}

.ops-alert-inline-action {
  height: auto;
  padding: 0;
  border: 0;
  color: var(--lx-clay-text-muted);
  background: transparent;
}

.ops-alert-inline-action:hover:not(:disabled),
.ops-alert-action-button:hover:not(:disabled) {
  color: var(--lx-clay-accent);
  border-color: color-mix(in srgb, var(--lx-clay-accent) 35%, var(--lx-clay-border));
}

.ops-alert-inline-action:disabled,
.ops-alert-action-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ops-alert-no-selection {
  display: flex;
  min-height: 360px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 48px;
  text-align: center;
}

.ops-alert-no-selection__icon {
  display: flex;
  width: 56px;
  height: 56px;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  margin-bottom: 20px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-recessed);
}

.ops-alert-no-selection__icon :deep(svg) {
  width: 28px;
  height: 28px;
}

.ops-alert-no-selection__title {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 16px;
  font-weight: 800;
  line-height: 24px;
}

.ops-alert-no-selection__copy {
  max-width: 384px;
  margin: 8px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  line-height: 19px;
}

.ops-alert-no-selection__tip {
  max-width: 448px;
  margin: 24px 0 0;
  padding: 8px 12px;
  border-radius: 8px;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-recessed);
  font-size: 10px;
  font-weight: 500;
  line-height: 15px;
}

[data-testid="ops-alert-selected-detail"] {
  padding: 24px;
}

.ops-alert-fact {
  padding: 16px;
}

.ops-alert-history--embedded {
  display: none;
}

.ops-alert-evidence {
  margin-bottom: 24px;
}

.ops-alert-evidence > h3 {
  margin: 0 0 16px 8px;
  color: var(--lx-clay-text-muted);
  font-size: 12px;
  font-weight: 900;
  line-height: 18px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.ops-alert-evidence :deep(.ops-section-heading--evidence) {
  display: none !important;
}

.ops-alert-evidence :deep(.ops-quality-grid) {
  display: grid !important;
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr) !important;
  grid-template-areas:
    "trend distribution"
    "trend switch" !important;
  gap: 24px !important;
  align-items: stretch !important;
}

.ops-alert-evidence :deep(.ops-panel) {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border) !important;
  border-radius: var(--lx-clay-radius-ops, 14px) !important;
  background: #fff !important;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04) !important;
}

.ops-alert-evidence :deep(.ops-panel--error-trend) {
  grid-area: trend;
}

.ops-alert-evidence :deep(.ops-panel--switch-rate) {
  grid-area: switch;
}

.ops-alert-evidence :deep(.ops-panel--error-distribution) {
  grid-area: distribution;
}

.ops-alert-history-card {
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops, 14px);
  margin-bottom: 8px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.ops-alert-history-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 24px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fcfcfc;
}

.ops-alert-history-card__header h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
}

.ops-alert-history-card__header p {
  margin: 2px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  line-height: 16px;
}

.ops-alert-history-ranges {
  display: flex;
  padding: 2px;
  border-radius: 8px;
  background: var(--lx-clay-recessed);
}

.ops-alert-history-ranges button {
  padding: 4px 12px;
  border: 0;
  border-radius: 6px;
  color: var(--lx-clay-text-muted);
  background: transparent;
  font: inherit;
  font-size: 10px;
  font-weight: 700;
  cursor: pointer;
}

.ops-alert-history-ranges button.is-active {
  color: var(--lx-clay-text);
  background: #fff;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.ops-alert-history-table-wrap {
  overflow-x: auto;
}

.ops-alert-history-table-wrap table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.ops-alert-history-table-wrap thead {
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fcfcfc;
}

.ops-alert-history-table-wrap th {
  padding: 12px 24px;
  color: var(--lx-clay-text-muted);
  font-size: 11px;
  font-weight: 900;
  line-height: 16px;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.ops-alert-history-table-wrap td {
  padding: 10px 24px;
  border-top: 1px solid rgba(91, 80, 112, 0.08);
  color: var(--lx-clay-text-secondary);
  font-size: 12px;
  line-height: 18px;
  font-variant-numeric: tabular-nums;
}

.ops-alert-history-table-wrap td.ops-alert-history-empty {
  height: 112px;
  padding: 32px 24px;
  color: var(--lx-clay-text-muted);
  text-align: center;
  opacity: 0.5;
}

.ops-alert-history-empty svg {
  margin: 0 auto 8px;
}

.ops-alert-history-empty p {
  margin: 0;
}

.ops-alert-status-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
}

@media (max-width: 1180px) {
  .ops-alert-evidence :deep(.ops-quality-grid) {
    grid-template-columns: minmax(0, 1fr) !important;
    grid-template-areas: "trend" "distribution" "switch" !important;
  }
}

@media (max-width: 1023px) {
  .ops-alert-workbench {
    height: auto;
    min-height: 0;
  }

  .ops-alert-layout {
    height: auto;
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-alert-rail {
    height: auto;
    max-height: 620px;
    border-right: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-alert-canvas {
    height: auto;
    overflow: visible;
  }
}

@media (max-width: 640px) {
  .ops-alert-canvas {
    padding: 20px 16px;
  }

  .ops-alert-page-heading,
  .ops-alert-actions,
  .ops-alert-history-card__header {
    align-items: flex-start;
    flex-direction: column;
  }

  .ops-alert-actions__primary,
  .ops-alert-actions__secondary {
    width: 100%;
  }

  .ops-alert-action-button {
    min-height: 40px;
  }

  .ops-alert-history-table-wrap th,
  .ops-alert-history-table-wrap td {
    padding-right: 14px;
    padding-left: 14px;
  }
}
</style>
