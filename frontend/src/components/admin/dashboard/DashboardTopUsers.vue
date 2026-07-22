<template>
  <section
    class="dashboard-top-users"
    data-testid="dashboard-top-users"
    :aria-labelledby="titleId"
    :aria-busy="loading"
  >
    <header class="top-users-heading">
      <div>
        <span>{{ t('admin.dashboard.topUsersEyebrow') }}</span>
        <h2 :id="titleId">{{ t('admin.dashboard.topUsersTitle') }}</h2>
      </div>
      <button type="button" class="top-users-view-all" @click="emit('view-all')">
        {{ t('admin.dashboard.topUsersViewAll') }}
        <Icon name="arrowRight" size="sm" aria-hidden="true" />
      </button>
    </header>

    <div v-if="loading" class="top-users-loading" role="status" aria-live="polite">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <div v-for="index in visibleLimit" :key="index" class="top-users-skeleton-row" aria-hidden="true">
        <i></i><i></i><i></i><i></i>
      </div>
    </div>

    <div v-else-if="error" class="top-users-state" role="alert">
      <span class="top-users-state-icon"><Icon name="exclamationTriangle" size="lg" /></span>
      <div>
        <strong>{{ t('admin.dashboard.topUsersErrorTitle') }}</strong>
        <p>{{ t('admin.dashboard.topUsersErrorDescription') }}</p>
      </div>
      <button type="button" @click="emit('retry')">
        <Icon name="refresh" size="sm" aria-hidden="true" />
        {{ t('admin.dashboard.retry') }}
      </button>
    </div>

    <div v-else-if="displayItems.length === 0" class="top-users-state top-users-empty">
      <span class="top-users-state-icon"><Icon name="users" size="lg" /></span>
      <div>
        <strong>{{ t('admin.dashboard.noUsageRecords') }}</strong>
        <p>{{ t('admin.dashboard.startUsingApi') }}</p>
      </div>
    </div>

    <div v-else class="top-users-table-wrap">
      <table>
        <caption class="sr-only">{{ t('admin.dashboard.topUsersCaption') }}</caption>
        <thead>
          <tr>
            <th scope="col">{{ t('admin.dashboard.topUsersRank') }}</th>
            <th scope="col">{{ t('admin.dashboard.spendingRankingUser') }}</th>
            <th scope="col">{{ t('admin.dashboard.topUsersMainModel') }}</th>
            <th scope="col">{{ t('admin.dashboard.spendingRankingTokens') }}</th>
            <th scope="col">{{ t('admin.dashboard.topUsersActualCost') }}</th>
            <th scope="col">{{ t('admin.dashboard.spendingRankingRequests') }}</th>
            <th scope="col">{{ t('admin.dashboard.topUsersRecentPeriods') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, index) in displayItems" :key="item.user_id">
            <td>
              <span class="top-users-rank" :class="{ top: index < 3 }">{{ index + 1 }}</span>
            </td>
            <td>
              <button
                type="button"
                class="top-users-user"
                :aria-label="t('admin.dashboard.topUsersOpenUser', { user: getUserLabel(item) })"
                @click="emit('select', item)"
              >
                <span class="top-users-avatar" aria-hidden="true">{{ getUserInitial(item) }}</span>
                <span>
                  <strong>{{ getUserLabel(item) }}</strong>
                  <small v-if="getUserSecondaryLabel(item)">{{ getUserSecondaryLabel(item) }}</small>
                </span>
              </button>
            </td>
            <td><span class="top-users-model">{{ item.main_model?.trim() || '—' }}</span></td>
            <td><strong>{{ formatTokens(item.tokens) }}</strong></td>
            <td>{{ formatCost(item.actual_cost) }}</td>
            <td>{{ formatNumber(item.requests) }}</td>
            <td>
              <div
                v-if="getActivity(item.user_id).length"
                class="top-users-bars"
                :aria-label="t('admin.dashboard.topUsersTrendLabel', { user: getUserLabel(item) })"
                role="img"
              >
                <span
                  v-for="(height, activityIndex) in getActivity(item.user_id)"
                  :key="activityIndex"
                  :style="{ height: `${height}%` }"
                  aria-hidden="true"
                ></span>
              </div>
              <span v-else class="top-users-no-trend">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserSpendingRankingItem, UserUsageTrendPoint } from '@/types'

