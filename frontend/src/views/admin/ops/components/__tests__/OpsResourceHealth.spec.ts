import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import OpsResourceHealth from '../OpsResourceHealth.vue'
import resourceHealthSource from '../OpsResourceHealth.vue?raw'
import type {
  AlertEvent,
  OpsAccountPoolAnomaly,
  OpsAccountPoolGroupSummary,
  OpsAccountPoolResponse,
  OpsProxyHealthItem,
  OpsProxyHealthResponse,
  OpsResourceView
} from '@/api/admin/ops'

const mocks = vi.hoisted(() => ({
  getAccountPool: vi.fn(),
  getProxyHealth: vi.fn(),
  getProxyHealthDetail: vi.fn(),
  listAlertEvents: vi.fn(),
  reprobeProxyHealth: vi.fn(),
  routerPush: vi.fn()
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getAccountPool: mocks.getAccountPool,
    getProxyHealth: mocks.getProxyHealth,
    getProxyHealthDetail: mocks.getProxyHealthDetail,
    listAlertEvents: mocks.listAlertEvents,
    reprobeProxyHealth: mocks.reprobeProxyHealth
  }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.routerPush })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'en' },
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key
  })
}))

const COLLECTED_AT = '2026-01-02T12:00:00Z'

function anomaly(overrides: Partial<OpsAccountPoolAnomaly> = {}): OpsAccountPoolAnomaly {
  return {
    account_id: 1,
    account_name: 'Manual account',
    platform: 'openai',
    proxy_id: 11,
    group_ids: [42],
    group_names: ['Critical group'],
    status: 'error',
    schedulable: false,
    category: 'actionable',
    primary_reason: 'account_error',
    reason_codes: ['account_error'],
    detail: 'Account is in an error state',
    ...overrides
  }
}

function group(overrides: Partial<OpsAccountPoolGroupSummary> = {}): OpsAccountPoolGroupSummary {
  return {
    group_id: 42,
    group_name: 'Critical group',
    platform: 'openai',
    total_accounts: 2,
    base_schedulable_count: 0,
    actionable_count: 1,
    auto_recovering_count: 1,
    inactive_count: 0,
    manual_unschedulable_count: 0,
    quota_coverage_unknown_count: 0,
    zero_capacity: true,
    low_redundancy: true,
    ...overrides
  }
}

function accountResponse(
  actionable: OpsAccountPoolAnomaly[] = [anomaly()],
  automatic: OpsAccountPoolAnomaly[] = [
    anomaly({
      account_id: 2,
      account_name: 'Rate-limited account',
      proxy_id: undefined,
      group_ids: [43],
      group_names: ['Low redundancy group'],
      status: 'active',
      category: 'auto_recovering',
      primary_reason: 'rate_limited',
      reason_codes: ['rate_limited'],
      detail: 'Rate limit will recover automatically',
      recover_at: '2026-01-02T12:10:00Z',
      remaining_seconds: 600
    })
  ]
): OpsAccountPoolResponse {
  return {
    summary: {
      total_accounts: 6,
      base_schedulable_count: 4,
      actionable_count: actionable.length,
      auto_recovering_count: automatic.length,
      inactive_count: 0,
      manual_unschedulable_count: 0,
      quota_coverage_unknown_count: 1,
      zero_capacity_group_count: 1,
      low_redundancy_group_count: 2,
      low_redundancy_threshold: 1
    },
    platforms: [],
    groups: [
      group(),
      group({
        group_id: 43,
        group_name: 'Low redundancy group',
        total_accounts: 3,
        base_schedulable_count: 1,
        actionable_count: 0,
        zero_capacity: false
      }),
      group({
        group_id: 44,
        group_name: 'Healthy group',
        total_accounts: 3,
        base_schedulable_count: 3,
        actionable_count: 0,
        auto_recovering_count: 0,
        zero_capacity: false,
        low_redundancy: false
      })
    ],
    anomalies: [...actionable, ...automatic],
    actionable_anomalies: actionable,
    auto_recovering_anomalies: automatic,
    group_counts_additive: false,
    collected_at: COLLECTED_AT
  }
}

