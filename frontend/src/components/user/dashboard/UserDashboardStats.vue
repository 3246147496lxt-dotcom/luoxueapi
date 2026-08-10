<template>
  <section
    class="dashboard-metric-grid"
    data-testid="dashboard-metric-grid"
    :aria-label="t('dashboard.accountMetrics')"
  >
    <article class="dashboard-metric-card dashboard-metric-card--balance">
      <div class="dashboard-metric-card__header">
        <span>{{ t('dashboard.workspace.balance') }}</span>
        <span class="dashboard-metric-card__icon-well" aria-hidden="true">
          <Icon name="wallet" size="sm" :stroke-width="1.7" />
        </span>
      </div>
      <div class="dashboard-metric-card__value dashboard-metric-card__value--credit">
        <CreditAmount
          :value="formatCredit(balance)"
          icon-size="md"
          :label="`${t('dashboard.workspace.balance')} ${formatCredit(balance)}`"
        />
      </div>
      <p class="dashboard-metric-card__hint">{{ t('dashboard.workspace.balanceHint') }}</p>
      <RouterLink to="/purchase" class="dashboard-metric-card__link">
        {{ t('dashboard.workspace.manageBalance') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </article>

    <article class="dashboard-metric-card dashboard-metric-card--usage">
      <div class="dashboard-metric-card__header">
        <span>{{ t('dashboard.workspace.todayUsage') }}</span>
        <span class="dashboard-metric-card__icon-well" aria-hidden="true">
          <Icon name="activity" size="sm" :stroke-width="1.7" />
        </span>
      </div>
      <div class="dashboard-metric-card__value dashboard-metric-card__value--credit">
        <CreditAmount
          :value="formatCredit(stats.today_actual_cost)"
          icon-size="md"
          :label="`${t('dashboard.workspace.todayUsage')} ${formatCredit(stats.today_actual_cost)}`"
        />
      </div>
      <p class="dashboard-metric-card__hint">
        {{ t('dashboard.workspace.todayRequestsHint', { count: formatNumber(stats.today_requests) }) }}
      </p>
      <RouterLink to="/usage" class="dashboard-metric-card__link">
        {{ t('dashboard.workspace.viewUsage') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </article>

    <article class="dashboard-metric-card dashboard-metric-card--tokens">
      <div class="dashboard-metric-card__header">
        <span>{{ t('dashboard.workspace.tokenConsumption') }}</span>
        <span class="dashboard-metric-card__icon-well" aria-hidden="true">
          <Icon name="type" size="sm" :stroke-width="1.7" />
        </span>
      </div>
      <strong class="dashboard-metric-card__value">
        {{ formatNumber(stats.today_tokens) }}
      </strong>
      <p class="dashboard-metric-card__hint">
        {{ t('dashboard.workspace.tokenBreakdownHint', {
          input: formatNumber(stats.today_input_tokens),
          output: formatNumber(stats.today_output_tokens),
        }) }}
      </p>
      <RouterLink to="/usage" class="dashboard-metric-card__link">
        {{ t('dashboard.workspace.viewUsage') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </article>

    <article class="dashboard-metric-card dashboard-metric-card--plan">
      <div class="dashboard-metric-card__header">
        <span>{{ t('dashboard.workspace.currentPlan') }}</span>
        <span class="dashboard-metric-card__icon-well" aria-hidden="true">
          <Icon name="creditCard" size="sm" :stroke-width="1.7" />
        </span>
      </div>
      <div v-if="planLoading && !subscriptionsLoaded" class="dashboard-plan-loading" aria-live="polite">
        <span class="skeleton h-8 w-36" />
        <span class="sr-only">{{ t('dashboard.workspace.planLoading') }}</span>
      </div>
      <strong v-else class="dashboard-metric-card__value dashboard-metric-card__value--plan">
        <span class="dashboard-plan-dot" aria-hidden="true" />
        <span class="truncate" :title="planName">{{ planName }}</span>
      </strong>
      <p class="dashboard-metric-card__hint">
        {{ planExpiresAt
          ? t('dashboard.workspace.planExpires', { date: formatPlanDate(planExpiresAt) })
          : hasActiveSubscription
            ? t('dashboard.workspace.planNoExpiry')
            : t('dashboard.workspace.planFlexible') }}
      </p>
      <RouterLink to="/subscriptions" class="dashboard-metric-card__link">
        {{ t('dashboard.workspace.managePlan') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </article>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'

defineProps<{
  stats: UserStatsType
  balance: number
  planName: string
  planExpiresAt: string | null
  planLoading: boolean
  subscriptionsLoaded: boolean
  hasActiveSubscription: boolean
}>()

const { t, locale } = useI18n()

function finiteNumber(value: number | null | undefined): number {
  return Number.isFinite(value) ? Number(value) : 0
}

function formatCredit(value: number | null | undefined): string {
  return finiteNumber(value).toFixed(2)
}

function formatNumber(value: number | null | undefined): string {
  return new Intl.NumberFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US')
    .format(finiteNumber(value))
}

function formatPlanDate(value: string): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return value
  return new Intl.DateTimeFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}
</script>

<style scoped>
.dashboard-metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--workspace-space-4);
}

.dashboard-metric-card {
  min-width: 0;
  padding: var(--workspace-space-5);
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-card);
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-work-shadow-card);
}

.dashboard-metric-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.25rem;
}

.dashboard-metric-card__icon-well {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-button);
  color: var(--workspace-work-accent);
  background: var(--workspace-work-accent-soft);
}

.dashboard-metric-card--tokens .dashboard-metric-card__icon-well {
  color: var(--workspace-work-info);
  background: var(--workspace-work-info-soft);
}

.dashboard-metric-card--plan .dashboard-metric-card__icon-well {
  color: var(--workspace-work-text-secondary);
  background: var(--workspace-surface-subtle);
}

.dashboard-metric-card__value {
  display: flex;
  min-height: 2.25rem;
  align-items: center;
  margin-top: 18px;
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-numeric-size);
  font-weight: var(--workspace-type-numeric-weight);
  line-height: 2.25rem;
  letter-spacing: -0.025em;
  font-variant-numeric: tabular-nums;
}

