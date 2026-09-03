<template>
  <section
    class="dashboard-metric-grid"
    data-testid="dashboard-metric-grid"
    :aria-label="t('dashboard.accountMetrics')"
  >
    <article class="dashboard-metric-card dashboard-metric-card--balance">
      <h2 class="dashboard-metric-card__title">
        {{ t('dashboard.workspace.accountBalance') }}
      </h2>
      <p class="dashboard-metric-card__value">
        <strong>{{ formatCredit(balance) }}</strong>
        <span>{{ t('dashboard.workspace.creditsUnit') }}</span>
      </p>
      <RouterLink
        to="/purchase"
        class="dashboard-metric-card__action"
        data-testid="dashboard-balance-recharge"
      >
        {{ t('payment.tabTopUp') }}
      </RouterLink>
    </article>

    <article class="dashboard-metric-card dashboard-metric-card--quota">
      <h2 class="dashboard-metric-card__title">
        {{ t('dashboard.workspace.planQuota', { plan: planName }) }}
      </h2>
      <div v-if="planLoading" class="dashboard-metric-card__loading" aria-live="polite">
        <span class="skeleton h-8 w-28" aria-hidden="true" />
        <span class="sr-only">{{ t('dashboard.workspace.planLoading') }}</span>
      </div>
      <template v-else>
        <p class="dashboard-metric-card__value">
          <strong>{{ quotaDisplay }}</strong>
          <span>{{ t('dashboard.workspace.remaining') }}</span>
        </p>
        <div
          class="dashboard-quota-track"
          role="progressbar"
          :aria-label="t('dashboard.workspace.planQuota', { plan: planName })"
          :aria-valuemin="0"
          :aria-valuemax="100"
          :aria-valuenow="quotaRemainingPercent ?? undefined"
        >
          <span
            class="dashboard-quota-track__value"
            :class="`dashboard-quota-track__value--${quotaTone}`"
            :style="{ width: `${quotaProgress}%` }"
          />
        </div>
      </template>
    </article>

    <article class="dashboard-metric-card">
      <h2 class="dashboard-metric-card__title">
        {{ t('dashboard.workspace.cumulativeTokens') }}
      </h2>
      <p class="dashboard-metric-card__value">
        <strong>{{ tokenMagnitude.value }}</strong>
        <span>{{ tokenMagnitude.unit }} {{ t('dashboard.workspace.tokenUnit') }}</span>
      </p>
      <p class="dashboard-metric-card__detail">
        {{ t('dashboard.workspace.tokenBreakdownHint', {
          input: formatNumber(cumulativeInputTokens),
          output: formatNumber(stats.total_output_tokens),
        }) }}
      </p>
    </article>

    <article class="dashboard-metric-card">
      <h2 class="dashboard-metric-card__title">
        {{ t('dashboard.workspace.cumulativeSpend') }}
      </h2>
      <p class="dashboard-metric-card__value">
        <strong>{{ formatCredit(stats.total_actual_cost) }}</strong>
        <span>{{ t('dashboard.workspace.creditsUnit') }}</span>
      </p>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'

const props = defineProps<{
  stats: UserStatsType
  balance: number
  planName: string
  quotaRemainingPercent: number | null
  planLoading: boolean
}>()

const { t, locale } = useI18n()

const numberLocale = computed(() => locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')
const cumulativeInputTokens = computed(() => (
  finiteNumber(props.stats.total_input_tokens)
  + finiteNumber(props.stats.total_cache_creation_tokens)
  + finiteNumber(props.stats.total_cache_read_tokens)
))

const quotaProgress = computed(() => {
  if (props.quotaRemainingPercent == null) return 0
  return Math.min(100, Math.max(0, props.quotaRemainingPercent))
})

const quotaDisplay = computed(() => (
  props.quotaRemainingPercent == null
    ? '—'
    : `${new Intl.NumberFormat(numberLocale.value, { maximumFractionDigits: 1 }).format(quotaProgress.value)}%`
))

const quotaTone = computed<'healthy' | 'attention' | 'critical' | 'neutral'>(() => {
  if (props.quotaRemainingPercent == null) return 'neutral'
  if (quotaProgress.value >= 50) return 'healthy'
  if (quotaProgress.value >= 20) return 'attention'
  return 'critical'
})

const tokenMagnitude = computed(() => {
  const value = finiteNumber(props.stats.total_tokens)
  if (locale.value.startsWith('zh') && Math.abs(value) >= 10_000) {
    return {
      value: new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 1 }).format(value / 10_000),
      unit: '万',
    }
  }
  if (!locale.value.startsWith('zh') && Math.abs(value) >= 1_000) {
    return {
      value: new Intl.NumberFormat('en-US', {
        notation: 'compact',
        maximumFractionDigits: 1,
      }).format(value),
      unit: '',
    }
  }
  return { value: formatNumber(value), unit: '' }
})