function proxy(overrides: Partial<OpsProxyHealthItem> = {}): OpsProxyHealthItem {
  return {
    id: 13,
    name: 'Fresh exit',
    protocol: 'socks5',
    lifecycle: 'active',
    health: 'healthy',
    health_reason: 'all_supported_targets_passed',
    account_count: 3,
    active_account_count: 3,
    platforms: [{ platform: 'openai', account_count: 3, coverage: 'supported', status: 'pass' }],
    latency_ms: 120,
    exit_ip: '203.0.113.13',
    country: 'Japan',
    country_code: 'JP',
    city: 'Tokyo',
    expiry_warn_days: 7,
    connectivity_checked_at: '2026-01-02T11:59:00Z',
    quality_checked_at: '2026-01-02T11:58:00Z',
    connectivity_stale: false,
    quality_stale: false,
    quality_score: 96,
    quality_grade: 'A',
    ...overrides
  }
}

function proxyResponse(
  items: OpsProxyHealthItem[] = [
    proxy({
      id: 11,
      name: 'Tokyo exit',
      health: 'failed',
      health_reason: 'connectivity_failed',
      account_count: 4,
      active_account_count: 4
    }),
    proxy({
      id: 12,
      name: 'Stale exit',
      health: 'unknown',
      health_reason: 'quality_stale',
      account_count: 2,
      active_account_count: 2,
      quality_stale: true,
      quality_checked_at: '2026-01-01T10:00:00Z',
      platforms: [
        { platform: 'openai', account_count: 2, coverage: 'supported', status: 'warn' },
        { platform: 'anthropic', account_count: 1, coverage: 'supported', status: 'challenge' },
        { platform: 'gemini', account_count: 1, coverage: 'supported', status: 'fail' },
        { platform: 'bedrock', account_count: 1, coverage: 'uncovered', status: 'uncovered' }
      ]
    }),
    proxy()
  ],
  dataStatus: OpsProxyHealthResponse['data_status'] = 'complete'
): OpsProxyHealthResponse {
  return {
    summary: {
      total: items.length,
      operational: items.filter((item) => item.lifecycle === 'active' || item.lifecycle === 'expiring_soon').length,
      inactive: items.filter((item) => item.lifecycle === 'inactive').length,
      expired: items.filter((item) => item.lifecycle === 'expired').length,
      expiring_soon: items.filter((item) => item.lifecycle === 'expiring_soon').length,
      healthy: items.filter((item) => item.health === 'healthy').length,
      degraded: items.filter((item) => item.health === 'degraded').length,
      suspected_restricted: items.filter((item) => item.health === 'suspected_restricted').length,
      failed: items.filter((item) => item.health === 'failed').length,
      unknown: items.filter((item) => item.health === 'unknown').length,
      affected_accounts: items
        .filter((item) => item.lifecycle !== 'active' || item.health !== 'healthy')
        .reduce((total, item) => total + item.active_account_count, 0)
    },
    items,
    generated_at: COLLECTED_AT,
    data_status: dataStatus
  }
}

function alert(overrides: Partial<AlertEvent> = {}): AlertEvent {
  return {
    id: 90,
    rule_id: 9,
    severity: 'P0',
    status: 'firing',
    title: 'Capacity exhausted',
    fired_at: COLLECTED_AT,
    email_sent: false,
    created_at: COLLECTED_AT,
    ...overrides
  }
}

