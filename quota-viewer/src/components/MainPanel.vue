<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  CalendarDays,
  Crown,
  Info,
  RefreshCw,
  Settings2,
  SunMedium,
  WalletCards,
  X
} from 'lucide-vue-next'
import { startQuotaViewerDrag } from '@/lib/desktop'
import type {
  QuotaOverview,
  QuotaState,
  ViewerDataStatus
} from '@/types'

const props = defineProps<{
  overview: QuotaOverview
  selectedQuotaId: string | null
  refreshing?: boolean
  dataStatus?: ViewerDataStatus
  connected?: boolean
}>()

const emit = defineEmits<{
  selectQuota: [quotaId: string]
  refresh: []
  disconnect: []
  close: []
}>()

const settingsOpen = ref(false)

const primaryQuota = computed(() => props.overview.quotas[0] ?? null)
const ringCircumference = 2 * Math.PI * 62

const remainingPercent = computed(() => {
  if (!primaryQuota.value || primaryQuota.value.usedPercent == null) return null
  return Math.max(0, Math.min(100, 100 - primaryQuota.value.usedPercent))
})

const ringDashOffset = computed(
  () =>
    remainingPercent.value == null
      ? ringCircumference
      : ringCircumference * (1 - remainingPercent.value / 100)
)

const statusLabel: Record<QuotaState, string> = {
  available: '可用',
  warning: '即将耗尽',
  exhausted: '已耗尽',
  expired: '已失效',
  unknown: '待确认'
}

const isInactive = computed(() => primaryQuota.value?.state === 'expired')
const isUnknown = computed(() => primaryQuota.value?.state === 'unknown')
const inactiveTitle = computed(() => {
  switch (primaryQuota.value?.membershipStatus) {
    case 'suspended':
      return '已暂停'
    case 'revoked':
      return '已撤销'
    default:
      return '已到期'
  }
})
const primaryStatusLabel = computed(() =>
  isInactive.value
    ? inactiveTitle.value
    : statusLabel[primaryQuota.value?.state ?? 'unknown']
)
const quotaAriaLabel = computed(() => {
  if (!primaryQuota.value) return ''
  if (isInactive.value) {
    return `查看${primaryQuota.value.name}详情，${primaryQuota.value.statusDetailLabel}`
  }
  if (remainingPercent.value == null) {
    return `查看${primaryQuota.value.name}详情，额度状态待确认`
  }

  return `查看${primaryQuota.value.name}详情，剩余 ${remainingPercent.value}%，${primaryQuota.value.remainingLabel}`
})

const formatCredit = (amount: string | null) => {
  if (amount == null) return '—'
  const parsed = Number(amount)
  return Number.isFinite(parsed) ? `❄${parsed.toFixed(2)}` : '—'
}
const formatNumber = (value: number | null) =>
  value == null ? '—' : new Intl.NumberFormat('zh-CN').format(value)

const stripLabel = (value: string, prefix: string) =>
  value.replace(new RegExp(`^${prefix}\\s*`), '')

const walletLabel = computed(() => {
  if (props.dataStatus === 'stale') {
    if (props.overview.walletState === 'available') return '上次可用'
    if (props.overview.walletState === 'exhausted') return '上次不足'
    return '上次未知'
  }
  if (props.overview.walletState === 'available') return '可用'
  if (props.overview.walletState === 'exhausted') return '余额不足'
  return '待确认'
})
</script>

