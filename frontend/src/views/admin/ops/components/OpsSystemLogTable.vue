<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  opsAPI,
  type OpsRuntimeLogConfig,
  type OpsSystemLog,
  type OpsSystemLogCleanupRequest,
  type OpsSystemLogQuery,
  type OpsSystemLogSinkHealth
} from '@/api/admin/ops'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'

type LogTimeRange = '5m' | '30m' | '1h' | '6h' | '24h' | '7d' | '30d'

interface InvestigationPreset {
  key?: string | number
  alertId?: string | number
  platform?: string | null
  startTime?: string | null
  endTime?: string | null
  requestId?: string | null
  groupId?: string | number | null
  region?: string | null
}

interface LogFilters {
  time_range: LogTimeRange
  start_time: string
  end_time: string
  host: string
  level: string
  component: string
  request_id: string
  client_request_id: string
  user_id: string
  api_key_id: string
  account_id: string
  platform: string
  model: string
  q: string
}

const appStore = useAppStore()
const { t } = useI18n()

const props = withDefaults(defineProps<{
  platformFilter?: string
  refreshToken?: number
  investigationPreset?: InvestigationPreset | null
}>(), {
  platformFilter: '',
  refreshToken: 0,
  investigationPreset: null
})

const emit = defineEmits<{
  (event: 'clear-investigation'): void
  (event: 'open-request-details', row: OpsSystemLog): void
}>()

const loading = ref(false)
const logs = ref<OpsSystemLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const selectedLog = ref<OpsSystemLog | null>(null)
const advancedFiltersOpen = ref(false)
const healthDetailsOpen = ref(false)
const logManagementOpen = ref(false)

const health = ref<OpsSystemLogSinkHealth | null>(null)
const healthLoaded = ref(false)
const healthLoading = ref(false)

const runtimeLoading = ref(false)
const runtimeSaving = ref(false)
const runtimeConfig = reactive<OpsRuntimeLogConfig>({
  level: 'info',
  enable_sampling: false,
  sampling_initial: 100,
  sampling_thereafter: 100,
  caller: true,
  stacktrace_level: 'error',
  retention_days: 30
})

const filters = reactive<LogFilters>({
  time_range: '1h',
  start_time: '',
  end_time: '',
  host: '',
  level: '',
  component: '',
  request_id: '',
  client_request_id: '',
  user_id: '',
  api_key_id: '',
  account_id: '',
  platform: '',
  model: '',
  q: ''
})

const runtimeLevelOptions = [
  { value: 'debug', label: 'debug' },
  { value: 'info', label: 'info' },
  { value: 'warn', label: 'warn' },
  { value: 'error', label: 'error' }
]

const stacktraceLevelOptions = [
  { value: 'none', label: 'none' },
  { value: 'error', label: 'error' },
  { value: 'fatal', label: 'fatal' }
]

const timeRangeOptions: Array<{ value: LogTimeRange; label: string }> = [
  { value: '5m', label: '5m' },
  { value: '30m', label: '30m' },
  { value: '1h', label: '1h' },
  { value: '6h', label: '6h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' }
]

const timeRangeMs: Record<LogTimeRange, number> = {
  '5m': 5 * 60 * 1000,
  '30m': 30 * 60 * 1000,
  '1h': 60 * 60 * 1000,
  '6h': 6 * 60 * 60 * 1000,
  '24h': 24 * 60 * 60 * 1000,
  '7d': 7 * 24 * 60 * 60 * 1000,
  '30d': 30 * 24 * 60 * 60 * 1000
}

const filterLevelOptions = computed(() => [
  { value: '', label: t('admin.ops.systemLogs.all') },
  { value: 'debug', label: 'debug' },
  { value: 'info', label: 'info' },
  { value: 'warn', label: 'warn' },
  { value: 'error', label: 'error' }
])

const runtimeConfigTitle = computed(() => t('admin.ops.systemLogs.runtimeConfig'))

const investigationDismissed = ref(false)
const investigationVisible = computed(() => Boolean(props.investigationPreset) && !investigationDismissed.value)

const normalizeString = (value: unknown): string => {
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

const sinkHealthTone = computed<'loading' | 'unknown' | 'healthy' | 'warning' | 'danger'>(() => {
  if (healthLoading.value && !healthLoaded.value) return 'loading'
  if (!healthLoaded.value || !health.value) return 'unknown'
  if (Number(health.value.write_failed_count || 0) > 0 || normalizeString(health.value.last_error)) return 'danger'
  if (Number(health.value.dropped_count || 0) > 0) return 'warning'
  return 'healthy'
})

const sinkHealthLabel = computed(() => {
  if (sinkHealthTone.value === 'loading') return t('common.loading')
  if (sinkHealthTone.value === 'healthy') return t('admin.ops.systemLogs.healthHealthy')
  if (sinkHealthTone.value === 'warning') return t('admin.ops.systemLogs.healthWarning')
  if (sinkHealthTone.value === 'danger') return t('admin.ops.systemLogs.healthUnhealthy')
  return t('admin.ops.systemLogs.healthUnavailable')
})

const sinkHealthIcon = computed(() => {
  if (sinkHealthTone.value === 'healthy') return 'checkCircle' as const
  if (sinkHealthTone.value === 'warning') return 'exclamationTriangle' as const
  if (sinkHealthTone.value === 'danger') return 'xCircle' as const
  if (sinkHealthTone.value === 'loading') return 'refresh' as const
  return 'infoCircle' as const
})

const formatTime = (value: string | null | undefined) => {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

const toDateTimeLocal = (value: string | null | undefined): string => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60 * 1000)
  return localDate.toISOString().slice(0, 16)
}

const toRFC3339 = (value: string): string | undefined => {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

const toPositiveInteger = (value: string): number | undefined => {
  const trimmed = value.trim()
  if (!/^\d+$/.test(trimmed)) return undefined
  const parsed = Number(trimmed)
  if (!Number.isSafeInteger(parsed) || parsed <= 0) return undefined
  return parsed
}

const formatExtraValue = (value: unknown, pretty = false): string => {
  if (value == null) return '-'
  if (typeof value === 'string') return value.trim() || '-'
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  try {
    return JSON.stringify(value, null, pretty ? 2 : 0)
  } catch {
    return String(value)
  }
}

const getExtraValue = (row: OpsSystemLog | null, key: string): string => {
  if (!row?.extra || !(key in row.extra)) return '-'
  return formatExtraValue(row.extra[key])
}

const getCompactExtraValue = (row: OpsSystemLog, key: string): string => {
  const value = getExtraValue(row, key)
  return value === '-' ? '' : value
}

const getRequestRoute = (row: OpsSystemLog): string => {
  const method = getCompactExtraValue(row, 'method')
  const path = getCompactExtraValue(row, 'path')
  return [method, path].filter(Boolean).join(' ')
}

const getStatusCode = (row: OpsSystemLog): string => getCompactExtraValue(row, 'status_code')

const getLatency = (row: OpsSystemLog): string => {
  const latency = getCompactExtraValue(row, 'latency_ms')
  return latency ? `${latency}ms` : ''
}

const isErrorStatus = (row: OpsSystemLog): boolean => {
  const status = Number(getStatusCode(row))
  return Number.isFinite(status) && status >= 400
}

const selectedExtra = computed(() => {
  const extra = selectedLog.value?.extra
  if (!extra || Object.keys(extra).length === 0) return '-'
  return formatExtraValue(extra, true)
})

const selectedErrors = computed(() => {
  const extra = selectedLog.value?.extra
  if (!extra) return '-'
  const values = ['errors', 'err', 'error']
    .filter((key) => key in extra)
    .map((key) => `${key}: ${formatExtraValue(extra[key], true)}`)
  return values.length > 0 ? values.join('\n') : '-'
})

const levelBadgeClass = (level: string) => {
  const value = String(level || '').toLowerCase()
  if (value === 'error' || value === 'fatal') return 'ops-log-level--error'
  if (value === 'warn' || value === 'warning') return 'ops-log-level--warn'
  if (value === 'debug') return 'ops-log-level--debug'
  return 'ops-log-level--info'
}

const activeFilterChips = computed(() => {
  const chips: Array<{ key: string; label: string; value: string }> = []
  if (filters.start_time || filters.end_time) {
    const start = filters.start_time ? formatTime(toRFC3339(filters.start_time)) : '-'
    const end = filters.end_time ? formatTime(toRFC3339(filters.end_time)) : '-'
    chips.push({ key: 'time', label: t('admin.ops.systemLogs.timeRange'), value: `${start} — ${end}` })
  } else if (filters.time_range !== '1h') {
    chips.push({ key: 'time', label: t('admin.ops.systemLogs.timeRange'), value: filters.time_range })
  }

  const textFilters: Array<[keyof LogFilters, string]> = [
    ['level', t('admin.ops.systemLogs.level')],
    ['component', t('admin.ops.systemLogs.component')],
    ['host', t('admin.ops.systemLogs.host')],
    ['platform', t('admin.ops.systemLogs.platform')],
    ['model', t('admin.ops.systemLogs.model')],
    ['request_id', 'request_id'],
    ['client_request_id', 'client_request_id'],
    ['user_id', 'user_id'],
    ['api_key_id', t('admin.ops.systemLogs.keyId')],
    ['account_id', 'account_id'],
    ['q', t('admin.ops.systemLogs.keyword')]
  ]

  textFilters.forEach(([key, label]) => {
    const value = String(filters[key] || '').trim()
    if (value) chips.push({ key, label, value })
  })
  return chips
})

const advancedFilterCount = computed(() => [
  filters.start_time,
  filters.end_time,
  filters.host,
  filters.platform,
  filters.model,
  filters.request_id,
  filters.client_request_id,
  filters.user_id,
  filters.api_key_id,
  filters.account_id
].filter((value) => String(value || '').trim()).length)

const hasData = computed(() => logs.value.length > 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))

