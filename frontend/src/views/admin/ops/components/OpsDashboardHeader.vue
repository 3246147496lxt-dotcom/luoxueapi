<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api'
import {
  opsAPI,
  type OpsDashboardOverview,
  type OpsMetricThresholds,
  type OpsRealtimeTrafficSummary,
  type OpsResourceView
} from '@/api/admin/ops'
import type { OpsRequestDetailsPreset } from './OpsRequestDetailsModal.vue'
import { useAdminSettingsStore } from '@/stores'
import { formatNumber } from '@/utils/format'

type RealtimeWindow = '1min' | '5min' | '30min' | '1h'

interface Props {
  overview?: OpsDashboardOverview | null
  platform: string
  groupId: number | null
  timeRange: string
  queryMode: string
  loading: boolean
  lastUpdated: Date | null
  thresholds?: OpsMetricThresholds | null // 阈值配置
  autoRefreshEnabled?: boolean
  autoRefreshCountdown?: number
  fullscreen?: boolean
  customStartTime?: string | null
  customEndTime?: string | null
  workspace?: string
  resource?: OpsResourceView
}

interface Emits {
  (e: 'update:platform', value: string): void
  (e: 'update:group', value: number | null): void
  (e: 'update:timeRange', value: string): void
  (e: 'update:queryMode', value: string): void
  (e: 'update:customTimeRange', startTime: string, endTime: string): void
  (e: 'refresh'): void
  (e: 'openRequestDetails', preset?: OpsRequestDetailsPreset): void
  (e: 'openErrorDetails', kind: 'request' | 'upstream'): void
  (e: 'openSettings'): void
  (e: 'openAlertRules'): void
  (e: 'enterFullscreen'): void
  (e: 'exitFullscreen'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const adminSettingsStore = useAdminSettingsStore()

const realtimeWindow = ref<RealtimeWindow>('1min')

const overview = computed(() => props.overview ?? null)
const systemMetrics = computed(() => overview.value?.system_metrics ?? null)

const REALTIME_WINDOW_MINUTES: Record<RealtimeWindow, number> = {
  '1min': 1,
  '5min': 5,
  '30min': 30,
  '1h': 60
}

const TOOLBAR_RANGE_MINUTES: Record<string, number> = {
  '5m': 5,
  '30m': 30,
  '1h': 60,
  '6h': 6 * 60,
  '24h': 24 * 60
}

const availableRealtimeWindows = computed(() => {
  const toolbarMinutes = TOOLBAR_RANGE_MINUTES[props.timeRange] ?? 60
  return (['1min', '5min', '30min', '1h'] as const).filter((w) => REALTIME_WINDOW_MINUTES[w] <= toolbarMinutes)
})

watch(
  () => props.timeRange,
  () => {
    // The realtime window must be inside the toolbar window; reset to keep UX predictable.
    realtimeWindow.value = '1min'
    // Keep realtime traffic consistent with toolbar changes even when the window is already 1min.
    loadRealtimeTrafficSummary()
  }
)

// --- Filters ---

const showCustomTimeRangeDialog = ref(false)
const customStartTimeInput = ref('')
const customEndTimeInput = ref('')

function formatCustomTimeRangeLabel(startTime: string, endTime: string): string {
  const start = new Date(startTime)
  const end = new Date(endTime)
  const formatDate = (d: Date) => {
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const hour = String(d.getHours()).padStart(2, '0')
    const minute = String(d.getMinutes()).padStart(2, '0')
    return `${month}-${day} ${hour}:${minute}`
  }
  return `${formatDate(start)} ~ ${formatDate(end)}`
}

const groups = ref<Array<{ id: number; name: string; platform: string }>>([])

const platformOptions = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
  { value: 'grok', label: 'Grok' }
])

const timeRangeOptions = computed(() => [
  { value: '5m', label: t('admin.ops.timeRange.5m') },
  { value: '30m', label: t('admin.ops.timeRange.30m') },
  { value: '1h', label: t('admin.ops.timeRange.1h') },
  { value: '6h', label: t('admin.ops.timeRange.6h') },
  { value: '24h', label: t('admin.ops.timeRange.24h') },
  {
    value: 'custom',
    label: props.timeRange === 'custom' && props.customStartTime && props.customEndTime
      ? `${t('admin.ops.timeRange.custom')} (${formatCustomTimeRangeLabel(props.customStartTime, props.customEndTime)})`
      : t('admin.ops.timeRange.custom')
  }
])

const queryModeOptions = computed(() => [
  { value: 'auto', label: t('admin.ops.queryMode.auto') },
  { value: 'raw', label: t('admin.ops.queryMode.raw') },
  { value: 'preagg', label: t('admin.ops.queryMode.preagg') }
])

const groupOptions = computed(() => {
  const filtered = props.platform ? groups.value.filter((g) => g.platform === props.platform) : groups.value
  return [{ value: null, label: t('common.all') }, ...filtered.map((g) => ({ value: g.id, label: g.name }))]
})

watch(
  () => props.platform,
  (newPlatform) => {
    if (!newPlatform) return
    const currentGroup = groups.value.find((g) => g.id === props.groupId)
    if (currentGroup && currentGroup.platform !== newPlatform) {
      emit('update:group', null)
    }
  }
)

onMounted(async () => {
  try {
    const list = await adminAPI.groups.getAll()
    groups.value = list.map((g) => ({ id: g.id, name: g.name, platform: g.platform }))
  } catch (e) {
    console.error('[OpsDashboardHeader] Failed to load groups', e)
    groups.value = []
  }
})

function handlePlatformChange(val: string | number | boolean | null) {
  emit('update:platform', String(val || ''))
}

function handleGroupChange(val: string | number | boolean | null) {
  if (val === null || val === '' || typeof val === 'boolean') {
    emit('update:group', null)
    return
  }
  const id = typeof val === 'number' ? val : Number.parseInt(String(val), 10)
  emit('update:group', Number.isFinite(id) && id > 0 ? id : null)
}

function handleTimeRangeChange(val: string | number | boolean | null) {
  const newValue = String(val || '1h')
  if (newValue === 'custom') {
    // 初始化为最近1小时
    const now = new Date()
    const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000)
    customStartTimeInput.value = oneHourAgo.toISOString().slice(0, 16)
    customEndTimeInput.value = now.toISOString().slice(0, 16)
    showCustomTimeRangeDialog.value = true
  } else {
    emit('update:timeRange', newValue)
  }
}

function handleCustomTimeRangeConfirm() {
  if (!customStartTimeInput.value || !customEndTimeInput.value) return
  const startTime = new Date(customStartTimeInput.value).toISOString()
  const endTime = new Date(customEndTimeInput.value).toISOString()
  // Emit custom time range first so the parent can build correct API params
  // when it reacts to timeRange switching to "custom".
  emit('update:customTimeRange', startTime, endTime)
  emit('update:timeRange', 'custom')
  showCustomTimeRangeDialog.value = false
}

function handleCustomTimeRangeCancel() {
  showCustomTimeRangeDialog.value = false
  // 如果当前不是 custom，不需要做任何事
  // 如果当前是 custom，保持不变
}

function handleQueryModeChange(val: string | number | boolean | null) {
  emit('update:queryMode', String(val || 'auto'))
}

function openDetails(preset?: OpsRequestDetailsPreset) {
  emit('openRequestDetails', preset)
}

function openErrorDetails(kind: 'request' | 'upstream') {
  emit('openErrorDetails', kind)
}

// --- Threshold checking helpers ---
type ThresholdLevel = 'normal' | 'warning' | 'critical'

function getSLAThresholdLevel(slaPercent: number | null): ThresholdLevel {
  if (slaPercent == null) return 'normal'
  const threshold = props.thresholds?.sla_percent_min
  if (threshold == null) return 'normal'

  // SLA is "higher is better":
  // - below threshold => critical
  // - within +0.1% buffer => warning
  const warningBuffer = 0.1

  if (slaPercent < threshold) return 'critical'
  if (slaPercent < threshold + warningBuffer) return 'warning'
  return 'normal'
}

function getTTFTThresholdLevel(ttftMs: number | null): ThresholdLevel {
  if (ttftMs == null) return 'normal'
  const threshold = props.thresholds?.ttft_p99_ms_max
  if (threshold == null) return 'normal'
  if (ttftMs >= threshold) return 'critical'
  if (ttftMs >= threshold * 0.8) return 'warning'
  return 'normal'
}

function getRequestErrorRateThresholdLevel(errorRatePercent: number | null): ThresholdLevel {
  if (errorRatePercent == null) return 'normal'
  const threshold = props.thresholds?.request_error_rate_percent_max
  if (threshold == null) return 'normal'
  if (errorRatePercent >= threshold) return 'critical'
  if (errorRatePercent >= threshold * 0.8) return 'warning'
  return 'normal'
}

function getUpstreamErrorRateThresholdLevel(upstreamErrorRatePercent: number | null): ThresholdLevel {
  if (upstreamErrorRatePercent == null) return 'normal'
  const threshold = props.thresholds?.upstream_error_rate_percent_max
  if (threshold == null) return 'normal'
  if (upstreamErrorRatePercent >= threshold) return 'critical'
  if (upstreamErrorRatePercent >= threshold * 0.8) return 'warning'
  return 'normal'
}

function getThresholdColorClass(level: ThresholdLevel): string {
  switch (level) {
    case 'critical':
      return 'metric-tone-critical'
    case 'warning':
      return 'metric-tone-warning'
    default:
      return 'metric-tone-normal'
  }
}

// --- Realtime / Overview labels ---

const totalRequestsLabel = computed(() => formatNumber(overview.value?.request_count_total ?? 0))
const totalTokensLabel = computed(() => formatNumber(overview.value?.token_consumed ?? 0))

const realtimeTrafficSummary = ref<OpsRealtimeTrafficSummary | null>(null)
const realtimeTrafficLoading = ref(false)

function makeZeroRealtimeTrafficSummary(): OpsRealtimeTrafficSummary {
  const now = new Date().toISOString()
  return {
    window: realtimeWindow.value,
    start_time: now,
    end_time: now,
    platform: props.platform,
    group_id: props.groupId,
    qps: { current: 0, peak: 0, avg: 0 },
    tps: { current: 0, peak: 0, avg: 0 }
  }
}

