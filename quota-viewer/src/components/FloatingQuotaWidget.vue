<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import { LoaderCircle, RefreshCw } from 'lucide-vue-next'
import type { ViewerUiStatus } from '@/composables/useQuotaViewer'
import { startQuotaViewerDrag } from '@/lib/desktop'
import type { QuotaItem, QuotaOverview, ViewerDataStatus } from '@/types'

type FloatingQuotaVisualState =
  | 'available'
  | 'warning'
  | 'exhausted'
  | 'expired'
  | 'unavailable'
  | 'no-membership'
  | 'stale'

const props = withDefaults(
  defineProps<{
    overview: QuotaOverview | null
    status?: ViewerUiStatus
    dataStatus?: ViewerDataStatus
    refreshing?: boolean
    lastRefreshAt?: number | null
    errorCode?: string | null
    errorMessage?: string | null
    pairingCode?: string | null
    collapsed?: boolean
  }>(),
  {
    status: 'ready',
    dataStatus: 'ready',
    refreshing: false,
    lastRefreshAt: null,
    errorCode: null,
    errorMessage: null,
    pairingCode: null,
    collapsed: false
  }
)

const emit = defineEmits<{
  refresh: []
  connect: []
  reopen: []
  toggle: []
}>()

const circumference = 2 * Math.PI * 68
const ringGradientId = `quota-ring-${useId().replace(/:/g, '')}`
const dragThreshold = 6
const now = ref(Date.now())
let clock: number | undefined
let mouseGesture:
  | {
      startScreenX: number
      startScreenY: number
    }
  | undefined
let suppressNextClick = false

onMounted(() => {
  clock = window.setInterval(() => {
    now.value = Date.now()
  }, 60_000)
})

onBeforeUnmount(() => {
  if (clock != null) window.clearInterval(clock)
})

const quota = computed<QuotaItem | null>(
  () =>
    props.overview?.quotas.find((item) => item.isCurrentMembership) ?? null
)

const visualState = computed<FloatingQuotaVisualState>(() => {
  if (props.dataStatus === 'stale' || props.status === 'stale') return 'stale'
  if (!quota.value) {
    return props.overview ? 'no-membership' : 'unavailable'
  }
  if (
    quota.value.membershipStatus !== 'active' ||
    quota.value.state === 'expired'
  ) {
    return 'expired'
  }
  if (quota.value.state === 'warning') return 'warning'
  if (quota.value.state === 'exhausted') return 'exhausted'
  if (quota.value.state === 'unknown') return 'unavailable'
  return 'available'
})

const planName = computed(() => {
  const rawName = quota.value?.name.trim()
  if (!rawName) return 'MEMBERSHIP'

  const conciseName = rawName
    .replace(/\s*(?:月度会员|周度会员|年度会员|会员套餐|会员)$/u, '')
    .trim()
  return (conciseName || rawName).toLocaleUpperCase('zh-CN')
})

const remainingPercent = computed<number | null>(() => {
  const current = quota.value
  if (
    !current ||
    current.membershipStatus !== 'active' ||
    current.state === 'expired' ||
    current.state === 'unknown' ||
    current.usedPercent == null ||
    !Number.isFinite(current.usedPercent)
  ) {
    return null
  }

  return Math.max(0, Math.min(100, 100 - current.usedPercent))
})

const displayPercent = computed(() =>
  remainingPercent.value == null ? null : Math.round(remainingPercent.value)
)

const dashOffset = computed(() =>
  circumference * (1 - (displayPercent.value ?? 0) / 100)
)

const ringValueVisible = computed(
  () => displayPercent.value != null && displayPercent.value > 0
)

const usesThreeDigitPercent = computed(
  () => displayPercent.value != null && displayPercent.value >= 100
)