const buildQuery = (): OpsSystemLogQuery => {
  const query: OpsSystemLogQuery = {
    page: page.value,
    page_size: pageSize.value,
    time_range: filters.time_range
  }

  const startTime = toRFC3339(filters.start_time)
  const endTime = toRFC3339(filters.end_time)
  if (startTime) query.start_time = startTime
  if (endTime) query.end_time = endTime
  if (filters.host.trim()) query.host = filters.host.trim()
  if (filters.level.trim()) query.level = filters.level.trim()
  if (filters.component.trim()) query.component = filters.component.trim()
  if (filters.request_id.trim()) query.request_id = filters.request_id.trim()
  if (filters.client_request_id.trim()) query.client_request_id = filters.client_request_id.trim()

  const userID = toPositiveInteger(filters.user_id)
  const apiKeyID = toPositiveInteger(filters.api_key_id)
  const accountID = toPositiveInteger(filters.account_id)
  if (userID) query.user_id = userID
  if (apiKeyID) query.api_key_id = apiKeyID
  if (accountID) query.account_id = accountID
  if (filters.platform.trim()) query.platform = filters.platform.trim()
  if (filters.model.trim()) query.model = filters.model.trim()
  if (filters.q.trim()) query.q = filters.q.trim()
  return query
}

const materializeCleanupWindow = (now = new Date()): { startTime: string; endTime: string } => {
  const explicitStart = toRFC3339(filters.start_time)
  const explicitEnd = toRFC3339(filters.end_time)

  if (explicitStart || explicitEnd) {
    const end = explicitEnd ? new Date(explicitEnd) : now
    const start = explicitStart ? new Date(explicitStart) : new Date(end.getTime() - timeRangeMs['1h'])
    return { startTime: start.toISOString(), endTime: end.toISOString() }
  }

  const end = now
  const start = new Date(end.getTime() - timeRangeMs[filters.time_range])
  return { startTime: start.toISOString(), endTime: end.toISOString() }
}

const buildCleanupSnapshot = (now = new Date()): OpsSystemLogCleanupRequest => {
  const window = materializeCleanupWindow(now)
  const payload: OpsSystemLogCleanupRequest = {
    start_time: window.startTime,
    end_time: window.endTime
  }

  if (filters.host.trim()) payload.host = filters.host.trim()
  if (filters.level.trim()) payload.level = filters.level.trim()
  if (filters.component.trim()) payload.component = filters.component.trim()
  if (filters.request_id.trim()) payload.request_id = filters.request_id.trim()
  if (filters.client_request_id.trim()) payload.client_request_id = filters.client_request_id.trim()

  const userID = toPositiveInteger(filters.user_id)
  const apiKeyID = toPositiveInteger(filters.api_key_id)
  const accountID = toPositiveInteger(filters.account_id)
  if (userID) payload.user_id = userID
  if (apiKeyID) payload.api_key_id = apiKeyID
  if (accountID) payload.account_id = accountID
  if (filters.platform.trim()) payload.platform = filters.platform.trim()
  if (filters.model.trim()) payload.model = filters.model.trim()
  if (filters.q.trim()) payload.q = filters.q.trim()
  return payload
}

const buildCleanupConfirmation = (snapshot: OpsSystemLogCleanupRequest): string => {
  const conditions = Object.entries(snapshot)
    .filter(([, value]) => value != null && String(value).trim() !== '')
    .map(([key, value]) => `${key}=${String(value)}`)
    .join('\n')
  return `${t('admin.ops.systemLogs.cleanupConfirm')}\n\n${conditions}`
}

let logsFetchSequence = 0