async function loadRealtimeTrafficSummary() {
  if (props.workspace && props.workspace !== 'traffic') {
    realtimeTrafficSummary.value = null
    return
  }
  if (realtimeTrafficLoading.value) return
  if (!adminSettingsStore.opsRealtimeMonitoringEnabled) {
    realtimeTrafficSummary.value = makeZeroRealtimeTrafficSummary()
    return
  }
  realtimeTrafficLoading.value = true
  try {
    const res = await opsAPI.getRealtimeTrafficSummary(realtimeWindow.value, props.platform, props.groupId)
    if (res && res.enabled === false) {
      adminSettingsStore.setOpsRealtimeMonitoringEnabledLocal(false)
    }
    realtimeTrafficSummary.value = res?.summary ?? null
  } catch (err) {
    console.error('[OpsDashboardHeader] Failed to load realtime traffic summary', err)
    realtimeTrafficSummary.value = null
  } finally {
    realtimeTrafficLoading.value = false
  }
}

watch(
  () => [realtimeWindow.value, props.platform, props.groupId, props.workspace] as const,
  () => {
    loadRealtimeTrafficSummary()
  },
  { immediate: true }
)

watch(
  () => adminSettingsStore.opsRealtimeMonitoringEnabled,
  (enabled) => {
    if (!enabled) {
      // Keep UI stable when realtime monitoring is turned off.
      realtimeTrafficSummary.value = makeZeroRealtimeTrafficSummary()
    } else {
      loadRealtimeTrafficSummary()
    }
  },
  { immediate: true }
)

// Realtime traffic refresh follows the parent (OpsDashboard) refresh cadence.
watch(
  () => [props.autoRefreshEnabled, props.autoRefreshCountdown, props.loading] as const,
  ([enabled, countdown, loading]) => {
    if (!enabled) return
    if (loading) return
    // Treat countdown reset (or reaching 0) as a refresh boundary.
    if (countdown === 0) {
      loadRealtimeTrafficSummary()
    }
  }
)

// no-op: parent controls refresh cadence

const displayRealTimeQps = computed(() => {
  const v = realtimeTrafficSummary.value?.qps?.current
  return typeof v === 'number' && Number.isFinite(v) ? v : 0
})

const displayRealTimeTps = computed(() => {
  const v = realtimeTrafficSummary.value?.tps?.current
  return typeof v === 'number' && Number.isFinite(v) ? v : 0
})

const realtimeQpsPeakLabel = computed(() => {
  const v = realtimeTrafficSummary.value?.qps?.peak
  return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
})
const realtimeTpsPeakLabel = computed(() => {
  const v = realtimeTrafficSummary.value?.tps?.peak
  return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
})
const realtimeQpsAvgLabel = computed(() => {
  const v = realtimeTrafficSummary.value?.qps?.avg
  return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
})
const realtimeTpsAvgLabel = computed(() => {
  const v = realtimeTrafficSummary.value?.tps?.avg
  return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
})

const qpsAvgLabel = computed(() => {
  const v = overview.value?.qps?.avg
  if (typeof v !== 'number') return '-'
  return v.toFixed(1)
})

const tpsAvgLabel = computed(() => {
  const v = overview.value?.tps?.avg
  if (typeof v !== 'number') return '-'
  return v.toFixed(1)
})

const slaPercent = computed(() => {
  if ((overview.value?.request_count_sla ?? 0) <= 0) return null
  const v = overview.value?.sla
  if (typeof v !== 'number') return null
  return v * 100
})

const errorRatePercent = computed(() => {
  if ((overview.value?.request_count_sla ?? 0) <= 0) return null
  const v = overview.value?.error_rate
  if (typeof v !== 'number') return null
  return v * 100
})

const upstreamErrorRatePercent = computed(() => {
  if ((overview.value?.request_count_total ?? 0) <= 0) return null
  const v = overview.value?.upstream_error_rate
  if (typeof v !== 'number') return null
  return v * 100
})

function getMetricToneClass(value: number | null, level: ThresholdLevel): string {
  if (value == null) return 'metric-tone-neutral'
  return getThresholdColorClass(level)
}

function getMetricFillClass(value: number | null, level: ThresholdLevel): string {
  if (value == null) return 'metric-fill-neutral'
  if (level === 'critical') return 'metric-fill-critical'
  if (level === 'warning') return 'metric-fill-warning'
  return 'metric-fill-normal'
}

const slaMetricToneClass = computed(() => getMetricToneClass(slaPercent.value, getSLAThresholdLevel(slaPercent.value)))
const slaMetricFillClass = computed(() => getMetricFillClass(slaPercent.value, getSLAThresholdLevel(slaPercent.value)))
const requestErrorMetricToneClass = computed(() =>
  getMetricToneClass(errorRatePercent.value, getRequestErrorRateThresholdLevel(errorRatePercent.value))
)
const upstreamErrorMetricToneClass = computed(() =>
  getMetricToneClass(upstreamErrorRatePercent.value, getUpstreamErrorRateThresholdLevel(upstreamErrorRatePercent.value))
)

const durationP99Ms = computed(() => overview.value?.duration?.p99_ms ?? null)
const durationP95Ms = computed(() => overview.value?.duration?.p95_ms ?? null)
const durationP90Ms = computed(() => overview.value?.duration?.p90_ms ?? null)
const durationP50Ms = computed(() => overview.value?.duration?.p50_ms ?? null)
const durationAvgMs = computed(() => overview.value?.duration?.avg_ms ?? null)
const durationMaxMs = computed(() => overview.value?.duration?.max_ms ?? null)

const ttftP99Ms = computed(() => overview.value?.ttft?.p99_ms ?? null)
const ttftP95Ms = computed(() => overview.value?.ttft?.p95_ms ?? null)
const ttftP90Ms = computed(() => overview.value?.ttft?.p90_ms ?? null)
const ttftP50Ms = computed(() => overview.value?.ttft?.p50_ms ?? null)
const ttftAvgMs = computed(() => overview.value?.ttft?.avg_ms ?? null)
const ttftMaxMs = computed(() => overview.value?.ttft?.max_ms ?? null)

// --- Health Score & Diagnosis (primary) ---

const isSystemIdle = computed(() => {
  const ov = overview.value
  if (!ov) return true
  const qps = ov.qps?.current
  const errorRate = ov.error_rate ?? 0
  return (qps ?? 0) === 0 && errorRate === 0
})

