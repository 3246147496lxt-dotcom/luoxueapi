import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const {
  list,
  getStats,
  getSnapshotV2,
  getById,
  getModelStats,
  listErrorLogs,
  listBillingReceipts,
  routeQuery,
} = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    list: vi.fn(),
    getStats: vi.fn(),
    getSnapshotV2: vi.fn(),
    getById: vi.fn(),
    getModelStats: vi.fn(),
    listErrorLogs: vi.fn(),
    listBillingReceipts: vi.fn(),
    routeQuery: {} as Record<string, string | undefined>,
  }
})

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time Range',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour',
  'admin.usage.failedToLoadUser': 'Failed to load user',
  'admin.usage.workspace.columns.auditSubject': 'Audit Subject',
  'admin.usage.workspace.columns.modelMapping': 'Model Mapping (In/Out)',
  'admin.usage.workspace.columns.tokenPayload': 'Token Payload',
  'admin.usage.workspace.columns.actualCharge': 'Actual Charge',
  'admin.usage.workspace.columns.latencyResponse': 'Latency',
  'admin.usage.workspace.columns.channelAccount': 'Channel Account',
  'admin.usage.workspace.columns.timeSerial': 'Time / Request ID',
  'admin.usage.workspace.columns.ipGeo': 'IP Location',
}

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      list,
      getStats,
    },
    dashboard: {
      getSnapshotV2,
      getModelStats,
    },
    users: {
      getById,
    },
  },
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    list: vi.fn(),
    listBillingReceipts,
  },
}))

vi.mock('@/api/admin/ops', () => ({
  listErrorLogs,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/utils/format', () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? '-',
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery
  })
}))