const props = withDefaults(defineProps<{
  items?: UserSpendingRankingItem[]
  trend?: UserUsageTrendPoint[]
  loading?: boolean
  error?: boolean
  visibleLimit?: number
  granularity?: 'day' | 'hour'
}>(), {
  items: () => [],
  trend: () => [],
  loading: false,
  error: false,
  visibleLimit: 6,
  granularity: undefined
})

const emit = defineEmits<{
  select: [item: UserSpendingRankingItem]
  retry: []
  'view-all': []
}>()

const { t, locale } = useI18n()
const titleId = `${useId()}-top-users-title`

const displayItems = computed(() => props.items.slice(0, Math.max(1, props.visibleLimit)))

const formatPeriod = (date: Date, granularity: 'day' | 'hour'): string => {
  const day = [
    date.getFullYear(),
    String(date.getMonth() + 1).padStart(2, '0'),
    String(date.getDate()).padStart(2, '0')
  ].join('-')
  return granularity === 'hour'
    ? `${day} ${String(date.getHours()).padStart(2, '0')}:00`
    : day
}

const getRecentPeriods = (): string[] => {
  const availablePeriods = [...new Set(props.trend.map((point) => point.date))].sort()
  if (!props.granularity || availablePeriods.length === 0) return availablePeriods.slice(-7)

  const latestPeriod = availablePeriods[availablePeriods.length - 1]
  const parsedLatest = new Date(
    props.granularity === 'hour'
      ? `${latestPeriod.replace(' ', 'T')}:00`
      : `${latestPeriod}T00:00:00`
  )
  if (Number.isNaN(parsedLatest.getTime())) return availablePeriods.slice(-7)

  return Array.from({ length: 7 }, (_, index) => {
    const period = new Date(parsedLatest)
    const offset = 6 - index
    if (props.granularity === 'hour') period.setHours(period.getHours() - offset)
    else period.setDate(period.getDate() - offset)
    return formatPeriod(period, props.granularity!)
  })
}

const trendByUser = computed(() => {
  const allPeriods = getRecentPeriods()
  const values = new Map<number, Map<string, number>>()

  props.trend.forEach((point) => {
    if (!values.has(point.user_id)) values.set(point.user_id, new Map())
    values.get(point.user_id)!.set(point.date, Number.isFinite(Number(point.tokens)) ? Number(point.tokens) : 0)
  })

  const result = new Map<number, number[]>()
  values.forEach((userValues, userId) => {
    const periods = allPeriods.map((period) => userValues.get(period) || 0)
    const maximum = Math.max(...periods, 0)
    result.set(userId, periods.map((value) => {
      if (value <= 0 || maximum <= 0) return 0
      return Math.max(14, Math.round((value / maximum) * 100))
    }))
  })
  return result
})

const trendIdentity = computed(() => {
  const result = new Map<number, UserUsageTrendPoint>()
  props.trend.forEach((point) => {
    if (!result.has(point.user_id)) result.set(point.user_id, point)
  })
  return result
})

const getActivity = (userId: number): number[] => trendByUser.value.get(userId) || []

const getUserLabel = (item: UserSpendingRankingItem): string => {
  const username = item.username?.trim() || trendIdentity.value.get(item.user_id)?.username?.trim()
  if (username) return username
  const email = item.email?.trim() || trendIdentity.value.get(item.user_id)?.email?.trim()
  return email || t('admin.dashboard.topUsersFallbackUser', { id: item.user_id })
}

const getUserSecondaryLabel = (item: UserSpendingRankingItem): string => {
  const email = item.email?.trim() || trendIdentity.value.get(item.user_id)?.email?.trim() || ''
  return email && email !== getUserLabel(item) ? email : ''
}

const getUserInitial = (item: UserSpendingRankingItem): string => {
  const label = getUserLabel(item).trim()
  return (Array.from(label)[0] || 'U').toLocaleUpperCase(locale.value)
}

const toFiniteNumber = (value: unknown): number => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const formatNumber = (value: unknown): string => new Intl.NumberFormat(locale.value).format(toFiniteNumber(value))