<template>
  <section
    class="monitor-panel monitor-panel--main window-drag-region"
    aria-labelledby="quota-viewer-title"
    @mousedown="startQuotaViewerDrag"
  >
    <header class="panel-header">
      <div class="brand-lockup">
        <span class="brand-mark" aria-hidden="true">
          <img src="/logo.png" alt="" />
        </span>
        <div>
          <h1 id="quota-viewer-title">落雪额度</h1>
          <p :class="{ 'data-time--stale': dataStatus === 'stale' }">
            <template v-if="refreshing">正在更新 · </template>
            <template v-if="dataStatus === 'stale'">上次数据 · </template>
            {{ overview.updatedAt }}
          </p>
        </div>
      </div>

      <nav class="header-actions" aria-label="面板操作">
        <button
          type="button"
          class="icon-button"
          :class="{ 'icon-button--spinning': refreshing }"
          :disabled="refreshing"
          aria-label="刷新额度"
          title="刷新额度"
          @click="emit('refresh')"
        >
          <RefreshCw :size="15" :stroke-width="2.1" />
        </button>
        <button
          type="button"
          class="icon-button"
          :aria-expanded="settingsOpen"
          aria-label="查看设置"
          title="设置"
          @click="settingsOpen = !settingsOpen"
        >
          <Settings2 :size="16" :stroke-width="2" />
        </button>
        <button
          type="button"
          class="icon-button"
          aria-label="隐藏额度面板"
          title="隐藏"
          @click="emit('close')"
        >
          <X :size="17" :stroke-width="2" />
        </button>
      </nav>

      <Transition name="popover">
        <div v-if="settingsOpen" class="settings-popover">
          <div class="settings-popover__heading">
            <strong>查看器设置</strong>
            <span>{{ connected ? '账号已安全连接' : '视觉演示数据' }}</span>
          </div>
          <div class="settings-row">
            <span>自动刷新</span>
            <strong>每 5 分钟</strong>
          </div>
          <div class="settings-row">
            <span>数据范围</span>
            <strong>{{ overview.dataScopeLabel }}</strong>
          </div>
          <button
            v-if="connected"
            type="button"
            class="settings-disconnect"
            @click="emit('disconnect')"
          >
            退出连接
          </button>
        </div>
      </Transition>
    </header>

    <div
      v-if="dataStatus === 'stale'"
      class="freshness-banner"
      role="status"
    >
      当前无法确认最新额度，以下为上次数据。
      <button type="button" :disabled="refreshing" @click="emit('refresh')">
        重新获取
      </button>
    </div>

    <section class="panel-card balance-card" aria-labelledby="balance-title">
      <div class="balance-card__heading">
        <div class="eyebrow" id="balance-title">
          <WalletCards :size="14" aria-hidden="true" />
          <span>账户余额</span>
        </div>
        <span
          class="availability-badge"
          :class="{
            'availability-badge--error': overview.walletState === 'exhausted',
            'availability-badge--unknown': overview.walletState === 'unknown'
          }"
        >
          <i aria-hidden="true" />
          {{ walletLabel }}
        </span>
      </div>

      <strong class="balance-value">{{ formatCredit(overview.balance) }}</strong>

      <div class="spend-grid">
        <div class="spend-metric">
          <div>
            <SunMedium :size="14" aria-hidden="true" />
            <span>今日消费</span>
          </div>
          <strong>{{ formatCredit(overview.todaySpend) }}</strong>
        </div>
        <div class="spend-metric">
          <div>
            <CalendarDays :size="14" aria-hidden="true" />
            <span>本月消费</span>
          </div>
          <strong>{{ formatCredit(overview.monthSpend) }}</strong>
        </div>
      </div>
    </section>

    <button
      v-if="primaryQuota"
      type="button"
      class="panel-card membership-card"
      :class="[
        `membership-card--${primaryQuota.state}`,
        { 'membership-card--selected': selectedQuotaId === primaryQuota.id }
      ]"
      :aria-pressed="selectedQuotaId === primaryQuota.id"
      :aria-label="quotaAriaLabel"
      @click="emit('selectQuota', primaryQuota.id)"
    >
      <span class="membership-top">
        <span class="membership-identity">
          <span class="membership-icon" aria-hidden="true">
            <Crown :size="17" :stroke-width="2.1" />
          </span>
          <span class="membership-copy">
            <strong>{{ primaryQuota.name }}</strong>
            <small>
              <i aria-hidden="true" />
              {{ primaryStatusLabel }}
            </small>
          </span>
        </span>
        <Info class="membership-info" :size="16" :stroke-width="2" aria-hidden="true" />
      </span>

      <span class="quota-ring" aria-hidden="true">
        <svg width="140" height="140" viewBox="0 0 140 140">
          <defs>
            <linearGradient
              id="quota-ring-gradient"
              x1="0%"
              y1="0%"
              x2="100%"
              y2="100%"
            >
              <stop offset="0%" stop-color="var(--membership-strong)" />
              <stop offset="100%" stop-color="var(--membership-accent)" />
            </linearGradient>
          </defs>
          <circle
            class="quota-ring__track"
            cx="70"
            cy="70"
            r="62"
            fill="transparent"
            stroke-width="14"
          />
          <circle
            class="quota-ring__progress"
            cx="70"
            cy="70"
            r="62"
            fill="transparent"
            stroke-width="14"
            stroke-linecap="round"
            :stroke-dasharray="ringCircumference"
            :stroke-dashoffset="ringDashOffset"
          />
        </svg>
        <span
          class="quota-ring__copy"
          :class="{ 'quota-ring__copy--status': isInactive }"
        >
          <template v-if="isInactive">
            <small>会员状态</small>
            <strong>{{ inactiveTitle }}</strong>
            <span>当前不可用</span>
          </template>
          <template v-else-if="isUnknown">
            <small>会员额度</small>
            <strong>待确认</strong>
            <span>—</span>
          </template>
          <template v-else>
            <small>剩余</small>
            <strong>{{ remainingPercent }}%</strong>
            <span>{{ stripLabel(primaryQuota.remainingLabel, '剩余') }}</span>
          </template>
        </span>
      </span>

      <span v-if="!isInactive" class="membership-summary">
        <span>
          <small>本周期已用</small>
          <strong>{{ stripLabel(primaryQuota.usedLabel, '已用') }}</strong>
        </span>
        <span>
          <small>本周期 Token</small>
          <strong :class="{ 'metric-unavailable': !primaryQuota.usageAvailable }">
            {{ formatNumber(primaryQuota.totalTokens) }}
          </strong>
        </span>
        <span>
          <small>本周期请求</small>
          <strong :class="{ 'metric-unavailable': !primaryQuota.usageAvailable }">
            {{ formatNumber(primaryQuota.totalRequests) }}
          </strong>
        </span>
      </span>
      <span v-else class="membership-inactive-summary">
        <small>会员状态</small>
        <strong>{{ primaryQuota.statusDetailLabel }}</strong>
      </span>
    </button>

    <section v-else class="panel-card membership-empty" aria-label="暂无会员订阅">
      <span class="membership-icon" aria-hidden="true">
        <Crown :size="18" :stroke-width="2" />
      </span>
      <strong>暂无会员订阅</strong>
      <p>账户余额仍可用于余额计费的 API Key。</p>
    </section>
  </section>
</template>