const fetchLogs = async () => {
  const sequence = ++logsFetchSequence
  loading.value = true
  try {
    const response = await opsAPI.listSystemLogs(buildQuery())
    if (sequence !== logsFetchSequence) return
    logs.value = response.items || []
    total.value = response.total || 0

    if (selectedLog.value) {
      selectedLog.value = logs.value.find((item) => item.id === selectedLog.value?.id) || null
    }
  } catch (err: any) {
    if (sequence !== logsFetchSequence) return
    console.error('[OpsSystemLogTable] Failed to fetch logs', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.systemLogs.loadFailed'))
    logs.value = []
    total.value = 0
    selectedLog.value = null
  } finally {
    if (sequence === logsFetchSequence) loading.value = false
  }
}

const fetchHealth = async () => {
  healthLoading.value = true
  try {
    health.value = await opsAPI.getSystemLogSinkHealth()
    healthLoaded.value = true
  } catch {
    health.value = null
    healthLoaded.value = false
  } finally {
    healthLoading.value = false
  }
}

const assignRuntimeConfig = (config: OpsRuntimeLogConfig) => {
  runtimeConfig.level = config.level
  runtimeConfig.enable_sampling = config.enable_sampling
  runtimeConfig.sampling_initial = config.sampling_initial
  runtimeConfig.sampling_thereafter = config.sampling_thereafter
  runtimeConfig.caller = config.caller
  runtimeConfig.stacktrace_level = config.stacktrace_level
  runtimeConfig.retention_days = config.retention_days
}

const loadRuntimeConfig = async () => {
  runtimeLoading.value = true
  try {
    assignRuntimeConfig(await opsAPI.getRuntimeLogConfig())
  } catch (err: any) {
    console.error('[OpsSystemLogTable] Failed to load runtime log config', err)
  } finally {
    runtimeLoading.value = false
  }
}

const saveRuntimeConfig = async () => {
  runtimeSaving.value = true
  try {
    assignRuntimeConfig(await opsAPI.updateRuntimeLogConfig({ ...runtimeConfig }))
    appStore.showSuccess(t('admin.ops.systemLogs.runtimeConfigActive'))
  } catch (err: any) {
    console.error('[OpsSystemLogTable] Failed to save runtime log config', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.systemLogs.runtimeConfigSaveFailed'))
  } finally {
    runtimeSaving.value = false
  }
}

const resetRuntimeConfig = async () => {
  const confirmed = window.confirm(t('admin.ops.systemLogs.resetRuntimeConfigConfirm'))
  if (!confirmed) return

  runtimeSaving.value = true
  try {
    assignRuntimeConfig(await opsAPI.resetRuntimeLogConfig())
    appStore.showSuccess(t('admin.ops.systemLogs.runtimeConfigReset'))
    await fetchHealth()
  } catch (err: any) {
    console.error('[OpsSystemLogTable] Failed to reset runtime log config', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.systemLogs.runtimeConfigResetFailed'))
  } finally {
    runtimeSaving.value = false
  }
}

const cleanupCurrentFilter = async () => {
  const snapshot = buildCleanupSnapshot()
  const confirmed = window.confirm(buildCleanupConfirmation(snapshot))
  if (!confirmed) return

  try {
    const response = await opsAPI.cleanupSystemLogs(snapshot)
    appStore.showSuccess(t('admin.ops.systemLogs.cleanupSuccess', { count: response.deleted || 0 }))
    page.value = 1
    selectedLog.value = null
    await Promise.all([fetchLogs(), fetchHealth()])
  } catch (err: any) {
    console.error('[OpsSystemLogTable] Failed to cleanup logs', err)
    appStore.showError(err?.response?.data?.detail || t('admin.ops.systemLogs.cleanupFailed'))
  }
}

const resetNonContextFilters = () => {
  filters.time_range = '1h'
  filters.start_time = ''
  filters.end_time = ''
  filters.host = ''
  filters.level = ''
  filters.component = ''
  filters.request_id = ''
  filters.client_request_id = ''
  filters.user_id = ''
  filters.api_key_id = ''
  filters.account_id = ''
  filters.platform = ''
  filters.model = ''
  filters.q = ''
}

const applyInvestigationPreset = (preset: InvestigationPreset | null | undefined, refresh: boolean) => {
  if (preset) resetNonContextFilters()
  filters.platform = preset ? normalizeString(preset.platform) : normalizeString(props.platformFilter)
  filters.start_time = preset ? toDateTimeLocal(preset.startTime) : ''
  filters.end_time = preset ? toDateTimeLocal(preset.endTime) : ''
  filters.request_id = preset ? normalizeString(preset.requestId) : ''
  page.value = 1
  selectedLog.value = null
  if (refresh) void fetchLogs()
}

const resetFilters = () => {
  resetNonContextFilters()
  applyInvestigationPreset(investigationVisible.value ? props.investigationPreset : null, false)
  void fetchLogs()
}

const removeFilterChip = (key: string) => {
  if (key === 'time') {
    filters.time_range = '1h'
    filters.start_time = ''
    filters.end_time = ''
  } else if (key in filters) {
    Object.assign(filters, { [key]: '' })
  }
  applyFilters()
}

const clearInvestigation = () => {
  investigationDismissed.value = true
  emit('clear-investigation')
  applyInvestigationPreset(null, true)
}

const selectTimeRange = (value: LogTimeRange) => {
  filters.time_range = value
  filters.start_time = ''
  filters.end_time = ''
}

const onTimeRangeChange = (event: Event) => {
  selectTimeRange((event.target as HTMLSelectElement).value as LogTimeRange)
}

const toggleAdvancedFilters = () => {
  advancedFiltersOpen.value = !advancedFiltersOpen.value
  healthDetailsOpen.value = false
  logManagementOpen.value = false
}

const toggleHealthDetails = () => {
  healthDetailsOpen.value = !healthDetailsOpen.value
  if (healthDetailsOpen.value) {
    logManagementOpen.value = false
    advancedFiltersOpen.value = false
  }
}

const toggleLogManagement = () => {
  logManagementOpen.value = !logManagementOpen.value
  if (logManagementOpen.value) {
    healthDetailsOpen.value = false
    advancedFiltersOpen.value = false
  }
}

const selectLog = (row: OpsSystemLog) => {
  selectedLog.value = row
}

const closeInspector = () => {
  selectedLog.value = null
}

const onPageChange = (next: number) => {
  page.value = next
  selectedLog.value = null
  void fetchLogs()
}

const onPageSizeChange = (next: number) => {
  pageSize.value = next
  page.value = 1
  selectedLog.value = null
  void fetchLogs()
}

const applyFilters = () => {
  page.value = 1
  selectedLog.value = null
  void fetchLogs()
}

let mounted = false

watch(() => props.platformFilter, (value) => {
  if (investigationVisible.value) return
  filters.platform = normalizeString(value)
  page.value = 1
  selectedLog.value = null
  if (mounted) void fetchLogs()
})

watch(() => props.investigationPreset, (preset) => {
  investigationDismissed.value = false
  applyInvestigationPreset(preset, mounted)
}, { deep: true })

watch(() => props.refreshToken, () => {
  if (!mounted) return
  void Promise.all([fetchLogs(), fetchHealth()])
})

onMounted(async () => {
  applyInvestigationPreset(props.investigationPreset, false)
  mounted = true
  await Promise.all([fetchLogs(), fetchHealth(), loadRuntimeConfig()])
})
</script>

<template>
  <section
    class="ops-log-workbench"
    :class="{ 'ops-log-workbench--with-context': investigationVisible }"
    data-testid="ops-system-log-workbench"
  >
    <div
      v-if="investigationVisible"
      class="ops-log-context-bar"
      data-testid="ops-log-investigation-context"
    >
      <div class="ops-log-context-bar__content">
        <Icon name="link" size="sm" aria-hidden="true" />
        <strong>{{ t('admin.ops.systemLogs.investigationFromAlert') }}</strong>
        <span v-if="props.investigationPreset?.alertId != null" class="ops-log-context-id">
          #{{ props.investigationPreset.alertId }}
        </span>
        <span v-if="filters.platform" class="ops-log-context-chip">platform={{ filters.platform }}</span>
        <span v-if="filters.request_id" class="ops-log-context-chip">request_id={{ filters.request_id }}</span>
        <span v-if="props.investigationPreset?.groupId != null" class="ops-log-context-chip">
          group_id={{ props.investigationPreset.groupId }}
        </span>
        <span v-if="props.investigationPreset?.region" class="ops-log-context-chip">
          region={{ props.investigationPreset.region }}
        </span>
        <span v-if="filters.start_time || filters.end_time" class="ops-log-context-window">
          {{ formatTime(toRFC3339(filters.start_time)) }} — {{ formatTime(toRFC3339(filters.end_time)) }}
        </span>
      </div>
      <button
        type="button"
        class="ops-log-icon-button"
        data-testid="ops-log-clear-investigation"
        :aria-label="t('admin.ops.systemLogs.clearAlertContext')"
        @click="clearInvestigation"
      >
        <Icon name="x" size="sm" aria-hidden="true" />
      </button>
    </div>

    <section class="ops-log-query-shell" data-testid="ops-log-filter-rail">
      <div class="ops-log-query-toolbar" data-testid="ops-log-query-toolbar">
        <label class="ops-log-toolbar-control ops-log-toolbar-time">
          <span class="sr-only">{{ t('admin.ops.systemLogs.timeRange') }}</span>
          <Icon name="clock" size="xs" aria-hidden="true" />
          <select
            :value="filters.time_range"
            data-testid="system-log-time-range"
            :aria-label="t('admin.ops.systemLogs.timeRange')"
            @change="onTimeRangeChange"
          >
            <option v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </label>

        <label class="ops-log-toolbar-search">
          <span class="sr-only">{{ t('admin.ops.systemLogs.keyword') }}</span>
          <Icon name="search" size="sm" aria-hidden="true" />
          <input
            v-model="filters.q"
            data-testid="system-log-keyword"
            type="search"
            :placeholder="t('admin.ops.systemLogs.keywordPlaceholder')"
            @keydown.enter.prevent="applyFilters"
          />
        </label>

        <label class="ops-log-toolbar-select">
          <span class="sr-only">{{ t('admin.ops.systemLogs.level') }}</span>
          <Select
            v-model="filters.level"
            :options="filterLevelOptions"
            :placeholder="t('admin.ops.systemLogs.level')"
          />
        </label>

        <label class="ops-log-toolbar-control ops-log-toolbar-component">
          <span class="sr-only">{{ t('admin.ops.systemLogs.component') }}</span>
          <input
            v-model="filters.component"
            data-testid="system-log-component"
            type="text"
            :placeholder="t('admin.ops.systemLogs.component')"
            @keydown.enter.prevent="applyFilters"
          />
        </label>

        <button
          type="button"
          class="ops-log-search-button"
          data-testid="system-log-search"
          :disabled="loading"
          @click="applyFilters"
        >
          <Icon name="search" size="xs" aria-hidden="true" />
          {{ t('admin.ops.systemLogs.search') }}
        </button>

        <button
          type="button"
          class="ops-log-icon-button ops-log-reset-button"
          data-testid="system-log-reset"
          :disabled="loading"
          :aria-label="t('common.reset')"
          :title="t('common.reset')"
          @click="resetFilters"
        >
          <Icon name="refresh" size="sm" aria-hidden="true" />
        </button>

        <button
          type="button"
          class="ops-log-secondary-button"
          data-testid="ops-log-advanced-toggle"
          :aria-expanded="advancedFiltersOpen"
          aria-controls="ops-log-advanced-filters"
          @click="toggleAdvancedFilters"
        >
          <Icon name="slidersHorizontal" size="xs" aria-hidden="true" />
          {{ t('admin.ops.systemLogs.advancedFilters') }}
          <span v-if="advancedFilterCount" class="ops-log-count-badge">{{ advancedFilterCount }}</span>
          <Icon
            name="chevronDown"
            size="xs"
            class="ops-log-chevron"
            :class="{ 'ops-log-chevron--open': advancedFiltersOpen }"
            aria-hidden="true"
          />
        </button>

        <div
          class="ops-log-health-strip"
          data-testid="ops-log-health-strip"
        >
          <button
            type="button"
            class="ops-log-health-status"
            :class="'ops-log-health-status--' + sinkHealthTone"
            data-testid="ops-log-health-status"
            :data-health-tone="sinkHealthTone"
            :aria-expanded="healthDetailsOpen"
            aria-controls="ops-log-health-details"
            @click="toggleHealthDetails"
          >
            <Icon
              :name="sinkHealthIcon"
              size="xs"
              :class="{ 'ops-log-spin': sinkHealthTone === 'loading' }"
              aria-hidden="true"
            />
            <span>{{ t('admin.ops.systemLogs.sinkHealth') }}</span>
            <b>{{ sinkHealthLabel }}</b>
          </button>

          <Transition name="ops-log-popover">
            <section
              v-if="healthDetailsOpen"
              id="ops-log-health-details"
              class="ops-log-health-popover"
              data-testid="ops-log-health-details"
            >
              <header>
                <div>
                  <strong>{{ t('admin.ops.systemLogs.sinkHealth') }}</strong>
                  <span>{{ sinkHealthLabel }}</span>
                </div>
                <button
                  type="button"
                  class="ops-log-icon-button"
                  :disabled="healthLoading"
                  :aria-label="t('admin.ops.systemLogs.refreshHealth')"
                  @click="fetchHealth"
                >
                  <Icon name="refresh" size="sm" aria-hidden="true" />
                </button>
              </header>
              <dl>
                <div>
                  <dt>{{ t('admin.ops.systemLogs.queue') }}</dt>
                  <dd>{{ healthLoaded ? health?.queue_depth ?? 0 : '—' }} / {{ healthLoaded ? health?.queue_capacity ?? 0 : '—' }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.systemLogs.written') }}</dt>
                  <dd>{{ healthLoaded ? health?.written_count : '—' }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.systemLogs.dropped') }}</dt>
                  <dd :class="{ 'ops-log-warning': healthLoaded && Boolean(health?.dropped_count) }">
                    {{ healthLoaded ? health?.dropped_count : '—' }}
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.systemLogs.failed') }}</dt>
                  <dd :class="{ 'ops-log-danger': healthLoaded && Boolean(health?.write_failed_count) }">
                    {{ healthLoaded ? health?.write_failed_count : '—' }}
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.ops.systemLogs.avgWriteDelay') }}</dt>
                  <dd>{{ healthLoaded ? health?.avg_write_delay_ms : '—' }}<span v-if="healthLoaded"> ms</span></dd>
                </div>
              </dl>
              <p v-if="healthLoaded && health?.last_error">
                <strong>{{ t('admin.ops.systemLogs.latestWriteError') }}</strong>
                {{ health.last_error }}
              </p>
            </section>
          </Transition>
        </div>

        <button
          type="button"
          class="ops-log-icon-button ops-log-management-toggle"
          data-testid="ops-log-management-toggle"
          :aria-label="t('admin.ops.systemLogs.logManagement')"
          :title="t('admin.ops.systemLogs.logManagement')"
          :aria-expanded="logManagementOpen"
          aria-controls="ops-log-management"
          @click="toggleLogManagement"
        >
          <Icon name="cog" size="sm" aria-hidden="true" />
        </button>
      </div>

      <Transition name="ops-log-disclosure">
        <section
          v-if="advancedFiltersOpen"
          id="ops-log-advanced-filters"
          class="ops-log-advanced-filters"
          data-testid="ops-log-advanced-filters"
        >
          <div class="ops-log-advanced-grid">
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.startTime') }}
              <input
                v-model="filters.start_time"
                data-testid="system-log-start-time"
                type="datetime-local"
                class="ops-log-input"
              />
            </label>
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.endTime') }}
              <input
                v-model="filters.end_time"
                data-testid="system-log-end-time"
                type="datetime-local"
                class="ops-log-input"
              />
            </label>
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.host') }}
              <input v-model="filters.host" data-testid="system-log-host" type="text" class="ops-log-input" />
            </label>
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.platform') }}
              <input v-model="filters.platform" data-testid="system-log-platform" type="text" class="ops-log-input" />
            </label>
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.model') }}
              <input v-model="filters.model" data-testid="system-log-model" type="text" class="ops-log-input" />
            </label>
            <label class="ops-log-field-label">
              request_id
              <input
                v-model="filters.request_id"
                data-testid="system-log-request-id"
                type="text"
                class="ops-log-input ops-log-mono"
              />
            </label>
            <label class="ops-log-field-label">
              client_request_id
              <input
                v-model="filters.client_request_id"
                data-testid="system-log-client-request-id"
                type="text"
                class="ops-log-input ops-log-mono"
              />
            </label>
            <label class="ops-log-field-label">
              user_id
              <input
                v-model="filters.user_id"
                data-testid="system-log-user-id"
                inputmode="numeric"
                type="text"
                class="ops-log-input ops-log-mono"
              />
            </label>
            <label class="ops-log-field-label">
              {{ t('admin.ops.systemLogs.keyId') }}
              <input
                v-model="filters.api_key_id"
                data-testid="system-log-api-key-id"
                inputmode="numeric"
                type="text"
                class="ops-log-input ops-log-mono"
              />
            </label>
            <label class="ops-log-field-label">
              account_id
              <input
                v-model="filters.account_id"
                data-testid="system-log-account-id"
                inputmode="numeric"
                type="text"
                class="ops-log-input ops-log-mono"
              />
            </label>
          </div>
          <div
            v-if="props.investigationPreset?.groupId != null || props.investigationPreset?.region"
            class="ops-log-context-only"
          >
            <span>{{ t('admin.ops.systemLogs.contextOnly') }}</span>
            <code v-if="props.investigationPreset?.groupId != null">group_id={{ props.investigationPreset.groupId }}</code>
            <code v-if="props.investigationPreset?.region">region={{ props.investigationPreset.region }}</code>
          </div>
        </section>
      </Transition>

      <div
        v-if="activeFilterChips.length"
        class="ops-log-filter-chips"
        data-testid="ops-log-filter-chips"
      >
        <span class="ops-log-filter-chips__label">{{ t('admin.ops.systemLogs.appliedFilters') }}</span>
        <button
          v-for="chip in activeFilterChips"
          :key="chip.key"
          type="button"
          :title="chip.label + '=' + chip.value"
          @click="removeFilterChip(chip.key)"
        >
          <span>{{ chip.label }}={{ chip.value }}</span>
          <Icon name="x" size="xs" aria-hidden="true" />
        </button>
      </div>

      <Transition name="ops-log-popover">
        <aside
          v-if="logManagementOpen"
          id="ops-log-management"
          class="ops-log-management"
          data-testid="ops-log-management"
          tabindex="-1"
          @keydown.esc="logManagementOpen = false"
        >
          <header class="ops-log-management__header">
            <div>
              <span>{{ t('admin.ops.systemLogs.logManagement') }}</span>
              <strong>{{ runtimeConfigTitle }}</strong>
            </div>
            <button
              type="button"
              class="ops-log-icon-button"
              :aria-label="t('common.close')"
              @click="logManagementOpen = false"
            >
              <Icon name="x" size="sm" aria-hidden="true" />
            </button>
          </header>

          <div class="ops-log-management__scroll">
            <section class="ops-log-management__section">
              <p>{{ t('admin.ops.systemLogs.runtimeConfigHint') }}</p>
              <span v-if="runtimeLoading" class="ops-log-muted">{{ t('common.loading') }}</span>

              <div class="ops-log-management-grid">
                <label class="ops-log-field-label">
                  {{ t('admin.ops.systemLogs.level') }}
                  <Select v-model="runtimeConfig.level" :options="runtimeLevelOptions" />
                </label>
                <label class="ops-log-field-label">
                  {{ t('admin.ops.systemLogs.stacktraceThreshold') }}
                  <Select v-model="runtimeConfig.stacktrace_level" :options="stacktraceLevelOptions" />
                </label>
                <label class="ops-log-field-label">
                  {{ t('admin.ops.systemLogs.samplingInitial') }}
                  <input v-model.number="runtimeConfig.sampling_initial" type="number" min="1" class="ops-log-input" />
                </label>
                <label class="ops-log-field-label">
                  {{ t('admin.ops.systemLogs.samplingThereafter') }}
                  <input v-model.number="runtimeConfig.sampling_thereafter" type="number" min="1" class="ops-log-input" />
                </label>
                <label class="ops-log-field-label">
                  {{ t('admin.ops.systemLogs.retentionDays') }}
                  <input
                    v-model.number="runtimeConfig.retention_days"
                    type="number"
                    min="1"
                    max="3650"
                    class="ops-log-input"
                  />
                </label>
              </div>

              <p class="ops-log-retention-note" data-testid="runtime-retention-clarification">
                {{ t('admin.ops.systemLogs.retentionClarification') }}
              </p>

              <div class="ops-log-checks">
                <label>
                  <input v-model="runtimeConfig.enable_sampling" type="checkbox" />
                  {{ t('admin.ops.systemLogs.sampling') }}
                </label>
                <label>
                  <input v-model="runtimeConfig.caller" type="checkbox" />
                  {{ t('admin.ops.systemLogs.caller') }}
                </label>
              </div>

              <div class="ops-log-runtime-actions">
                <button type="button" :disabled="runtimeSaving" @click="saveRuntimeConfig">
                  {{ runtimeSaving ? t('common.saving') : t('admin.ops.systemLogs.saveAndApply') }}
                </button>
                <button type="button" :disabled="runtimeSaving" @click="resetRuntimeConfig">
                  {{ t('admin.ops.systemLogs.resetDefaults') }}
                </button>
              </div>
            </section>

            <section class="ops-log-danger-zone">
              <div>
                <Icon name="exclamationTriangle" size="sm" aria-hidden="true" />
                <div>
                  <strong>{{ t('admin.ops.systemLogs.cleanCurrentFilters') }}</strong>
                  <p>{{ t('admin.ops.systemLogs.cleanupDangerHint') }}</p>
                </div>
              </div>
              <button
                type="button"
                data-testid="system-log-cleanup"
                @click="cleanupCurrentFilter"
              >
                <Icon name="trash" size="xs" aria-hidden="true" />
                {{ t('admin.ops.systemLogs.cleanCurrentFilters') }}
              </button>
            </section>
          </div>
        </aside>
      </Transition>
    </section>

    <main class="ops-log-evidence" data-testid="ops-log-evidence">
      <div
        class="ops-log-split"
        :class="{ 'ops-log-split--inspecting': selectedLog }"
        data-testid="ops-log-split"
      >
        <section class="ops-log-results" aria-labelledby="ops-log-results-heading">
          <header class="ops-log-results-header">
            <div>
              <h3 id="ops-log-results-heading">{{ t('admin.ops.systemLogs.results') }}</h3>
              <span>{{ t('admin.ops.systemLogs.newestFirstCount', { count: total }) }}</span>
            </div>
            <div class="ops-log-results-paging">
              <button
                type="button"
                :disabled="page <= 1"
                :aria-label="t('pagination.previous')"
                @click="onPageChange(page - 1)"
              >
                <Icon name="chevronLeft" size="sm" aria-hidden="true" />
              </button>
              <b>{{ page }}</b>
              <span>/ {{ totalPages }}</span>
              <button
                type="button"
                :disabled="page >= totalPages"
                :aria-label="t('pagination.next')"
                @click="onPageChange(page + 1)"
              >
                <Icon name="chevronRight" size="sm" aria-hidden="true" />
              </button>
            </div>
          </header>

          <div class="ops-log-table-scroll" role="region" tabindex="0" :aria-label="t('admin.ops.systemLogs.results')">
            <table class="ops-log-table" data-testid="ops-system-log-table">
              <colgroup>
                <col class="ops-log-col-time" />
                <col class="ops-log-col-level" />
                <col class="ops-log-col-component" />
                <col class="ops-log-col-summary" />
                <col class="ops-log-col-status" />
                <col class="ops-log-col-request" />
              </colgroup>
              <thead>
                <tr>
                  <th>{{ t('admin.ops.systemLogs.time') }}</th>
                  <th>{{ t('admin.ops.systemLogs.level') }}</th>
                  <th>{{ t('admin.ops.systemLogs.component') }}</th>
                  <th>{{ t('admin.ops.systemLogs.requestSummary') }}</th>
                  <th>{{ t('admin.ops.systemLogs.statusDuration') }}</th>
                  <th>request_id</th>
                </tr>
              </thead>
              <tbody v-if="hasData && !loading">
                <tr
                  v-for="row in logs"
                  :key="row.id"
                  tabindex="0"
                  class="ops-log-row"
                  :class="{ 'ops-log-row--selected': selectedLog?.id === row.id }"
                  :aria-selected="selectedLog?.id === row.id"
                  :aria-controls="selectedLog?.id === row.id ? 'ops-log-inspector' : undefined"
                  :data-testid="'system-log-row-' + row.id"
                  @click="selectLog(row)"
                  @keydown.enter.prevent="selectLog(row)"
                  @keydown.space.prevent="selectLog(row)"
                >
                  <td class="ops-log-time-cell">{{ formatTime(row.created_at) }}</td>
                  <td>
                    <span class="ops-log-level" :class="levelBadgeClass(row.level)">
                      {{ row.level || '—' }}
                    </span>
                  </td>
                  <td class="ops-log-component-cell">
                    <span :title="row.component || '—'">{{ row.component || '—' }}</span>
                    <small v-if="row.host" :title="row.host">{{ row.host }}</small>
                  </td>
                  <td>
                    <div class="ops-log-request-summary">
                      <strong v-if="getRequestRoute(row)" :title="getRequestRoute(row)">
                        {{ getRequestRoute(row) }}
                      </strong>
                      <span :title="row.message || '—'">{{ row.message || '—' }}</span>
                    </div>
                  </td>
                  <td>
                    <div class="ops-log-status-duration">
                      <strong
                        v-if="getStatusCode(row)"
                        :class="{ 'ops-log-danger': isErrorStatus(row), 'ops-log-success': !isErrorStatus(row) }"
                      >
                        {{ getStatusCode(row) }}
                      </strong>
                      <span v-if="getStatusCode(row) && getLatency(row)">/</span>
                      <span v-if="getLatency(row)">{{ getLatency(row) }}</span>
                      <span v-if="!getStatusCode(row) && !getLatency(row)">—</span>
                    </div>
                  </td>
                  <td>
                    <code class="ops-log-request-id" :title="row.request_id || '—'">
                      {{ row.request_id || '—' }}
                    </code>
                  </td>
                </tr>
              </tbody>
            </table>

            <div v-if="loading" class="ops-log-empty" role="status">
              <span class="ops-log-loading-mark" aria-hidden="true"></span>
              <strong>{{ t('common.loading') }}</strong>
            </div>
            <div v-else-if="!hasData" class="ops-log-empty" role="status">
              <Icon name="terminal" size="xl" aria-hidden="true" />
              <strong>{{ t('admin.ops.systemLogs.empty') }}</strong>
              <span>{{ t('admin.ops.systemLogs.emptyHint') }}</span>
            </div>
          </div>

          <footer class="ops-log-results-footer">
            <span>{{ logs.length }} / {{ total }}</span>
            <label>
              <span class="sr-only">{{ t('admin.ops.systemLogs.perPage') }}</span>
              <select
                :value="pageSize"
                :aria-label="t('admin.ops.systemLogs.perPage')"
                @change="onPageSizeChange(Number(($event.target as HTMLSelectElement).value))"
              >
                <option :value="20">20</option>
                <option :value="50">50</option>
                <option :value="100">100</option>
              </select>
              <span>{{ t('admin.ops.systemLogs.perPage') }}</span>
            </label>
          </footer>
        </section>

        <aside
            v-if="selectedLog"
            id="ops-log-inspector"
            class="ops-log-inspector"
            aria-labelledby="ops-log-inspector-heading"
            data-testid="ops-log-inspector"
          >
            <header class="ops-log-inspector-header">
              <div>
                <span class="ops-log-inspector-kicker">{{ t('admin.ops.systemLogs.inspectorTitle') }}</span>
                <h4 id="ops-log-inspector-heading">
                  <code>{{ selectedLog.request_id || '#' + selectedLog.id }}</code>
                </h4>
              </div>
              <button
                type="button"
                class="ops-log-icon-button"
                data-testid="ops-log-inspector-close"
                :aria-label="t('admin.ops.systemLogs.closeDetails')"
                @click="closeInspector"
              >
                <Icon name="x" size="sm" aria-hidden="true" />
              </button>
            </header>

            <div class="ops-log-inspector-scroll" data-testid="ops-log-inspector-content">
              <section class="ops-log-detail-section">
                <h5>{{ t('admin.ops.systemLogs.logDetails') }}</h5>
                <dl class="ops-log-detail-grid">
                  <div>
                    <dt>{{ t('admin.ops.systemLogs.time') }}</dt>
                    <dd>{{ formatTime(selectedLog.created_at) }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('admin.ops.systemLogs.level') }}</dt>
                    <dd>
                      <span class="ops-log-level" :class="levelBadgeClass(selectedLog.level)">
                        {{ selectedLog.level || '—' }}
                      </span>
                    </dd>
                  </div>
                  <div>
                    <dt>{{ t('admin.ops.systemLogs.host') }}</dt>
                    <dd>{{ selectedLog.host || '—' }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('admin.ops.systemLogs.component') }}</dt>
                    <dd>{{ selectedLog.component || '—' }}</dd>
                  </div>
                </dl>
              </section>

              <section class="ops-log-detail-section">
                <h5>{{ t('admin.ops.systemLogs.message') }}</h5>
                <pre>{{ selectedLog.message || '—' }}</pre>
              </section>

              <section class="ops-log-detail-section">
                <h5>{{ t('admin.ops.systemLogs.relatedIdentifiers') }}</h5>
                <dl class="ops-log-detail-list">
                  <div><dt>request_id</dt><dd>{{ selectedLog.request_id || '—' }}</dd></div>
                  <div><dt>client_request_id</dt><dd>{{ selectedLog.client_request_id || '—' }}</dd></div>
                  <div><dt>user_id</dt><dd>{{ selectedLog.user_id ?? '—' }}</dd></div>
                  <div><dt>api_key_id</dt><dd>{{ selectedLog.api_key_id ?? '—' }}</dd></div>
                  <div><dt>account_id</dt><dd>{{ selectedLog.account_id ?? '—' }}</dd></div>
                </dl>
              </section>

              <section class="ops-log-detail-section">
                <h5>{{ t('admin.ops.systemLogs.requestContext') }}</h5>
                <dl class="ops-log-detail-grid ops-log-chain-grid">
                  <div>
                    <dt>method / path</dt>
                    <dd>{{ getRequestRoute(selectedLog) || '—' }}</dd>
                  </div>
                  <div>
                    <dt>client_ip / protocol</dt>
                    <dd>{{ getExtraValue(selectedLog, 'client_ip') }} / {{ getExtraValue(selectedLog, 'protocol') }}</dd>
                  </div>
                  <div>
                    <dt>platform / model</dt>
                    <dd>{{ selectedLog.platform || '—' }} / {{ selectedLog.model || '—' }}</dd>
                  </div>
                  <div>
                    <dt>status_code / latency_ms</dt>
                    <dd>{{ getExtraValue(selectedLog, 'status_code') }} / {{ getExtraValue(selectedLog, 'latency_ms') }}</dd>
                  </div>
                </dl>
              </section>

              <section v-if="selectedErrors !== '-'" class="ops-log-detail-section ops-log-error-section">
                <h5>{{ t('admin.ops.systemLogs.errorDetails') }}</h5>
                <pre>{{ selectedErrors }}</pre>
              </section>

              <details v-if="selectedExtra !== '-'" class="ops-log-detail-section ops-log-raw-extra">
                <summary>{{ t('admin.ops.systemLogs.rawExtra') }}</summary>
                <pre>{{ selectedExtra }}</pre>
              </details>
            </div>

            <footer class="ops-log-inspector-footer">
              <button
                type="button"
                data-testid="ops-log-open-request-details"
                @click="emit('open-request-details', selectedLog)"
              >
                <Icon name="externalLink" size="xs" aria-hidden="true" />
                {{ t('admin.ops.diagnosticsSectionOpenRequestDetails') }}
              </button>
            </footer>
        </aside>
      </div>
    </main>
  </section>
</template>

<style scoped>
.ops-log-workbench {
  --ops-log-context-height: 0px;
  --ops-log-management-anchor: 54px;

  position: relative;
  display: flex;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  color: var(--lx-clay-text);
  background: #ffffff;
  font-family: var(--lx-clay-font-ui);
}

.ops-log-workbench--with-context {
  --ops-log-context-height: 42px;
}

.ops-log-context-bar {
  display: flex;
  min-height: 42px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 16px;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-warning) 22%, var(--lx-clay-border));
  color: var(--lx-clay-text-secondary);
  background: color-mix(in srgb, var(--lx-clay-warning-soft) 74%, #ffffff);
  font-size: 11px;
}

.ops-log-context-bar__content {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
}

.ops-log-context-bar__content::-webkit-scrollbar {
  display: none;
}

.ops-log-context-bar__content > svg {
  flex: 0 0 auto;
  color: var(--lx-clay-warning);
}

.ops-log-context-bar__content strong {
  flex: 0 0 auto;
  color: var(--lx-clay-text);
  font-weight: 800;
}

.ops-log-context-id,
.ops-log-context-chip,
.ops-log-context-window {
  flex: 0 0 auto;
  font-family: var(--lx-clay-font-mono);
}

.ops-log-context-id {
  color: var(--lx-clay-warning);
  font-weight: 800;
}

.ops-log-context-chip {
  padding: 3px 7px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-warning) 18%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--lx-clay-warning-soft) 68%, #ffffff);
}

