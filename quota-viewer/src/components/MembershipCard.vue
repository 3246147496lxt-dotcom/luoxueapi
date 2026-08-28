<script setup lang="ts">
import { computed } from 'vue'
import { Crown, Info } from 'lucide-vue-next'
import type { QuotaItem, QuotaState } from '@/types'

const props = withDefaults(
  defineProps<{
    quota: QuotaItem
    selected?: boolean
    stale?: boolean
  }>(),
  {
    selected: false,
    stale: false
  }
)

const emit = defineEmits<{
  select: [quotaId: string]
}>()

const ringCircumference = 2 * Math.PI * 62

const remainingPercent = computed(() => {
  if (props.quota.usedPercent == null) return null
  return Math.max(0, Math.min(100, 100 - props.quota.usedPercent))
})

const displayedRemainingPercent = computed(() =>
  remainingPercent.value == null ? null : Math.round(remainingPercent.value)
)

const remainingCreditText = computed(() =>
  stripLabel(props.quota.remainingLabel, '剩余')
)

const ringDashOffset = computed(() =>
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

const isInactive = computed(
  () =>
    props.quota.membershipStatus !== 'active' ||
    props.quota.state === 'expired'
)
const isUnknown = computed(() => props.quota.state === 'unknown')
const inactiveTitle = computed(() => {
  switch (props.quota.membershipStatus) {
    case 'suspended':
      return '已暂停'
    case 'revoked':
      return '已撤销'
    default:
      return '已到期'
  }
})
const primaryStatusLabel = computed(() => {
  if (props.stale) return '上次同步数据'
  return isInactive.value
    ? inactiveTitle.value
    : statusLabel[props.quota.state]
})
const quotaAriaLabel = computed(() => {
  if (props.stale) {
    return `查看${props.quota.name}详情，上次同步数据`
  }
  if (isInactive.value) {
    return `查看${props.quota.name}详情，${props.quota.statusDetailLabel}`
  }
  if (displayedRemainingPercent.value == null) {
    return `查看${props.quota.name}详情，积分状态待确认`
  }

  return `查看${props.quota.name}详情，周剩余 ${displayedRemainingPercent.value}%`
})

const formatNumber = (value: number | null) =>
  value == null ? '—' : new Intl.NumberFormat('zh-CN').format(value)

const stripLabel = (value: string, prefix: string) =>
  value.replace(new RegExp(`^${prefix}\\s*`), '')
</script>

<template>
  <button
    type="button"
    class="membership-card tray-membership-card"
    :class="[
      `membership-card--${quota.state}`,
      {
        'membership-card--selected': selected,
        'membership-card--stale': stale
      }
    ]"
    :aria-expanded="selected"
    :aria-controls="selected ? `quota-detail-${quota.id}` : undefined"
    :aria-label="quotaAriaLabel"
    @click="emit('select', quota.id)"
  >
    <span class="membership-top">
      <span class="membership-identity">
        <span class="membership-icon" aria-hidden="true">
          <Crown :size="17" :stroke-width="2.1" />
        </span>
        <span class="membership-copy">
          <strong>{{ quota.name }}</strong>
          <small>
            <i aria-hidden="true" />
            {{ primaryStatusLabel }}
          </small>
          <span class="membership-period-label">周剩余：</span>
        </span>
      </span>
      <Info class="membership-info" :size="16" :stroke-width="2" aria-hidden="true" />
    </span>

    <span class="quota-ring" aria-hidden="true">
      <svg width="140" height="140" viewBox="0 0 140 140">
        <defs>
          <linearGradient
            id="tray-quota-ring-gradient"
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
          <strong>—</strong>
        </template>
        <template v-else>
          <strong>{{ displayedRemainingPercent }}%</strong>
        </template>
      </span>
    </span>

    <span v-if="!isInactive" class="membership-summary">
      <span>
        <small>本周期已用</small>
        <strong>{{ stripLabel(quota.usedLabel, '已用') }}</strong>
      </span>
      <span>
        <small>本周期 Token</small>
        <strong :class="{ 'metric-unavailable': !quota.usageAvailable }">
          {{ formatNumber(quota.totalTokens) }}
        </strong>
      </span>
      <span>
        <small>本周期剩余积分</small>
        <strong
          :class="{ 'metric-unavailable': remainingCreditText === '—' }"
        >
          {{ remainingCreditText }}
        </strong>
      </span>
    </span>
    <span v-else class="membership-inactive-summary">
      <small>会员状态</small>
      <strong>{{ quota.statusDetailLabel }}</strong>
    </span>
  </button>
</template>

<style scoped>
.tray-membership-card {
  --membership-accent: #8b6cf6;
  --membership-strong: #6842df;
  --membership-dot: #10b981;
  --membership-track: rgba(85, 112, 142, 0.11);
  position: relative;
  display: flex;
  width: 100%;
  min-height: 0;
  padding: 14px 16px 12px;
  flex: 1;
  flex-direction: column;
  align-items: stretch;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.78);
  border-radius: 20px;
  appearance: none;
  background:
    radial-gradient(circle at 18% 84%, rgba(255, 255, 255, 0.7), transparent 34%),
    radial-gradient(circle at 84% 10%, rgba(191, 204, 242, 0.44), transparent 42%),
    linear-gradient(145deg, #f7f3fb 0%, #f1f0fa 50%, #edf4fa 100%);
  box-shadow:
    0 10px 28px rgba(72, 84, 101, 0.11),
    inset 0 1px rgba(255, 255, 255, 0.92);
  color: #2f2a36;
  cursor: pointer;
  text-align: left;
}