const formatTokens = (value: unknown): string => new Intl.NumberFormat(locale.value, {
  notation: 'compact',
  maximumFractionDigits: 1
}).format(toFiniteNumber(value))

const formatCost = (value: unknown): string => new Intl.NumberFormat(locale.value, {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
  maximumFractionDigits: 4
}).format(toFiniteNumber(value))
</script>

<style scoped>
.dashboard-top-users {
  min-width: 0;
  padding: 22px 22px 10px;
  border: 1px solid var(--clay-border, var(--lx-clay-border));
  border-radius: 26px;
  color: var(--clay-text, var(--lx-clay-text));
  background: var(--clay-surface, var(--lx-clay-surface));
  box-shadow: var(--clay-soft-shadow, var(--lx-clay-shadow-surface));
}

.top-users-heading {
  display: flex;
  min-height: 50px;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding: 0 3px 16px 0;
}

.top-users-heading span {
  display: block;
  margin-bottom: 5px;
  color: var(--clay-muted, var(--lx-clay-text-muted));
  font-size: 9px;
  font-weight: 900;
  letter-spacing: 0.16em;
}

.top-users-heading h2 {
  margin: 0;
  color: var(--clay-text, var(--lx-clay-text));
  font-family: var(--lx-clay-font-display);
  font-size: 18px;
  font-weight: 900;
  line-height: 1.2;
  letter-spacing: -0.025em;
}

.top-users-view-all,
.top-users-state button {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 7px;
  border: 0;
  border-radius: 10px;
  color: var(--clay-violet, var(--lx-clay-accent));
  background: transparent;
  font-size: 11px;
  font-weight: 800;
  cursor: pointer;
}

.top-users-view-all:hover,
.top-users-state button:hover {
  background: var(--clay-violet-soft, var(--lx-clay-accent-soft));
}

.top-users-table-wrap,
.top-users-loading,
.top-users-state {
  border-top: 1px solid var(--clay-border, var(--lx-clay-border));
}

.top-users-table-wrap {
  overflow-x: auto;
  overscroll-behavior-inline: contain;
}

table {
  width: 100%;
  min-width: 850px;
  border-collapse: collapse;
  table-layout: fixed;
}

th,
td {
  padding: 13px 10px;
  border-bottom: 1px solid var(--clay-border, var(--lx-clay-border));
  text-align: left;
  white-space: nowrap;
}

th {
  color: var(--clay-muted, var(--lx-clay-text-muted));
  font-size: 9px;
  font-weight: 800;
  letter-spacing: 0.05em;
}

td {
  color: var(--clay-secondary, var(--lx-clay-text-secondary));
  font-size: 10px;
}

tbody tr {
  transition: background-color 150ms ease;
}

tbody tr:hover {
  background: var(--clay-surface-soft, var(--lx-clay-surface-subtle));
}

tbody tr:last-child td {
  border-bottom: 0;
}

th:nth-child(1),
td:nth-child(1) { width: 60px; }
th:nth-child(2),
td:nth-child(2) { width: 205px; }
th:nth-child(3),
td:nth-child(3) { width: 160px; }
th:last-child,
td:last-child { width: 112px; }

td > strong {
  color: var(--clay-text, var(--lx-clay-text));
  font-size: 10px;
}

.top-users-rank {
  display: grid;
  width: 25px;
  height: 25px;
  place-items: center;
  border-radius: 9px;
  color: var(--clay-muted, var(--lx-clay-text-muted));
  background: var(--clay-recessed, var(--lx-clay-recessed));
  font-weight: 800;
}

.top-users-rank.top {
  color: var(--clay-violet, var(--lx-clay-accent));
  background: var(--clay-violet-soft, var(--lx-clay-accent-soft));
}