.dashboard-metric-card--balance .dashboard-metric-card__value {
  color: var(--workspace-work-accent-deep);
}

.dashboard-metric-card--balance .dashboard-metric-card__value--credit :deep([data-testid="snowflake-credit-icon"]) {
  opacity: 1;
}

.dashboard-metric-card__value--plan {
  gap: 9px;
}

.dashboard-plan-dot {
  width: 7px;
  height: 7px;
  flex: 0 0 7px;
  border-radius: 999px;
  background: var(--workspace-work-success);
}

.dashboard-plan-loading {
  display: flex;
  min-height: 2.25rem;
  align-items: center;
  margin-top: 18px;
}

.dashboard-metric-card__hint {
  min-height: 2.5rem;
  margin-top: 8px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.25rem;
}

.dashboard-metric-card__link {
  display: inline-flex;
  min-height: 32px;
  align-items: center;
  gap: 5px;
  margin-top: 12px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  transition: color 140ms ease, background-color 140ms ease;
}

.dashboard-metric-card__link:hover {
  color: var(--workspace-work-accent-hover);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.dashboard-metric-card__link:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 3px;
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
    padding: 17px 18px;
  }

  .dashboard-metric-card__value {
    margin-top: 12px;
  }

  .dashboard-metric-card__hint {
    min-height: auto;
  }

  .dashboard-metric-card__link {
    min-height: 44px;
  }
}

:global(html.dark) .dashboard-metric-card {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-metric-card__header,
:global(html.dark) .dashboard-metric-card__hint {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-metric-card__value {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-metric-card__link {
  color: var(--workspace-dark-text-secondary);
}

:global(html.dark) .dashboard-metric-card__link:hover {
  color: var(--workspace-light-surface);
}
</style>