function mountResource(resource: OpsResourceView = 'overview'): VueWrapper {
  return mount(OpsResourceHealth, {
    props: {
      resource,
      platformFilter: 'openai',
      groupIdFilter: 42,
      refreshToken: 1
    },
    attachTo: document.body
  })
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

beforeEach(() => {
  mocks.routerPush.mockResolvedValue(undefined)
  mocks.getAccountPool.mockResolvedValue(accountResponse())
  mocks.getProxyHealth.mockResolvedValue(proxyResponse())
  mocks.getProxyHealthDetail.mockImplementation((proxyId: number) =>
    Promise.resolve(proxyResponse().items.find((item) => item.id === proxyId))
  )
  mocks.listAlertEvents.mockResolvedValue([
    alert(),
    alert({ id: 91, severity: 'P2' }),
    alert({ id: 92, severity: 'P1', status: 'resolved' })
  ])
  mocks.reprobeProxyHealth.mockResolvedValue(
    proxy({ id: 12, name: 'Stale exit', exit_ip: '203.0.113.12' })
  )
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.clearAllMocks()
})

describe('OpsResourceHealth', () => {
  it('keeps sticky and account workspaces on the shared admin canvas', () => {
    const sharedCanvasUses = resourceHealthSource.match(
      /background: var\(--app-shell-canvas, var\(--lx-clay-canvas\)\);/g,
    )

    expect(sharedCanvasUses).toHaveLength(2)
  })

  it('keeps the compact layout usable at 390px with accessible navigation, records, and drawer targets', () => {
    expect(resourceHealthSource).toContain('@media (max-width: 420px)')
    expect(resourceHealthSource).toContain('@container ops-dashboard (max-width: 1080px)')
    expect(resourceHealthSource).toContain('grid-template-columns: repeat(2, minmax(0, 1fr))')
    expect(resourceHealthSource).toContain('min-height: 44px')
    expect(resourceHealthSource).toContain('role="region"')
    expect(resourceHealthSource).toContain('role="dialog"')
    expect(resourceHealthSource).toContain('<details')
    expect(resourceHealthSource).toContain(
      ".ops-resource-health__row-actions .ops-resource-health__retest-button {\n    display: none;"
    )
    expect(resourceHealthSource).toContain('data-testid="detail-reprobe"')
    expect(resourceHealthSource).not.toContain('24 * 60 * 60')
  })

  it('exposes controlled Overview, Account Pool, and IP Resources navigation without dangling tab panels', async () => {
    const wrapper = mountResource()
    await flushPromises()

    const navigation = wrapper.get('nav')
    const tabs = navigation.findAll('button')
    expect(tabs).toHaveLength(3)
    expect(tabs[0].attributes('aria-current')).toBe('page')
    expect(wrapper.get('[role="region"]').attributes()).toMatchObject({
      id: 'ops-workspace-panel-overview',
      'aria-label': 'admin.ops.resourceHealth.nav.overview',
      tabindex: '0'
    })
    expect(tabs[1].attributes('aria-controls')).toBeUndefined()

    await tabs[1].trigger('click')
    expect(wrapper.emitted('update:resource')).toEqual([['accounts']])

    wrapper.unmount()
  })

  it('renders the four planned summaries and a five-item queue in strict impact order', async () => {
    const wrapper = mountResource()
    await flushPromises()

    const metrics = wrapper.get('[data-testid="overview-metrics"]')
    expect(metrics.findAll('.ops-resource-health__metric')).toHaveLength(4)
    expect(metrics.text()).toContain('admin.ops.resourceHealth.metrics.schedulableAccounts')
    expect(metrics.text()).toContain('admin.ops.resourceHealth.metrics.groupCapacity')
    expect(metrics.text()).toContain('admin.ops.resourceHealth.metrics.proxyHealth')
    expect(metrics.text()).toContain('admin.ops.resourceHealth.metrics.severeAlerts')
    expect(wrapper.find('[data-testid="overview-capacity-viz"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="account-composition-stack"]').attributes('role')).toBe('img')
    expect(wrapper.get('[data-testid="account-composition-stack"]').findAll('i')).toHaveLength(4)
    expect(wrapper.find('[data-testid="account-composition-fallback"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="overview-quota-coverage"]').text()).toContain(
      'admin.ops.resourceHealth.composition.quotaUnknown:{"count":1}'
    )
    expect(wrapper.get('[data-testid="overview-capacity-viz"]').text()).toContain('Jan 02')
    expect(wrapper.find('[data-testid="overview-cockpit"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="overview-capacity-list"]').text()).toContain('Critical group')
    expect(wrapper.get('[data-testid="overview-healthy-group-disclosure"]').attributes('open')).toBeUndefined()
    expect(wrapper.find('[data-testid="account-ledger"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="proxy-ledger"]').exists()).toBe(false)

    const rows = wrapper.get('[data-testid="pending-list"]').findAll('li')
    expect(rows).toHaveLength(5)
    expect(rows.map((row) => row.text())).toEqual([
      expect.stringContaining('Critical group'),
      expect.stringContaining('Tokyo exit'),
      expect.stringContaining('Stale exit'),
      expect.stringContaining('Manual account'),
      expect.stringContaining('Rate-limited account')
    ])
    expect(mocks.getAccountPool).toHaveBeenCalledWith(
      'openai',
      42,
      20,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(mocks.getProxyHealth).toHaveBeenCalledWith(
      {},
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(mocks.listAlertEvents).toHaveBeenCalledWith(
      { limit: 100, status: 'firing', platform: 'openai', group_id: 42 },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )

    wrapper.unmount()
  })

  it('does not normalize an incomplete account classification into a fake 100% stack', async () => {
    const response = accountResponse()
    response.summary.total_accounts = 7
    mocks.getAccountPool.mockResolvedValue(response)

    const wrapper = mountResource()
    await flushPromises()

    expect(wrapper.find('[data-testid="account-composition-stack"]').exists()).toBe(false)
    const fallback = wrapper.get('[data-testid="account-composition-fallback"]')
    expect(fallback.findAll('.ops-resource-health__composition-track')).toHaveLength(4)
    expect(wrapper.get('.ops-resource-health__composition-note').text()).toContain(
      'admin.ops.resourceHealth.composition.incomplete:{"sum":6,"total":7}'
    )

    wrapper.unmount()
  })

  it('reloads all sources when the dashboard refresh token or account scope changes', async () => {
    const wrapper = mountResource()
    await flushPromises()
    mocks.getAccountPool.mockClear()
    mocks.getProxyHealth.mockClear()
    mocks.listAlertEvents.mockClear()

    await wrapper.setProps({
      refreshToken: 2,
      platformFilter: 'gemini',
      groupIdFilter: null
    })
    await flushPromises()

    expect(mocks.getAccountPool).toHaveBeenCalledWith(
      'gemini',
      null,
      20,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(mocks.getProxyHealth).toHaveBeenCalledTimes(1)
    expect(mocks.listAlertEvents).toHaveBeenCalledWith(
      { limit: 100, status: 'firing', platform: 'gemini' },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )

    wrapper.unmount()
  })

  it('loads and refreshes only the active resource view while keeping visited snapshots alive', async () => {
    const wrapper = mountResource('accounts')
    await flushPromises()

    expect(mocks.getAccountPool).toHaveBeenCalledTimes(1)
    expect(mocks.getProxyHealth).not.toHaveBeenCalled()
    expect(mocks.listAlertEvents).not.toHaveBeenCalled()

    mocks.getAccountPool.mockClear()
    await wrapper.setProps({ refreshToken: 2 })
    await flushPromises()
    expect(mocks.getAccountPool).toHaveBeenCalledTimes(1)
    expect(mocks.getProxyHealth).not.toHaveBeenCalled()
    expect(mocks.listAlertEvents).not.toHaveBeenCalled()

    mocks.getAccountPool.mockClear()
    await wrapper.setProps({ resource: 'proxies' })
    await flushPromises()
    expect(mocks.getProxyHealth).toHaveBeenCalledTimes(1)
    expect(mocks.getAccountPool).not.toHaveBeenCalled()
    expect(mocks.listAlertEvents).not.toHaveBeenCalled()

    mocks.getProxyHealth.mockClear()
    await wrapper.setProps({ resource: 'accounts' })
    await flushPromises()
    expect(mocks.getAccountPool).not.toHaveBeenCalled()
    expect(mocks.getProxyHealth).not.toHaveBeenCalled()

    wrapper.unmount()
  })

  it('shows non-additive group capacity, keeps healthy groups collapsed, and deep-links through account details', async () => {
    const wrapper = mountResource('accounts')
    await flushPromises()

    expect(wrapper.get('[data-testid="group-counts-note"]').attributes('role')).toBe('note')
    expect(wrapper.get('[data-testid="capacity-risk-list"]').text()).toContain('Critical group')
    expect(wrapper.get('[data-testid="healthy-group-disclosure"]').attributes('open')).toBeUndefined()
    expect(wrapper.get('[data-testid="quota-coverage-note"]').text()).toContain(
      'admin.ops.resourceHealth.accounts.quotaCoverageUnknown:{"count":1}'
    )

    expect(mocks.getProxyHealth).not.toHaveBeenCalled()
    expect(mocks.listAlertEvents).not.toHaveBeenCalled()

    const segments = wrapper.get('[data-testid="account-ledger"]').findAll('.ops-resource-health__segments button')
    expect(
      wrapper
        .get('[data-testid="capacity-risk-list"]')
        .findAll('th')
        .map((cell) => cell.text())
    ).toEqual([
      'admin.ops.resourceHealth.columns.status',
      'admin.ops.resourceHealth.columns.name',
      'admin.ops.resourceHealth.columns.platform',
      'admin.ops.resourceHealth.columns.capacity',
      'admin.ops.resourceHealth.columns.reason',
      'admin.ops.resourceHealth.columns.actions'
    ])
    expect(wrapper.get('[data-testid="capacity-risk-list"] tbody tr').attributes('data-severity')).toBe('critical')
    expect(wrapper.get('[data-testid="account-ledger"] > .ops-resource-health__ledger-toolbar > button').text()).toContain(
      'admin.ops.resourceHealth.accounts.openAll'
    )
    await segments[1].trigger('click')
    expect(wrapper.get('[data-testid="manual-account-list"]').text()).toContain('Manual account')
    expect(wrapper.get('[data-testid="ledger-preview"]').text()).toContain('Manual account')
    expect(wrapper.get('[data-testid="ledger-preview"]').text()).toContain('admin.ops.resourceHealth.detail.blockingReason')
    await segments[2].trigger('click')
    expect(wrapper.get('[data-testid="automatic-account-list"]').text()).toContain('Rate-limited account')
    await segments[1].trigger('click')

    expect(mocks.getAccountPool).toHaveBeenCalledTimes(1)
    const detailTrigger = wrapper.get('[data-testid="manual-account-list"] button')
    const detailTriggerElement = detailTrigger.element as HTMLButtonElement
    detailTriggerElement.focus()
    await detailTrigger.trigger('click')
    const drawer = wrapper.get('[data-testid="resource-detail-drawer"]')
    expect(drawer.attributes('aria-modal')).toBe('true')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.reason')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.detectedAt')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.countdown')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.relatedObjects')

    await drawer.get('[data-testid="detail-open-management"]').trigger('click')
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: 'AdminAccounts',
      query: {
        health: 'error',
        account_id: '1',
        platform: 'openai',
        group: '42',
        proxy_id: '11'
      }
    })
    expect(mocks.reprobeProxyHealth).not.toHaveBeenCalled()
    expect(document.body.style.overflow).toBe('hidden')

    await drawer.trigger('keydown', { key: 'Escape' })
    await flushPromises()
    expect(wrapper.find('[data-testid="resource-detail-drawer"]').exists()).toBe(false)
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(detailTriggerElement)

    wrapper.unmount()
  })

  it('closes and aborts an open detail drawer when the active resource changes', async () => {
    const proxyDetail = deferred<OpsProxyHealthItem>()
    mocks.getProxyHealthDetail.mockReturnValue(proxyDetail.promise)
    const wrapper = mountResource('proxies')
    await flushPromises()

    await wrapper.get('[data-testid="proxy-issues-list"] button').trigger('click')
    await flushPromises()
    const requestOptions = mocks.getProxyHealthDetail.mock.calls.at(-1)?.[1] as { signal: AbortSignal }
    expect(requestOptions.signal.aborted).toBe(false)
    expect(document.body.style.overflow).toBe('hidden')

    await wrapper.setProps({ resource: 'accounts' })
    await flushPromises()

    expect(wrapper.find('[data-testid="resource-detail-drawer"]').exists()).toBe(false)
    expect(document.body.style.overflow).toBe('')
    expect(requestOptions.signal.aborted).toBe(true)

    wrapper.unmount()
  })

  it('keeps healthy IPs collapsed, opens precise IP details, and re-probes only the selected resource', async () => {
    const wrapper = mountResource('proxies')
    await flushPromises()

    const issueRows = wrapper.get('[data-testid="proxy-issues-list"]').findAll('tbody tr')
    expect(issueRows).toHaveLength(2)
    expect(new Set(issueRows.map((row) => row.get('td strong').text())).size).toBe(2)
    expect(wrapper.get('[data-testid="proxy-issues-list"]').text()).toContain('Tokyo exit')
    expect(wrapper.get('[data-testid="proxy-issues-list"]').text()).toContain('Stale exit')
    expect(mocks.getAccountPool).not.toHaveBeenCalled()
    expect(mocks.listAlertEvents).not.toHaveBeenCalled()
    expect(mocks.getProxyHealthDetail).not.toHaveBeenCalled()

    const segments = wrapper.get('[data-testid="proxy-ledger"]').findAll('.ops-resource-health__segments button')
    await segments[1].trigger('click')
    const staleRow = wrapper.get('[data-testid="proxy-stale-list"] tbody tr')
    await staleRow.get('button').trigger('click')
    await flushPromises()
    const drawer = wrapper.get('[data-testid="resource-detail-drawer"]')
    expect(mocks.getProxyHealthDetail).toHaveBeenCalledWith(
      12,
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.coverage.warn')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.coverage.challenge')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.coverage.fail')
    expect(drawer.text()).toContain('admin.ops.resourceHealth.detail.coverage.uncovered')
    expect(drawer.text()).toContain('bedrock')
    await drawer.get('[data-testid="detail-open-management"]').trigger('click')
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: 'AdminProxies',
      query: {
        open: 'health',
        health: 'stale',
        status: 'active',
        protocol: 'socks5',
        focus_id: '12'
      }
    })

    await wrapper.get('[data-testid="detail-reprobe"]').trigger('click')
    await flushPromises()
    expect(mocks.reprobeProxyHealth).toHaveBeenCalledTimes(1)
    expect(mocks.reprobeProxyHealth).toHaveBeenCalledWith(12)
    expect(wrapper.find('[data-testid="proxy-stale-list"]').exists()).toBe(false)

    wrapper.unmount()
  })

  it('uses controlled lifecycle reasons for inactive and expired IP resources', async () => {
    mocks.getProxyHealth.mockResolvedValue(
      proxyResponse([
        proxy({ id: 20, name: 'Inactive exit', lifecycle: 'inactive', health: 'unknown', health_reason: 'not_checked' }),
        proxy({ id: 21, name: 'Expired exit', lifecycle: 'expired', health: 'unknown', health_reason: 'not_checked' })
      ])
    )

    const wrapper = mountResource('proxies')
    await flushPromises()

    const anomalies = wrapper.get('[data-testid="proxy-issues-list"]').text()
    expect(anomalies).toContain('admin.ops.resourceHealth.proxies.lifecycle.inactive')
    expect(anomalies).toContain('admin.ops.resourceHealth.proxies.lifecycle.expired')

    wrapper.unmount()
  })

  it('degrades each source independently and preserves successful resource data', async () => {
    mocks.listAlertEvents.mockRejectedValue(new Error('alert source unavailable'))

    const wrapper = mountResource()
    await flushPromises()

    expect(wrapper.get('[data-testid="alert-source-error"]').attributes('role')).toBe('alert')
    expect(wrapper.get('[data-testid="alert-source-error"]').text()).toContain('alert source unavailable')
    expect(wrapper.find('[data-testid="overview-metrics"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="pending-list"]').text()).toContain('Tokyo exit')

    wrapper.unmount()
  })

  it.each([
    {
      source: 'account',
      reject: () => mocks.getAccountPool.mockRejectedValueOnce(new Error('account unavailable')),
      errorTestId: 'account-source-error',
      metricTestIds: ['overview-metric-schedulableAccounts', 'overview-metric-groupCapacity']
    },
    {
      source: 'proxy',
      reject: () => mocks.getProxyHealth.mockRejectedValueOnce(new Error('proxy unavailable')),
      errorTestId: 'proxy-source-error',
      metricTestIds: ['overview-metric-proxyHealth']
    },
    {
      source: 'alert',
      reject: () => mocks.listAlertEvents.mockRejectedValueOnce(new Error('alert unavailable')),
      errorTestId: 'alert-source-error',
      metricTestIds: ['overview-metric-severeAlerts']
    }
  ])('marks $source summaries unavailable instead of presenting a healthy zero', async ({ reject, errorTestId, metricTestIds }) => {
    reject()
    const wrapper = mountResource()
    await flushPromises()

    expect(wrapper.find(`[data-testid="${errorTestId}"]`).exists()).toBe(true)
    for (const testId of metricTestIds) {
      const metric = wrapper.get(`[data-testid="${testId}"]`)
      expect(metric.attributes('data-tone')).toBe('neutral')
      expect(metric.get('strong').text()).toBe('—')
      expect(metric.get('small').text()).toBe('admin.ops.resourceHealth.metrics.dataUnavailable')
    }

    wrapper.unmount()
  })

  it('reveals successful snapshots without waiting for a slower independent source', async () => {
    const alertsDeferred = deferred<AlertEvent[]>()
    mocks.listAlertEvents.mockReturnValue(alertsDeferred.promise)

    const wrapper = mountResource()
    await flushPromises()

    expect(wrapper.find('[data-testid="resource-loading"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="overview-metrics"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="ops-resource-health"]').attributes('aria-busy')).toBe('true')

    alertsDeferred.resolve([])
    await flushPromises()
    expect(wrapper.get('[data-testid="ops-resource-health"]').attributes('aria-busy')).toBe('false')

    wrapper.unmount()
  })

  it('shows partial proxy data without discarding lifecycle and account impact', async () => {
    mocks.getProxyHealth.mockResolvedValue(proxyResponse(undefined, 'partial'))

    const wrapper = mountResource('proxies')
    await flushPromises()

    expect(wrapper.get('[data-testid="proxy-data-partial"]').attributes('role')).toBe('status')
    expect(wrapper.find('[data-testid="proxy-metrics"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="proxy-issues-list"]').text()).toContain('Tokyo exit')

    wrapper.unmount()
  })

  it('provides shaped loading and empty states across all three data sources', async () => {
    const accountsDeferred = deferred<OpsAccountPoolResponse>()
    const proxiesDeferred = deferred<OpsProxyHealthResponse>()
    const alertsDeferred = deferred<AlertEvent[]>()
    mocks.getAccountPool.mockReturnValue(accountsDeferred.promise)
    mocks.getProxyHealth.mockReturnValue(proxiesDeferred.promise)
    mocks.listAlertEvents.mockReturnValue(alertsDeferred.promise)

    const wrapper = mountResource()
    expect(wrapper.get('[data-testid="resource-loading"]').attributes('role')).toBe('status')
    expect(wrapper.get('[data-testid="resource-loading"]').findAll('.ops-resource-health__skeleton-metrics span')).toHaveLength(4)

    const emptyAccounts = accountResponse([], [])
    emptyAccounts.summary = {
      ...emptyAccounts.summary,
      total_accounts: 0,
      base_schedulable_count: 0,
      zero_capacity_group_count: 0,
      low_redundancy_group_count: 0
    }
    emptyAccounts.groups = []
    accountsDeferred.resolve(emptyAccounts)
    proxiesDeferred.resolve(proxyResponse([]))
    alertsDeferred.resolve([])
    await flushPromises()

    expect(wrapper.get('[data-testid="pending-empty"]').attributes('role')).toBe('status')
    expect(wrapper.find('[data-testid="pending-list"]').exists()).toBe(false)
    expect(
      wrapper
        .get('[data-testid="overview-metrics"]')
        .findAll('strong')
        .map((value) => value.text())
    ).toEqual(['0', '0 / 0', '0 / 0', '0'])

    wrapper.unmount()
  })
})