.ops-log-context-window {
  color: var(--lx-clay-text-muted);
}

.ops-log-query-shell {
  position: static;
  z-index: 15;
  flex: 0 0 auto;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #ffffff;
}

.ops-log-query-toolbar {
  display: flex;
  min-width: 0;
  min-height: 60px;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
}

.ops-log-toolbar-control,
.ops-log-toolbar-search {
  display: flex;
  min-width: 0;
  height: 36px;
  align-items: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 9px;
  color: var(--lx-clay-text-secondary);
  background: #ffffff;
  transition:
    border-color 150ms ease,
    box-shadow 150ms ease;
}

.ops-log-toolbar-control:focus-within,
.ops-log-toolbar-search:focus-within {
  border-color: var(--lx-clay-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 16%, transparent);
}

.ops-log-toolbar-time {
  width: 94px;
  flex: 0 0 94px;
  gap: 5px;
  padding: 0 8px;
}

.ops-log-toolbar-time select,
.ops-log-toolbar-control input,
.ops-log-toolbar-search input {
  width: 100%;
  min-width: 0;
  height: 100%;
  border: 0;
  outline: 0;
  color: var(--lx-clay-text);
  background: transparent;
  font: inherit;
  font-size: 12px;
}

.ops-log-toolbar-time select {
  appearance: none;
  cursor: pointer;
}