beforeEach(() => {
  Object.keys(routeQuery).forEach((key) => delete routeQuery[key])
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = {
  name: 'UsageFilters',
  props: ['layout', 'mode', 'showActions'],
  template: `
    <div
      data-test="usage-filters"
      :data-layout="layout"
      :data-mode="mode"
      :data-show-actions="showActions"
    >
      <slot name="after-reset" />
    </div>
  `,
}
const CreditModeConsumerStub = {
  props: { creditMode: Boolean },
  template: '<div data-test="credit-mode-consumer" :data-credit-mode="String(creditMode)" />',
}
const UsageTableStub = {
  props: {
    columns: { type: Array, default: () => [] },
    auditLayout: Boolean,
    showUpstreamModelAudit: Boolean,
  },
  emits: ['userClick'],
  template: '<div data-test="usage-table"><button class="user-click" @click="$emit(\'userClick\', 2)">user</button></div>',
}
const UserTokenRankingStub = {
  name: 'UserTokenRanking',
  props: ['initialSortBy'],
  emits: ['select-user'],
  template: '<div data-test="ranking"><button class="pick-user" @click="$emit(\'select-user\', 5, \'rank@test.com\')">pick</button></div>',
}
const ModelDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="model-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}
const GroupDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="group-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}
const AdminBillingReceiptsPanelStub = {
  name: 'AdminBillingReceiptsPanel',
  props: ['startDate', 'endDate'],
  template: '<div data-test="billing-receipts">{{ startDate }}|{{ endDate }}</div>',
}

describe('admin UsageView distribution metric toggles', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getById.mockReset()
    getModelStats.mockReset()

    list.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
    })
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({
      trend: [],
      models: [],
      groups: [],
    })
    getModelStats.mockResolvedValue({ models: [] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('enables snow-credit mode for every actual-cost consumer on the admin usage page', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub,
        UsageStatsCards: CreditModeConsumerStub,
        UsageFilters: UsageFiltersStub,
        UsageTable: CreditModeConsumerStub,
        UsageExportProgress: true,
        UsageCleanupDialog: true,
        UserBalanceHistoryModal: true,
        Pagination: true,
        Select: true,
        DateRangePicker: true,
        Icon: true,
        TokenUsageTrend: CreditModeConsumerStub,
        ModelDistributionChart: CreditModeConsumerStub,
        GroupDistributionChart: CreditModeConsumerStub,
        EndpointDistributionChart: CreditModeConsumerStub,
        UserTokenRanking: true,
        OpsErrorLogTable: true,
        OpsErrorDetailModal: true,
      } },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    const consumers = wrapper.findAll('[data-test="credit-mode-consumer"]')
    expect(consumers).toHaveLength(6)
    expect(consumers.every((consumer) => consumer.attributes('data-credit-mode') === 'true')).toBe(true)
  })

  it('composes the usage workbench as a filter rail and evidence area with ordered analytics', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: true,
          GroupDistributionChart: true,
          EndpointDistributionChart: true,
          UserTokenRanking: true,
          OpsErrorLogTable: true,
          OpsErrorDetailModal: true,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    const workbench = wrapper.get('[data-testid="usage-workbench"]')
    const header = workbench.get('.usage-workbench__header')
    const body = workbench.get('.usage-workbench__body')
    const rail = body.get('[data-testid="usage-filter-rail"]')
    const evidence = body.get('[data-testid="usage-evidence-area"]')

    expect(
      Array.from(body.element.children).map(element => element.getAttribute('data-testid')),
    ).toEqual(['usage-filter-rail', 'usage-evidence-area'])
    expect(rail.element.tagName).toBe('ASIDE')
    expect(evidence.element.tagName).toBe('SECTION')

    expect(header.find('usage-stats-cards-stub').exists()).toBe(true)
    expect(body.find('usage-stats-cards-stub').exists()).toBe(false)

    const filters = rail.getComponent(UsageFiltersStub)
    expect(wrapper.findAllComponents(UsageFiltersStub)).toHaveLength(1)
    expect(filters.props('layout')).toBe('rail')
    expect(filters.props('showActions')).toBe(false)
    expect(evidence.findComponent(UsageFiltersStub).exists()).toBe(false)

    const toolbar = evidence.get('[data-testid="usage-tab-toolbar"]')
    const tabs = toolbar.findAll('[data-testid="usage-detail-tab"]')
    expect(tabs).toHaveLength(4)
    expect(tabs.every(tab => tab.element.tagName === 'BUTTON')).toBe(true)
    expect(tabs.every(tab => tab.attributes('role') === 'tab')).toBe(true)
    expect(tabs.map(tab => tab.attributes('aria-controls'))).toEqual([
      'usage-panel-usage',
      'usage-panel-errors',
      'usage-panel-ranking',
      'usage-panel-billing',
    ])
    expect(tabs[0].attributes('aria-current')).toBe('page')
    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(tabs[0].classes()).toContain('usage-workbench__tab--active')
    expect(tabs.slice(1).every(tab => tab.attributes('aria-current') === undefined)).toBe(true)
    expect(tabs.slice(1).every(tab => tab.attributes('aria-selected') === 'false')).toBe(true)
    expect(tabs.slice(1).every(tab => !tab.classes().includes('usage-workbench__tab--active'))).toBe(true)

    expect(evidence.find('[data-test="usage-table"]').exists()).toBe(true)
    const usageTable = evidence.getComponent(UsageTableStub)
    expect(usageTable.props('auditLayout')).toBe(true)
    expect(usageTable.props('showUpstreamModelAudit')).toBe(true)
    expect((usageTable.props('columns') as Array<{ key: string }>).map(column => column.key)).toEqual([
      'user',
      'model',
      'tokens',
      'cost',
      'latency',
      'account',
      'created_at',
      'ip_address',
    ])
    expect((usageTable.props('columns') as Array<{ label: string }>).map(column => column.label)).toEqual([
      'Audit Subject',
      'Model Mapping (In/Out)',
      'Token Payload',
      'Actual Charge',
      'Latency',
      'Channel Account',
      'Time / Request ID',
      'IP Location',
    ])
    const analytics = evidence.get('[data-testid="usage-analytics"]')
    const chartGrid = analytics.get('.usage-workbench__chart-grid')
    expect(chartGrid.attributes('aria-labelledby')).toBe('usage-analytics-heading')
    expect(
      Array.from(chartGrid.element.children).map(element => element.tagName.toLowerCase()),
    ).toEqual([
      'token-usage-trend-stub',
      'model-distribution-chart-stub',
      'group-distribution-chart-stub',
      'endpoint-distribution-chart-stub',
    ])
  })

  it('keeps previous model stats visible during refresh until new data arrives', async () => {
    // 首次加载返回 A
    getModelStats.mockResolvedValueOnce({ models: [{ model: 'A', total_tokens: 10 }] })

    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, AuditLogModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: ModelDistributionChartStub, GroupDistributionChart: GroupDistributionChartStub,
        EndpointDistributionChart: true, UserTokenRanking: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'A', total_tokens: 10 }])

    // 刷新:让第二次 getModelStats 处于 pending,断言旧数据 A 仍在(不被清空成 [])
    let resolveSecond: (v: any) => void = () => {}
    getModelStats.mockReturnValueOnce(new Promise((res) => { resolveSecond = res }))
    ;(wrapper.vm as any).refreshData()
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'A', total_tokens: 10 }])

    // 新数据到达后替换为 B
    resolveSecond({ models: [{ model: 'B', total_tokens: 20 }] })
    await flushPromises()
    expect((wrapper.vm as any).requestedModelStats).toEqual([{ model: 'B', total_tokens: 20 }])
  })

  it('keeps model and group metric toggles independent without refetching chart data', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: true,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
          UserTokenRanking: true,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))

    const modelChart = wrapper.find('[data-test="model-chart"]')
    const groupChart = wrapper.find('[data-test="group-chart"]')

    expect(modelChart.find('.metric').text()).toBe('tokens')
    expect(groupChart.find('.metric').text()).toBe('tokens')

    await modelChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('tokens')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)

    await groupChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('actual_cost')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
  })

  it('forwards the response-model mismatch filter to every supported usage query', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true,
        EndpointDistributionChart: true, UserTokenRanking: true,
      } },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()
    list.mockClear()
    getStats.mockClear()
    getModelStats.mockClear()
    getSnapshotV2.mockClear()

    const vm = wrapper.vm as any
    vm.filters.upstream_model_mismatch = true
    vm.applyFilters()
    await flushPromises()

    expect(list).toHaveBeenCalledWith(
      expect.objectContaining({ upstream_model_mismatch: true }),
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({ upstream_model_mismatch: true }))
    expect(getModelStats).toHaveBeenCalledWith(expect.objectContaining({ upstream_model_mismatch: true }))
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({ upstream_model_mismatch: true }))
  })
})