const healthScoreValue = computed<number | null>(() => {
  const v = overview.value?.health_score
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const healthScoreColor = computed(() => {
  if (isSystemIdle.value) return 'var(--lx-clay-text-subtle)'
  const score = healthScoreValue.value
  if (score == null) return 'var(--lx-clay-text-subtle)'
  if (score >= 90) return 'var(--lx-clay-success-bright)'
  if (score >= 60) return 'var(--lx-clay-warning-bright)'
  return 'var(--lx-clay-danger)'
})

const healthScoreClass = computed(() => {
  if (isSystemIdle.value) return 'metric-tone-neutral'
  const score = healthScoreValue.value
  if (score == null) return 'metric-tone-neutral'
  if (score >= 90) return 'metric-tone-normal'
  if (score >= 60) return 'metric-tone-warning'
  return 'metric-tone-critical'
})

const circleSize = computed(() => props.fullscreen ? 64 : 56)
const strokeWidth = computed(() => props.fullscreen ? 6 : 5)
const radius = computed(() => (circleSize.value - strokeWidth.value) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const dashOffset = computed(() => {
  if (isSystemIdle.value) return 0
  if (healthScoreValue.value == null) return 0
  const score = Math.max(0, Math.min(100, healthScoreValue.value))
  return circumference.value - (score / 100) * circumference.value
})

interface DiagnosisItem {
  type: 'critical' | 'warning' | 'info'
  message: string
  impact: string
  action?: string
}

const diagnosisReport = computed<DiagnosisItem[]>(() => {
  const ov = overview.value
  if (!ov) return []

  const report: DiagnosisItem[] = []

  if (isSystemIdle.value) {
    report.push({
      type: 'info',
      message: t('admin.ops.diagnosis.idle'),
      impact: t('admin.ops.diagnosis.idleImpact')
    })
    return report
  }

  // Resource diagnostics (highest priority)
  const sm = ov.system_metrics
  if (sm) {
    if (sm.db_ok === false) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.dbDown'),
        impact: t('admin.ops.diagnosis.dbDownImpact'),
        action: t('admin.ops.diagnosis.dbDownAction')
      })
    }
    if (sm.redis_ok === false) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.redisDown'),
        impact: t('admin.ops.diagnosis.redisDownImpact'),
        action: t('admin.ops.diagnosis.redisDownAction')
      })
    }

    const cpuPct = sm.cpu_usage_percent ?? 0
    if (cpuPct > 90) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.cpuCritical', { usage: cpuPct.toFixed(1) }),
        impact: t('admin.ops.diagnosis.cpuCriticalImpact'),
        action: t('admin.ops.diagnosis.cpuCriticalAction')
      })
    } else if (cpuPct > 80) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.cpuHigh', { usage: cpuPct.toFixed(1) }),
        impact: t('admin.ops.diagnosis.cpuHighImpact'),
        action: t('admin.ops.diagnosis.cpuHighAction')
      })
    }

    const memPct = sm.memory_usage_percent ?? 0
    if (memPct > 90) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.memoryCritical', { usage: memPct.toFixed(1) }),
        impact: t('admin.ops.diagnosis.memoryCriticalImpact'),
        action: t('admin.ops.diagnosis.memoryCriticalAction')
      })
    } else if (memPct > 85) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.memoryHigh', { usage: memPct.toFixed(1) }),
        impact: t('admin.ops.diagnosis.memoryHighImpact'),
        action: t('admin.ops.diagnosis.memoryHighAction')
      })
    }
  }

  const ttftP99 = ov.ttft?.p99_ms ?? 0
  if (ttftP99 > 500) {
    report.push({
      type: 'warning',
      message: t('admin.ops.diagnosis.ttftHigh', { ttft: ttftP99.toFixed(0) }),
      impact: t('admin.ops.diagnosis.ttftHighImpact'),
      action: t('admin.ops.diagnosis.ttftHighAction')
    })
  }

  // Error rate diagnostics (adjusted thresholds)
  const upstreamRatePct = (ov.upstream_error_rate ?? 0) * 100
  if (upstreamRatePct > 5) {
    report.push({
      type: 'critical',
      message: t('admin.ops.diagnosis.upstreamCritical', { rate: upstreamRatePct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.upstreamCriticalImpact'),
      action: t('admin.ops.diagnosis.upstreamCriticalAction')
    })
  } else if (upstreamRatePct > 2) {
    report.push({
      type: 'warning',
      message: t('admin.ops.diagnosis.upstreamHigh', { rate: upstreamRatePct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.upstreamHighImpact'),
      action: t('admin.ops.diagnosis.upstreamHighAction')
    })
  }

  const errorPct = (ov.error_rate ?? 0) * 100
  if (errorPct > 3) {
    report.push({
      type: 'critical',
      message: t('admin.ops.diagnosis.errorHigh', { rate: errorPct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.errorHighImpact'),
      action: t('admin.ops.diagnosis.errorHighAction')
    })
  } else if (errorPct > 0.5) {
    report.push({
      type: 'warning',
      message: t('admin.ops.diagnosis.errorElevated', { rate: errorPct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.errorElevatedImpact'),
      action: t('admin.ops.diagnosis.errorElevatedAction')
    })
  }

  // SLA diagnostics
  const slaPct = (ov.sla ?? 0) * 100
  if (slaPct < 90) {
    report.push({
      type: 'critical',
      message: t('admin.ops.diagnosis.slaCritical', { sla: slaPct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.slaCriticalImpact'),
      action: t('admin.ops.diagnosis.slaCriticalAction')
    })
  } else if (slaPct < 98) {
    report.push({
      type: 'warning',
      message: t('admin.ops.diagnosis.slaLow', { sla: slaPct.toFixed(2) }),
      impact: t('admin.ops.diagnosis.slaLowImpact'),
      action: t('admin.ops.diagnosis.slaLowAction')
    })
  }

  // Health score diagnostics (lowest priority)
  if (healthScoreValue.value != null) {
    if (healthScoreValue.value < 60) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.healthCritical', { score: healthScoreValue.value }),
        impact: t('admin.ops.diagnosis.healthCriticalImpact'),
        action: t('admin.ops.diagnosis.healthCriticalAction')
      })
    } else if (healthScoreValue.value < 90) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.healthLow', { score: healthScoreValue.value }),
        impact: t('admin.ops.diagnosis.healthLowImpact'),
        action: t('admin.ops.diagnosis.healthLowAction')
      })
    }
  }

  if (report.length === 0) {
    report.push({
      type: 'info',
      message: t('admin.ops.diagnosis.healthy'),
      impact: t('admin.ops.diagnosis.healthyImpact')
    })
  }

  return report
})

// --- System health (secondary) ---

function formatTimeShort(ts?: string | null): string {
  if (!ts) return '-'
  const d = new Date(ts)
  if (Number.isNaN(d.getTime())) return '-'
  return d.toLocaleTimeString()
}

const cpuPercentValue = computed<number | null>(() => {
  const v = systemMetrics.value?.cpu_usage_percent
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const cpuPercentClass = computed(() => {
  const v = cpuPercentValue.value
  if (v == null) return 'metric-tone-neutral'
  if (v >= 95) return 'metric-tone-critical'
  if (v >= 80) return 'metric-tone-warning'
  return 'metric-tone-normal'
})

const memPercentValue = computed<number | null>(() => {
  const v = systemMetrics.value?.memory_usage_percent
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const memPercentClass = computed(() => {
  const v = memPercentValue.value
  if (v == null) return 'metric-tone-neutral'
  if (v >= 95) return 'metric-tone-critical'
  if (v >= 85) return 'metric-tone-warning'
  return 'metric-tone-normal'
})

const dbConnActiveValue = computed<number | null>(() => {
  const v = systemMetrics.value?.db_conn_active
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const dbConnIdleValue = computed<number | null>(() => {
  const v = systemMetrics.value?.db_conn_idle
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const dbConnWaitingValue = computed<number | null>(() => {
  const v = systemMetrics.value?.db_conn_waiting
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const dbConnOpenValue = computed<number | null>(() => {
  if (dbConnActiveValue.value == null || dbConnIdleValue.value == null) return null
  return dbConnActiveValue.value + dbConnIdleValue.value
})

const dbMaxOpenConnsValue = computed<number | null>(() => {
  const v = systemMetrics.value?.db_max_open_conns
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const dbUsagePercent = computed<number | null>(() => {
  if (dbConnOpenValue.value == null || dbMaxOpenConnsValue.value == null || dbMaxOpenConnsValue.value <= 0) return null
  return Math.min(100, Math.max(0, (dbConnOpenValue.value / dbMaxOpenConnsValue.value) * 100))
})

const dbMiddleLabel = computed(() => {
  if (systemMetrics.value?.db_ok === false) return 'FAIL'
  if (dbUsagePercent.value != null) return `${dbUsagePercent.value.toFixed(0)}%`
  if (systemMetrics.value?.db_ok === true) return t('admin.ops.ok')
  return t('admin.ops.noData')
})

const dbMiddleClass = computed(() => {
  if (systemMetrics.value?.db_ok === false) return 'metric-tone-critical'
  if (dbUsagePercent.value != null) {
    if (dbUsagePercent.value >= 90) return 'metric-tone-critical'
    if (dbUsagePercent.value >= 70) return 'metric-tone-warning'
    return 'metric-tone-normal'
  }
  if (systemMetrics.value?.db_ok === true) return 'metric-tone-normal'
  return 'metric-tone-neutral'
})

const redisConnTotalValue = computed<number | null>(() => {
  const v = systemMetrics.value?.redis_conn_total
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const redisConnIdleValue = computed<number | null>(() => {
  const v = systemMetrics.value?.redis_conn_idle
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const redisConnActiveValue = computed<number | null>(() => {
  if (redisConnTotalValue.value == null || redisConnIdleValue.value == null) return null
  return Math.max(redisConnTotalValue.value - redisConnIdleValue.value, 0)
})

const redisPoolSizeValue = computed<number | null>(() => {
  const v = systemMetrics.value?.redis_pool_size
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const redisUsagePercent = computed<number | null>(() => {
  if (redisConnTotalValue.value == null || redisPoolSizeValue.value == null || redisPoolSizeValue.value <= 0) return null
  return Math.min(100, Math.max(0, (redisConnTotalValue.value / redisPoolSizeValue.value) * 100))
})

const redisMiddleLabel = computed(() => {
  if (systemMetrics.value?.redis_ok === false) return 'FAIL'
  if (redisUsagePercent.value != null) return `${redisUsagePercent.value.toFixed(0)}%`
  if (systemMetrics.value?.redis_ok === true) return t('admin.ops.ok')
  return t('admin.ops.noData')
})

const redisMiddleClass = computed(() => {
  if (systemMetrics.value?.redis_ok === false) return 'metric-tone-critical'
  if (redisUsagePercent.value != null) {
    if (redisUsagePercent.value >= 90) return 'metric-tone-critical'
    if (redisUsagePercent.value >= 70) return 'metric-tone-warning'
    return 'metric-tone-normal'
  }
  if (systemMetrics.value?.redis_ok === true) return 'metric-tone-normal'
  return 'metric-tone-neutral'
})

const goroutineCountValue = computed<number | null>(() => {
  const v = systemMetrics.value?.goroutine_count
  return typeof v === 'number' && Number.isFinite(v) ? v : null
})

const goroutinesWarnThreshold = 8_000
const goroutinesCriticalThreshold = 15_000

const goroutineStatus = computed<'ok' | 'warning' | 'critical' | 'unknown'>(() => {
  const n = goroutineCountValue.value
  if (n == null) return 'unknown'
  if (n >= goroutinesCriticalThreshold) return 'critical'
  if (n >= goroutinesWarnThreshold) return 'warning'
  return 'ok'
})

const goroutineStatusLabel = computed(() => {
  switch (goroutineStatus.value) {
    case 'ok':
      return t('admin.ops.ok')
    case 'warning':
      return t('common.warning')
    case 'critical':
      return t('common.critical')
    default:
      return t('admin.ops.noData')
  }
})

const goroutineStatusClass = computed(() => {
  switch (goroutineStatus.value) {
    case 'ok':
      return 'metric-tone-normal'
    case 'warning':
      return 'metric-tone-warning'
    case 'critical':
      return 'metric-tone-critical'
    default:
      return 'metric-tone-neutral'
  }
})

const jobHeartbeats = computed(() => overview.value?.job_heartbeats ?? [])

const jobsStatus = computed<'ok' | 'warn' | 'unknown'>(() => {
  const list = jobHeartbeats.value
  if (!list.length) return 'unknown'
  for (const hb of list) {
    if (!hb) continue
    if (hb.last_error_at && (!hb.last_success_at || hb.last_error_at > hb.last_success_at)) return 'warn'
  }
  return 'ok'
})

const jobsWarnCount = computed(() => {
  let warn = 0
  for (const hb of jobHeartbeats.value) {
    if (!hb) continue
    if (hb.last_error_at && (!hb.last_success_at || hb.last_error_at > hb.last_success_at)) warn++
  }
  return warn
})

const jobsStatusLabel = computed(() => {
  switch (jobsStatus.value) {
    case 'ok':
      return t('admin.ops.ok')
    case 'warn':
      return t('common.warning')
    default:
      return t('admin.ops.noData')
  }
})

const jobsStatusClass = computed(() => {
  switch (jobsStatus.value) {
    case 'ok':
      return 'metric-tone-normal'
    case 'warn':
      return 'metric-tone-warning'
    default:
      return 'metric-tone-neutral'
  }
})

const showJobsDetails = ref(false)
const showDiagnosticDetails = ref(false)

function openJobsDetails() {
  showJobsDetails.value = true
}

function handleToolbarRefresh() {
  loadRealtimeTrafficSummary()
  emit('refresh')
}

type OperationalStatusTone = 'healthy' | 'warning' | 'critical' | 'idle' | 'loading'

const operationalStatus = computed<{ tone: OperationalStatusTone; label: string }>(() => {
  if (props.loading) return { tone: 'loading', label: t('admin.ops.loadingText') }
  if (isSystemIdle.value) return { tone: 'idle', label: t('admin.ops.idleStatus') }
  if (diagnosisReport.value.some((item) => item.type === 'critical')) {
    return { tone: 'critical', label: t('common.critical') }
  }
  if (diagnosisReport.value.some((item) => item.type === 'warning')) {
    return { tone: 'warning', label: t('common.warning') }
  }
  return { tone: 'healthy', label: t('admin.ops.ready') }
})

const lastUpdatedLabel = computed(() => {
  if (!props.lastUpdated) return t('common.unknown')
  return props.lastUpdated
    .toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
    .replace(/\//g, '-')
})

const concurrencyQueueDepth = computed<number | null>(() => {
  const value = systemMetrics.value?.concurrency_queue_depth
  return typeof value === 'number' && Number.isFinite(value) ? value : null
})

const concurrencyQueueTone = computed(() => {
  if (concurrencyQueueDepth.value == null) return 'metric-tone-neutral'
  return concurrencyQueueDepth.value > 0 ? 'metric-tone-warning' : 'metric-tone-normal'
})

function blurDiagnosisTrigger(event: KeyboardEvent) {
  (event.currentTarget as HTMLElement).blur()
}
</script>

<template>
  <div
    class="ops-command-center"
    :class="{
      'ops-command-center--fullscreen': props.fullscreen,
      'ops-command-center--resources': props.workspace === 'resources'
    }"
    data-testid="ops-dashboard-header"
  >
    <header class="ops-masthead">
      <div class="ops-masthead__identity">
        <span class="ops-masthead__icon" aria-hidden="true">
          <Icon name="activity" size="lg" :stroke-width="2" />
        </span>
        <div class="ops-masthead__copy">
          <div class="ops-masthead__title-row">
            <h1>{{ t('admin.ops.title') }}</h1>
            <span class="ops-status-badge" :class="`ops-status-badge--${operationalStatus.tone}`">
              <i aria-hidden="true"></i>
              {{ operationalStatus.label }}
            </span>
          </div>
          <p v-if="!props.fullscreen && props.workspace !== 'resources'">{{ t('admin.ops.description') }}</p>
          <div
            v-if="!props.fullscreen"
            class="ops-status-line"
            data-testid="ops-status-line"
            role="status"
            aria-live="polite"
            aria-atomic="true"
          >
            <span>{{ t('common.refresh') }}: {{ lastUpdatedLabel }}</span>
            <span
              v-if="props.autoRefreshEnabled && props.autoRefreshCountdown !== undefined"
              class="ops-status-line__countdown"
            >
              {{ t('admin.ops.autoRefreshRemaining', { seconds: props.autoRefreshCountdown }) }}
            </span>
          </div>
        </div>
      </div>

      <button
        v-if="props.fullscreen"
        type="button"
        class="ops-icon-button"
        :title="t('common.close')"
        :aria-label="t('common.close')"
        @click="emit('exitFullscreen')"
      >
        <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 9H4m0 0V4m0 5 5-5m6 5h5m0 0V4m0 5-5-5M9 15H4m0 0v5m0-5 5 5m6-5h5m0 0v5m0-5-5 5" />
        </svg>
      </button>
    </header>

    <div v-if="!props.fullscreen" class="ops-command-bar" data-testid="ops-command-bar">
      <div class="ops-command-bar__label">
        <Icon name="filter" size="sm" aria-hidden="true" />
        <span>{{ t('common.filter') }}</span>
      </div>

      <div class="ops-command-bar__controls">
        <template v-if="props.workspace !== 'resources' || props.resource !== 'proxies'">
          <Select
            :model-value="platform"
            :options="platformOptions"
            :aria-label="t('admin.ops.errorLog.platform')"
            class="ops-filter-control ops-filter-control--platform"
            @update:model-value="handlePlatformChange"
          />

          <Select
            :model-value="groupId"
            :options="groupOptions"
            :aria-label="t('admin.ops.errorLog.group')"
            class="ops-filter-control ops-filter-control--group"
            @update:model-value="handleGroupChange"
          />
        </template>

        <Select
          v-if="!props.workspace || props.workspace !== 'resources'"
          :model-value="timeRange"
          :options="timeRangeOptions"
          :aria-label="t('admin.ops.systemLogs.timeRange')"
          class="ops-filter-control ops-filter-control--range"
          @update:model-value="handleTimeRangeChange"
        />

        <Select
          v-if="false"
          :model-value="queryMode"
          :options="queryModeOptions"
          class="ops-filter-control"
          @update:model-value="handleQueryModeChange"
        />

        <button
          type="button"
          class="ops-command-button ops-command-button--primary"
          :disabled="loading"
          @click="handleToolbarRefresh"
        >
          <Icon name="refresh" size="sm" :class="{ 'ops-spin': loading }" aria-hidden="true" />
          <span>{{ t('common.refresh') }}</span>
        </button>

        <button
          v-if="!props.workspace || props.workspace === 'incidents'"
          type="button"
          class="ops-command-button"
          :title="t('admin.ops.alertRules.title')"
          @click="emit('openAlertRules')"
        >
          <Icon name="bell" size="sm" aria-hidden="true" />
          <span>{{ t('admin.ops.alertRules.manage') }}</span>
        </button>

        <button
          type="button"
          class="ops-icon-button"
          :title="t('admin.ops.settings.title')"
          :aria-label="t('admin.ops.settings.title')"
          @click="emit('openSettings')"
        >
          <Icon name="cog" size="sm" aria-hidden="true" />
        </button>

        <button
          type="button"
          class="ops-icon-button"
          :title="t('admin.ops.fullscreen.enter')"
          :aria-label="t('admin.ops.fullscreen.enter')"
          @click="emit('enterFullscreen')"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0-5 5M4 16v4m0 0h4m-4 0 5-5m11 5-5-5m5 5v-4m0 4h-4" />
          </svg>
        </button>
      </div>
    </div>

    <section
      v-if="overview"
      class="ops-signal-strip"
      :aria-label="t('admin.ops.overview')"
      data-testid="ops-signal-strip"
    >
      <article class="ops-signal">
        <div class="ops-signal__heading">
          <span>{{ t('admin.ops.health') }}</span>
          <i class="ops-signal__dot" :class="healthScoreClass" aria-hidden="true"></i>
        </div>
        <strong :class="healthScoreClass">
          {{ isSystemIdle ? t('admin.ops.idleStatus') : (healthScoreValue ?? '-') }}
        </strong>
        <small>{{ operationalStatus.label }}</small>
      </article>

      <button
        type="button"
        class="ops-signal ops-signal--action"
        data-testid="ops-signal-sla"
        @click="openDetails({ title: t('admin.ops.requestDetails.title'), kind: 'error' })"
      >
        <span class="ops-signal__heading">
          <span>{{ t('admin.ops.sla') }}</span>
          <i
            class="ops-signal__dot"
            :class="slaMetricToneClass"
            aria-hidden="true"
          ></i>
        </span>
        <strong :class="slaMetricToneClass">
          {{ slaPercent == null ? '-' : `${slaPercent.toFixed(3)}%` }}
        </strong>
        <small>{{ t('admin.ops.exceptions') }} {{ formatNumber((overview.request_count_sla ?? 0) - (overview.success_count ?? 0)) }}</small>
      </button>

      <button
        type="button"
        class="ops-signal ops-signal--action"
        data-testid="ops-signal-request-errors"
        @click="openErrorDetails('request')"
      >
        <span class="ops-signal__heading">
          <span>{{ t('admin.ops.requestErrors') }}</span>
          <i
            class="ops-signal__dot"
            :class="requestErrorMetricToneClass"
            aria-hidden="true"
          ></i>
        </span>
        <strong :class="requestErrorMetricToneClass">
          {{ errorRatePercent == null ? '-' : `${errorRatePercent.toFixed(2)}%` }}
        </strong>
        <small>{{ t('admin.ops.errorCount') }} {{ formatNumber(overview.error_count_sla ?? 0) }}</small>
      </button>

      <article class="ops-signal">
        <div class="ops-signal__heading">
          <span>{{ t('admin.ops.realtime.title') }}</span>
          <i class="ops-signal__dot metric-tone-normal" aria-hidden="true"></i>
        </div>
        <strong>{{ displayRealTimeQps.toFixed(1) }} <em>QPS</em></strong>
        <small>{{ displayRealTimeTps.toFixed(1) }} {{ t('admin.ops.tps') }}</small>
      </article>

      <article class="ops-signal">
        <div class="ops-signal__heading">
          <span>{{ t('admin.ops.concurrency.title') }}</span>
          <i class="ops-signal__dot" :class="concurrencyQueueTone" aria-hidden="true"></i>
        </div>
        <strong :class="concurrencyQueueTone">{{ concurrencyQueueDepth ?? '-' }}</strong>
        <small>{{ t('admin.ops.queue') }}</small>
      </article>
    </section>

    <button
      v-if="overview"
      type="button"
      class="ops-diagnostic-toggle"
      :aria-expanded="showDiagnosticDetails"
      aria-controls="ops-diagnostic-details"
      data-testid="ops-diagnostic-toggle"
      @click="showDiagnosticDetails = !showDiagnosticDetails"
    >
      <span>
        <Icon name="brain" size="sm" aria-hidden="true" />
        <strong>{{ t('admin.ops.diagnosis.title') }}</strong>
        <small>{{ t('admin.ops.requestDetails.details') }}</small>
      </span>
      <Icon
        name="chevronDown"
        size="sm"
        class="ops-diagnostic-toggle__chevron"
        :class="{ 'ops-diagnostic-toggle__chevron--open': showDiagnosticDetails }"
        aria-hidden="true"
      />
    </button>

    <div
      v-show="showDiagnosticDetails"
      id="ops-diagnostic-details"
      class="ops-diagnostic-details"
      data-testid="ops-diagnostic-details"
    >
    <div v-if="overview" class="ops-overview-grid grid grid-cols-1 gap-6 lg:grid-cols-12">
      <!-- Left: Health + Realtime -->
      <div :class="['ops-health-panel lg:col-span-5', props.fullscreen ? 'p-6' : 'p-4']">
        <div class="grid h-full grid-cols-1 gap-6 md:grid-cols-[200px_1fr] md:items-center">
          <!-- 1) Health Score -->
          <div
            class="ops-diagnosis group relative flex flex-col items-center justify-center rounded-xl py-2 transition-colors md:border-r md:pr-6"
          >
            <!-- Diagnosis Popover: available on hover, keyboard focus and touch focus. -->
            <div
              id="ops-diagnosis-popover"
              role="tooltip"
              class="pointer-events-none absolute left-1/2 top-full z-50 mt-2 w-72 -translate-x-1/2 opacity-0 transition-opacity duration-200 group-hover:pointer-events-auto group-hover:opacity-100 group-focus-within:pointer-events-auto group-focus-within:opacity-100 md:left-full md:top-0 md:ml-2 md:mt-0 md:translate-x-0"
            >
              <div class="rounded-xl bg-white p-4 shadow-xl ring-1 ring-black/5 dark:bg-gray-800 dark:ring-white/10">
                <h4 class="mb-3 border-b border-gray-100 pb-2 text-sm font-bold text-gray-900 dark:border-gray-700 dark:text-white flex items-center gap-2">
                  <Icon name="brain" size="sm" class="ops-diagnosis-info" />
                  {{ t('admin.ops.diagnosis.title') }}
                </h4>

                <div class="space-y-3">
                  <div v-for="(item, idx) in diagnosisReport" :key="idx" class="flex gap-3">
                    <div class="mt-0.5 shrink-0">
                      <svg v-if="item.type === 'critical'" class="h-4 w-4 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fill-rule="evenodd"
                          d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                          clip-rule="evenodd"
                        />
                      </svg>
                      <svg v-else-if="item.type === 'warning'" class="h-4 w-4 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fill-rule="evenodd"
                          d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                          clip-rule="evenodd"
                        />
                      </svg>
                      <svg v-else class="ops-diagnosis-info h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                        <path
                          fill-rule="evenodd"
                          d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-8-3a1 1 0 100 2 1 1 0 000-2zm-1 3a1 1 0 012 0v4a1 1 0 11-2 0v-4z"
                          clip-rule="evenodd"
                        />
                      </svg>
                    </div>
                    <div class="flex-1">
                      <div class="text-xs font-semibold text-gray-900 dark:text-white">{{ item.message }}</div>
                      <div class="mt-0.5 text-[11px] text-gray-500 dark:text-gray-400">{{ item.impact }}</div>
                      <div v-if="item.action" class="ops-diagnosis-info mt-1 flex items-center gap-1 text-[11px]">
                        <Icon name="lightbulb" size="xs" />
                        {{ item.action }}
                      </div>
                    </div>
                  </div>
                </div>

                <div class="mt-3 border-t border-gray-100 pt-2 text-[10px] text-gray-400 dark:border-gray-700">
                  {{ t('admin.ops.diagnosis.footer') }}
                </div>
              </div>
            </div>

            <button
              type="button"
              class="relative flex items-center justify-center rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-4 focus-visible:ring-offset-gray-50 dark:focus-visible:ring-offset-dark-900"
              :aria-label="t('admin.ops.diagnosis.title')"
              aria-describedby="ops-diagnosis-popover"
              @keydown.esc="blurDiagnosisTrigger"
            >
              <svg :width="circleSize" :height="circleSize" class="-rotate-90 transform">
                <circle
                  :cx="circleSize / 2"
                  :cy="circleSize / 2"
                  :r="radius"
                  :stroke-width="strokeWidth"
                  fill="transparent"
                  class="text-gray-200 dark:text-dark-700"
                  stroke="currentColor"
                />
                <circle
                  :cx="circleSize / 2"
                  :cy="circleSize / 2"
                  :r="radius"
                  :stroke-width="strokeWidth"
                  fill="transparent"
                  :stroke="healthScoreColor"
                  stroke-linecap="round"
                  :stroke-dasharray="circumference"
                  :stroke-dashoffset="dashOffset"
                  class="transition-all duration-1000 ease-out"
                />
              </svg>

              <div class="absolute flex flex-col items-center">
                <span :class="[props.fullscreen ? 'text-5xl' : 'text-3xl', 'font-black', healthScoreClass]">
                  {{ isSystemIdle ? t('admin.ops.idleStatus') : (overview.health_score ?? '--') }}
                </span>
                <span :class="[props.fullscreen ? 'text-xs' : 'text-[10px]', 'font-bold uppercase tracking-wider text-gray-400']">{{ t('admin.ops.health') }}</span>
              </div>
            </button>

            <div class="mt-4 text-center" v-if="!props.fullscreen">
              <div class="flex items-center justify-center gap-1 text-xs font-medium text-gray-500">
                {{ t('admin.ops.healthCondition') }}
                <HelpTooltip :content="t('admin.ops.healthHelp')" />
              </div>
              <div class="mt-1 text-xs font-bold" :class="healthScoreClass">
                {{
                  isSystemIdle
                    ? t('admin.ops.idleStatus')
                    : typeof overview.health_score === 'number' && overview.health_score >= 90
                      ? t('admin.ops.healthyStatus')
                      : t('admin.ops.riskyStatus')
                }}
              </div>
            </div>
          </div>

          <!-- 2) Realtime Traffic -->
          <div class="ops-realtime-panel flex h-full flex-col justify-center py-2">
            <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2">
                <div class="relative flex h-3 w-3 shrink-0">
                  <span class="ops-live-dot ops-live-dot--halo absolute inline-flex h-full w-full rounded-full"></span>
                  <span class="ops-live-dot relative inline-flex h-3 w-3 rounded-full"></span>
                </div>
                <h3 class="text-xs font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.realtime.title') }}</h3>
                <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.qps')" />
              </div>

              <!-- Time Window Selector -->
              <div class="flex flex-wrap gap-1">
                <button
                  v-for="window in availableRealtimeWindows"
                  :key="window"
                  type="button"
                  class="ops-window-chip"
                  :class="{ 'ops-window-chip--selected': realtimeWindow === window }"
                  :aria-pressed="realtimeWindow === window"
                  @click="realtimeWindow = window"
                >
                  {{ window }}
                </button>
              </div>
            </div>

            <div :class="props.fullscreen ? 'space-y-4' : 'space-y-3'">
              <!-- Row 1: Current -->
              <div>
                <div :class="[props.fullscreen ? 'text-xs' : 'text-[10px]', 'font-bold uppercase text-gray-400']">{{ t('admin.ops.current') }}</div>
                <div class="mt-1 flex flex-wrap items-baseline gap-x-4 gap-y-2">
                  <div class="flex items-baseline gap-1.5">
                    <span :class="[props.fullscreen ? 'text-4xl' : 'text-xl sm:text-2xl', 'font-black text-gray-900 dark:text-white']">{{ displayRealTimeQps.toFixed(1) }}</span>
                    <span :class="[props.fullscreen ? 'text-sm' : 'text-xs', 'font-bold text-gray-500']">QPS</span>
                  </div>
                  <div class="flex items-baseline gap-1.5">
                    <span :class="[props.fullscreen ? 'text-4xl' : 'text-xl sm:text-2xl', 'font-black text-gray-900 dark:text-white']">{{ displayRealTimeTps.toFixed(1) }}</span>
                    <span :class="[props.fullscreen ? 'text-sm' : 'text-xs', 'font-bold text-gray-500']">{{ t('admin.ops.tps') }}</span>
                  </div>
                </div>
              </div>

              <!-- Row 2: Peak + Average -->
              <div class="grid grid-cols-2 gap-3">
                <!-- Peak -->
                <div>
                  <div :class="[props.fullscreen ? 'text-xs' : 'text-[10px]', 'font-bold uppercase text-gray-400']">{{ t('admin.ops.peak') }}</div>
                  <div :class="[props.fullscreen ? 'text-base' : 'text-sm', 'mt-1 space-y-0.5 font-medium text-gray-600 dark:text-gray-400']">
                    <div class="flex items-baseline gap-1.5">
                      <span class="font-black text-gray-900 dark:text-white">{{ realtimeQpsPeakLabel }}</span>
                      <span class="text-xs">QPS</span>
                    </div>
                    <div class="flex items-baseline gap-1.5">
                      <span class="font-black text-gray-900 dark:text-white">{{ realtimeTpsPeakLabel }}</span>
                      <span class="text-xs">{{ t('admin.ops.tps') }}</span>
                    </div>
                  </div>
                </div>

                <!-- Average -->
                <div>
                  <div :class="[props.fullscreen ? 'text-xs' : 'text-[10px]', 'font-bold uppercase text-gray-400']">{{ t('admin.ops.average') }}</div>
                  <div :class="[props.fullscreen ? 'text-base' : 'text-sm', 'mt-1 space-y-0.5 font-medium text-gray-600 dark:text-gray-400']">
                    <div class="flex items-baseline gap-1.5">
                      <span class="font-black text-gray-900 dark:text-white">{{ realtimeQpsAvgLabel }}</span>
                      <span class="text-xs">QPS</span>
                    </div>
                    <div class="flex items-baseline gap-1.5">
                      <span class="font-black text-gray-900 dark:text-white">{{ realtimeTpsAvgLabel }}</span>
                      <span class="text-xs">{{ t('admin.ops.tps') }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div class="ops-heartbeat h-8 w-full overflow-hidden opacity-50" aria-hidden="true">
                <svg class="h-full w-full" viewBox="0 0 280 32" preserveAspectRatio="none">
                  <path
                    d="M0 16 Q 20 16, 40 16 T 80 16 T 120 10 T 160 22 T 200 16 T 240 16 T 280 16"
                    fill="none"
                    stroke="var(--lx-clay-success-bright)"
                    stroke-width="2"
                    vector-effect="non-scaling-stroke"
                  />
                </svg>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: 6 cards (3 cols x 2 rows) -->
      <div class="ops-detail-grid grid h-full grid-cols-1 content-center sm:grid-cols-2 lg:col-span-7 lg:grid-cols-3">
        <!-- Card 1: Requests -->
        <div class="ops-detail-card" style="order: 1;">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1">
              <span class="text-[10px] font-bold uppercase text-gray-400">{{ t('admin.ops.requestsTitle') }}</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.totalRequests')" />
            </div>
            <button
              v-if="!props.fullscreen"
              class="ops-detail-link text-[10px] font-bold"
              type="button"
              @click="openDetails({ title: t('admin.ops.requestDetails.title') })"
            >
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 space-y-2 text-xs">
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.requests') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ totalRequestsLabel }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.tokens') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ totalTokensLabel }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.avgQps') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ qpsAvgLabel }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.avgTps') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ tpsAvgLabel }}</span>
            </div>
          </div>
        </div>

        <!-- Card 2: SLA -->
        <div class="ops-detail-card" style="order: 2;" data-testid="ops-detail-sla">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <span class="text-[10px] font-bold uppercase text-gray-400">{{ t('admin.ops.sla') }}</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.sla')" />
              <span class="ops-threshold-dot" :class="slaMetricFillClass"></span>
            </div>
            <button
              v-if="!props.fullscreen"
              class="ops-detail-link text-[10px] font-bold"
              type="button"
              @click="openDetails({ title: t('admin.ops.requestDetails.title'), kind: 'error' })"
            >
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 text-3xl font-black" :class="slaMetricToneClass">
            {{ slaPercent == null ? '-' : `${slaPercent.toFixed(3)}%` }}
          </div>
          <div class="mt-3 h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
            <div class="h-full transition-all" :class="slaMetricFillClass" :style="{ width: `${Math.max((slaPercent ?? 0) - 90, 0) * 10}%` }"></div>
          </div>
          <div class="mt-3 text-xs">
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.exceptions') }}:</span>
              <span class="font-bold text-red-600 dark:text-red-400">{{ formatNumber((overview.request_count_sla ?? 0) - (overview.success_count ?? 0)) }}</span>
            </div>
          </div>
        </div>

        <!-- Card 4: Request Duration -->
        <div class="ops-detail-card" style="order: 4;">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1">
              <span class="text-[10px] font-bold uppercase text-gray-400">{{ t('admin.ops.latencyDuration') }}</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.latency')" />
            </div>
            <button
              v-if="!props.fullscreen"
              class="ops-detail-link text-[10px] font-bold"
              type="button"
              @click="openDetails({ title: t('admin.ops.latencyDuration'), sort: 'duration_desc' })"
            >
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 flex items-baseline gap-2">
            <div class="text-3xl font-black text-gray-900 dark:text-white">
              {{ durationP99Ms ?? '-' }}
            </div>
            <span class="text-xs font-bold text-gray-400">ms (P99)</span>
          </div>
          <div class="mt-3 grid grid-cols-1 gap-x-3 gap-y-1 text-xs 2xl:grid-cols-2">
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P95:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ durationP95Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P90:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ durationP90Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P50:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ durationP50Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">Avg:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ durationAvgMs ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">Max:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ durationMaxMs ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
          </div>
        </div>

        <!-- Card 5: TTFT -->
        <div class="ops-detail-card" style="order: 5;">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1">
              <span class="text-[10px] font-bold uppercase text-gray-400">TTFT</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.ttft')" />
            </div>
            <button
              v-if="!props.fullscreen"
              class="ops-detail-link text-[10px] font-bold"
              type="button"
              @click="openDetails({ title: t('admin.ops.ttftLabel'), sort: 'duration_desc' })"
            >
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 flex items-baseline gap-2">
            <div class="text-3xl font-black" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftP99Ms))">
              {{ ttftP99Ms ?? '-' }}
            </div>
            <span class="text-xs font-bold text-gray-400">ms (P99)</span>
          </div>
          <div class="mt-3 grid grid-cols-1 gap-x-3 gap-y-1 text-xs 2xl:grid-cols-2">
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P95:</span>
              <span class="font-bold" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftP95Ms))">{{ ttftP95Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P90:</span>
              <span class="font-bold" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftP90Ms))">{{ ttftP90Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">P50:</span>
              <span class="font-bold" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftP50Ms))">{{ ttftP50Ms ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">Avg:</span>
              <span class="font-bold" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftAvgMs))">{{ ttftAvgMs ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
            <div class="flex items-baseline gap-1 whitespace-nowrap">
              <span class="text-gray-500">Max:</span>
              <span class="font-bold" :class="getThresholdColorClass(getTTFTThresholdLevel(ttftMaxMs))">{{ ttftMaxMs ?? '-' }}</span>
              <span class="text-gray-400">ms</span>
            </div>
          </div>
        </div>

        <!-- Card 3: Request Errors -->
        <div class="ops-detail-card" style="order: 3;" data-testid="ops-detail-request-errors">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1">
              <span class="text-[10px] font-bold uppercase text-gray-400">{{ t('admin.ops.requestErrors') }}</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.errors')" />
            </div>
            <button v-if="!props.fullscreen" class="ops-detail-link text-[10px] font-bold" type="button" @click="openErrorDetails('request')">
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 text-3xl font-black" :class="requestErrorMetricToneClass">
            {{ errorRatePercent == null ? '-' : `${errorRatePercent.toFixed(2)}%` }}
          </div>
          <div class="mt-3 space-y-1 text-xs">
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.errorCount') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ formatNumber(overview.error_count_sla ?? 0) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.businessLimited') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ formatNumber(overview.business_limited_count ?? 0) }}</span>
            </div>
          </div>
        </div>

        <!-- Card 6: Upstream Errors -->
        <div class="ops-detail-card" style="order: 6;" data-testid="ops-detail-upstream-errors">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1">
              <span class="text-[10px] font-bold uppercase text-gray-400">{{ t('admin.ops.upstreamErrors') }}</span>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.upstreamErrors')" />
            </div>
            <button v-if="!props.fullscreen" class="ops-detail-link text-[10px] font-bold" type="button" @click="openErrorDetails('upstream')">
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>
          <div class="mt-2 text-3xl font-black" :class="upstreamErrorMetricToneClass">
            {{ upstreamErrorRatePercent == null ? '-' : `${upstreamErrorRatePercent.toFixed(2)}%` }}
          </div>
          <div class="mt-3 space-y-1 text-xs">
            <div class="flex justify-between">
              <span class="text-gray-500">{{ t('admin.ops.errorCountExcl429529') }}:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ formatNumber(overview.upstream_error_count_excl_429_529 ?? 0) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-gray-500">429/529:</span>
              <span class="font-bold text-gray-900 dark:text-white">{{ formatNumber((overview.upstream_429_count ?? 0) + (overview.upstream_529_count ?? 0)) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Integrated: System health (cards) -->
    <div v-if="overview" class="ops-system-strip">
      <div class="ops-system-strip__heading">
        <span>{{ t('admin.ops.systemHealth') }}</span>
        <small v-if="systemMetrics?.created_at">{{ t('admin.ops.collectedAt') }} {{ formatTimeShort(systemMetrics.created_at) }}</small>
      </div>
      <div class="ops-system-grid grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6">
        <!-- CPU -->
        <div class="ops-system-item">
          <div class="flex items-center gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">CPU</div>
            <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.cpu')" />
          </div>
          <div class="mt-1 text-lg font-black" :class="cpuPercentClass">
            {{ cpuPercentValue == null ? '-' : `${cpuPercentValue.toFixed(1)}%` }}
          </div>
          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{ t('common.warning') }} 80% · {{ t('common.critical') }} 95%
          </div>
        </div>

        <!-- MEM -->
        <div class="ops-system-item">
          <div class="flex items-center gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.memory') }}</div>
            <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.memory')" />
          </div>
          <div class="mt-1 text-lg font-black" :class="memPercentClass">
            {{ memPercentValue == null ? '-' : `${memPercentValue.toFixed(1)}%` }}
          </div>
          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{
              systemMetrics?.memory_used_mb == null || systemMetrics?.memory_total_mb == null
                ? '-'
                : `${formatNumber(systemMetrics.memory_used_mb)} / ${formatNumber(systemMetrics.memory_total_mb)} MB`
            }}
          </div>
        </div>

        <!-- DB -->
        <div class="ops-system-item">
          <div class="flex items-center gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.db') }}</div>
            <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.db')" />
          </div>
          <div class="mt-1 text-lg font-black" :class="dbMiddleClass">
            {{ dbMiddleLabel }}
          </div>
          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{ t('admin.ops.conns') }} {{ dbConnOpenValue ?? '-' }} / {{ dbMaxOpenConnsValue ?? '-' }}
            · {{ t('admin.ops.active') }} {{ dbConnActiveValue ?? '-' }}
            · {{ t('admin.ops.idle') }} {{ dbConnIdleValue ?? '-' }}
            <span v-if="dbConnWaitingValue != null"> · {{ t('admin.ops.waiting') }} {{ dbConnWaitingValue }} </span>
          </div>
        </div>

        <!-- Redis -->
        <div class="ops-system-item">
          <div class="flex items-center gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">Redis</div>
            <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.redis')" />
          </div>
          <div class="mt-1 text-lg font-black" :class="redisMiddleClass">
            {{ redisMiddleLabel }}
          </div>
          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{ t('admin.ops.conns') }} {{ redisConnTotalValue ?? '-' }} / {{ redisPoolSizeValue ?? '-' }}
            <span v-if="redisConnActiveValue != null"> · {{ t('admin.ops.active') }} {{ redisConnActiveValue }} </span>
            <span v-if="redisConnIdleValue != null"> · {{ t('admin.ops.idle') }} {{ redisConnIdleValue }} </span>
          </div>
        </div>

        <!-- Goroutines -->
        <div class="ops-system-item">
          <div class="flex items-center gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.goroutines') }}</div>
            <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.goroutines')" />
          </div>
          <div class="mt-1 text-lg font-black" :class="goroutineStatusClass">
            {{ goroutineStatusLabel }}
          </div>
          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{ t('admin.ops.current') }} <span class="font-mono">{{ goroutineCountValue ?? '-' }}</span>
            · {{ t('common.warning') }} <span class="font-mono">{{ goroutinesWarnThreshold }}</span>
            · {{ t('common.critical') }} <span class="font-mono">{{ goroutinesCriticalThreshold }}</span>
            <span v-if="systemMetrics?.concurrency_queue_depth != null">
              · {{ t('admin.ops.queue') }} <span class="font-mono">{{ systemMetrics.concurrency_queue_depth }}</span>
            </span>
          </div>
        </div>

        <!-- Jobs -->
        <div class="ops-system-item">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-1">
              <div class="text-[10px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.jobs') }}</div>
              <HelpTooltip v-if="!props.fullscreen" :content="t('admin.ops.tooltips.jobs')" />
            </div>
            <button v-if="!props.fullscreen" class="ops-detail-link text-[10px] font-bold" type="button" @click="openJobsDetails">
              {{ t('admin.ops.requestDetails.details') }}
            </button>
          </div>

          <div class="mt-1 text-lg font-black" :class="jobsStatusClass">
            {{ jobsStatusLabel }}
          </div>

          <div v-if="!props.fullscreen" class="mt-1 text-[10px] text-gray-500 dark:text-gray-400">
            {{ t('common.total') }} <span class="font-mono">{{ jobHeartbeats.length }}</span>
            · {{ t('common.warning') }} <span class="font-mono">{{ jobsWarnCount }}</span>
          </div>
        </div>
      </div>
    </div>

    </div>

    <BaseDialog :show="showJobsDetails" :title="t('admin.ops.jobs')" width="wide" @close="showJobsDetails = false">
      <div v-if="!jobHeartbeats.length" class="text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.ops.noData') }}
      </div>
      <div v-else class="space-y-3">
        <div
          v-for="hb in jobHeartbeats"
          :key="hb.job_name"
          class="rounded-xl border border-gray-100 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ hb.job_name }}</div>
            <div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400">
              <span v-if="hb.last_duration_ms != null" class="font-mono">{{ hb.last_duration_ms }}ms</span>
              <span>{{ formatTimeShort(hb.updated_at) }}</span>
            </div>
          </div>

          <div class="mt-2 grid grid-cols-1 gap-2 text-xs text-gray-600 dark:text-gray-300 sm:grid-cols-2">
            <div>
              {{ t('admin.ops.lastSuccess') }} <span class="font-mono">{{ formatTimeShort(hb.last_success_at) }}</span>
            </div>
            <div>
              {{ t('admin.ops.lastError') }} <span class="font-mono">{{ formatTimeShort(hb.last_error_at) }}</span>
            </div>
            <div>
              {{ t('admin.ops.result') }} <span class="font-mono">{{ hb.last_result || '-' }}</span>
            </div>
          </div>

          <div
            v-if="hb.last_error"
            class="mt-3 rounded-lg bg-rose-50 p-2 text-xs text-rose-700 dark:bg-rose-900/20 dark:text-rose-300"
          >
            {{ hb.last_error }}
          </div>
        </div>
      </div>
    </BaseDialog>

    <!-- Custom Time Range Dialog -->
    <BaseDialog :show="showCustomTimeRangeDialog" :title="t('admin.ops.timeRange.custom')" width="narrow" @close="handleCustomTimeRangeCancel">
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('admin.ops.customTimeRange.startTime') }}
          </label>
          <input
            v-model="customStartTimeInput"
            type="datetime-local"
            class="ops-dialog-input w-full"
          />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ t('admin.ops.customTimeRange.endTime') }}
          </label>
          <input
            v-model="customEndTimeInput"
            type="datetime-local"
            class="ops-dialog-input w-full"
          />
        </div>
        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            class="ops-dialog-button"
            @click="handleCustomTimeRangeCancel"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="ops-dialog-button ops-dialog-button--primary"
            @click="handleCustomTimeRangeConfirm"
          >
            {{ t('common.confirm') }}
          </button>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>