.ops-log-toolbar-search {
  position: relative;
  max-width: 360px;
  flex: 1 1 240px;
  gap: 7px;
  padding: 0 10px;
}

.ops-log-toolbar-search > svg {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
}

.ops-log-toolbar-search input::placeholder,
.ops-log-toolbar-control input::placeholder {
  color: var(--lx-clay-text-muted);
}

.ops-log-toolbar-select {
  width: 112px;
  min-width: 96px;
  flex: 0 1 112px;
}

.ops-log-toolbar-select :deep(.select-trigger) {
  min-height: 36px;
  height: 36px;
  padding: 0 9px;
  border-radius: 9px;
  background: #ffffff;
  font-size: 12px;
  box-shadow: none;
}

.ops-log-toolbar-component {
  width: 132px;
  min-width: 108px;
  flex: 0 1 132px;
  padding: 0 10px;
}

.ops-log-search-button,
.ops-log-secondary-button,
.ops-log-runtime-actions button,
.ops-log-inspector-footer button {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border-radius: 9px;
  font: inherit;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    color 150ms ease,
    transform 150ms ease;
}

.ops-log-search-button {
  flex: 0 0 auto;
  padding: 0 16px;
  border: 1px solid var(--lx-clay-accent);
  color: #ffffff;
  background: var(--lx-clay-accent);
}