describe('admin UsageView handleUserClick', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getById.mockReset()

    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0, total_input_tokens: 0, total_output_tokens: 0,
      total_cache_tokens: 0, total_tokens: 0, total_cost: 0, total_actual_cost: 0, average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('opens user via include_deleted when clicking a usage row user', async () => {
    getById.mockResolvedValue({ id: 2, email: 'd@test.com', deleted_at: '2026-05-28T00:00:00Z' })

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: UsageTableStub,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          AuditLogModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: true,
          GroupDistributionChart: true,
          EndpointDistributionChart: true,
          UserTokenRanking: true,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    await wrapper.find('[data-test="usage-table"] .user-click').trigger('click')
    await flushPromises()

    expect(getById).toHaveBeenCalledWith(2, true)
  })
})

describe('admin UsageView errors tab filter forwarding', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()
    listErrorLogs.mockReset()

    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0, total_input_tokens: 0, total_output_tokens: 0,
      total_cache_tokens: 0, total_tokens: 0, total_cost: 0, total_actual_cost: 0, average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue({ models: [] })
    listErrorLogs.mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('forwards model/account_id/group_id to listErrorLogs on the errors tab', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, AuditLogModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: true, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    // 模拟用户在过滤器里选择了模型/账户/分组
    const vm = wrapper.vm as any
    vm.filters.model = 'gpt-5.3-codex'
    vm.filters.account_id = 7
    vm.filters.group_id = 3
    await flushPromises()

    // 切换到「错误请求」标签（第二个 tab 按钮）触发 loadAdminErrors
    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    await tabs[1].trigger('click')
    await flushPromises()

    const rail = wrapper.get('[data-testid="usage-filter-rail"]')
    const evidence = wrapper.get('[data-testid="usage-evidence-area"]')
    const toolbar = evidence.get('[data-testid="usage-tab-toolbar"]')
    const activeTabs = toolbar.findAll('[data-testid="usage-detail-tab"]')

    expect(rail.element.tagName).toBe('ASIDE')
    expect(activeTabs).toHaveLength(4)
    expect(activeTabs[1].attributes('aria-current')).toBe('page')
    expect(activeTabs[1].attributes('aria-selected')).toBe('true')
    expect(activeTabs[1].classes()).toContain('usage-workbench__tab--active')
    expect(activeTabs[0].attributes('aria-current')).toBeUndefined()
    expect(activeTabs[0].attributes('aria-selected')).toBe('false')
    expect(activeTabs[0].classes()).not.toContain('usage-workbench__tab--active')
    expect(rail.getComponent(UsageFiltersStub).props('layout')).toBe('rail')
    expect(rail.getComponent(UsageFiltersStub).props('mode')).toBe('errors')
    expect(wrapper.find('usage-stats-cards-stub').exists()).toBe(false)
    expect(evidence.find('[data-testid="usage-analytics"]').exists()).toBe(false)
    expect(evidence.find('ops-error-log-table-stub').exists()).toBe(true)

    expect(listErrorLogs).toHaveBeenCalledWith(expect.objectContaining({
      view: 'all',
      model: 'gpt-5.3-codex',
      account_id: 7,
      group_id: 3,
    }))
  })

  it('only lets the latest error request update rows, total, and loading state', async () => {
    let resolveOlder: (value: { items: Array<{ id: number }>; total: number; pages: number }) => void = () => {}
    let resolveLatest: (value: { items: Array<{ id: number }>; total: number; pages: number }) => void = () => {}
    const olderRequest = new Promise<{ items: Array<{ id: number }>; total: number; pages: number }>((resolve) => {
      resolveOlder = resolve
    })
    const latestRequest = new Promise<{ items: Array<{ id: number }>; total: number; pages: number }>((resolve) => {
      resolveLatest = resolve
    })
    listErrorLogs
      .mockImplementationOnce(() => olderRequest)
      .mockImplementationOnce(() => latestRequest)

    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, AuditLogModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: true, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    const vm = wrapper.vm as any
    await wrapper.findAll('[data-testid="usage-detail-tab"]')[1].trigger('click')
    await flushPromises()
    expect(vm.errLoading).toBe(true)

    vm.filters.model = 'latest-model'
    vm.applyFilters()
    await flushPromises()
    expect(listErrorLogs).toHaveBeenCalledTimes(2)

    resolveOlder({ items: [{ id: 1 }], total: 1, pages: 1 })
    await flushPromises()
    expect(vm.errRows).toEqual([])
    expect(vm.errTotal).toBe(0)
    expect(vm.errLoading).toBe(true)

    resolveLatest({ items: [{ id: 2 }], total: 2, pages: 1 })
    await flushPromises()
    expect(vm.errRows).toEqual([{ id: 2 }])
    expect(vm.errTotal).toBe(2)
    expect(vm.errLoading).toBe(false)
  })
})