const parseTimestamp = (value: string | null | undefined) => {
  if (!value) return null
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const formatCompactDateTime = (value: Date | null) => {
  if (!value) return null
  const month = value.getMonth() + 1
  const day = value.getDate()
  const hours = String(value.getHours()).padStart(2, '0')
  const minutes = String(value.getMinutes()).padStart(2, '0')
  return `${month}/${day} ${hours}:${minutes}`
}

const resetAt = computed(() => parseTimestamp(quota.value?.resetsAt))
const formattedResetAt = computed(() => formatCompactDateTime(resetAt.value))
const expiresAt = computed(() => parseTimestamp(quota.value?.expiresAt))
const formattedExpiresAt = computed(() =>
  formatCompactDateTime(expiresAt.value)
)

const resetCountdown = computed(() => {
  if (!resetAt.value) return null
  const delta = resetAt.value.getTime() - now.value
  if (delta <= 0) return '即将重置'

  const totalMinutes = Math.ceil(delta / 60_000)
  const days = Math.floor(totalMinutes / 1440)
  const hours = Math.floor((totalMinutes % 1440) / 60)
  const minutes = totalMinutes % 60

  if (days > 0) return `${days} 天 ${hours} 小时后重置`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟后重置`
  return `${Math.max(1, minutes)} 分钟后重置`
})

const resetLine = computed(() => {
  switch (visualState.value) {
    case 'no-membership':
      return '暂无会员订阅'
    case 'expired':
      return quota.value?.statusDetailLabel || '会员已失效'
    case 'unavailable':
      switch (props.status) {
        case 'disconnected':
          return '尚未连接落雪账户'
        case 'connecting':
          return '等待浏览器确认'
        case 'loading':
          return '正在读取会员额度'
        case 'auth-invalid':
          return '账户连接已失效'
        case 'pairing-expired':
          return '本次授权已超时'
        default:
          return props.errorMessage || '暂时无法读取额度'
      }
    default:
      if (!resetCountdown.value || !formattedResetAt.value) return '重置时间待确认'
      return `${resetCountdown.value} · ${formattedResetAt.value}`
  }
})

const monthlyRemainingPercent = computed<number | null>(() => {
  const value = quota.value?.monthlyRemainingPercent
  if (value == null || !Number.isFinite(value)) return null
  return Math.round(Math.max(0, Math.min(100, value)))
})

const monthlyRemainingLine = computed(() => {
  const remaining = `月剩余 ${monthlyRemainingPercent.value ?? '—'}%`
  return formattedExpiresAt.value
    ? `${remaining} · ${formattedExpiresAt.value} 到期`
    : remaining
})

const showsMonthlyRemaining = computed(() =>
  ['available', 'warning', 'exhausted', 'stale'].includes(visualState.value)
)

const refreshFeedbackLine = computed(() => {
  if (props.refreshing) return '正在重新获取额度'
  if (props.lastRefreshAt == null) return null

  const refreshedAt = new Date(props.lastRefreshAt)
  if (!Number.isFinite(refreshedAt.getTime())) return null
  const checkedAt = new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).format(refreshedAt)

  if (props.dataStatus === 'stale' || props.status === 'stale') {
    return `${checkedAt} 已检查，当前显示旧数据`
  }
  if (!props.overview || props.status === 'unavailable') {
    return `${checkedAt} 获取失败，可再次重试`
  }
  if (visualState.value === 'unavailable') {
    return `${checkedAt} 已检查，服务端额度待更新`
  }
  return `${checkedAt} 已更新`
})

const statusLabel: Record<FloatingQuotaVisualState, string> = {
  available: '可用',
  warning: '即将耗尽',
  exhausted: '已耗尽',
  expired: '已失效',
  unavailable: '暂不可用',
  'no-membership': '暂无会员',
  stale: '旧数据'
}

const supportingLine = computed(() => {
  switch (visualState.value) {
    case 'available':
    case 'warning':
    case 'exhausted':
      return monthlyRemainingLine.value
    case 'expired':
      return '会员已失效，请前往网站查看'
    case 'unavailable':
      if (props.status === 'connecting') {
        return props.pairingCode
          ? `浏览器确认码 ${props.pairingCode}`
          : '请在浏览器完成只读授权'
      }
      if (props.status === 'loading') return '正在同步最新额度'
      if (props.status === 'disconnected') return '连接后显示会员周剩余额度'
      if (props.status === 'auth-invalid') return '重新连接后获取最新额度'
      if (props.status === 'pairing-expired') return '请重新发起连接'
      return refreshFeedbackLine.value ?? '暂时无法获取额度'
    case 'no-membership':
      return '开通后显示每周剩余额度与重置时间'
    case 'stale':
      return `${monthlyRemainingLine.value} · 旧数据`
  }
})

const cardAriaLabel = computed(() => {
  if (displayPercent.value == null) {
    return `CODEX · ${planName.value}，${statusLabel[visualState.value]}，${resetLine.value}`
  }
  return `CODEX · ${planName.value}，周剩余 ${displayPercent.value}%，${monthlyRemainingLine.value}，${resetLine.value}`
})

const recoveryAction = computed<null | {
  label: string
  event: 'refresh' | 'connect' | 'reopen'
}>(() => {
  if (visualState.value !== 'unavailable' || props.collapsed) {
    return null
  }

  switch (props.status) {
    case 'disconnected':
    case 'auth-invalid':
    case 'pairing-expired':
      return { label: '连接账户', event: 'connect' }
    case 'connecting':
      return { label: '重新打开授权页', event: 'reopen' }
    case 'loading':
      return null
    default:
      return {
        label: props.refreshing ? '获取中' : '重新获取',
        event: 'refresh'
      }
  }
})

const runRecoveryAction = () => {
  if (!recoveryAction.value || props.refreshing) return
  switch (recoveryAction.value.event) {
    case 'connect':
      emit('connect')
      break
    case 'reopen':
      emit('reopen')
      break
    default:
      emit('refresh')
  }
}

const isInteractiveTarget = (target: EventTarget | null) =>
  target instanceof Element &&
  Boolean(
    target.closest(
      'button, a, input, select, textarea, [data-no-window-drag]'
    )
  )

const onCardMouseDown = (event: MouseEvent) => {
  if (event.button !== 0 || isInteractiveTarget(event.target)) return

  suppressNextClick = false
  mouseGesture = {
    startScreenX: event.screenX,
    startScreenY: event.screenY
  }
  startQuotaViewerDrag(event)
}

const gestureMoved = (event: MouseEvent) => {
  const gesture = mouseGesture
  if (!gesture) return false

  return Math.hypot(
    event.screenX - gesture.startScreenX,
    event.screenY - gesture.startScreenY
  ) >= dragThreshold
}

const onCardMouseMove = (event: MouseEvent) => {
  if (!gestureMoved(event)) return
  suppressNextClick = true
}

const onCardMouseUp = (event: MouseEvent) => {
  if (gestureMoved(event)) suppressNextClick = true
}

const onCardClick = (event: MouseEvent) => {
  if (gestureMoved(event)) suppressNextClick = true
  mouseGesture = undefined

  if (suppressNextClick) {
    suppressNextClick = false
    return
  }
  emit('toggle')
}

</script>

<template>
  <article
    class="floating-quota-widget"
    :class="[
      `floating-quota-widget--${visualState}`,
      {
        'floating-quota-widget--collapsed': collapsed,
        'floating-quota-widget--three-digits': usesThreeDigitPercent
      }
    ]"
    :aria-label="cardAriaLabel"
    role="button"
    tabindex="0"
    @mousedown="onCardMouseDown"
    @mousemove="onCardMouseMove"
    @mouseup="onCardMouseUp"
    @click="onCardClick"
    @keydown.enter.prevent="emit('toggle')"
    @keydown.space.prevent="emit('toggle')"
  >
    <template v-if="collapsed">
      <div class="floating-quota-widget__compact" aria-hidden="true">
        <strong>{{ displayPercent ?? '—' }}</strong>
        <span v-if="displayPercent != null">%</span>
      </div>
      <span class="sr-only">{{ statusLabel[visualState] }}</span>
    </template>

    <template v-else>
      <header class="floating-quota-widget__header">
        <div>
          <strong class="floating-quota-widget__plan">CODEX <span>·</span> {{ planName }}</strong>
          <p>周剩余</p>
        </div>
        <span
          class="floating-quota-widget__status-dot"
          :title="statusLabel[visualState]"
          aria-hidden="true"
        />
        <span class="sr-only">{{ statusLabel[visualState] }}</span>
      </header>

      <section
        class="floating-quota-widget__metric"
        :aria-label="displayPercent == null ? statusLabel[visualState] : `周剩余 ${displayPercent}%`"
      >
        <svg
          class="floating-quota-widget__ring"
          viewBox="0 0 160 160"
          role="progressbar"
          aria-label="本周剩余额度"
          aria-valuemin="0"
          aria-valuemax="100"
          :aria-valuenow="displayPercent ?? undefined"
        >
          <defs>
            <linearGradient
              :id="ringGradientId"
              x1="20%"
              y1="15%"
              x2="82%"
              y2="88%"
            >
              <stop offset="0%" stop-color="var(--ring-start)" />
              <stop offset="100%" stop-color="var(--ring-end)" />
            </linearGradient>
          </defs>
          <circle class="floating-quota-widget__ring-track" cx="80" cy="80" r="68" />
          <circle
            class="floating-quota-widget__ring-value"
            :class="{ 'floating-quota-widget__ring-value--hidden': !ringValueVisible }"
            cx="80"
            cy="80"
            r="68"
            :stroke="`url(#${ringGradientId})`"
            :style="{
              strokeDasharray: `${circumference}px`,
              strokeDashoffset: `${dashOffset}px`
            }"
          />
        </svg>
        <div class="floating-quota-widget__percent">
          <strong>{{ displayPercent ?? '—' }}</strong>
          <span v-if="displayPercent != null">%</span>
        </div>
      </section>

      <p class="floating-quota-widget__reset">{{ resetLine }}</p>
      <footer class="floating-quota-widget__footer">
        <span
          v-if="showsMonthlyRemaining"
          class="floating-quota-widget__monthly-remaining"
        >
          <span>月剩余</span>
          <strong>{{ monthlyRemainingPercent ?? '—' }}%</strong>
          <span
            v-if="formattedExpiresAt"
            class="floating-quota-widget__monthly-expiry"
          >
            · {{ formattedExpiresAt }} 到期
          </span>
          <em v-if="visualState === 'stale'">旧数据</em>
        </span>
        <span v-else aria-live="polite">{{ supportingLine }}</span>
        <button
          v-if="recoveryAction"
          type="button"
          class="floating-quota-widget__refresh"
          data-no-window-drag
          :disabled="refreshing"
          :aria-busy="refreshing"
          @click.stop="runRecoveryAction"
        >
          <LoaderCircle
            v-if="refreshing"
            class="floating-quota-widget__refresh-spinner"
            :size="13"
            aria-hidden="true"
          />
          <RefreshCw v-else :size="13" aria-hidden="true" />
          <span>{{ recoveryAction.label }}</span>
        </button>
      </footer>
    </template>
  </article>
</template>

<style scoped>
.floating-quota-widget {
  --widget-surface: rgba(247, 247, 249, 0.68);
  --ring-start: #8b5cf6;
  --ring-end: #5b21b6;
  --metric-color: #7c3aed;
  --status-color: #63f58c;
  position: relative;
  box-sizing: border-box;
  width: 306px;
  height: 306px;
  display: grid;
  grid-template-rows: 44px minmax(0, 1fr) auto auto;
  row-gap: 5px;
  overflow: hidden;
  isolation: isolate;
  border: 1px solid rgba(91, 80, 112, 0.14);
  border-radius: 32px;
  color: #332f3a;
  background: var(--widget-surface);
  background-clip: padding-box;
  backdrop-filter: blur(20px) saturate(1.08);
  -webkit-backdrop-filter: blur(20px) saturate(1.08);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.28),
    inset 0 0 0 1px rgba(255, 255, 255, 0.2);
  padding: 22px 22px 20px;
  font-family: "SF Pro Display", "SF Pro Text", "Segoe UI Variable Display", "Segoe UI", system-ui, sans-serif;
  font-variant-numeric: tabular-nums;
  cursor: grab;
  touch-action: none;
  user-select: none;
}

.floating-quota-widget:active {
  cursor: grabbing;
}

.floating-quota-widget__header {
  grid-row: 1;
  display: flex;
  min-width: 0;
  min-height: 0;
  align-items: flex-start;
  justify-content: space-between;
}

.floating-quota-widget__plan {
  display: block;
  max-width: 208px;
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.floating-quota-widget__plan span {
  padding: 0 2px;
}

.floating-quota-widget__header p {
  margin: 5px 0 0;
  color: rgba(17, 20, 27, 0.9);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: 0;
  line-height: 1.2;
}

.floating-quota-widget__status-dot {
  box-sizing: border-box;
  width: 25px;
  height: 25px;
  border: 1px solid rgba(255, 255, 255, 0.32);
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.12);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.26);
  backdrop-filter: blur(8px);
}

.floating-quota-widget__status-dot::after {
  display: block;
  width: 8px;
  height: 8px;
  margin: 7px;
  border-radius: 50%;
  background: var(--status-color);
  box-shadow: 0 0 7px color-mix(in srgb, var(--status-color) 72%, transparent);
  content: '';
}

.floating-quota-widget__metric {
  position: absolute;
  top: 50%;
  left: 50%;
  z-index: 1;
  width: 160px;
  height: 160px;
  margin: 0;
  display: grid;
  place-items: center;
  transform: translate(-50%, -50%);
}

.floating-quota-widget__ring {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
  transform: rotate(-90deg);
}

.floating-quota-widget__ring-track,
.floating-quota-widget__ring-value {
  fill: none;
  stroke-width: 12;
}

.floating-quota-widget__ring-track {
  stroke: rgba(255, 255, 255, 0.88);
}

.floating-quota-widget__ring-value {
  stroke-linecap: round;
  opacity: 1;
  transition: stroke-dashoffset 0.5s ease;
}

.floating-quota-widget__ring-value--hidden {
  opacity: 0;
}

.floating-quota-widget__percent {
  display: flex;
  align-items: flex-end;
  color: var(--metric-color);
  line-height: 0.82;
  letter-spacing: 0;
}

.floating-quota-widget__percent strong {
  font-size: 64px;
  font-weight: 500;
}

.floating-quota-widget__percent span {
  margin: 0 0 5px 5px;
  font-size: 21px;
  font-weight: 700;
  letter-spacing: 0;
}

.floating-quota-widget--three-digits .floating-quota-widget__percent strong {
  font-size: 50px;
}

.floating-quota-widget--three-digits .floating-quota-widget__percent span {
  margin-bottom: 4px;
  margin-left: 3px;
  font-size: 18px;
}

.floating-quota-widget__reset {
  grid-row: 3;
  overflow: hidden;
  margin: 0;
  color: rgba(17, 20, 27, 0.52);
  font-size: 12px;
  letter-spacing: 0;
  line-height: 1.4;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.floating-quota-widget__footer {
  grid-row: 4;
  min-width: 0;
  min-height: 16px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  color: rgba(17, 20, 27, 0.68);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0;
  line-height: 1.2;
  text-align: left;
}

.floating-quota-widget__footer > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.floating-quota-widget__monthly-remaining {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 4px;
}

.floating-quota-widget__monthly-remaining strong {
  color: var(--metric-color);
  font-size: 13px;
  font-weight: 750;
  letter-spacing: 0;
}

.floating-quota-widget__monthly-remaining em {
  flex: 0 0 auto;
  margin-left: 3px;
  color: rgba(17, 20, 27, 0.48);
  font-style: normal;
}

.floating-quota-widget__monthly-expiry {
  min-width: 0;
  overflow: hidden;
  color: rgba(17, 20, 27, 0.6);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.floating-quota-widget__refresh {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: 0;
  color: #234977;
  background: transparent;
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.floating-quota-widget__refresh:disabled {
  cursor: wait;
  opacity: 0.72;
}

.floating-quota-widget__refresh-spinner {
  animation: floating-quota-refresh-spin 0.8s linear infinite;
}

@keyframes floating-quota-refresh-spin {
  to {
    transform: rotate(360deg);
  }
}

.floating-quota-widget__refresh:hover {
  text-decoration: underline;
  text-underline-offset: 2px;
}

.floating-quota-widget--warning {
  --widget-surface: rgba(249, 244, 231, 0.68);
  --ring-start: #d28a1d;
  --ring-end: #f2c96d;
  --metric-color: #9a610e;
  --status-color: #f4bc36;
}

.floating-quota-widget--exhausted {
  --widget-surface: rgba(249, 238, 240, 0.68);
  --ring-start: #dc2626;
  --ring-end: #f39a9a;
  --metric-color: #c92f2f;
  --status-color: #dc2626;
}

.floating-quota-widget--expired,
.floating-quota-widget--unavailable,
.floating-quota-widget--no-membership,
.floating-quota-widget--stale {
  --widget-surface: rgba(237, 240, 242, 0.68);
  --ring-start: #8f9094;
  --ring-end: #b8c0ca;
  --metric-color: #737b84;
  --status-color: #8f9094;
}

.floating-quota-widget--collapsed {
  width: 72px;
  height: 72px;
  display: grid;
  grid-template: 1fr / 1fr;
  place-items: center;
  border-radius: 24px;
  padding: 0;
  cursor: grab;
}

.floating-quota-widget__compact {
  display: flex;
  align-items: flex-end;
  color: var(--metric-color);
  line-height: 0.86;
  letter-spacing: 0;
}

.floating-quota-widget__compact strong {
  font-size: 27px;
  font-weight: 560;
}

.floating-quota-widget__compact span {
  margin: 0 0 3px 1px;
  font-size: 10px;
  font-weight: 750;
  letter-spacing: 0;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (prefers-reduced-motion: reduce) {
  .floating-quota-widget__ring-value {
    transition: none;
  }
}
</style>