.ops-log-search-button:hover:not(:disabled) {
  background: var(--lx-clay-accent-deep);
}

.ops-log-secondary-button {
  flex: 0 0 auto;
  padding: 0 10px;
  border: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: #ffffff;
}

.ops-log-secondary-button:hover {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-log-icon-button {
  display: inline-grid;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  place-items: center;
  border: 1px solid transparent;
  border-radius: 9px;
  color: var(--lx-clay-text-muted);
  background: transparent;
  cursor: pointer;
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    color 150ms ease;
}

.ops-log-icon-button:hover:not(:disabled) {
  border-color: var(--lx-clay-border);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.ops-log-reset-button {
  border-color: var(--lx-clay-border);
}

.ops-log-count-badge {
  display: inline-grid;
  min-width: 18px;
  height: 18px;
  place-items: center;
  padding: 0 5px;
  border-radius: 999px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  font-size: 10px;
  font-weight: 900;
}

.ops-log-chevron {
  transition: transform 160ms ease;
}

.ops-log-chevron--open {
  transform: rotate(180deg);
}

.ops-log-health-strip {
  position: relative;
  margin-left: auto;
}

.ops-log-health-status {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 999px;
  color: var(--lx-clay-text-secondary);
  background: #ffffff;
  font: inherit;
  font-size: 10px;
  cursor: pointer;
  white-space: nowrap;
}

.ops-log-health-status span {
  font-weight: 650;
}

.ops-log-health-status b {
  font-weight: 850;
}

.ops-log-health-status--healthy {
  border-color: color-mix(in srgb, var(--lx-clay-success) 22%, transparent);
  color: var(--lx-clay-success);
  background: var(--lx-clay-success-soft);
}

.ops-log-health-status--warning {
  border-color: color-mix(in srgb, var(--lx-clay-warning) 24%, transparent);
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.ops-log-health-status--danger {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 24%, transparent);
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-log-health-status--loading,
.ops-log-health-status--unknown {
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-recessed);
}

.ops-log-health-popover {
  position: absolute;
  z-index: 80;
  top: calc(100% + 8px);
  right: 0;
  width: 318px;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 14px;
  color: var(--lx-clay-text);
  background: #ffffff;
  box-shadow: var(--lx-clay-shadow-overlay);
}

.ops-log-health-popover header {
  display: flex;
  min-height: 52px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px 8px 14px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fafafa;
}

.ops-log-health-popover header > div {
  display: grid;
  gap: 1px;
}

.ops-log-health-popover header strong {
  font-size: 12px;
}

.ops-log-health-popover header span {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-health-popover dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
  padding: 8px 14px 12px;
}

.ops-log-health-popover dl > div {
  display: grid;
  gap: 2px;
  padding: 8px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 65%, transparent);
}

.ops-log-health-popover dl > div:nth-last-child(-n + 2) {
  border-bottom: 0;
}

.ops-log-health-popover dt {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-health-popover dd {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 12px;
  font-weight: 700;
}

.ops-log-health-popover > p {
  margin: 0 12px 12px;
  padding: 9px 10px;
  border-radius: 9px;
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
  font-size: 10px;
  line-height: 1.5;
}

.ops-log-management-toggle {
  border-color: var(--lx-clay-border);
}

.ops-log-advanced-filters {
  padding: 12px 16px 14px;
  border-top: 1px solid var(--lx-clay-border);
  background: #fcfcfc;
}

.ops-log-advanced-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(130px, 1fr));
  gap: 10px 12px;
}

