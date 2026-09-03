<script setup lang="ts">
import { computed } from 'vue'
import {
  ChevronLeft,
  Clock3,
  Crown,
  Gauge,
  TrendingUp
} from 'lucide-vue-next'
import { startQuotaViewerDrag } from '@/lib/desktop'
import type { DailyUsagePoint, QuotaItem } from '@/types'

const props = defineProps<{
  quota: QuotaItem
  draggable?: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const statusLabel: Record<QuotaItem['state'], string> = {
  available: '可用',
  warning: '即将耗尽',
  exhausted: '已耗尽',
  expired: '已失效',
  unknown: '待确认'
}

const isInactive = computed(() => props.quota.state === 'expired')
const inactiveHeading = computed(() => {
  switch (props.quota.membershipStatus) {
    case 'suspended':
      return '会员已暂停'
    case 'revoked':
      return '会员资格已撤销'
    default:
      return '会员已到期'
  }
})
const detailStatusLabel = computed(() => {
  if (!isInactive.value) return statusLabel[props.quota.state]
  switch (props.quota.membershipStatus) {
    case 'suspended':
      return '已暂停'
    case 'revoked':
      return '已撤销'
    default:
      return '已到期'
  }
})
const periodRangeLabel = computed(() =>
  props.quota.periodStartLabel ? `${props.quota.periodStartLabel} 至今` : null
)

const formatNumber = (value: number | null) =>
  value == null ? '—' : new Intl.NumberFormat('zh-CN').format(value)

const formatCompact = (value: number | null) => {
  if (value == null) return '—'
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${Math.round(value / 1_000)}K`
  return String(value)
}

const segmentHeight = (point: DailyUsagePoint, value: number) => {
  if (point.totalTokens <= 0 || value <= 0) return 0
  return (value / point.totalTokens) * 100
}
</script>

<template>
  <section
    class="monitor-panel monitor-panel--detail"
    :class="[
      `monitor-panel--${quota.tone}`,
      `monitor-panel--state-${quota.state}`,
      { 'window-drag-region': draggable !== false }
    ]"
    :aria-labelledby="`quota-detail-${quota.id}`"
    @mousedown="draggable !== false && startQuotaViewerDrag($event)"
  >
    <div class="detail-header-card">
      <span class="detail-icon" aria-hidden="true">
        <Crown v-if="quota.icon === 'crown'" :size="27" :stroke-width="2.1" />
        <Gauge v-else :size="27" :stroke-width="2.1" />
      </span>

      <div class="detail-header-card__copy">
        <span>PRO 会员详情 · {{ detailStatusLabel }}</span>
        <h2 :id="`quota-detail-${quota.id}`">{{ quota.name }}</h2>
        <p>
          <Clock3 :size="10" :stroke-width="2" aria-hidden="true" />
          {{ quota.resetLabel }}
        </p>
      </div>

      <button
        type="button"
        class="detail-collapse"
        aria-label="收起详情"
        title="收起详情"
        @click="emit('close')"
      >
        <ChevronLeft :size="18" :stroke-width="2.2" />
      </button>
    </div>

    <div v-if="!isInactive" class="detail-metrics">
      <div class="detail-metric">
        <span>本周期请求</span>
        <strong>{{ formatNumber(quota.totalRequests) }}</strong>
      </div>
      <div class="detail-metric">
        <span>本周期 Token</span>
        <strong>{{ formatNumber(quota.totalTokens) }}</strong>
      </div>
    </div>

    <section
      v-if="isInactive"
      class="detail-lifecycle-card"
      aria-labelledby="detail-lifecycle-title"
    >
      <span class="detail-lifecycle-card__icon" aria-hidden="true">
        <Clock3 :size="24" :stroke-width="1.8" />
      </span>
      <div>
        <strong id="detail-lifecycle-title">{{ inactiveHeading }}</strong>
        <p>{{ quota.statusDetailLabel }}</p>
        <small>当前没有可用的周期积分和用量统计。</small>
      </div>
    </section>

    <section
      v-else
      class="detail-chart-card"
      aria-labelledby="detail-chart-title"
    >
      <div class="detail-chart-card__heading">
        <div>
          <h3 id="detail-chart-title">本周期 Token 消耗</h3>
          <p v-if="periodRangeLabel">{{ periodRangeLabel }}</p>
        </div>
        <span class="detail-total-tag">
          <TrendingUp :size="10" :stroke-width="2" aria-hidden="true" />
          <strong>{{ formatCompact(quota.totalTokens) }}</strong>
        </span>
      </div>

      <div v-if="quota.usageAvailable" class="chart-legend" aria-label="Token 分类图例">
        <span><i class="legend-dot legend-dot--hit" />缓存命中</span>
        <span><i class="legend-dot legend-dot--miss" />未命中</span>
        <span><i class="legend-dot legend-dot--output" />输出</span>
      </div>

      <div
        v-if="quota.usageAvailable"
        class="detail-chart"
        role="img"
        aria-label="本周期 Token 分类堆叠柱状图"
      >
        <div
          v-for="point in quota.points"
          :key="point.date"
          class="detail-chart__column"
          :class="{ 'detail-chart__column--selected': point.state === 'partial' }"
          :title="`${point.date}：${formatNumber(point.totalTokens)} Tokens`"
        >
          <span class="detail-chart__value">
            {{ point.totalTokens > 0 ? formatCompact(point.totalTokens) : '' }}
          </span>
          <span class="detail-chart__rail">
            <i
              class="token-segment token-segment--output"
              :style="{ height: `${segmentHeight(point, point.outputTokens)}%` }"
            />
            <i
              class="token-segment token-segment--miss"
              :style="{ height: `${segmentHeight(point, point.cacheMissTokens)}%` }"
            />
            <i
              class="token-segment token-segment--hit"
              :style="{ height: `${segmentHeight(point, point.cacheHitTokens)}%` }"
            />
          </span>
          <span class="detail-chart__label">{{ point.label }}</span>
        </div>
      </div>
      <div v-else class="detail-chart-empty" role="status">
        <TrendingUp :size="24" :stroke-width="1.8" aria-hidden="true" />
        <strong>本周期用量暂时无法获取</strong>
        <p>不影响余额和会员积分判断。</p>
      </div>
    </section>
  </section>
</template>
