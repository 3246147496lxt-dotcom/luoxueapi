import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineComponent } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import OpsDashboard from '../OpsDashboard.vue'
import OpsDashboardHeader from '../components/OpsDashboardHeader.vue'
import opsDashboardHeaderSource from '../components/OpsDashboardHeader.vue?raw'

const opsDashboardStyles = readFileSync(
  resolve(dirname(fileURLToPath(import.meta.url)), '../OpsDashboard.clay.css'),
  'utf8',
)

const mocks = vi.hoisted(() => {
  const route = { query: {} as Record<string, string> }
  const adminSettingsStore = {
    opsMonitoringEnabled: true,
    opsRealtimeMonitoringEnabled: true,
    opsQueryModeDefault: 'auto',
    fetch: vi.fn(),
    setOpsRealtimeMonitoringEnabledLocal: vi.fn(),
  }

  return {
    route,
    adminSettingsStore,
    routerReplace: vi.fn(),
    showError: vi.fn(),
    getAdvancedSettings: vi.fn(),
    getGroups: vi.fn(),
    getDashboardOverview: vi.fn(),
    getDashboardSnapshotV2: vi.fn(),
    getErrorDistribution: vi.fn(),
    getErrorTrend: vi.fn(),
    getLatencyHistogram: vi.fn(),
    getMetricThresholds: vi.fn(),
    getRealtimeTrafficSummary: vi.fn(),
    getThroughputTrend: vi.fn(),
    pauseCountdown: vi.fn(),
    resumeCountdown: vi.fn(),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ replace: mocks.routerReplace }),
}))

vi.mock('@vueuse/core', () => ({
  useDebounceFn: (fn: (...args: unknown[]) => unknown) => fn,
  useIntervalFn: () => ({
    pause: mocks.pauseCountdown,
    resume: mocks.resumeCountdown,
  }),
}))

vi.mock('@/stores', () => ({
  useAdminSettingsStore: () => mocks.adminSettingsStore,
  useAppStore: () => ({ showError: mocks.showError }),
}))

vi.mock('@/api/admin/ops', () => ({
  default: {
    getAdvancedSettings: mocks.getAdvancedSettings,
    getDashboardOverview: mocks.getDashboardOverview,
    getDashboardSnapshotV2: mocks.getDashboardSnapshotV2,
    getErrorDistribution: mocks.getErrorDistribution,
    getErrorTrend: mocks.getErrorTrend,
    getLatencyHistogram: mocks.getLatencyHistogram,
    getMetricThresholds: mocks.getMetricThresholds,
    getRealtimeTrafficSummary: mocks.getRealtimeTrafficSummary,
    getThroughputTrend: mocks.getThroughputTrend,
  },
  opsAPI: {
    getAdvancedSettings: mocks.getAdvancedSettings,
    getDashboardOverview: mocks.getDashboardOverview,
    getDashboardSnapshotV2: mocks.getDashboardSnapshotV2,
    getErrorDistribution: mocks.getErrorDistribution,
    getErrorTrend: mocks.getErrorTrend,
    getLatencyHistogram: mocks.getLatencyHistogram,
    getMetricThresholds: mocks.getMetricThresholds,
    getRealtimeTrafficSummary: mocks.getRealtimeTrafficSummary,
    getThroughputTrend: mocks.getThroughputTrend,
  },
}))