.ops-log-field-label {
  display: grid;
  min-width: 0;
  gap: 5px;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  font-weight: 750;
}

.ops-log-input {
  width: 100%;
  min-width: 0;
  min-height: 34px;
  padding: 0 9px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  outline: 0;
  color: var(--lx-clay-text);
  background: #ffffff;
  font: inherit;
  font-size: 11px;
}

.ops-log-input:hover {
  border-color: var(--lx-clay-border-strong);
}

.ops-log-input:focus {
  border-color: var(--lx-clay-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 15%, transparent);
}

.ops-log-mono {
  font-family: var(--lx-clay-font-mono);
}

.ops-log-context-only {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 10px;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-context-only code {
  padding: 3px 7px;
  border-radius: 6px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-family: var(--lx-clay-font-mono);
}

.ops-log-filter-chips {
  display: flex;
  min-height: 38px;
  align-items: center;
  gap: 6px;
  padding: 5px 16px;
  overflow-x: auto;
  border-top: 1px solid var(--lx-clay-border);
  background: #ffffff;
  scrollbar-width: thin;
}

.ops-log-filter-chips__label {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  font-weight: 750;
}

.ops-log-filter-chips button {
  display: inline-flex;
  min-height: 26px;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  max-width: 260px;
  padding: 0 8px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 17%, transparent);
  border-radius: 999px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  font: inherit;
  font-size: 10px;
  cursor: pointer;
}

.ops-log-filter-chips button span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-log-management {
  position: absolute;
  z-index: 70;
  top: calc(var(--ops-log-context-height) + var(--ops-log-management-anchor));
  right: 14px;
  width: min(448px, calc(100% - 28px));
  max-height: min(
    570px,
    calc(100% - var(--ops-log-context-height) - var(--ops-log-management-anchor) - 14px)
  );
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: 14px;
  color: var(--lx-clay-text);
  background: #ffffff;
  box-shadow: var(--lx-clay-shadow-overlay);
}

.ops-log-management__header {
  display: flex;
  min-height: 60px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 8px 10px 8px 16px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #fafafa;
}

.ops-log-management__header > div {
  display: grid;
  gap: 2px;
}

.ops-log-management__header span {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  font-weight: 750;
}

.ops-log-management__header strong {
  font-size: 13px;
}

.ops-log-management__scroll {
  max-height: inherit;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.ops-log-management__section {
  padding: 16px;
}

.ops-log-management__section > p:first-child {
  margin: 0 0 14px;
  color: var(--lx-clay-text-secondary);
  font-size: 11px;
  line-height: 1.55;
}

.ops-log-management-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.ops-log-management-grid :deep(.select-trigger) {
  min-height: 36px;
  border-radius: 8px;
  background: #ffffff;
  font-size: 11px;
}

.ops-log-retention-note {
  margin: 12px 0 0;
  padding: 8px 10px;
  border-radius: 8px;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-recessed);
  font-size: 10px;
  line-height: 1.5;
}

.ops-log-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-top: 12px;
}

.ops-log-checks label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--lx-clay-text-secondary);
  font-size: 11px;
}

.ops-log-checks input {
  accent-color: var(--lx-clay-accent);
}

.ops-log-runtime-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}

.ops-log-runtime-actions button {
  padding: 0 12px;
  border: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: #ffffff;
}

.ops-log-runtime-actions button:first-child {
  border-color: var(--lx-clay-accent);
  color: #ffffff;
  background: var(--lx-clay-accent);
}

.ops-log-danger-zone {
  display: grid;
  gap: 12px;
  padding: 14px 16px 16px;
  border-top: 1px solid color-mix(in srgb, var(--lx-clay-danger) 18%, var(--lx-clay-border));
  background: color-mix(in srgb, var(--lx-clay-danger-soft) 42%, #ffffff);
}

.ops-log-danger-zone > div {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  color: var(--lx-clay-danger);
}

.ops-log-danger-zone strong {
  font-size: 12px;
}

.ops-log-danger-zone p {
  margin: 2px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
  line-height: 1.45;
}

.ops-log-danger-zone > button {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-danger) 24%, transparent);
  border-radius: 8px;
  color: var(--lx-clay-danger);
  background: #ffffff;
  font: inherit;
  font-size: 11px;
  font-weight: 800;
  cursor: pointer;
}

.ops-log-danger-zone > button:hover {
  color: #ffffff;
  background: var(--lx-clay-danger);
}

.ops-log-evidence {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
  background: #fcfcfc;
}

.ops-log-split {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.ops-log-results {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: #ffffff;
}

.ops-log-results-header {
  display: flex;
  min-height: 44px;
  flex: 0 0 44px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 14px 0 16px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #ffffff;
}

.ops-log-results-header > div:first-child {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 8px;
}

.ops-log-results-header h3 {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 12px;
  font-weight: 850;
}

.ops-log-results-header span {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-results-paging {
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-results-paging button {
  display: inline-grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border: 1px solid var(--lx-clay-border);
  border-radius: 7px;
  color: var(--lx-clay-text-secondary);
  background: #ffffff;
  cursor: pointer;
}

.ops-log-results-paging button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.ops-log-results-paging b {
  color: var(--lx-clay-text);
}

.ops-log-table-scroll {
  position: relative;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  overflow: auto;
  outline: 0;
  background: #ffffff;
  scrollbar-gutter: stable;
}

.ops-log-table-scroll:focus-visible {
  box-shadow: inset 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 20%, transparent);
}

.ops-log-table {
  width: 100%;
  min-width: 670px;
  border-collapse: separate;
  border-spacing: 0;
  table-layout: fixed;
  color: var(--lx-clay-text);
  font-size: 11px;
}

/* The inspector already anchors the selected request id. Remove the repeated
   column while inspecting so the evidence list keeps its scan rhythm without
   introducing a horizontal scrollbar. */
.ops-log-split--inspecting .ops-log-table {
  min-width: 0;
}

.ops-log-split--inspecting .ops-log-col-request,
.ops-log-split--inspecting .ops-log-table th:nth-child(6),
.ops-log-split--inspecting .ops-log-table td:nth-child(6) {
  display: none;
}

.ops-log-col-time {
  width: 142px;
}

.ops-log-col-level {
  width: 76px;
}

.ops-log-col-component {
  width: 118px;
}

.ops-log-col-summary {
  width: auto;
}

.ops-log-col-status {
  width: 98px;
}

.ops-log-col-request {
  width: 140px;
}

.ops-log-table th {
  position: sticky;
  z-index: 5;
  top: 0;
  height: 38px;
  padding: 0 10px;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-muted);
  background: #f7f7f8;
  font-size: 9px;
  font-weight: 850;
  letter-spacing: 0.035em;
  text-align: left;
  text-transform: uppercase;
}

.ops-log-table td {
  height: 52px;
  padding: 7px 10px;
  overflow: hidden;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 64%, transparent);
  color: var(--lx-clay-text);
  vertical-align: middle;
}

.ops-log-row {
  cursor: pointer;
  outline: 0;
  transition: background-color 120ms ease;
}

.ops-log-row:hover {
  background: #f8f7fa;
}

.ops-log-row--selected,
.ops-log-row--selected:hover {
  background: var(--lx-clay-accent-soft);
}

.ops-log-row:focus-visible {
  position: relative;
  z-index: 2;
  box-shadow: inset 3px 0 var(--lx-clay-accent);
}

.ops-log-time-cell {
  color: var(--lx-clay-text-muted) !important;
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
  white-space: nowrap;
}

.ops-log-level {
  display: inline-flex;
  min-height: 20px;
  align-items: center;
  padding: 0 7px;
  border-radius: 999px;
  font-size: 9px;
  font-weight: 900;
  letter-spacing: 0.025em;
  text-transform: uppercase;
}