<style scoped>
.ops-command-center {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: visible;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-surface);
  font-family: var(--lx-clay-font-ui);
}

.ops-command-center--fullscreen {
  min-height: min(calc(100vh - 3rem), 900px);
}

.ops-command-center--resources .ops-masthead {
  align-items: center;
  padding: 0.75rem 1rem;
}

.ops-command-center--resources .ops-masthead__identity {
  align-items: center;
}

.ops-command-center--resources .ops-masthead__icon {
  width: 2.5rem;
  height: 2.5rem;
}

.ops-command-center--resources .ops-status-line {
  min-width: 0;
  flex-wrap: wrap;
  margin-top: 0.125rem;
  overflow-wrap: anywhere;
}

.ops-command-center--resources .ops-command-bar {
  gap: 0.5rem;
  padding: 0.5rem 1rem;
}

.ops-command-center--resources .ops-command-bar__label {
  display: none;
}

.ops-command-center--resources .ops-command-bar__controls {
  flex-basis: auto;
}

.ops-command-center--resources .ops-filter-control {
  max-width: 18rem;
  flex: 1 1 12rem;
}

.ops-masthead {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  padding: 1.125rem 1.25rem 1rem;
  border-bottom: 1px solid var(--lx-clay-border);
}

.ops-masthead__identity {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 0.875rem;
}