vi.mock('@/api', () => ({
  adminAPI: {
    groups: {
      getAll: mocks.getGroups,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  props: {
    variant: { type: String, default: 'default' },
    contentMode: { type: String, default: 'contained' },
  },
  template: '<div data-testid="app-layout" :data-variant="variant" :data-content-mode="contentMode"><slot /></div>',
})

const OpsDashboardSkeletonStub = defineComponent({
  name: 'OpsDashboardSkeleton',
  props: { fullscreen: Boolean },
  template: '<div data-testid="ops-skeleton" :data-fullscreen="String(fullscreen)" />',
})

const OpsDashboardHeaderStub = defineComponent({
  name: 'OpsDashboardHeader',
  props: {
    overview: { type: Object, default: null },
    platform: { type: String, default: '' },
    groupId: { type: Number, default: null },
    timeRange: { type: String, default: '' },
    queryMode: { type: String, default: '' },
    workspace: { type: String, default: '' },
    resource: { type: String, default: '' },
    loading: Boolean,
    fullscreen: Boolean,
    compact: Boolean,
  },
  emits: [
    'update:timeRange',
    'update:platform',
    'update:group',
    'refresh',
    'openRequestDetails',
    'openErrorDetails',
    'openSettings',
    'openAlertRules',
    'enterFullscreen',
    'exitFullscreen',
  ],
  template: `
    <header
      data-testid="ops-header"
      :data-platform="platform"
      :data-group-id="groupId == null ? '' : String(groupId)"
      :data-time-range="timeRange"
      :data-query-mode="queryMode"
      :data-workspace="workspace"
      :data-resource="resource"
      :data-has-overview="String(overview != null)"
      :data-loading="String(loading)"
      :data-fullscreen="String(fullscreen)"
      :data-compact="String(compact)"
    >
      <button data-testid="refresh" type="button" :disabled="loading" @click="$emit('refresh')">refresh</button>
      <button data-testid="platform" type="button" @click="$emit('update:platform', 'openai')">platform</button>
      <button data-testid="time-range" type="button" @click="$emit('update:timeRange', '24h')">time</button>
      <button data-testid="enter-fullscreen" type="button" @click="$emit('enterFullscreen')">enter</button>
      <button data-testid="exit-fullscreen" type="button" @click="$emit('exitFullscreen')">exit</button>
    </header>
  `,
})

const OpsThroughputTrendChartStub = defineComponent({
  name: 'OpsThroughputTrendChart',
  props: {
    overview: { type: Object, default: null },
    points: { type: Array, default: () => [] },
    loading: Boolean,
    timeRange: { type: String, default: '' },
    fullscreen: Boolean,
  },
  emits: ['selectPlatform', 'selectGroup', 'openDetails'],
  template: `
    <section
      data-testid="throughput-trend"
      :data-has-overview="String(overview != null)"
      :data-average-qps="overview?.qps?.avg ?? ''"
      :data-point-count="String(points.length)"
      :data-loading="String(loading)"
      :data-time-range="timeRange"
      :data-fullscreen="String(fullscreen)"
    >
      <button data-testid="select-group" type="button" @click="$emit('selectGroup', 42)">group</button>
      <button data-testid="open-request-details" type="button" @click="$emit('openDetails')">details</button>
    </section>
  `,
})

const componentStub = (name: string, testId: string) => defineComponent({
  name,
  template: `<section data-testid="${testId}" />`,
})

const OpsWorkbenchShellStub = defineComponent({
  name: 'OpsWorkbenchShell',
  template: '<section data-testid="ops-workbench-shell"><aside><slot name="rail" /></aside><div><slot name="evidence" /></div></section>',
})

const OpsTrafficInvestigationRailStub = defineComponent({
  name: 'OpsTrafficInvestigationRail',
  emits: ['selectSignal'],
  template: '<button data-testid="traffic-investigation-rail" type="button" @click="$emit(\'selectSignal\', \'throughput\')">rail</button>',
})

const OpsAlertEventsCardStub = defineComponent({
  name: 'OpsAlertEventsCard',
  props: {
    enabled: { type: Boolean, default: true },
    platformFilter: { type: String, default: '' },
    groupIdFilter: { type: Number, default: null },
  },
  emits: ['viewRelatedLogs', 'update:platform', 'update:group'],
  template: `
    <section data-testid="alert-events" :data-enabled="String(enabled)">
      <p v-if="!enabled" role="status">admin.ops.workspace.alertsDisabled</p>
      <button
        data-testid="alert-view-related-logs"
        type="button"
        @click="$emit('viewRelatedLogs', {
          alertId: 17,
          firedAt: '2026-07-21T03:00:00.000Z',
          platform: 'openai',
          groupId: 42,
          title: 'Latency alert'
        })"
      >logs</button>
      <slot name="evidence" />
    </section>
  `,
})

const OpsSystemLogTableStub = defineComponent({
  name: 'OpsSystemLogTable',
  props: {
    platformFilter: { type: String, default: '' },
    refreshToken: { type: Number, default: 0 },
    investigationPreset: { type: Object, default: null },
  },
  template: `
    <section
      data-testid="system-log"
      :data-alert-id="investigationPreset?.alertId ?? ''"
      :data-platform="investigationPreset?.platform ?? ''"
      :data-start-time="investigationPreset?.startTime ?? ''"
      :data-end-time="investigationPreset?.endTime ?? ''"
    />
  `,
})

const OpsErrorDistributionChartStub = defineComponent({
  name: 'OpsErrorDistributionChart',
  emits: ['openDetails'],
  template: '<section data-testid="error-distribution" />',
})

const OpsErrorTrendChartStub = defineComponent({
  name: 'OpsErrorTrendChart',
  emits: ['openRequestErrors', 'openUpstreamErrors'],
  template: '<section data-testid="error-trend" />',
})

const OpsErrorDetailsModalStub = defineComponent({
  name: 'OpsErrorDetailsModal',
  props: {
    show: Boolean,
    errorType: { type: String, default: 'request' },
  },
  template: '<div data-testid="error-details-modal" :data-show="String(show)" :data-error-type="errorType" />',
})

const OpsRequestDetailsModalStub = defineComponent({
  name: 'OpsRequestDetailsModal',
  props: { modelValue: Boolean },
  template: '<div data-testid="request-details-modal" :data-show="String(modelValue)" />',
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: Boolean },
  template: '<div data-testid="base-dialog" :data-show="String(show)"><slot /></div>',
})

const globalStubs = {
  AppLayout: AppLayoutStub,
  BaseDialog: BaseDialogStub,
  OpsDashboardHeader: OpsDashboardHeaderStub,
  OpsDashboardSkeleton: OpsDashboardSkeletonStub,
  OpsConcurrencyCard: componentStub('OpsConcurrencyCard', 'concurrency-card'),
  OpsSwitchRateTrendChart: componentStub('OpsSwitchRateTrendChart', 'switch-rate-trend'),
  OpsThroughputTrendChart: OpsThroughputTrendChartStub,
  OpsLatencyChart: componentStub('OpsLatencyChart', 'latency-chart'),
  OpsErrorDistributionChart: OpsErrorDistributionChartStub,
  OpsErrorTrendChart: OpsErrorTrendChartStub,
  OpsOpenAITokenStatsCard: componentStub('OpsOpenAITokenStatsCard', 'openai-token-stats'),
  OpsAlertEventsCard: OpsAlertEventsCardStub,
  OpsSystemLogTable: OpsSystemLogTableStub,
  OpsWorkbenchShell: OpsWorkbenchShellStub,
  OpsTrafficInvestigationRail: OpsTrafficInvestigationRailStub,
  OpsSettingsDialog: componentStub('OpsSettingsDialog', 'settings-dialog'),
  OpsAlertRulesCard: componentStub('OpsAlertRulesCard', 'alert-rules'),
  OpsErrorDetailsModal: OpsErrorDetailsModalStub,
  OpsErrorDetailModal: componentStub('OpsErrorDetailModal', 'error-detail-modal'),
  OpsRequestDetailsModal: OpsRequestDetailsModalStub,
}

const overview = {
  health_score: 100,
  request_count_total: 12,
  request_count_sla: 12,
  success_count: 12,
  error_count_sla: 0,
  business_limited_count: 0,
  upstream_error_count_excl_429_529: 0,
  upstream_429_count: 0,
  upstream_529_count: 0,
  token_consumed: 1_200_000,
  qps: { current: 0.42, peak: 1.84, avg: 0.31 },
  tps: { current: 19_900, peak: 22_400, avg: 18_200 },
}

const trend = {
  points: [],
  by_platform: [],
  top_groups: [],
}

function mountDashboard(): VueWrapper {
  return mount(OpsDashboard, {
    global: {
      stubs: globalStubs,
    },
  })
}

describe('OpsDashboard integration shell', () => {
  it('uses the shared admin canvas in both embedded and light fullscreen modes', () => {
    expect(opsDashboardStyles).toContain(
      'background: var(--app-shell-canvas, var(--lx-clay-canvas));',
    )
    expect(opsDashboardStyles).toContain('html:not(.dark) .ops-fullscreen-shell')
    expect(opsDashboardStyles).toContain('html:not(.dark) body.admin-ops-fullscreen')
    expect(opsDashboardStyles).toContain('--app-shell-canvas: #ffffff;')
  })

  beforeEach(() => {
    mocks.route.query = {}
    mocks.adminSettingsStore.opsMonitoringEnabled = true
    mocks.adminSettingsStore.opsRealtimeMonitoringEnabled = true
    mocks.adminSettingsStore.opsQueryModeDefault = 'auto'

    mocks.routerReplace.mockReset().mockImplementation(async (location: unknown) => {
      if (location && typeof location === 'object' && 'query' in location) {
        mocks.route.query = { ...((location as { query?: Record<string, string> }).query ?? {}) }
      }
    })
    mocks.showError.mockReset()
    mocks.pauseCountdown.mockReset()
    mocks.resumeCountdown.mockReset()
    mocks.adminSettingsStore.fetch.mockReset().mockResolvedValue(undefined)
    mocks.adminSettingsStore.setOpsRealtimeMonitoringEnabledLocal.mockReset()

    mocks.getAdvancedSettings.mockReset().mockResolvedValue({
      display_alert_events: true,
      display_openai_token_stats: false,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30,
    })
    mocks.getGroups.mockReset().mockResolvedValue([])
    mocks.getDashboardOverview.mockReset().mockResolvedValue(overview)
    mocks.getDashboardSnapshotV2.mockReset().mockResolvedValue({
      overview,
      throughput_trend: trend,
      error_trend: { points: [] },
    })
    mocks.getErrorDistribution.mockReset().mockResolvedValue({ total: 0, items: [] })
    mocks.getErrorTrend.mockReset().mockResolvedValue({ points: [] })
    mocks.getLatencyHistogram.mockReset().mockResolvedValue({ buckets: [] })
    mocks.getMetricThresholds.mockReset().mockResolvedValue(null)
    mocks.getRealtimeTrafficSummary.mockReset().mockResolvedValue({
      enabled: true,
      summary: {
        window: '1min',
        start_time: '2026-07-21T04:00:00Z',
        end_time: '2026-07-21T04:01:00Z',
        platform: '',
        group_id: null,
        qps: { current: 0, peak: 0, avg: 0 },
        tps: { current: 0, peak: 0, avg: 0 },
      },
    })
    mocks.getThroughputTrend.mockReset().mockResolvedValue(trend)
  })

  afterEach(() => {
    document.body.classList.remove('admin-ops-fullscreen')
    vi.restoreAllMocks()
  })

  it('keeps a stable page-kind contract and initially mounts only the traffic workspace', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    const page = wrapper.find('[data-admin-page-kind="ops"]')
    expect(page.exists()).toBe(true)
    expect(page.attributes('aria-busy')).toBe('false')
    expect(wrapper.get('[data-testid="app-layout"]').attributes('data-variant')).toBe('home-clay')
    expect(wrapper.get('[data-testid="app-layout"]').attributes('data-content-mode')).toBe('workbench')
    expect(wrapper.find('[data-testid="ops-dashboard-loading"]').exists()).toBe(false)

    expect(wrapper.get('[data-testid="ops-dashboard-body"]').classes()).toContain('ops-dashboard-body')
    expect(wrapper.find('#ops-workspace-tab-resources').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-workspace-panel-resources"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="ops-traffic-section"]').attributes('aria-labelledby')).toBe('ops-traffic-heading')
    expect(wrapper.find('[data-testid="ops-incidents-section"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-log-section"]').exists()).toBe(false)

    for (const workspace of ['traffic', 'incidents', 'diagnostics']) {
      const tab = wrapper.get(`#ops-workspace-tab-${workspace}`)
      const panel = wrapper.get(`#ops-workspace-panel-${workspace}`)
      expect(tab.attributes('aria-controls')).toBe(`ops-workspace-panel-${workspace}`)
      expect(panel.attributes()).toMatchObject({
        role: 'tabpanel',
        'aria-labelledby': `ops-workspace-tab-${workspace}`,
      })
    }

    expect(wrapper.get('#ops-workspace-tab-traffic').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="ops-workspace-panel-traffic"]').attributes('style')).toBeUndefined()
    expect(wrapper.find('[data-testid="ops-header"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-compact')).toBe('true')
    expect(wrapper.find('[data-testid="concurrency-card"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="throughput-trend"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="throughput-trend"]').attributes()).toMatchObject({
      'data-has-overview': 'true',
      'data-average-qps': '0.31',
      'data-point-count': '0',
      'data-loading': 'false',
      'data-time-range': '1h',
      'data-fullscreen': 'false',
    })
    expect(wrapper.find('[data-testid="latency-chart"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="alert-events"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="system-log"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="openai-token-stats"]').exists()).toBe(false)
    expect(mocks.getDashboardSnapshotV2).not.toHaveBeenCalled()
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(1)
    expect(mocks.getLatencyHistogram).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorTrend).not.toHaveBeenCalled()
    expect(mocks.getErrorDistribution).not.toHaveBeenCalled()
  })

  it('loads each chart workspace on first visit and keeps visited workspaces mounted without refetching', async () => {
    mocks.route.query = { platform: 'gemini', tr: '6h', fullscreen: '1' }
    const wrapper = mountDashboard()
    await flushPromises()
    mocks.routerReplace.mockClear()

    expect(wrapper.get('#ops-workspace-tab-traffic').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="ops-traffic-section"]').attributes('aria-labelledby')).toBe('ops-traffic-heading')
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(1)
    expect(mocks.getLatencyHistogram).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorTrend).not.toHaveBeenCalled()
    expect(mocks.getErrorDistribution).not.toHaveBeenCalled()
    await wrapper.get('#ops-workspace-tab-incidents').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="ops-incidents-section"]').exists()).toBe(true)
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getLatencyHistogram).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorTrend).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorDistribution).toHaveBeenCalledTimes(1)
    // One throughput request belongs to traffic and one to the incident switch-rate chart.
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(2)

    await wrapper.get('#ops-workspace-tab-diagnostics').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="system-log"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="ops-traffic-section"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="ops-workspace-panel-traffic"]').attributes('style')).toContain('display: none')
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorTrend).toHaveBeenCalledTimes(1)

    await wrapper.get('#ops-workspace-tab-traffic').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="ops-traffic-section"]').exists()).toBe(true)
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getLatencyHistogram).toHaveBeenCalledTimes(1)
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(2)
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({
      query: {
        platform: 'gemini',
        tr: '6h',
        fullscreen: '1',
      },
    })
  })

  it('passes the active workspace to the header and exposes overview only in traffic', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    const header = () => wrapper.get('[data-testid="ops-header"]')
    expect(header().attributes()).toMatchObject({
      'data-workspace': 'traffic',
      'data-has-overview': 'true',
    })

    await wrapper.get('#ops-workspace-tab-incidents').trigger('click')
    await flushPromises()
    expect(header().attributes()).toMatchObject({
      'data-workspace': 'incidents',
      'data-has-overview': 'false',
    })

    await wrapper.get('#ops-workspace-tab-diagnostics').trigger('click')
    await flushPromises()
    expect(header().attributes()).toMatchObject({
      'data-workspace': 'diagnostics',
      'data-has-overview': 'false',
    })

    await wrapper.get('#ops-workspace-tab-traffic').trigger('click')
    await flushPromises()
    expect(header().attributes()).toMatchObject({
      'data-workspace': 'traffic',
      'data-has-overview': 'true',
    })
  })

  it.each([
    ['removed workspace and resource view', { section: 'resources', resource: 'accounts', retained: 'yes' }],
    ['resource-only deep link', { resource: 'proxies', retained: 'yes' }],
  ])('removes the %s and falls back to traffic', async (_case, query) => {
    mocks.route.query = query

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.find('#ops-workspace-tab-resources').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-workspace-panel-resources"]').exists()).toBe(false)
    expect(wrapper.get('#ops-workspace-tab-traffic').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-workspace')).toBe('traffic')
    expect(mocks.routerReplace).toHaveBeenCalledWith({ query: { retained: 'yes' } })
    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(1)
  })

  it.each([
    ['live', 'traffic'],
    ['quality', 'incidents'],
    ['alerts', 'incidents'],
    ['logs', 'diagnostics'],
  ])('canonicalizes the legacy %s workspace to %s', async (legacyWorkspace, canonicalWorkspace) => {
    mocks.route.query = { section: legacyWorkspace, retained: 'yes' }

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get(`#ops-workspace-tab-${canonicalWorkspace}`).attributes('aria-selected')).toBe('true')
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      query: { section: canonicalWorkspace, retained: 'yes' },
    })
  })

  it('normalizes invalid workspace and resource queries without discarding unrelated query state', async () => {
    mocks.route.query = {
      section: 'unknown',
      resource: 'unknown',
      fullscreen: '1',
      retained: 'yes',
    }

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get('#ops-workspace-tab-traffic').attributes('aria-selected')).toBe('true')
    expect(mocks.routerReplace).toHaveBeenCalledWith({
      query: {
        fullscreen: '1',
        retained: 'yes',
      },
    })
  })

  it('shows a clear alert-workspace message when alert events are disabled', async () => {
    mocks.route.query = { section: 'alerts' }
    mocks.getAdvancedSettings.mockResolvedValue({
      display_alert_events: false,
      display_openai_token_stats: false,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30,
    })

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.get('#ops-workspace-tab-incidents').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="alert-events"]').attributes('data-enabled')).toBe('false')
    expect(wrapper.get('[data-testid="ops-alerts-section"] [role="status"]').text()).toBe(
      'admin.ops.workspace.alertsDisabled',
    )
  })

  it('hands an alert to diagnostics as a materialized one-hour log investigation', async () => {
    mocks.route.query = { section: 'incidents' }
    const wrapper = mountDashboard()
    await flushPromises()

    await wrapper.get('[data-testid="alert-view-related-logs"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('#ops-workspace-tab-diagnostics').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="system-log"]').attributes()).toMatchObject({
      'data-alert-id': '17',
      'data-platform': 'openai',
      'data-start-time': '2026-07-21T02:30:00.000Z',
      'data-end-time': '2026-07-21T03:30:00.000Z',
    })
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({ query: { section: 'diagnostics' } })
  })

  it('shows the dedicated loading skeleton until feature settings resolve', async () => {
    let resolveSettings!: () => void
    mocks.adminSettingsStore.fetch.mockImplementation(() => new Promise<void>((resolve) => {
      resolveSettings = resolve
    }))

    const wrapper = mountDashboard()

    const page = wrapper.get('[data-admin-page-kind="ops"]')
    const loadingStatus = wrapper.get('[data-testid="ops-dashboard-loading"]')
    expect(page.attributes('aria-busy')).toBe('true')
    expect(loadingStatus.attributes()).toMatchObject({
      role: 'status',
      'aria-live': 'polite',
      'aria-busy': 'true',
      'aria-label': 'common.loading',
    })
    expect(wrapper.find('[data-testid="ops-header"]').exists()).toBe(false)

    resolveSettings()
    await flushPromises()

    expect(wrapper.find('[data-testid="ops-dashboard-loading"]').exists()).toBe(false)
    expect(page.attributes('aria-busy')).toBe('false')
    expect(wrapper.find('[data-testid="ops-header"]').exists()).toBe(true)
  })

  it('redirects to settings without calling monitoring endpoints when Ops is disabled', async () => {
    mocks.adminSettingsStore.opsMonitoringEnabled = false

    mountDashboard()
    await flushPromises()

    expect(mocks.routerReplace).toHaveBeenCalledWith('/admin/settings')
    expect(mocks.getDashboardSnapshotV2).not.toHaveBeenCalled()
    expect(mocks.getThroughputTrend).not.toHaveBeenCalled()
  })

  it('hydrates filters and the error detail deep link from the route before the initial request', async () => {
    mocks.route.query = {
      tr: '24h',
      platform: 'anthropic',
      group_id: '12',
      mode: 'raw',
      open_error_details: '1',
      error_type: 'upstream',
    }

    const wrapper = mountDashboard()
    await flushPromises()

    const header = wrapper.get('[data-testid="ops-header"]')
    expect(header.attributes()).toMatchObject({
      'data-platform': 'anthropic',
      'data-group-id': '12',
      'data-time-range': '24h',
      'data-query-mode': 'raw',
    })
    expect(wrapper.get('[data-testid="error-details-modal"]').attributes()).toMatchObject({
      'data-show': 'true',
      'data-error-type': 'upstream',
    })
    expect(wrapper.get('#ops-workspace-tab-incidents').attributes('aria-selected')).toBe('true')
    expect(wrapper.find('[data-testid="ops-incidents-section"]').exists()).toBe(true)
    expect(mocks.getErrorTrend).toHaveBeenCalledWith(
      {
        platform: 'anthropic',
        group_id: 12,
        mode: 'raw',
        time_range: '24h',
      },
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getErrorDistribution).toHaveBeenCalledWith(
      {
        platform: 'anthropic',
        group_id: 12,
        mode: 'raw',
        time_range: '24h',
      },
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getThroughputTrend).toHaveBeenCalledWith(
      expect.objectContaining({
        platform: 'anthropic',
        group_id: 12,
        mode: 'raw',
        start_time: expect.any(String),
        end_time: expect.any(String),
      }),
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getDashboardOverview).not.toHaveBeenCalled()
    expect(mocks.getLatencyHistogram).not.toHaveBeenCalled()
  })

  it('preserves filter and drill-down events while refreshing the matching data contracts', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    await wrapper.get('#ops-workspace-tab-traffic').trigger('click')
    await flushPromises()

    mocks.getDashboardOverview.mockClear()
    mocks.getThroughputTrend.mockClear()
    mocks.getLatencyHistogram.mockClear()
    await wrapper.get('[data-testid="platform"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-platform')).toBe('openai')
    expect(mocks.getDashboardOverview).toHaveBeenLastCalledWith(
      expect.objectContaining({ platform: 'openai' }),
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getThroughputTrend).toHaveBeenLastCalledWith(
      expect.objectContaining({ platform: 'openai' }),
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getLatencyHistogram).toHaveBeenLastCalledWith(
      expect.objectContaining({ platform: 'openai' }),
      { signal: expect.any(AbortSignal) },
    )
    expect(mocks.getErrorTrend).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="select-group"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-group-id')).toBe('42')
    expect(mocks.getDashboardOverview).toHaveBeenLastCalledWith(
      expect.objectContaining({ platform: 'openai', group_id: 42 }),
      { signal: expect.any(AbortSignal) },
    )

    await wrapper.get('[data-testid="open-request-details"]').trigger('click')
    expect(wrapper.get('[data-testid="request-details-modal"]').attributes('data-show')).toBe('true')
  })

  it('disables refresh while a manual refresh is in flight and re-enables it when data settles', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    await wrapper.get('#ops-workspace-tab-traffic').trigger('click')
    await flushPromises()

    mocks.getDashboardOverview.mockClear()
    mocks.getThroughputTrend.mockClear()
    mocks.getLatencyHistogram.mockClear()
    mocks.getErrorTrend.mockClear()
    mocks.getErrorDistribution.mockClear()

    let resolveRefresh!: (value: unknown) => void
    mocks.getDashboardOverview.mockImplementationOnce(() => new Promise((resolve) => {
      resolveRefresh = resolve
    }))

    await wrapper.get('[data-testid="refresh"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-testid="refresh"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-loading')).toBe('true')

    expect(mocks.getDashboardOverview).toHaveBeenCalledTimes(1)
    expect(mocks.getThroughputTrend).toHaveBeenCalledTimes(1)
    expect(mocks.getLatencyHistogram).toHaveBeenCalledTimes(1)
    expect(mocks.getErrorTrend).not.toHaveBeenCalled()
    expect(mocks.getErrorDistribution).not.toHaveBeenCalled()

    resolveRefresh(overview)
    await flushPromises()

    expect(wrapper.get('[data-testid="refresh"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-loading')).toBe('false')
  })

  it('keeps the dashboard usable and reports partial endpoint failures through the app error channel', async () => {
    mocks.route.query = { section: 'traffic' }
    mocks.getLatencyHistogram.mockRejectedValue(new Error('latency unavailable'))

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.find('[data-testid="ops-header"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="latency-chart"]').exists()).toBe(true)
    expect(mocks.showError).toHaveBeenCalledWith('latency unavailable')
  })

  it('announces a page-level failure assertively when the dashboard error channel is populated', async () => {
    const wrapper = mountDashboard()
    await flushPromises()

    const internalInstance = (wrapper.vm as unknown as {
      $: { setupState: { errorMessage: string } }
    }).$
    internalInstance.setupState.errorMessage = 'dashboard unavailable'
    await wrapper.vm.$nextTick()

    const alert = wrapper.get('[data-testid="ops-error-alert"]')
    expect(alert.text()).toBe('dashboard unavailable')
    expect(alert.attributes()).toMatchObject({
      role: 'alert',
      'aria-live': 'assertive',
    })
  })

  it('uses the route-backed fullscreen shell and supports both toolbar and Escape exits', async () => {
    mocks.route.query = { fullscreen: '1', section: 'logs' }

    const wrapper = mountDashboard()
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.get('.ops-fullscreen-shell').classes()).not.toContain('justify-center')
    expect(wrapper.get('[data-admin-page-kind="ops"]').classes()).toContain('ops-dashboard-shell--fullscreen')
    expect(wrapper.get('[data-testid="ops-header"]').attributes('data-fullscreen')).toBe('true')
    expect(wrapper.find('[data-testid="settings-dialog"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="request-details-modal"]').exists()).toBe(false)

    await wrapper.get('[data-testid="exit-fullscreen"]').trigger('click')
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({ query: { section: 'diagnostics' } })

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(mocks.routerReplace).toHaveBeenLastCalledWith({ query: { section: 'diagnostics' } })

    expect(document.body.classList.contains('admin-ops-fullscreen')).toBe(true)
    wrapper.unmount()
    expect(document.body.classList.contains('admin-ops-fullscreen')).toBe(false)
  })

  it('keeps fullscreen entry route-driven so the current filters are not lost', async () => {
    mocks.route.query = { platform: 'gemini', tr: '6h', section: 'quality' }

    const wrapper = mountDashboard()
    await flushPromises()
    mocks.routerReplace.mockClear()

    await wrapper.get('[data-testid="enter-fullscreen"]').trigger('click')

    expect(mocks.routerReplace).toHaveBeenCalledWith({
      query: {
        platform: 'gemini',
        tr: '6h',
        section: 'incidents',
        fullscreen: '1',
      },
    })
  })

  it('announces header status changes and keeps the diagnosis available to keyboard users', async () => {
    const wrapper = mount(OpsDashboardHeader, {
      props: {
        overview: overview as never,
        platform: '',
        groupId: null,
        timeRange: '1h',
        queryMode: 'auto',
        loading: false,
        lastUpdated: new Date('2026-07-21T04:00:00Z'),
        autoRefreshEnabled: true,
        autoRefreshCountdown: 12,
      },
      global: {
        stubs: {
          Select: true,
          HelpTooltip: true,
          BaseDialog: true,
          Icon: true,
        },
      },
    })
    await flushPromises()

    const status = wrapper.get('[data-testid="ops-status-line"]')
    expect(status.attributes()).toMatchObject({
      role: 'status',
      'aria-live': 'polite',
      'aria-atomic': 'true',
    })
    expect(status.classes()).toContain('ops-status-line')

    const statusRule = opsDashboardHeaderSource.match(/\.ops-status-line\s*\{([\s\S]*?)\}/)?.[1] ?? ''
    expect(statusRule).toContain('min-width: 0')
    expect(statusRule).toContain('flex-wrap: wrap')
    expect(statusRule).toContain('overflow-wrap: anywhere')

    await wrapper.get('[data-testid="ops-diagnostic-toggle"]').trigger('click')

    const diagnosis = wrapper.get('#ops-diagnosis-popover')
    expect(diagnosis.attributes('role')).toBe('tooltip')
    expect(wrapper.get('button[aria-describedby="ops-diagnosis-popover"]').attributes('aria-label')).toBe(
      'admin.ops.diagnosis.title',
    )
  })
})