.tray-membership-card::before {
  position: absolute;
  width: 190px;
  height: 190px;
  border: 32px solid rgba(255, 255, 255, 0.11);
  border-radius: 50%;
  top: -112px;
  right: -70px;
  content: "";
  pointer-events: none;
}

.tray-membership-card > * {
  position: relative;
  z-index: 1;
}

.tray-membership-card:hover {
  border-color: rgba(124, 58, 237, 0.18);
  box-shadow:
    0 12px 30px rgba(72, 84, 101, 0.14),
    inset 0 1px rgba(255, 255, 255, 0.94);
}

.tray-membership-card.membership-card--selected {
  border-color: color-mix(in srgb, var(--membership-accent) 42%, white);
}

.tray-membership-card.membership-card--warning {
  --membership-accent: #e5a233;
  --membership-strong: #bd7510;
  --membership-dot: #f59e0b;
  --membership-track: rgba(180, 83, 9, 0.11);
  background: linear-gradient(145deg, #fffaf0, #fbf4e7 55%, #f4eadb);
}

.tray-membership-card.membership-card--exhausted {
  --membership-accent: #e57d7d;
  --membership-strong: #c63e3e;
  --membership-dot: #dc2626;
  --membership-track: rgba(185, 28, 28, 0.11);
  background: linear-gradient(145deg, #fff6f5, #f8eeee 55%, #efe4e5);
}

.tray-membership-card.membership-card--expired,
.tray-membership-card.membership-card--unknown,
.tray-membership-card.membership-card--stale {
  --membership-accent: #abb0b7;
  --membership-strong: #737b84;
  --membership-dot: #969da5;
  --membership-track: rgba(95, 89, 101, 0.11);
  background: linear-gradient(145deg, #f6f7f8, #eceff1 55%, #e3e7ea);
}

.membership-top,
.membership-identity,
.membership-copy {
  display: flex;
}

.membership-top {
  min-height: 36px;
  align-items: flex-start;
  justify-content: space-between;
}

.membership-identity {
  min-width: 0;
  align-items: center;
  gap: 9px;
}

.membership-icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  border: 1px solid rgba(255, 255, 255, 0.86);
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.66);
  color: var(--membership-strong);
  place-items: center;
}

.membership-copy {
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.membership-copy strong {
  overflow: hidden;
  max-width: 230px;
  color: var(--membership-strong);
  font-size: 13px;
  font-weight: 850;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.membership-copy small {
  display: flex;
  align-items: center;
  gap: 5px;
  color: #047857;
  font-size: 9px;
  font-weight: 750;
}

.membership-copy small i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--membership-dot);
}

.membership-period-label {
  color: #736d79;
  font-size: 8px;
  font-weight: 700;
}

.membership-info {
  color: #0b8bed;
}

.quota-ring {
  position: relative;
  display: grid;
  width: 140px;
  height: 140px;
  margin: 2px auto 4px;
  place-items: center;
}

.quota-ring svg {
  position: absolute;
  inset: 0;
  transform: rotate(-90deg);
}

.quota-ring__track {
  stroke: var(--membership-track);
}

.quota-ring__progress {
  stroke: url(#tray-quota-ring-gradient);
  transition: stroke-dashoffset 320ms ease;
}

.quota-ring__copy {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: #57515d;
  line-height: 1.05;
}

.quota-ring__copy small,
.quota-ring__copy span {
  color: #6f6875;
  font-size: 9px;
  font-weight: 650;
}

.quota-ring__copy strong {
  margin: 3px 0;
  color: var(--membership-strong);
  font-family: Nunito, "PingFang SC", sans-serif;
  font-size: 29px;
  font-weight: 900;
  letter-spacing: 0;
}

.quota-ring__copy--status strong {
  font-size: 21px;
  letter-spacing: 0;
}

.membership-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  min-height: 38px;
  padding-top: 8px;
  border-top: 1px solid rgba(91, 80, 112, 0.11);
}

.membership-summary > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: center;
  gap: 3px;
}

.membership-summary > span + span {
  border-left: 1px solid rgba(91, 80, 112, 0.1);
}

.membership-summary small,
.membership-inactive-summary small {
  color: #736d79;
  font-size: 8px;
  font-weight: 700;
}

.membership-summary strong,
.membership-inactive-summary strong {
  overflow: hidden;
  max-width: 100%;
  color: var(--membership-strong);
  font-size: 9.5px;
  font-variant-numeric: tabular-nums;
  font-weight: 850;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.membership-summary .metric-unavailable {
  color: #8d8791;
}

.membership-inactive-summary {
  display: flex;
  min-height: 38px;
  padding-top: 8px;
  border-top: 1px solid rgba(91, 80, 112, 0.11);
  flex-direction: column;
  align-items: center;
  gap: 3px;
}

@media (prefers-reduced-motion: reduce) {
  .quota-ring__progress {
    transition: none;
  }
}
</style>