.ops-masthead__icon {
  display: inline-flex;
  width: 2.75rem;
  height: 2.75rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--lx-clay-accent) 20%, transparent);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.ops-masthead__copy {
  min-width: 0;
}

.ops-masthead__title-row {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.625rem;
}

.ops-masthead h1 {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1.25rem;
  font-weight: 850;
  letter-spacing: -0.02em;
  line-height: 1.25;
  text-wrap: balance;
}

.ops-masthead p {
  margin: 0.25rem 0 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  line-height: 1.45;
}

.ops-status-badge {
  display: inline-flex;
  min-height: 1.75rem;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 999px;
  font-size: 0.6875rem;
  font-weight: 750;
  line-height: 1;
  white-space: nowrap;
}

.ops-status-badge i {
  width: 0.4375rem;
  height: 0.4375rem;
  border-radius: 999px;
  background: currentColor;
}

.ops-status-badge--healthy {
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.ops-status-badge--warning {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.ops-status-badge--critical {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.ops-status-badge--idle,
.ops-status-badge--loading {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.ops-status-line {
  display: flex;
  min-width: 0;
  max-width: 100%;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.25rem 0.75rem;
  margin-top: 0.375rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.ops-status-line__countdown {
  position: relative;
  padding-inline-start: 0.75rem;
}

.ops-status-line__countdown::before {
  position: absolute;
  top: 50%;
  left: 0;
  width: 0.25rem;
  height: 0.25rem;
  border-radius: 999px;
  background: var(--lx-clay-border-strong);
  content: '';
  transform: translateY(-50%);
}

.ops-command-bar {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1.25rem;
  border-bottom: 1px solid var(--lx-clay-border);
  background: color-mix(in srgb, var(--lx-clay-recessed) 58%, var(--lx-clay-surface));
}

.ops-command-bar__label {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.4rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  font-weight: 750;
}

.ops-command-bar__controls {
  display: flex;
  min-width: 0;
  flex: 1 1 40rem;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.ops-filter-control {
  min-width: 8.5rem;
  flex: 1 1 8.5rem;
}

.ops-filter-control--group {
  flex-basis: 9.5rem;
}

.ops-filter-control--range {
  flex-basis: 10rem;
}

.ops-command-center :deep(.select-trigger) {
  min-height: 2.75rem;
  border-color: var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: none;
}

.ops-command-center :deep(.select-trigger:hover) {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-surface-soft);
}

.ops-command-center :deep(.select-trigger:focus-visible) {
  border-color: var(--lx-clay-accent);
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 24%, transparent);
  outline-offset: 1px;
}

.ops-command-button,
.ops-icon-button {
  display: inline-flex;
  min-height: 2.75rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface);
  font-size: 0.75rem;
  font-weight: 750;
  transition: border-color 180ms ease, background-color 180ms ease, color 180ms ease, transform 180ms ease;
}

.ops-command-button {
  padding: 0.625rem 0.875rem;
}

.ops-icon-button {
  width: 2.75rem;
  padding: 0;
}

.ops-command-button:hover:not(:disabled),
.ops-icon-button:hover:not(:disabled) {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.ops-command-button:active:not(:disabled),
.ops-icon-button:active:not(:disabled) {
  transform: translateY(1px);
}

.ops-command-button:disabled,
.ops-icon-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.ops-command-button--primary {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 72%, transparent);
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent);
  box-shadow: var(--lx-clay-shadow-primary);
}

.ops-command-button--primary:hover:not(:disabled) {
  border-color: var(--lx-clay-accent-deep);
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent-deep);
}

.ops-command-center button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 32%, transparent);
  outline-offset: 2px;
}