function finiteNumber(value: number | null | undefined): number {
  return Number.isFinite(value) ? Number(value) : 0
}

function formatNumber(value: number | null | undefined): string {
  return new Intl.NumberFormat(numberLocale.value).format(finiteNumber(value))
}

function formatCredit(value: number | null | undefined): string {
  return new Intl.NumberFormat(numberLocale.value, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  }).format(finiteNumber(value))
}
</script>

<style scoped>
.dashboard-metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--workspace-space-6);
}

.dashboard-metric-card {
  min-width: 0;
  min-height: 138px;
  padding: var(--workspace-space-6);
  border: 1px solid var(--workspace-dashboard-card-border);
  border-radius: 24px;
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-dashboard-card-shadow);
}

.dashboard-metric-card--balance {
  position: relative;
}

.dashboard-metric-card__action {
  position: absolute;
  right: var(--workspace-space-6);
  bottom: var(--workspace-space-6);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--workspace-space-1) var(--workspace-space-3);
  border: 1px solid var(--workspace-work-accent-border);
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-accent);
  background: var(--workspace-work-accent-soft);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: 700;
  line-height: 1rem;
  text-decoration: none;
  transition: background-color 140ms ease;
}

.dashboard-metric-card__action:hover {
  border-color: var(--workspace-work-accent-border-strong);
  color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-soft-hover);
}

.dashboard-metric-card__action:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.dashboard-metric-card__title {
  overflow: hidden;
  color: var(--workspace-dashboard-text-muted);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-metric-card__value {
  display: flex;
  min-width: 0;
  min-height: 2rem;
  align-items: baseline;
  gap: var(--workspace-space-1);
  margin-top: var(--workspace-space-4);
  color: var(--workspace-dashboard-text-strong);
  white-space: nowrap;
}

.dashboard-metric-card__value strong {
  overflow: hidden;
  font-size: calc(var(--workspace-type-page-title-size) - 0.25rem);
  font-weight: 700;
  line-height: 2rem;
  letter-spacing: -0.025em;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}

.dashboard-metric-card__value span {
  flex: 0 0 auto;
  color: var(--workspace-dashboard-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.25rem;
}

.dashboard-metric-card__detail {
  overflow: hidden;
  margin-top: 8px;
  color: var(--workspace-dashboard-text-subtle);
  font-size: calc(var(--workspace-type-secondary-size) - 1px);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.dashboard-metric-card__loading {
  display: flex;
  min-height: 2.25rem;
  align-items: center;
  margin-top: 18px;
}

.dashboard-quota-track {
  width: 100%;
  height: 6px;
  overflow: hidden;
  margin-top: var(--workspace-space-4);
  border-radius: 999px;
  background: var(--workspace-dashboard-track);
}

.dashboard-quota-track__value {
  display: block;
  height: 100%;
  border-radius: inherit;
  transition: background-color 180ms ease;
}

.dashboard-quota-track__value--healthy {
  background: var(--workspace-dashboard-success);
}

.dashboard-quota-track__value--attention {
  background: var(--lx-clay-warning-bright);
}

.dashboard-quota-track__value--critical {
  background: var(--lx-clay-danger);
}

.dashboard-quota-track__value--neutral {
  background: var(--workspace-work-text-muted);
}

@media (max-width: 1279px) {
  .dashboard-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 639px) {
  .dashboard-metric-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }

  .dashboard-metric-card {
    min-height: 132px;
    padding: 17px 18px;
  }

  .dashboard-metric-card__value {
    margin-top: 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-metric-card__action,
  .dashboard-quota-track__value {
    transition-duration: 0.01ms;
  }
}

</style>