.top-users-user {
  display: flex;
  width: 100%;
  min-height: 36px;
  align-items: center;
  gap: 9px;
  padding: 0;
  overflow: hidden;
  border: 0;
  color: inherit;
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.top-users-user > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.top-users-user strong,
.top-users-user small {
  overflow: hidden;
  text-overflow: ellipsis;
}

.top-users-user strong {
  color: var(--clay-text, var(--lx-clay-text));
  font-size: 10px;
}

.top-users-user small {
  margin-top: 2px;
  color: var(--clay-muted, var(--lx-clay-text-muted));
  font-size: 8px;
}

.top-users-avatar {
  display: grid;
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 10px;
  color: var(--lx-clay-on-accent);
  background: linear-gradient(145deg, var(--lx-clay-accent-gradient-start), var(--clay-violet-deep, var(--lx-clay-accent-deep)));
  box-shadow: 0 5px 12px color-mix(in srgb, var(--clay-violet, var(--lx-clay-accent)) 18%, transparent);
  font-size: 9px;
  font-weight: 800;
}

.top-users-model {
  display: inline-block;
  max-width: 140px;
  padding: 4px 7px;
  overflow: hidden;
  border: 1px solid var(--clay-border, var(--lx-clay-border));
  border-radius: 8px;
  color: var(--clay-secondary, var(--lx-clay-text-secondary));
  background: var(--clay-surface-soft, var(--lx-clay-surface-subtle));
  font-size: 9px;
  text-overflow: ellipsis;
}

.top-users-bars {
  display: flex;
  height: 28px;
  align-items: flex-end;
  gap: 3px;
}

.top-users-bars span {
  width: 5px;
  min-height: 4px;
  border-radius: 3px 3px 1px 1px;
  background: linear-gradient(180deg, var(--lx-clay-accent-gradient-start), var(--clay-violet-deep, var(--lx-clay-accent-deep)));
  opacity: 0.78;
}

.top-users-no-trend {
  color: var(--clay-muted, var(--lx-clay-text-muted));
}

.top-users-loading {
  padding: 3px 10px 0;
}

.top-users-skeleton-row {
  display: grid;
  min-height: 54px;
  align-items: center;
  grid-template-columns: 38px minmax(140px, 1fr) 120px 90px;
  gap: 18px;
  border-bottom: 1px solid var(--clay-border, var(--lx-clay-border));
}

.top-users-skeleton-row:last-child {
  border-bottom: 0;
}

.top-users-skeleton-row i {
  height: 10px;
  border-radius: 6px;
  background: var(--clay-recessed, var(--lx-clay-recessed));
  animation: top-users-pulse 1.25s ease-in-out infinite alternate;
}

.top-users-skeleton-row i:first-child {
  width: 25px;
  height: 25px;
  border-radius: 9px;
}

.top-users-state {
  display: flex;
  min-height: 160px;
  align-items: center;
  justify-content: center;
  gap: 14px;
  padding: 26px;
  color: var(--clay-secondary, var(--lx-clay-text-secondary));
  text-align: left;
}

.top-users-state-icon {
  display: grid;
  width: 42px;
  height: 42px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 14px;
  color: var(--clay-violet, var(--lx-clay-accent));
  background: var(--clay-violet-soft, var(--lx-clay-accent-soft));
}

.top-users-state strong {
  display: block;
  color: var(--clay-text, var(--lx-clay-text));
  font-size: 12px;
}

.top-users-state p {
  margin: 3px 0 0;
  font-size: 10px;
}

.top-users-empty {
  text-align: center;
}

.top-users-state button {
  border: 1px solid var(--clay-border, var(--lx-clay-border));
  background: var(--clay-surface-soft, var(--lx-clay-surface-subtle));
}

.top-users-user:focus-visible,
.top-users-view-all:focus-visible,
.top-users-state button:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--clay-violet, var(--lx-clay-accent)) 42%, transparent);
  outline-offset: 3px;
}

@keyframes top-users-pulse {
  to { opacity: 0.42; }
}

@media (max-width: 767px) {
  .dashboard-top-users {
    padding-right: 17px;
    padding-left: 17px;
    border-radius: 22px;
  }

  .top-users-heading {
    gap: 10px;
  }

  .top-users-view-all {
    flex: 0 0 auto;
  }

  .top-users-state {
    min-height: 180px;
    flex-direction: column;
    text-align: center;
  }

  .top-users-user small {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .dashboard-top-users *,
  .dashboard-top-users *::before,
  .dashboard-top-users *::after {
    scroll-behavior: auto !important;
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>