.ops-spin {
  animation: ops-spin 900ms linear infinite;
}

.ops-signal-strip {
  display: grid;
  min-width: 0;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  border-bottom: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.ops-signal {
  display: flex;
  min-width: 0;
  min-height: 7rem;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 0.25rem;
  padding: 0.875rem 1rem;
  border: 0;
  border-inline-end: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text);
  background: transparent;
  text-align: start;
}

.ops-signal:last-child {
  border-inline-end: 0;
}

.ops-signal--action {
  cursor: pointer;
  transition: background-color 180ms ease;
}

.ops-signal--action:hover {
  background: color-mix(in srgb, var(--lx-clay-surface) 82%, var(--lx-clay-accent-soft));
}

.ops-signal__heading {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 750;
  line-height: 1.3;
}

.ops-signal__dot {
  width: 0.4375rem;
  height: 0.4375rem;
  flex: 0 0 auto;
  border-radius: 999px;
  background: currentColor;
}

.ops-signal > strong {
  max-width: 100%;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
  font-size: 1.45rem;
  font-weight: 850;
  letter-spacing: -0.025em;
  line-height: 1.15;
  overflow-wrap: anywhere;
}

.ops-signal strong em {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-style: normal;
  font-weight: 750;
  letter-spacing: 0;
}