describe('admin UsageView ranking tab', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()

    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0, total_input_tokens: 0, total_output_tokens: 0,
      total_cache_tokens: 0, total_tokens: 0, total_cost: 0, total_actual_cost: 0, average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue({ models: [] })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('parses dashboard ranking query and forwards actual-cost as the initial ranking sort', async () => {
    Object.assign(routeQuery, {
      tab: 'ranking',
      sort_by: 'actual_cost',
      start_date: '2026-07-19',
      end_date: '2026-07-20',
    })
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: UserTokenRankingStub, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    expect((wrapper.vm as any).activeTab).toBe('ranking')
    expect((wrapper.vm as any).startDate).toBe('2026-07-19')
    expect((wrapper.vm as any).endDate).toBe('2026-07-20')
    expect(wrapper.getComponent(UserTokenRankingStub).props('initialSortBy')).toBe('actual_cost')
  })

  it('mounts ranking lazily and drill-down sets user filter then jumps back to usage tab', async () => {
    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: UserTokenRankingStub, OpsErrorLogTable: true, OpsErrorDetailModal: true,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    // 懒挂载:切到排行 tab 前不渲染
    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(false)

    const tabs = wrapper.findAll('[data-testid="usage-detail-tab"]')
    expect(tabs).toHaveLength(4)
    await tabs[2].trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-test="ranking"]').exists()).toBe(true)

    // 下钻:设置 user_id、切回用量明细 tab 并按新筛选重新拉取列表
    list.mockClear()
    await wrapper.find('[data-test="ranking"] .pick-user').trigger('click')
    await flushPromises()

    expect((wrapper.vm as any).activeTab).toBe('usage')
    expect((wrapper.vm as any).filters.user_id).toBe(5)
    expect(list).toHaveBeenCalledWith(expect.objectContaining({ user_id: 5 }), expect.anything())
  })

  it('opens the read-only chat billing tab from a deep link', async () => {
    Object.assign(routeQuery, {
      tab: 'billing',
      start_date: '2026-07-22',
      end_date: '2026-07-25',
      receipt_id: 'rcpt-chat-42',
    })

    const wrapper = mount(UsageView, {
      global: { stubs: {
        AppLayout: AppLayoutStub, UsageStatsCards: true, UsageFilters: UsageFiltersStub,
        UsageTable: true, UsageExportProgress: true, UsageCleanupDialog: true,
        UserBalanceHistoryModal: true, Pagination: true, Select: true,
        DateRangePicker: true, Icon: true, TokenUsageTrend: true,
        ModelDistributionChart: true, GroupDistributionChart: true, EndpointDistributionChart: true,
        UserTokenRanking: UserTokenRankingStub, OpsErrorLogTable: true, OpsErrorDetailModal: true,
        AdminBillingReceiptsPanel: AdminBillingReceiptsPanelStub,
      } },
    })
    vi.advanceTimersByTime(120)
    await flushPromises()

    expect((wrapper.vm as any).activeTab).toBe('billing')
    expect(wrapper.get('[data-test="billing-receipts"]').text()).toBe(
      '2026-07-22|2026-07-25'
    )
    expect(wrapper.findComponent(UsageFiltersStub).exists()).toBe(false)
  })
})
