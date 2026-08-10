<template>
  <section class="dashboard-panel dashboard-model-usage" :aria-label="t('dashboard.workspace.modelUsage')">
    <header class="dashboard-panel-header">
      <div>
        <h2>{{ t('dashboard.workspace.modelUsage') }}</h2>
        <p>{{ t('dashboard.workspace.selectedRange', { start: startDate, end: endDate }) }}</p>
      </div>
      <RouterLink to="/usage" class="dashboard-panel-link">
        {{ t('dashboard.workspace.viewAll') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </RouterLink>
    </header>

    <div class="dashboard-usage-summary" :aria-label="t('dashboard.workspace.rangeSummary')">
      <div>
        <span>{{ t('dashboard.workspace.requests') }}</span>
        <strong>{{ formatNumber(rangeRequests) }}</strong>
      </div>
      <div>
        <span>{{ t('dashboard.workspace.tokens') }}</span>
        <strong>{{ formatNumber(rangeTokens) }}</strong>
      </div>
      <div>
        <span>{{ t('dashboard.workspace.spend') }}</span>
        <strong>{{ formatCredit(rangeCost) }}</strong>
      </div>
    </div>

    <div class="dashboard-model-list">
      <div class="dashboard-model-list__head" aria-hidden="true">
        <span>{{ t('dashboard.workspace.model') }}</span>
        <span>{{ t('dashboard.workspace.requests') }}</span>
        <span>{{ t('dashboard.workspace.tokens') }}</span>
        <span>{{ t('dashboard.workspace.spend') }}</span>
      </div>

      <div v-if="loading" class="dashboard-model-loading" aria-live="polite">
        <LoadingSpinner size="md" />
        <span>{{ t('dashboard.workspace.loadingModels') }}</span>
      </div>

      <div v-else-if="topModels.length === 0" class="dashboard-model-empty">
        <Icon name="destinationModels" size="lg" aria-hidden="true" />
        <p>{{ t('dashboard.workspace.noModelUsage') }}</p>
        <RouterLink to="/models">{{ t('dashboard.workspace.exploreModels') }}</RouterLink>
      </div>

      <ol v-else class="dashboard-model-rows">
        <li v-for="model in topModels" :key="model.model" class="dashboard-model-row">
          <div class="dashboard-model-row__identity">
            <span class="dashboard-model-row__name" :title="model.model">{{ model.model }}</span>
            <span class="dashboard-model-row__track" aria-hidden="true">
              <span :style="{ width: `${modelShare(model.requests)}%` }" />
            </span>
          </div>
          <span class="dashboard-model-row__metric" :data-label="t('dashboard.workspace.requests')">
            {{ formatNumber(model.requests) }}
          </span>
          <span class="dashboard-model-row__metric" :data-label="t('dashboard.workspace.tokens')">
            {{ formatNumber(model.total_tokens) }}
          </span>
          <span class="dashboard-model-row__metric" :data-label="t('dashboard.workspace.spend')">
            {{ formatCredit(model.actual_cost) }}
          </span>
        </li>
      </ol>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { ModelStat, TrendDataPoint } from '@/types'

const props = defineProps<{
  loading: boolean
  startDate: string
  endDate: string
  trend: TrendDataPoint[]
  models: ModelStat[]
}>()

const { t, locale } = useI18n()

const rangeRequests = computed(() => props.trend.reduce((sum, point) => sum + finiteNumber(point.requests), 0))
const rangeTokens = computed(() => props.trend.reduce((sum, point) => sum + finiteNumber(point.total_tokens), 0))
const rangeCost = computed(() => props.trend.reduce((sum, point) => sum + finiteNumber(point.actual_cost), 0))
const topModels = computed(() => [...props.models]
  .filter((model) => model.requests > 0 || model.total_tokens > 0 || model.actual_cost > 0)
  .sort((left, right) => right.requests - left.requests || right.total_tokens - left.total_tokens)
  .slice(0, 6))
const maxModelRequests = computed(() => Math.max(0, ...topModels.value.map((model) => finiteNumber(model.requests))))

function finiteNumber(value: number | null | undefined): number {
  return Number.isFinite(value) ? Number(value) : 0
}

function formatNumber(value: number | null | undefined): string {
  return new Intl.NumberFormat(locale.value.startsWith('zh') ? 'zh-CN' : 'en-US', {
    notation: finiteNumber(value) >= 100_000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(finiteNumber(value))
}

function formatCredit(value: number | null | undefined): string {
  return finiteNumber(value).toFixed(2)
}

function modelShare(requests: number): number {
  if (maxModelRequests.value <= 0) return 0
  return Math.max(3, Math.min(100, (finiteNumber(requests) / maxModelRequests.value) * 100))
}
</script>

<style scoped>
.dashboard-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-work-card);
  background: var(--workspace-card-surface);
  box-shadow: var(--workspace-work-shadow-card);
}

.dashboard-panel-header {
  display: flex;
  min-height: 78px;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: var(--workspace-space-4-5) var(--workspace-space-5);
  border-bottom: 1px solid var(--workspace-border);
}

.dashboard-panel-header h2 {
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.35rem;
}

.dashboard-panel-header p {
  margin-top: 3px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-panel-link {
  display: inline-flex;
  min-height: 34px;
  flex: 0 0 auto;
  align-items: center;
  gap: 5px;
  border-radius: var(--workspace-radius-compact);
  color: var(--workspace-work-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.dashboard-panel-link:hover {
  color: var(--workspace-work-accent-hover);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.dashboard-panel-link:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 3px;
}

.dashboard-usage-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-bottom: 1px solid var(--workspace-border);
}

.dashboard-usage-summary > div {
  min-width: 0;
  padding: 17px 20px;
}

.dashboard-usage-summary > div + div {
  border-left: 1px solid var(--workspace-border);
}

.dashboard-usage-summary span {
  display: block;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
}

.dashboard-usage-summary strong {
  display: block;
  overflow: hidden;
  margin-top: 5px;
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.5rem;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.dashboard-usage-summary > div:last-child strong {
  color: var(--workspace-work-accent-deep);
}

.dashboard-model-list__head,
.dashboard-model-row {
  display: grid;
  grid-template-columns: minmax(12rem, 1.6fr) repeat(3, minmax(5.5rem, 0.55fr));
  align-items: center;
  column-gap: var(--workspace-space-4);
}

.dashboard-model-list__head {
  min-height: 38px;
  padding: 0 20px;
  border-bottom: 1px solid var(--workspace-border);
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.dashboard-model-list__head span:not(:first-child) {
  text-align: right;
}

.dashboard-model-rows {
  padding: 0 20px;
}

.dashboard-model-row {
  min-height: 68px;
  border-bottom: 1px solid var(--workspace-divider);
}

.dashboard-model-row:last-child {
  border-bottom: 0;
}

.dashboard-model-row__identity {
  min-width: 0;
}

.dashboard-model-row__name {
  display: block;
  overflow: hidden;
  color: var(--workspace-work-text);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  line-height: 1.15rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-model-row__track {
  display: block;
  width: min(100%, 13rem);
  height: 3px;
  overflow: hidden;
  margin-top: 8px;
  border-radius: 999px;
  background: var(--workspace-work-track);
}

.dashboard-model-row__track > span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--workspace-work-accent);
}

.dashboard-model-row:nth-child(2) .dashboard-model-row__track > span {
  background: var(--workspace-work-chart-primary);
}

.dashboard-model-row:nth-child(3) .dashboard-model-row__track > span {
  background: var(--workspace-work-chart-secondary);
}

.dashboard-model-row:nth-child(n + 4) .dashboard-model-row__track > span {
  background: var(--workspace-work-accent-border-strong);
}

.dashboard-model-row__metric {
  overflow: hidden;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  font-variant-numeric: tabular-nums;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-model-loading,
.dashboard-model-empty {
  display: flex;
  min-height: 286px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 32px;
  color: var(--workspace-work-text-muted);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
  text-align: center;
}

.dashboard-model-empty :deep(svg) {
  color: var(--workspace-work-accent);
}

.dashboard-model-empty a {
  min-height: 32px;
  color: var(--workspace-work-accent-hover);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-decoration: underline;
  text-underline-offset: 3px;
}

@media (max-width: 767px) {
  .dashboard-panel-link,
  .dashboard-model-empty a {
    min-height: 44px;
  }

  .dashboard-model-empty a {
    display: inline-flex;
    align-items: center;
  }

  .dashboard-model-list__head {
    display: none;
  }

  .dashboard-model-rows {
    padding: 0 18px;
  }

  .dashboard-model-row {
    grid-template-columns: repeat(3, minmax(0, 1fr));
    row-gap: 11px;
    padding: 16px 0;
  }

  .dashboard-model-row__identity {
    grid-column: 1 / -1;
  }

  .dashboard-model-row__metric {
    display: flex;
    min-width: 0;
    flex-direction: column;
    gap: 3px;
    text-align: left;
  }

  .dashboard-model-row__metric::before {
    color: var(--workspace-work-text-muted);
    content: attr(data-label);
    font-size: var(--workspace-type-secondary-size);
    font-weight: var(--workspace-type-secondary-weight);
  }
}

@media (max-width: 479px) {
  .dashboard-panel-header {
    align-items: flex-start;
    padding: 16px 18px;
  }

  .dashboard-usage-summary > div {
    padding: 14px 12px;
  }
}

:global(html.dark) .dashboard-panel {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
  box-shadow: none;
}

:global(html.dark) .dashboard-panel-header,
:global(html.dark) .dashboard-usage-summary,
:global(html.dark) .dashboard-usage-summary > div + div,
:global(html.dark) .dashboard-model-list__head {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-panel-header h2,
:global(html.dark) .dashboard-usage-summary strong,
:global(html.dark) .dashboard-model-row__name {
  color: var(--workspace-dark-text);
}

:global(html.dark) .dashboard-panel-header p,
:global(html.dark) .dashboard-usage-summary span,
:global(html.dark) .dashboard-model-row__metric {
  color: var(--workspace-dark-text-muted);
}

:global(html.dark) .dashboard-model-row {
  border-color: var(--workspace-border);
}

:global(html.dark) .dashboard-model-row__track {
  background: var(--workspace-border);
}

</style>