.ops-signal small {
  max-width: 100%;
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.metric-tone-normal {
  color: var(--lx-clay-success-text) !important;
}

.metric-tone-warning {
  color: var(--lx-clay-warning) !important;
}

.metric-tone-critical {
  color: var(--lx-clay-danger) !important;
}

.metric-tone-neutral {
  color: var(--lx-clay-text-subtle) !important;
}

.metric-fill-normal {
  background: var(--lx-clay-success-bright) !important;
}

.metric-fill-warning {
  background: var(--lx-clay-warning-bright) !important;
}

.metric-fill-critical {
  background: var(--lx-clay-danger) !important;
}

.metric-fill-neutral {
  background: var(--lx-clay-text-subtle) !important;
}

.ops-threshold-dot {
  width: 0.375rem;
  height: 0.375rem;
  flex: 0 0 auto;
  border-radius: 999px;
}

.ops-signal__dot.metric-tone-normal {
  background: var(--lx-clay-success-bright);
}

.ops-signal__dot.metric-tone-warning {
  background: var(--lx-clay-warning-bright);
}

.ops-signal__dot.metric-tone-critical {
  background: var(--lx-clay-danger);
}

.ops-signal__dot.metric-tone-neutral {
  background: var(--lx-clay-text-subtle);
}

.ops-diagnostic-toggle {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 2.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.625rem 1.25rem;
  border: 0;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: color-mix(in srgb, var(--lx-clay-recessed) 42%, var(--lx-clay-surface));
  font-size: 0.75rem;
  text-align: start;
  transition: background-color 180ms ease, color 180ms ease;
}

.ops-diagnostic-toggle:hover {
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.ops-diagnostic-toggle > span {
  display: inline-flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.4rem;
}

.ops-diagnostic-toggle strong {
  font-weight: 800;
}

.ops-diagnostic-toggle small {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
}

.ops-diagnostic-toggle__chevron {
  flex: 0 0 auto;
  transition: transform 180ms ease;
}

.ops-diagnostic-toggle__chevron--open {
  transform: rotate(180deg);
}

.ops-diagnostic-details {
  min-width: 0;
}

.ops-overview-grid {
  min-width: 0;
  gap: 0.75rem !important;
  padding: 1rem 1.25rem 0;
}

.ops-overview-grid .gap-6 {
  gap: 0.75rem !important;
}

.ops-health-panel {
  min-width: 0;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: color-mix(in srgb, var(--lx-clay-recessed) 72%, var(--lx-clay-surface));
}

.ops-diagnosis {
  border-color: var(--lx-clay-border) !important;
}

.ops-diagnosis:hover,
.ops-diagnosis:focus-within {
  background: var(--lx-clay-surface-soft);
}

.ops-diagnosis .mt-4 {
  margin-top: 0.5rem !important;
}

.ops-diagnosis > div[role='tooltip'] > div {
  border: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

.ops-diagnosis-info {
  color: var(--lx-clay-info) !important;
}

.ops-realtime-panel {
  min-width: 0;
}

.ops-realtime-panel .mb-3 {
  margin-bottom: 0.5rem !important;
}

.ops-realtime-panel .space-y-3 > :not([hidden]) ~ :not([hidden]),
.ops-realtime-panel .space-y-4 > :not([hidden]) ~ :not([hidden]) {
  margin-top: 0.5rem !important;
}

.ops-live-dot {
  background: var(--lx-clay-success-bright);
}

.ops-live-dot--halo {
  background: var(--lx-clay-success-soft);
}

.ops-window-chip {
  min-width: 2.75rem;
  min-height: 2.75rem;
  padding: 0.35rem 0.5rem;
  border: 1px solid transparent;
  border-radius: 0.625rem;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.625rem;
  font-weight: 800;
  transition: border-color 180ms ease, background-color 180ms ease, color 180ms ease;
}

.ops-window-chip:hover {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-recessed-strong);
}

.ops-window-chip--selected {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 38%, transparent);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.ops-heartbeat {
  height: 1rem !important;
  border-bottom: 1px solid color-mix(in srgb, var(--lx-clay-success-bright) 18%, transparent);
}

.ops-detail-grid {
  min-width: 0;
  gap: 0 !important;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
}

.ops-detail-card {
  min-width: 0;
  padding: 0.75rem;
  border-right: 1px solid var(--lx-clay-border);
  border-bottom: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.ops-detail-card .text-3xl {
  font-size: 1.35rem !important;
  line-height: 1.15 !important;
}

.ops-detail-card .mt-2 {
  margin-top: 0.375rem !important;
}

.ops-detail-card .mt-3 {
  margin-top: 0.5rem !important;
}

.ops-detail-card .space-y-2 > :not([hidden]) ~ :not([hidden]) {
  margin-top: 0.25rem !important;
}

.ops-detail-card .uppercase,
.ops-realtime-panel .uppercase {
  text-transform: none !important;
  letter-spacing: 0 !important;
}

.ops-detail-card button,
.ops-system-item button {
  display: inline-flex;
  min-width: 2.75rem;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  color: var(--lx-clay-accent) !important;
}

.ops-detail-link:hover {
  text-decoration: underline;
  text-underline-offset: 0.18em;
}

.ops-detail-card:nth-child(3n) {
  border-right: 0;
}

.ops-detail-card:nth-last-child(-n + 3) {
  border-bottom: 0;
}

.ops-detail-card :is(.text-gray-400, .text-gray-500, .text-gray-600) {
  color: var(--lx-clay-text-muted) !important;
}

.ops-detail-card .text-gray-900 {
  color: var(--lx-clay-text) !important;
}

.ops-system-strip {
  min-width: 0;
  margin: 1rem 1.25rem 1.25rem;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
}

.ops-system-strip__heading {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.375rem 0.75rem;
  padding: 0.625rem 0.875rem;
  border-bottom: 1px solid var(--lx-clay-border);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.75rem;
  font-weight: 800;
}

.ops-system-strip__heading small {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
  font-weight: 500;
  overflow-wrap: anywhere;
}

.ops-system-grid {
  min-width: 0;
  gap: 0 !important;
}

.ops-system-item {
  min-width: 0;
  padding: 0.75rem;
  border-right: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.ops-system-item:last-child {
  border-right: 0;
}

.ops-system-item :is(.text-gray-400, .text-gray-500) {
  color: var(--lx-clay-text-muted) !important;
}


.ops-dialog-input {
  min-height: 2.75rem;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--lx-clay-border-strong);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  font-size: 0.875rem;
}

.ops-dialog-input:focus-visible {
  border-color: var(--lx-clay-accent);
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 22%, transparent);
  outline-offset: 1px;
}

.ops-dialog-button {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  padding: 0.625rem 1rem;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
  font-size: 0.875rem;
  font-weight: 700;
}

.ops-dialog-button:hover {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
}

.ops-dialog-button--primary {
  border-color: var(--lx-clay-accent);
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent);
}

.ops-dialog-button--primary:hover {
  border-color: var(--lx-clay-accent-deep);
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent-deep);
}

