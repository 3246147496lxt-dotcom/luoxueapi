import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import type {
  DashboardStats,
  ModelStat,
  TrendDataPoint,
  UserSpendingRankingItem,
  UserUsageTrendPoint,
} from '@/types'
import DashboardView from '../DashboardView.vue'

const dashboardClayCss = readFileSync(
  resolve(process.cwd(), 'src/views/admin/DashboardView.clay.css'),
  'utf8',
)

const {
  getSnapshotV2,
  getUserUsageTrend,
  getUserSpendingRanking,
  refreshBatchImageAccess,
  routerPush,
  showError,
} = vi.hoisted(() => ({
  getSnapshotV2: vi.fn(),
  getUserUsageTrend: vi.fn(),
  getUserSpendingRanking: vi.fn(),
  refreshBatchImageAccess: vi.fn(),
  routerPush: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    dashboard: {
      getSnapshotV2,
      getUserUsageTrend,
      getUserSpendingRanking,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('@/composables/useBatchImageAccess', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useBatchImageAccess: () => ({
      canUseBatchImage: ref(false),
      refreshBatchImageAccess,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.dashboard.apiKeyWeeklyActivitySummary': '{enabled} enabled · {weekly} active this week',
    'admin.dashboard.apiKeyWeeklyActivityUnavailableSummary': '{enabled} enabled · Weekly activity unavailable',
    'admin.dashboard.apiKeyComparedToPreviousWeek': 'vs. same period last week',
    'admin.dashboard.apiKeyFirstActive': 'First activity',
    'admin.dashboard.apiKeyPreviousWeekUnavailable': 'No same-period data last week',
    'admin.dashboard.apiKeyNoWeeklyComparison': 'No weekly comparison',
    'admin.dashboard.apiKeyBeijingTime': 'Beijing Time',
    'admin.dashboard.apiKeyWeekComparisonAria': '{currentWindow}: {current} current; {previousWindow}: {previous} previous; {result}; {comparisonLabel}; {timezone}',
    'admin.dashboard.accountHealthRate': 'Health rate',
    'admin.dashboard.accountNoAccounts': 'No accounts to evaluate',
    'admin.dashboard.accountStatusSummary': '{healthy} currently healthy · {error} errors',
    'admin.dashboard.accountHealthAria': 'Currently healthy {healthy} / all non-deleted accounts {total} = {rate}; errors {error}; rate-limited {ratelimit}; overloaded {overload}; effective rate limits, overloads, temporary cooldowns, and expiry auto-pauses are excluded; account quota windows are not included; statuses may overlap and are not added together.',
    'admin.dashboard.requestComparedToPreviousDay': 'vs. same period yesterday',
    'admin.dashboard.requestFirst': 'First requests',
    'admin.dashboard.requestPreviousDayNone': 'No requests in the same period yesterday',
    'admin.dashboard.requestNoComparison': 'No day-over-day comparison',
    'admin.dashboard.requestPreviousDayUnavailable': 'Same-period data for yesterday is unavailable',
    'admin.dashboard.requestDayComparisonAria': '{currentWindow}: {current} current; {previousWindow}: {previous} previous; {result}; {comparisonLabel}; {timezone}',
    'admin.dashboard.newUserComparedToPreviousDay': 'vs. same period yesterday',
    'admin.dashboard.newUserFirst': 'First sign-ups',
    'admin.dashboard.newUserPreviousDayNone': 'No sign-ups in the same period yesterday',
    'admin.dashboard.newUserNoComparison': 'No day-over-day comparison',
    'admin.dashboard.newUserPreviousDayUnavailable': 'Same-period data for yesterday is unavailable',
    'admin.dashboard.newUserDayComparisonAria': '{currentWindow}: {current} current; {previousWindow}: {previous} previous; {result}; {comparisonLabel}; {timezone}',
    'admin.dashboard.totalUsersSummary': 'Total users: {count}',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params: Record<string, string | number> = {}) => {
        const template = messages[key] ?? key
        return Object.entries(params).reduce(
          (value, [name, replacement]) => value.replaceAll(`{${name}}`, String(replacement)),
          template,
        )
      },
      locale: { value: 'en-US' },
    }),
  }
})

const AppLayoutStub = {
  name: 'AppLayout',
  props: {
    variant: { type: String, default: 'default' },
  },
  template: '<div data-testid="app-layout-stub" :data-variant="variant"><slot /></div>',
}

const ModelDistributionChartStub = {
  name: 'ModelDistributionChart',
  props: {
    variant: String,
    modelStats: Array,
    loading: Boolean,
    error: Boolean,
    creditMode: Boolean,
  },
  template: '<div data-testid="model-distribution-chart" />',
}

const CreditAmountStub = {
  name: 'CreditAmount',
  props: {
    value: [String, Number],
    iconSize: String,
  },
  template: '<span data-testid="credit-amount" :data-icon-size="iconSize">{{ value }}</span>',
}

const DashboardTopUsersStub = {
  name: 'DashboardTopUsers',
  props: {
    items: Array,
    trend: Array,
    loading: Boolean,
    error: Boolean,
  },
  emits: ['select', 'retry', 'view-all'],
  template: '<div data-testid="dashboard-top-users-stub"><button data-testid="top-users-view-all" @click="$emit(\'view-all\')" /></div>',
}

const TokenUsageTrendStub = {
  name: 'TokenUsageTrend',
  props: {
    variant: String,
    trendData: Array,
    loading: Boolean,
    error: Boolean,
    creditMode: Boolean,
  },
  template: '<div data-testid="token-usage-trend" />',
}

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const createDashboardStats = (
  overrides: Partial<DashboardStats> = {},
): DashboardStats => ({
  total_users: 8_642,
  today_new_users: 126,
  current_day_new_users: 126,
  previous_day_same_period_new_users: 116,
  active_users: 1_936,
  hourly_active_users: 438,
  stats_updated_at: '2026-07-20T03:42:00Z',
  stats_stale: false,
  total_api_keys: 1_284,
  active_api_keys: 1_106,
  current_week_active_api_keys: 210,
  previous_week_same_period_active_api_keys: 168,
  current_week_start_at: '2026-07-20T00:00:00+08:00',
  current_week_end_at: '2026-07-20T12:00:00+08:00',
  previous_week_same_period_start_at: '2026-07-13T00:00:00+08:00',
  previous_week_same_period_end_at: '2026-07-13T12:00:00+08:00',
  stats_timezone: 'Asia/Shanghai',
  total_accounts: 64,
  normal_accounts: 60,
  healthy_accounts: 58,
  error_accounts: 2,
  ratelimit_accounts: 3,
  overload_accounts: 1,
  total_requests: 28_930_412,
  total_input_tokens: 4_000_000_000,
  total_output_tokens: 1_500_000_000,
  total_cache_creation_tokens: 210_000_000,
  total_cache_read_tokens: 1_100_000_000,
  total_tokens: 6_810_000_000,
  total_cost: 1_234.56,
  total_actual_cost: 987.65,
  total_account_cost: 654.32,
  today_requests: 182_460,
  current_day_requests: 182_460,
  previous_day_same_period_requests: 162_331,
  current_day_start_at: '2026-07-20T00:00:00+08:00',
  current_day_end_at: '2026-07-20T12:00:00+08:00',
  previous_day_same_period_start_at: '2026-07-19T00:00:00+08:00',
  previous_day_same_period_end_at: '2026-07-19T12:00:00+08:00',
  today_input_tokens: 20_000_000,
  today_output_tokens: 8_000_000,
  today_cache_creation_tokens: 1_420_000,
  today_cache_read_tokens: 9_000_000,
  today_tokens: 38_420_000,
  today_cost: 234.56,
  today_actual_cost: 123.45,
  today_account_cost: 67.89,
  average_duration_ms: 842,
  uptime: 86_400,
  rpm: 286,
  tpm: 1_720_000,
  ...overrides,
})

const trend: TrendDataPoint[] = [
  {
    date: '2026-07-20 11:00',
    requests: 120,
    input_tokens: 1_000,
    output_tokens: 500,
    cache_creation_tokens: 100,
    cache_read_tokens: 400,
    total_tokens: 2_000,
    cost: 1.2,
    actual_cost: 0.8,
  },
]

const models: ModelStat[] = [
  {
    model: 'claude-sonnet-4',
    requests: 120,
    input_tokens: 1_000,
    output_tokens: 500,
    cache_creation_tokens: 100,
    cache_read_tokens: 400,
    total_tokens: 2_000,
    cost: 1.2,
    actual_cost: 0.8,
    account_cost: 0.6,
  },
]

const usersTrend: UserUsageTrendPoint[] = [
  {
    date: '2026-07-20 11:00',
    user_id: 7,
    email: 'operator@example.com',
    username: 'operator',
    requests: 20,
    tokens: 2_000,
    cost: 1.2,
    actual_cost: 0.8,
  },
]

const ranking: UserSpendingRankingItem[] = [
  {
    user_id: 7,
    email: 'operator@example.com',
    username: 'operator',
    main_model: 'claude-sonnet-4',
    actual_cost: 12.34,
    requests: 20,
    tokens: 2_000,
  },
]

const successfulSnapshot = () => ({
  stats: createDashboardStats(),
  trend,
  models,
})

const mountDashboard = () => mount(DashboardView, {
  global: {
    stubs: {
      AppLayout: AppLayoutStub,
      LoadingSpinner: true,
      Icon: true,
      DateRangePicker: true,
      Select: true,
      CreditAmount: CreditAmountStub,
      ModelDistributionChart: ModelDistributionChartStub,
      TokenUsageTrend: TokenUsageTrendStub,
      DashboardTopUsers: DashboardTopUsersStub,
    },
  },
})

describe('admin DashboardView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 6, 20, 12, 0, 0))
    setActivePinia(createPinia())

    getSnapshotV2.mockReset()
    getUserUsageTrend.mockReset()
    getUserSpendingRanking.mockReset()
    refreshBatchImageAccess.mockReset()
    routerPush.mockReset()
    showError.mockReset()

    getSnapshotV2.mockResolvedValue(successfulSnapshot())
    getUserUsageTrend.mockResolvedValue({
      trend: usersTrend,
      start_date: '2026-07-19',
      end_date: '2026-07-20',
      granularity: 'hour',
    })
    getUserSpendingRanking.mockResolvedValue({
      ranking,
      total_actual_cost: 18.5,
      total_requests: 30,
      total_tokens: 3_000,
      start_date: '2026-07-19',
      end_date: '2026-07-20',
    })
    refreshBatchImageAccess.mockResolvedValue(false)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('keeps overview summaries 2x2 on compact phones and falls back below 360px', () => {
    const compactPhoneBlock = dashboardClayCss.slice(
      dashboardClayCss.indexOf('@media (max-width: 767px)'),
      dashboardClayCss.indexOf('@media (max-width: 359px)'),
    )
    const narrowPhoneBlock = dashboardClayCss.slice(
      dashboardClayCss.indexOf('@media (max-width: 359px)'),
      dashboardClayCss.indexOf('@media (prefers-reduced-motion: reduce)'),
    )

    expect(compactPhoneBlock).toMatch(
      /\.overview-metrics,\s*\.loading-metrics\s*{\s*grid-template-columns:\s*repeat\(2, minmax\(0, 1fr\)\)/,
    )
    expect(narrowPhoneBlock).toMatch(
      /\.overview-metrics,\s*\.loading-metrics\s*{\s*grid-template-columns:\s*minmax\(0, 1fr\)/,
    )
  })

  it('opts into the home-clay shell and loads all real dashboard data contracts', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    const expectedRange = {
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
    }

    expect(wrapper.get('[data-testid="app-layout-stub"]').attributes('data-variant')).toBe('home-clay')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    expect(getSnapshotV2).toHaveBeenCalledWith({
      ...expectedRange,
      granularity: 'hour',
      include_stats: true,
      include_trend: true,
      include_model_stats: true,
      include_group_stats: false,
      include_users_trend: false,
    })
    expect(getUserUsageTrend).toHaveBeenCalledTimes(1)
    expect(getUserUsageTrend).toHaveBeenCalledWith({
      ...expectedRange,
      granularity: 'hour',
      limit: 12,
    })
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(1)
    expect(getUserSpendingRanking).toHaveBeenCalledWith({
      ...expectedRange,
      limit: 12,
    })
    expect(refreshBatchImageAccess).toHaveBeenCalledTimes(1)

    expect(wrapper.get('[data-testid="dashboard-overview"]').text()).toContain('1,284')
    expect(wrapper.get('[data-testid="dashboard-overview"]').text()).toContain('182,460')
    expect(wrapper.text()).toContain('38.42M')

    const apiKeyMetric = wrapper.get('[data-testid="api-key-overview-metric"]')
    expect(apiKeyMetric.get(':scope > strong').text()).toBe('1,284')
    expect(apiKeyMetric.get('.api-key-activity-summary').text()).toBe('1,106 enabled · 210 active this week')
    const apiKeyComparison = apiKeyMetric.get('[data-testid="api-key-week-comparison"]')
    expect(apiKeyComparison.get('strong').text()).toBe('+25.0%')
    expect(apiKeyComparison.get('small').text()).toBe('vs. same period last week')
    expect(apiKeyComparison.classes()).toContain('positive')
    expect(apiKeyComparison.attributes('title')).toBe(apiKeyComparison.attributes('aria-label'))
    expect(apiKeyComparison.attributes('aria-label')).toContain('07/20/2026')
    expect(apiKeyComparison.attributes('aria-label')).toContain('07/13/2026')
    expect(apiKeyComparison.attributes('aria-label')).toContain('Beijing Time')

    const accountMetric = wrapper.get('[data-testid="account-overview-metric"]')
    expect(accountMetric.get(':scope > strong').text()).toBe('64')
    expect(accountMetric.get('.account-status-summary').text()).toBe('58 currently healthy · 2 errors')
    const accountHealth = accountMetric.get('[data-testid="account-health-comparison"]')
    expect(accountHealth.get('strong').text()).toBe('90.6%')
    expect(accountHealth.get('small').text()).toBe('Health rate')
    expect(accountHealth.classes()).toContain('positive')
    expect(accountHealth.attributes('title')).toBe(accountHealth.attributes('aria-label'))
    expect(accountHealth.attributes('aria-label')).toContain('Currently healthy 58 / all non-deleted accounts 64 = 90.6%')
    expect(accountHealth.attributes('aria-label')).toContain('errors 2')
    expect(accountHealth.attributes('aria-label')).toContain('rate-limited 3')
    expect(accountHealth.attributes('aria-label')).toContain('overloaded 1')
    expect(accountHealth.attributes('aria-label')).toContain('may overlap and are not added together')

    const requestMetric = wrapper.get('[data-testid="request-overview-metric"]')
    expect(requestMetric.get(':scope > strong').text()).toBe('182,460')
    const requestComparison = requestMetric.get('[data-testid="request-day-comparison"]')
    expect(requestComparison.get('strong').text()).toBe('+12.4%')
    expect(requestComparison.get('small').text()).toBe('vs. same period yesterday')
    expect(requestComparison.classes()).toContain('positive')
    expect(requestComparison.attributes('title')).toBe(requestComparison.attributes('aria-label'))
    expect(requestComparison.attributes('aria-label')).toContain('182,460 current')
    expect(requestComparison.attributes('aria-label')).toContain('162,331 previous')
    expect(requestComparison.attributes('aria-label')).toContain('07/20/2026')
    expect(requestComparison.attributes('aria-label')).toContain('07/19/2026')
    expect(requestComparison.attributes('aria-label')).toContain('Beijing Time')

    const newUserMetric = wrapper.get('[data-testid="new-user-overview-metric"]')
    expect(newUserMetric.get(':scope > strong').text()).toBe('126')
    expect(newUserMetric.get(':scope > span').text()).toBe('Total users: 8,642')
    const newUserComparison = newUserMetric.get('[data-testid="new-user-day-comparison"]')
    expect(newUserComparison.get('strong').text()).toBe('+8.6%')
    expect(newUserComparison.get('small').text()).toBe('vs. same period yesterday')
    expect(newUserComparison.classes()).toContain('positive')
    expect(newUserComparison.attributes('title')).toBe(newUserComparison.attributes('aria-label'))
    expect(newUserComparison.attributes('aria-label')).toContain('126 current')
    expect(newUserComparison.attributes('aria-label')).toContain('116 previous')
    expect(newUserComparison.attributes('aria-label')).toContain('07/20/2026')
    expect(newUserComparison.attributes('aria-label')).toContain('07/19/2026')
    expect(newUserComparison.attributes('aria-label')).toContain('Beijing Time')

    const todayPerformance = wrapper.findAll('.performance-metrics article')[0]
    expect(todayPerformance.get('[title="admin.dashboard.actual"] [data-testid="credit-amount"]').text()).toBe('123.45')
    expect(todayPerformance.find('[title="admin.dashboard.actual"]').text()).not.toContain('$')
    expect(todayPerformance.find('[title="admin.dashboard.accountCost"]').text()).toBe('$67.89')
    expect(todayPerformance.find('[title="admin.dashboard.standard"]').text()).toBe('$234.56')

    const totalPerformance = wrapper.findAll('.performance-metrics article')[1]
    expect(totalPerformance.get('[title="admin.dashboard.actual"] [data-testid="credit-amount"]').text()).toBe('987.65')
    expect(totalPerformance.find('[title="admin.dashboard.actual"]').text()).not.toContain('$')
    expect(totalPerformance.find('[title="admin.dashboard.accountCost"]').text()).toBe('$654.32')
    expect(totalPerformance.find('[title="admin.dashboard.standard"]').text()).toBe('$1.23K')

    const modelChart = wrapper.getComponent(ModelDistributionChartStub)
    expect(modelChart.props('variant')).toBe('home-clay')
    expect(modelChart.props('modelStats')).toEqual(models)
    expect(modelChart.props('error')).toBeFalsy()
    expect(modelChart.props('creditMode')).toBe(true)

    const tokenChart = wrapper.getComponent(TokenUsageTrendStub)
    expect(tokenChart.props('variant')).toBe('home-clay')
    expect(tokenChart.props('trendData')).toEqual(trend)
    expect(tokenChart.props('error')).toBeFalsy()
    expect(tokenChart.props('creditMode')).toBe(true)

    const topUsers = wrapper.getComponent(DashboardTopUsersStub)
    expect(topUsers.props('items')).toEqual(ranking)
    expect(topUsers.props('trend')).toEqual(usersTrend)
    expect(topUsers.props('loading')).toBe(false)
    expect(topUsers.props('error')).toBe(false)

    wrapper.unmount()
  })

  it('shows no account health percentage when the account total is zero', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        total_accounts: 0,
        normal_accounts: 0,
        healthy_accounts: 0,
        error_accounts: 0,
        ratelimit_accounts: 0,
        overload_accounts: 0,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const accountMetric = wrapper.get('[data-testid="account-overview-metric"]')
    expect(accountMetric.get(':scope > strong').text()).toBe('0')
    expect(accountMetric.get('.account-status-summary').text()).toBe('0 currently healthy · 0 errors')
    const accountHealth = accountMetric.get('[data-testid="account-health-comparison"]')
    expect(accountHealth.get('strong').text()).toBe('—')
    expect(accountHealth.get('small').text()).toBe('No accounts to evaluate')
    expect(accountHealth.classes()).toContain('neutral')
    expect(accountHealth.attributes('aria-label')).toContain('Currently healthy 0 / all non-deleted accounts 0 = —')
    wrapper.unmount()
  })

  it('keeps the zero-error account summary structurally visible', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        total_accounts: 64,
        normal_accounts: 64,
        healthy_accounts: 64,
        error_accounts: 0,
        ratelimit_accounts: 2,
        overload_accounts: 1,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const accountMetric = wrapper.get('[data-testid="account-overview-metric"]')
    expect(accountMetric.get('.account-status-summary').text()).toBe('64 currently healthy · 0 errors')
    const accountHealth = accountMetric.get('[data-testid="account-health-comparison"]')
    expect(accountHealth.get('strong').text()).toBe('100.0%')
    expect(accountHealth.attributes('aria-label')).toContain('errors 0')
    expect(accountHealth.attributes('aria-label')).toContain('rate-limited 2')
    expect(accountHealth.attributes('aria-label')).toContain('overloaded 1')
    wrapper.unmount()
  })

  it.each([
    { healthy: 80, tone: 'warning' },
    { healthy: 50, tone: 'negative' },
  ])('uses the $tone health tone for $healthy% healthy accounts', async ({ healthy, tone }) => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        total_accounts: 100,
        normal_accounts: 90,
        healthy_accounts: healthy,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const accountHealth = wrapper.get('[data-testid="account-health-comparison"]')
    expect(accountHealth.get('strong').text()).toBe(`${healthy.toFixed(1)}%`)
    expect(accountHealth.classes()).toContain(tone)
    wrapper.unmount()
  })

  it.each([
    {
      current: 4,
      previous: 0,
      value: 'First requests',
      label: 'No requests in the same period yesterday',
      tone: 'first',
    },
    {
      current: 0,
      previous: 0,
      value: '—',
      label: 'No day-over-day comparison',
      tone: 'neutral',
    },
    {
      current: 90,
      previous: 120,
      value: '-25.0%',
      label: 'vs. same period yesterday',
      tone: 'negative',
    },
  ])('renders the request same-period comparison edge case: $value', async ({ current, previous, value, label, tone }) => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        current_day_requests: current,
        previous_day_same_period_requests: previous,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const comparison = wrapper.get('[data-testid="request-day-comparison"]')
    expect(comparison.get('strong').text()).toBe(value)
    expect(comparison.get('small').text()).toBe(label)
    expect(comparison.classes()).toContain(tone)
    expect(comparison.attributes('aria-label')).toContain(`${current} current`)
    expect(comparison.attributes('aria-label')).toContain(`${previous} previous`)
    wrapper.unmount()
  })

  it('treats the legacy request-comparison shape with zero counts and empty windows as unavailable', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        current_day_requests: 0,
        previous_day_same_period_requests: 0,
        current_day_start_at: '',
        current_day_end_at: '',
        previous_day_same_period_start_at: '',
        previous_day_same_period_end_at: '',
        stats_timezone: '',
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get('[data-testid="request-overview-metric"] > strong').text()).toBe('182,460')
    const comparison = wrapper.get('[data-testid="request-day-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('Same-period data for yesterday is unavailable')
    expect(comparison.classes()).toContain('neutral')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    expect(comparison.attributes('aria-label')).toContain('UTC')
    wrapper.unmount()
  })

  it('treats an unparseable request comparison window as unavailable', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({ current_day_start_at: 'not-a-date' }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const comparison = wrapper.get('[data-testid="request-day-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('Same-period data for yesterday is unavailable')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    wrapper.unmount()
  })

  it.each([
    {
      current: 4,
      previous: 0,
      value: 'First sign-ups',
      label: 'No sign-ups in the same period yesterday',
      tone: 'first',
    },
    {
      current: 0,
      previous: 0,
      value: '—',
      label: 'No day-over-day comparison',
      tone: 'neutral',
    },
    {
      current: 90,
      previous: 120,
      value: '-25.0%',
      label: 'vs. same period yesterday',
      tone: 'negative',
    },
    {
      current: 120,
      previous: 120,
      value: '0.0%',
      label: 'vs. same period yesterday',
      tone: 'neutral',
    },
  ])('renders the new-user same-period comparison edge case: $value', async ({ current, previous, value, label, tone }) => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        today_new_users: current,
        current_day_new_users: current,
        previous_day_same_period_new_users: previous,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const metric = wrapper.get('[data-testid="new-user-overview-metric"]')
    expect(metric.get(':scope > strong').text()).toBe(String(current))
    const comparison = metric.get('[data-testid="new-user-day-comparison"]')
    expect(comparison.get('strong').text()).toBe(value)
    expect(comparison.get('small').text()).toBe(label)
    expect(comparison.classes()).toContain(tone)
    expect(comparison.attributes('aria-label')).toContain(`${current} current`)
    expect(comparison.attributes('aria-label')).toContain(`${previous} previous`)
    wrapper.unmount()
  })

  it('treats missing new-user comparison counts as unavailable', async () => {
    const legacyStats: Partial<DashboardStats> = createDashboardStats()
    delete legacyStats.current_day_new_users
    delete legacyStats.previous_day_same_period_new_users
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: legacyStats,
    })

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get('[data-testid="new-user-overview-metric"] > strong').text()).toBe('126')
    const comparison = wrapper.get('[data-testid="new-user-day-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('Same-period data for yesterday is unavailable')
    expect(comparison.classes()).toContain('neutral')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    wrapper.unmount()
  })

  it('treats an unparseable new-user comparison window as unavailable', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({ current_day_end_at: 'not-a-date' }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const comparison = wrapper.get('[data-testid="new-user-day-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('Same-period data for yesterday is unavailable')
    expect(comparison.classes()).toContain('neutral')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    expect(comparison.attributes('aria-label')).toContain('Beijing Time')
    wrapper.unmount()
  })

  it.each([
    {
      current: 4,
      previous: 0,
      value: 'First activity',
      label: 'No same-period data last week',
      tone: 'first',
    },
    {
      current: 0,
      previous: 0,
      value: '—',
      label: 'No weekly comparison',
      tone: 'neutral',
    },
    {
      current: 90,
      previous: 120,
      value: '-25.0%',
      label: 'vs. same period last week',
      tone: 'negative',
    },
  ])('renders the weekly API-key comparison edge case: $value', async ({ current, previous, value, label, tone }) => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        current_week_active_api_keys: current,
        previous_week_same_period_active_api_keys: previous,
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const comparison = wrapper.get('[data-testid="api-key-week-comparison"]')
    expect(comparison.get('strong').text()).toBe(value)
    expect(comparison.get('small').text()).toBe(label)
    expect(comparison.classes()).toContain(tone)
    expect(comparison.attributes('aria-label')).toContain(`${current} current`)
    expect(comparison.attributes('aria-label')).toContain(`${previous} previous`)
    wrapper.unmount()
  })

  it('safely degrades when a legacy cached stats payload lacks all weekly API-key fields', async () => {
    const legacyStats: Partial<DashboardStats> = createDashboardStats()
    const weeklyFields = [
      'current_week_active_api_keys',
      'previous_week_same_period_active_api_keys',
      'current_week_start_at',
      'current_week_end_at',
      'previous_week_same_period_start_at',
      'previous_week_same_period_end_at',
      'stats_timezone',
    ] as const satisfies readonly (keyof DashboardStats)[]
    weeklyFields.forEach((field) => delete legacyStats[field])
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: legacyStats,
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const apiKeyMetric = wrapper.get('[data-testid="api-key-overview-metric"]')
    expect(apiKeyMetric.get(':scope > strong').text()).toBe('1,284')
    expect(apiKeyMetric.get('.api-key-activity-summary').text()).toBe('1,106 enabled · Weekly activity unavailable')
    const comparison = apiKeyMetric.get('[data-testid="api-key-week-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('No weekly comparison')
    expect(comparison.classes()).toContain('neutral')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    expect(comparison.attributes('aria-label')).toContain('UTC')
    wrapper.unmount()
  })

  it('treats the real legacy Go-cache shape with zero counts and empty windows as unavailable', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({
        current_week_active_api_keys: 0,
        previous_week_same_period_active_api_keys: 0,
        current_week_start_at: '',
        current_week_end_at: '',
        previous_week_same_period_start_at: '',
        previous_week_same_period_end_at: '',
        stats_timezone: '',
      }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    const apiKeyMetric = wrapper.get('[data-testid="api-key-overview-metric"]')
    expect(apiKeyMetric.get('.api-key-activity-summary').text()).toBe('1,106 enabled · Weekly activity unavailable')
    const comparison = apiKeyMetric.get('[data-testid="api-key-week-comparison"]')
    expect(comparison.get('strong').text()).toBe('—')
    expect(comparison.get('small').text()).toBe('No weekly comparison')
    expect(comparison.attributes('aria-label')).toContain('— current')
    expect(comparison.attributes('aria-label')).toContain('— previous')
    expect(comparison.attributes('aria-label')).toContain('UTC')
    wrapper.unmount()
  })

  it('shows an initial error state and retries all three live endpoints', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined)
    getSnapshotV2
      .mockRejectedValueOnce(new Error('snapshot unavailable'))
      .mockResolvedValueOnce(successfulSnapshot())

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.find('[data-testid="dashboard-loading"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-error"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="dashboard-overview"]').exists()).toBe(false)
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith('admin.dashboard.failedToLoad')

    await wrapper.get('[data-testid="dashboard-error"] button').trigger('click')
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(2)
    expect(getUserUsageTrend).toHaveBeenCalledTimes(2)
    expect(getUserSpendingRanking).toHaveBeenCalledTimes(2)
    expect(getSnapshotV2.mock.calls[1]?.[0]).toEqual(expect.objectContaining({
      granularity: 'hour',
      include_stats: true,
    }))
    expect(wrapper.find('[data-testid="dashboard-error"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="dashboard-overview"]').exists()).toBe(true)
    expect(showError).toHaveBeenCalledTimes(1)

    consoleError.mockRestore()
    wrapper.unmount()
  })

  it('keeps refresh controls disabled until all three live requests finish', async () => {
    let resolveUsersTrend!: (value: {
      trend: UserUsageTrendPoint[]
      start_date: string
      end_date: string
      granularity: string
    }) => void
    let resolveRanking!: (value: {
      ranking: UserSpendingRankingItem[]
      total_actual_cost: number
      total_requests: number
      total_tokens: number
      start_date: string
      end_date: string
    }) => void

    getUserUsageTrend.mockReturnValueOnce(new Promise((resolve) => {
      resolveUsersTrend = resolve
    }))
    getUserSpendingRanking.mockReturnValueOnce(new Promise((resolve) => {
      resolveRanking = resolve
    }))

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.find('[data-testid="dashboard-overview"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="dashboard-refresh"]').attributes('disabled')).toBeDefined()

    resolveUsersTrend({
      trend: usersTrend,
      start_date: '2026-07-19',
      end_date: '2026-07-20',
      granularity: 'hour',
    })
    resolveRanking({
      ranking,
      total_actual_cost: 18.5,
      total_requests: 30,
      total_tokens: 3_000,
      start_date: '2026-07-19',
      end_date: '2026-07-20',
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="dashboard-refresh"]').attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('uses the stale state consistently across overview and performance', async () => {
    getSnapshotV2.mockResolvedValueOnce({
      ...successfulSnapshot(),
      stats: createDashboardStats({ stats_stale: true }),
    })

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get('.dashboard-health').classes()).toContain('stale')
    expect(wrapper.get('.dashboard-health').text()).toContain('admin.dashboard.dataStale')
    expect(wrapper.get('.dashboard-sync-state').classes()).toContain('stale')
    expect(wrapper.get('.dashboard-sync-state').text()).toContain('admin.dashboard.dataStale')
    wrapper.unmount()
  })

  it('opens the full ranking with the active time range', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    await wrapper.get('[data-testid="top-users-view-all"]').trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      path: '/admin/usage',
      query: {
        tab: 'ranking',
        sort_by: 'actual_cost',
        start_date: '2026-07-19',
        end_date: '2026-07-20',
      },
    })
    wrapper.unmount()
  })
})