.ops-log-level--debug {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.ops-log-level--info {
  color: var(--lx-clay-info-deep);
  background: var(--lx-clay-info-soft);
}

.ops-log-level--warn {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.ops-log-level--error {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-log-component-cell > span,
.ops-log-component-cell > small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-log-component-cell > span {
  font-weight: 700;
}

.ops-log-component-cell > small {
  margin-top: 2px;
  color: var(--lx-clay-text-muted);
  font-family: var(--lx-clay-font-mono);
  font-size: 9px;
}

.ops-log-request-summary {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.ops-log-request-summary strong,
.ops-log-request-summary span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-log-request-summary strong {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
}

.ops-log-request-summary span {
  color: var(--lx-clay-text-secondary);
}

.ops-log-status-duration {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 4px;
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
  white-space: nowrap;
}

.ops-log-success {
  color: var(--lx-clay-success);
}

.ops-log-warning {
  color: var(--lx-clay-warning) !important;
}

.ops-log-danger {
  color: var(--lx-clay-danger) !important;
}

.ops-log-request-id {
  display: block;
  overflow: hidden;
  color: var(--lx-clay-accent-deep);
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
  text-overflow: ellipsis;
  user-select: all;
  white-space: nowrap;
}

.ops-log-empty {
  position: absolute;
  inset: 38px 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 7px;
  color: var(--lx-clay-text-muted);
  background: #ffffff;
  text-align: center;
}

.ops-log-empty > svg {
  color: color-mix(in srgb, var(--lx-clay-text-muted) 24%, transparent);
}

.ops-log-empty strong {
  color: var(--lx-clay-text-secondary);
  font-size: 12px;
}

.ops-log-empty span {
  max-width: 34ch;
  font-size: 10px;
  line-height: 1.5;
}

.ops-log-loading-mark {
  width: 24px;
  height: 24px;
  border: 2px solid var(--lx-clay-border);
  border-top-color: var(--lx-clay-accent);
  border-radius: 50%;
  animation: ops-log-spin 800ms linear infinite;
}

.ops-log-results-footer {
  display: flex;
  min-height: 40px;
  flex: 0 0 40px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 14px;
  border-top: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-muted);
  background: #ffffff;
  font-size: 10px;
}

.ops-log-results-footer label {
  display: flex;
  align-items: center;
  gap: 5px;
}

.ops-log-results-footer select {
  height: 26px;
  padding: 0 6px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 7px;
  color: var(--lx-clay-text);
  background: #ffffff;
  font: inherit;
}

.ops-log-inspector {
  display: flex;
  width: 420px;
  min-width: 380px;
  min-height: 0;
  flex: 0 0 420px;
  flex-direction: column;
  overflow: hidden;
  border-left: 1px solid var(--lx-clay-border);
  background: #ffffff;
}

.ops-log-inspector-header {
  display: flex;
  min-height: 54px;
  flex: 0 0 54px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 10px 7px 16px;
  border-bottom: 1px solid var(--lx-clay-border);
  background: #f7f7f8;
}

.ops-log-inspector-header > div {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.ops-log-inspector-kicker {
  color: var(--lx-clay-text-muted);
  font-size: 9px;
  font-weight: 850;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.ops-log-inspector-header h4 {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ops-log-inspector-header code {
  color: var(--lx-clay-accent-deep);
  font-family: var(--lx-clay-font-mono);
}

.ops-log-inspector-scroll {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  padding: 18px;
  overscroll-behavior: contain;
}

.ops-log-detail-section {
  margin: 0 0 20px;
}

.ops-log-detail-section h5,
.ops-log-raw-extra summary {
  margin: 0 0 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 68%, transparent);
  color: var(--lx-clay-text-muted);
  font-size: 9px;
  font-weight: 900;
  letter-spacing: 0.055em;
  text-transform: uppercase;
}

.ops-log-detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 14px;
  margin: 0;
}

.ops-log-detail-grid > div,
.ops-log-detail-list > div {
  min-width: 0;
}

.ops-log-detail-grid dt,
.ops-log-detail-list dt {
  color: var(--lx-clay-text-muted);
  font-size: 9px;
  font-weight: 750;
}

.ops-log-detail-grid dd {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 11px;
  font-weight: 650;
  text-overflow: ellipsis;
}

.ops-log-detail-section pre {
  margin: 0;
  padding: 10px 11px;
  overflow-x: auto;
  border: 1px solid var(--lx-clay-border);
  border-radius: 9px;
  color: var(--lx-clay-text-secondary);
  background: #fafafa;
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
}

.ops-log-detail-list {
  display: grid;
  gap: 0;
  margin: 0;
}

.ops-log-detail-list > div {
  display: grid;
  grid-template-columns: 124px minmax(0, 1fr);
  gap: 10px;
  padding: 7px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-border) 60%, transparent);
}

.ops-log-detail-list dd {
  margin: 0;
  overflow: hidden;
  color: var(--lx-clay-accent-deep);
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
  text-align: right;
  text-overflow: ellipsis;
  user-select: all;
  white-space: nowrap;
}

.ops-log-chain-grid dd {
  font-family: var(--lx-clay-font-mono);
  font-size: 10px;
}

.ops-log-error-section pre {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 18%, transparent);
  color: var(--lx-clay-danger);
  background: color-mix(in srgb, var(--lx-clay-danger-soft) 52%, #ffffff);
}

.ops-log-raw-extra {
  border: 0;
}

.ops-log-raw-extra summary {
  cursor: pointer;
  list-style-position: inside;
}

.ops-log-raw-extra pre {
  color: #d8d3df;
  background: var(--lx-clay-code-canvas);
}

.ops-log-inspector-footer {
  display: flex;
  min-height: 52px;
  flex: 0 0 52px;
  align-items: center;
  justify-content: flex-end;
  padding: 8px 12px;
  border-top: 1px solid var(--lx-clay-border);
  background: #ffffff;
}

.ops-log-inspector-footer button {
  padding: 0 12px;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 24%, transparent);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-log-muted {
  color: var(--lx-clay-text-muted);
  font-size: 10px;
}

.ops-log-popover-enter-active,
.ops-log-popover-leave-active,
.ops-log-disclosure-enter-active,
.ops-log-disclosure-leave-active,
.ops-log-inspector-enter-active,
.ops-log-inspector-leave-active {
  transition:
    opacity 160ms ease,
    transform 180ms cubic-bezier(0.22, 1, 0.36, 1);
}

.ops-log-popover-enter-from,
.ops-log-popover-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.985);
}

.ops-log-disclosure-enter-from,
.ops-log-disclosure-leave-to {
  opacity: 0;
  transform: translateY(-5px);
}

.ops-log-inspector-enter-from,
.ops-log-inspector-leave-to {
  opacity: 0;
  transform: translateX(12px);
}

.ops-log-spin {
  animation: ops-log-spin 800ms linear infinite;
}

@keyframes ops-log-spin {
  to {
    transform: rotate(360deg);
  }
}

@container ops-dashboard (max-width: 980px) {
  .ops-log-query-toolbar {
    gap: 6px;
    padding-inline: 10px;
  }

  .ops-log-toolbar-search {
    min-width: 160px;
  }

  .ops-log-toolbar-component {
    width: 112px;
    min-width: 90px;
  }

  .ops-log-health-status span {
    display: none;
  }

  .ops-log-secondary-button {
    padding-inline: 8px;
  }
}

@container ops-dashboard (max-width: 820px) {
  .ops-log-workbench {
    --ops-log-management-anchor: 96px;
  }

  .ops-log-query-toolbar {
    display: grid;
    min-height: 102px;
    grid-template-columns: 92px minmax(180px, 1fr) 106px 112px auto auto;
    align-content: center;
  }

  .ops-log-toolbar-time,
  .ops-log-toolbar-search,
  .ops-log-toolbar-select,
  .ops-log-toolbar-component {
    width: auto;
    max-width: none;
  }

  .ops-log-health-strip {
    grid-column: 1 / 3;
    margin-left: 0;
  }

  .ops-log-management-toggle {
    grid-column: 6;
    grid-row: 2;
  }

  .ops-log-advanced-grid {
    grid-template-columns: repeat(3, minmax(120px, 1fr));
  }

  .ops-log-inspector {
    position: absolute;
    z-index: 45;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(420px, 100%);
    min-width: 0;
    flex-basis: auto;
    box-shadow: -14px 0 34px rgba(48, 35, 70, 0.14);
  }

  .ops-log-split {
    position: relative;
  }
}

@container ops-dashboard (max-width: 640px) {
  .ops-log-workbench {
    --ops-log-management-anchor: 142px;
  }

  .ops-log-context-bar {
    padding-inline: 10px;
  }

  .ops-log-query-toolbar {
    min-height: 148px;
    grid-template-columns: 92px minmax(0, 1fr) 40px;
  }

  .ops-log-toolbar-search {
    grid-column: 2 / 4;
  }

  .ops-log-toolbar-select,
  .ops-log-toolbar-component {
    grid-column: span 1;
  }

  .ops-log-search-button {
    grid-column: 1 / 2;
  }

  .ops-log-reset-button {
    grid-column: 2 / 3;
  }

  .ops-log-secondary-button {
    grid-column: 2 / 4;
  }

  .ops-log-health-strip {
    grid-column: 1 / 3;
  }

  .ops-log-management-toggle {
    grid-column: 3;
    grid-row: auto;
  }

  .ops-log-advanced-filters {
    max-height: 42dvh;
    overflow-y: auto;
  }

  .ops-log-advanced-grid,
  .ops-log-management-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-log-health-popover {
    right: auto;
    left: 0;
    width: min(318px, calc(100vw - 24px));
  }

  .ops-log-management {
    right: 8px;
    width: calc(100% - 16px);
  }

  .ops-log-col-component,
  .ops-log-table th:nth-child(3),
  .ops-log-table td:nth-child(3) {
    display: none;
  }

  .ops-log-table {
    min-width: 520px;
  }

  .ops-log-detail-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-log-workbench *,
  .ops-log-workbench *::before,
  .ops-log-workbench *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>