@media (max-width: 1100px) {
  .ops-signal-strip {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .ops-signal:nth-child(3) {
    border-inline-end: 0;
  }

  .ops-signal:nth-child(-n + 3) {
    border-bottom: 1px solid var(--lx-clay-border);
  }
}

@media (max-width: 1023px) {
  .ops-detail-card:nth-child(3n) {
    border-right: 1px solid var(--lx-clay-border);
  }

  .ops-detail-card:nth-child(2n) {
    border-right: 0;
  }

  .ops-detail-card:nth-last-child(-n + 3) {
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-detail-card:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}

@media (max-width: 720px) {
  .ops-masthead {
    padding: 1rem;
  }

  .ops-command-bar {
    align-items: stretch;
    padding: 0.75rem 1rem;
  }

  .ops-command-bar__label {
    width: 100%;
  }

  .ops-signal-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .ops-signal,
  .ops-signal:nth-child(3) {
    border-inline-end: 1px solid var(--lx-clay-border);
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-signal:nth-child(2n) {
    border-inline-end: 0;
  }

  .ops-signal:last-child {
    border-bottom: 0;
  }

  .ops-overview-grid {
    padding: 0.875rem 1rem 0;
  }

  .ops-system-strip {
    margin: 0.875rem 1rem 1rem;
  }

  .ops-system-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr)) !important;
  }

  .ops-system-item,
  .ops-system-item:nth-child(3n) {
    border-right: 1px solid var(--lx-clay-border);
  }

  .ops-system-item:nth-child(2n) {
    border-right: 0;
  }

  .ops-system-item:nth-child(-n + 4) {
    border-bottom: 1px solid var(--lx-clay-border);
  }
}

@media (max-width: 560px) {
  .ops-masthead__identity {
    gap: 0.625rem;
  }

  .ops-masthead__icon {
    width: 2.5rem;
    height: 2.5rem;
  }

  .ops-status-line {
    align-items: flex-start;
    flex-direction: column;
    gap: 0.125rem;
  }

  .ops-status-line__countdown {
    padding-inline-start: 0;
  }

  .ops-status-line__countdown::before {
    display: none;
  }

  .ops-command-bar__controls {
    flex-basis: 100%;
  }

  .ops-command-center--resources .ops-command-bar__controls {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .ops-command-center--resources .ops-filter-control {
    width: auto;
    max-width: none;
    min-width: 0;
    flex-basis: auto;
    grid-column: span 2;
  }

  .ops-command-center--resources .ops-command-button--primary {
    width: 100%;
    grid-column: span 2;
  }

  .ops-command-center--resources .ops-command-button:not(.ops-command-button--primary) {
    width: 100%;
    min-width: 0;
    grid-column: span 1;
  }

  .ops-filter-control {
    width: 100%;
    min-width: 0;
    flex-basis: 100%;
  }

  .ops-command-button--primary,
  .ops-command-button:not(.ops-command-button--primary) {
    min-width: 0;
    flex: 1 1 8rem;
  }

  .ops-diagnosis > div[role='tooltip'] {
    width: min(18rem, calc(100vw - 2rem));
  }
}

@media (max-width: 430px) {
  .ops-signal-strip {
    grid-template-columns: minmax(0, 1fr);
  }

  .ops-signal,
  .ops-signal:nth-child(2n),
  .ops-signal:nth-child(3) {
    border-inline-end: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-detail-card,
  .ops-detail-card:nth-child(2n),
  .ops-detail-card:nth-child(3n),
  .ops-detail-card:nth-last-child(-n + 2),
  .ops-detail-card:nth-last-child(-n + 3) {
    border-right: 0;
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-detail-card:last-child {
    border-bottom: 0;
  }

  .ops-system-item,
  .ops-system-item:nth-child(3n) {
    border-right: 1px solid var(--lx-clay-border);
    border-bottom: 1px solid var(--lx-clay-border);
  }

  .ops-system-item:nth-child(2n) {
    border-right: 0;
  }

  .ops-system-item:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ops-command-center *,
  .ops-command-center *::before,
  .ops-command-center *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}

@keyframes ops-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
