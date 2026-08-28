<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Coins, Gauge, Timer } from 'lucide-vue-next'
import type { Component } from 'vue'
import type { DesktopSnapshot } from '@/types'

const props = defineProps<{ today: DesktopSnapshot['today'] }>()
const { t, locale } = useI18n()

function compact(value: number): string {
  return Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 2 }).format(value)
}

function formatPoints(value: number): string {
  return `${value.toFixed(2)} ${t('overview.pointsUnit')}`
}

const metrics: Array<{ key: keyof DesktopSnapshot['today']; label: string; icon: Component; format: (value: number) => string }> = [
  { key: 'requests', label: 'overview.todayRequests', icon: Gauge, format: compact },
  { key: 'tokens', label: 'overview.todayTokens', icon: Coins, format: compact },
  { key: 'cost', label: 'overview.todayCost', icon: Coins, format: formatPoints },
  { key: 'balance', label: 'overview.balance', icon: Coins, format: formatPoints },
  { key: 'averageFirstTokenMs', label: 'overview.latency', icon: Timer, format: (value) => `${Math.round(value)}ms` }
]
</script>

<template>
  <section class="metric-strip" :aria-label="t('common.todaySummary')">
    <div v-for="metric in metrics" :key="metric.key" class="metric-item">
      <component :is="metric.icon" :size="17" aria-hidden="true" />
      <div>
        <span>{{ t(metric.label) }}</span>
        <strong>{{ metric.format(Number(props.today[metric.key] ?? 0)) }}</strong>
      </div>
    </div>
  </section>
</template>
