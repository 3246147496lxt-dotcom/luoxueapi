import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OpsDashboardHeader from '../OpsDashboardHeader.vue'

const { mockGetGroups, mockGetRealtimeTrafficSummary, mockSetRealtimeEnabled } = vi.hoisted(() => ({
  mockGetGroups: vi.fn(),
  mockGetRealtimeTrafficSummary: vi.fn(),
  mockSetRealtimeEnabled: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    groups: {
      getAll: (...args: unknown[]) => mockGetGroups(...args)
    }
  }
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRealtimeTrafficSummary: (...args: unknown[]) => mockGetRealtimeTrafficSummary(...args)
  }
}))

vi.mock('@/stores', () => ({
  useAdminSettingsStore: () => ({
    opsRealtimeMonitoringEnabled: true,
    setOpsRealtimeMonitoringEnabledLocal: mockSetRealtimeEnabled
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.ops.autoRefreshRemaining') return `剩余 ${params?.seconds}s`
        return key
      }
    })
  }
})

const SelectStub = defineComponent({
  name: 'SelectControlStub',
  props: {
    modelValue: { type: [String, Number, Boolean], default: null },
    options: { type: Array, default: () => [] },
    ariaLabel: { type: String, default: '' }
  },
  emits: ['update:modelValue'],
  template: '<button type="button" class="select-stub" :aria-label="ariaLabel">{{ modelValue }}</button>'
})

const HelpTooltipStub = defineComponent({
  name: 'HelpTooltip',
  props: { content: { type: String, default: '' } },
  template: '<span class="help-tooltip-stub" />'
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' }
  },
  emits: ['close'],
  template: '<div v-if="show" class="base-dialog-stub"><slot /></div>'
})

const IconStub = defineComponent({
  name: 'Icon',
  props: { name: { type: String, required: true } },
  template: '<i class="icon-stub" :data-icon="name" />'
})

const overview = {
  start_time: '2026-07-21T00:00:00Z',
  end_time: '2026-07-21T01:00:00Z',
  platform: '',
  group_id: null,
  health_score: 96,
  success_count: 995,
  error_count_total: 10,
  business_limited_count: 5,
  error_count_sla: 5,
  request_count_total: 1005,
  request_count_sla: 1000,
  token_consumed: 120000,
  sla: 0.995,
  error_rate: 0.005,
  upstream_error_rate: 0.002,
  upstream_error_count_excl_429_529: 2,
  upstream_429_count: 1,
  upstream_529_count: 0,
  qps: { current: 8, peak: 16, avg: 9 },
  tps: { current: 1200, peak: 1800, avg: 1300 },
  duration: { p50_ms: 100, p90_ms: 180, p95_ms: 220, p99_ms: 400, avg_ms: 130, max_ms: 900 },
  ttft: { p50_ms: 80, p90_ms: 150, p95_ms: 190, p99_ms: 260, avg_ms: 100, max_ms: 500 },
  system_metrics: {
    id: 1,
    created_at: '2026-07-21T01:00:00Z',
    window_minutes: 1,
    cpu_usage_percent: 20,
    memory_usage_percent: 30,
    memory_used_mb: 300,
    memory_total_mb: 1000,
    db_ok: true,
    redis_ok: true,
    db_max_open_conns: 100,
    redis_pool_size: 100,
    redis_conn_total: 8,
    redis_conn_idle: 6,
    db_conn_active: 3,
    db_conn_idle: 7,
    db_conn_waiting: 0,
    goroutine_count: 120,
    concurrency_queue_depth: 2,
    account_switch_count: 0
  },
  job_heartbeats: []
}

const global = {
  stubs: {
    Select: SelectStub,
    HelpTooltip: HelpTooltipStub,
    BaseDialog: BaseDialogStub,
    Icon: IconStub
  }
}

function mountHeader(overrides: Record<string, unknown> = {}) {
  return mount(OpsDashboardHeader, {
    props: {
      overview,
      platform: '',
      groupId: null,
      timeRange: '1h',
      queryMode: 'auto',
      loading: false,
      lastUpdated: new Date('2026-07-21T01:02:03Z'),
      autoRefreshEnabled: true,
      autoRefreshCountdown: 24,
      fullscreen: false,
      ...overrides
    },
    global
  })
}

describe('OpsDashboardHeader', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetGroups.mockResolvedValue([
      { id: 7, name: 'OpenAI 主组', platform: 'openai' }
    ])
    mockGetRealtimeTrafficSummary.mockResolvedValue({
      enabled: true,
      summary: {
        window: '1min',
        start_time: '2026-07-21T01:01:00Z',
        end_time: '2026-07-21T01:02:00Z',
        platform: '',
        group_id: null,
        qps: { current: 12.5, peak: 20, avg: 10 },
        tps: { current: 1600, peak: 2000, avg: 1400 }
      }
    })
  })

  it('状态行对辅助技术实时播报，并保留可换行的稳定结构', async () => {
    const wrapper = mountHeader()
    await flushPromises()

    const statusLine = wrapper.get('[data-testid="ops-status-line"]')
    expect(statusLine.attributes()).toMatchObject({
      role: 'status',
      'aria-live': 'polite',
      'aria-atomic': 'true'
    })
    expect(statusLine.classes()).toContain('ops-status-line')
    expect(statusLine.text()).toContain('common.refresh')
    expect(statusLine.text()).toContain('剩余 24s')
  })

  it('首要信号带展示真实 SLA、错误、吞吐和并发排队数据', async () => {
    const wrapper = mountHeader()
    await flushPromises()

    const signalStrip = wrapper.get('[data-testid="ops-signal-strip"]')
    expect(signalStrip.text()).toContain('99.500%')
    expect(signalStrip.text()).toContain('0.50%')
    expect(signalStrip.text()).toContain('12.5 QPS')
    expect(signalStrip.text()).toContain('1600.0 admin.ops.tps')
    expect(signalStrip.text()).toContain('2')

    const actions = signalStrip.findAll('button')
    await actions[0].trigger('click')
    await actions[1].trigger('click')
    expect(wrapper.emitted('openRequestDetails')).toBeTruthy()
    expect(wrapper.emitted('openErrorDetails')?.[0]).toEqual(['request'])
  })

  it('二级诊断与系统指标默认折叠，并通过可访问控件按需展开', async () => {
    const wrapper = mountHeader()
    await flushPromises()

    const toggle = wrapper.get('[data-testid="ops-diagnostic-toggle"]')
    const details = wrapper.get('[data-testid="ops-diagnostic-details"]')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('ops-diagnostic-details')
    expect(details.attributes('id')).toBe('ops-diagnostic-details')
    expect(details.attributes('style')).toContain('display: none')

    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(details.attributes('style') ?? '').not.toContain('display: none')
    expect(details.text()).toContain('admin.ops.systemHealth')
  })

  it('无请求样本时将 SLA 与错误率显示为中性的无数据状态', async () => {
    const noSampleOverview = {
      ...overview,
      success_count: 0,
      error_count_total: 0,
      error_count_sla: 0,
      request_count_total: 0,
      request_count_sla: 0,
      sla: 0,
      error_rate: 0,
      upstream_error_rate: 0,
      qps: { current: 0, peak: 0, avg: 0 }
    }
    const wrapper = mountHeader({ overview: noSampleOverview })
    await flushPromises()

    const slaSignal = wrapper.get('[data-testid="ops-signal-sla"]')
    const requestErrorSignal = wrapper.get('[data-testid="ops-signal-request-errors"]')
    expect(slaSignal.get('strong').text()).toBe('-')
    expect(slaSignal.get('strong').classes()).toContain('metric-tone-neutral')
    expect(requestErrorSignal.get('strong').text()).toBe('-')
    expect(requestErrorSignal.get('strong').classes()).toContain('metric-tone-neutral')

    await wrapper.get('[data-testid="ops-diagnostic-toggle"]').trigger('click')
    for (const testId of ['ops-detail-sla', 'ops-detail-request-errors', 'ops-detail-upstream-errors']) {
      const value = wrapper.get(`[data-testid="${testId}"]`).get('.text-3xl')
      expect(value.text()).toBe('-')
      expect(value.classes()).toContain('metric-tone-neutral')
    }
  })

  it('保留筛选、刷新、告警、设置与全屏事件契约', async () => {
    const wrapper = mountHeader()
    await flushPromises()

    const selects = wrapper.findAllComponents(SelectStub)
    await selects[0].vm.$emit('update:modelValue', 'openai')
    await selects[1].vm.$emit('update:modelValue', 7)
    await selects[2].vm.$emit('update:modelValue', '30m')

    expect(wrapper.emitted('update:platform')?.[0]).toEqual(['openai'])
    expect(wrapper.emitted('update:group')?.[0]).toEqual([7])
    expect(wrapper.emitted('update:timeRange')?.[0]).toEqual(['30m'])

    await wrapper.get('.ops-command-button--primary').trigger('click')
    await wrapper.get('button[title="admin.ops.alertRules.title"]').trigger('click')
    await wrapper.get('button[title="admin.ops.settings.title"]').trigger('click')
    await wrapper.get('button[title="admin.ops.fullscreen.enter"]').trigger('click')

    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(wrapper.emitted('openAlertRules')).toHaveLength(1)
    expect(wrapper.emitted('openSettings')).toHaveLength(1)
    expect(wrapper.emitted('enterFullscreen')).toHaveLength(1)
  })

  it('资源工作区不请求实时流量，精简时间与告警控件但保留刷新', async () => {
    const wrapper = mountHeader({ workspace: 'resources', overview: null })
    await flushPromises()

    expect(mockGetRealtimeTrafficSummary).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="ops-signal-strip"]').exists()).toBe(false)

    const selects = wrapper.findAllComponents(SelectStub)
    expect(selects).toHaveLength(2)
    expect(selects.map((select) => select.attributes('aria-label'))).toEqual([
      'admin.ops.errorLog.platform',
      'admin.ops.errorLog.group'
    ])
    expect(wrapper.find('button[title="admin.ops.alertRules.title"]').exists()).toBe(false)

    await wrapper.get('.ops-command-button--primary').trigger('click')
    await flushPromises()
    expect(wrapper.emitted('refresh')).toHaveLength(1)
    expect(mockGetRealtimeTrafficSummary).not.toHaveBeenCalled()

    await wrapper.setProps({ workspace: 'traffic', overview })
    await flushPromises()

    expect(mockGetRealtimeTrafficSummary).toHaveBeenCalledTimes(1)
    expect(wrapper.findAllComponents(SelectStub)).toHaveLength(3)
    expect(wrapper.find('[data-testid="ops-signal-strip"]').exists()).toBe(true)
  })

  it('IP 资源页隐藏不会影响全局 IP 快照的平台和分组筛选', async () => {
    const wrapper = mountHeader({ workspace: 'resources', resource: 'proxies', overview: null })
    await flushPromises()

    expect(wrapper.findAllComponents(SelectStub)).toHaveLength(0)
    expect(wrapper.find('.ops-command-button--primary').exists()).toBe(true)

    await wrapper.setProps({ resource: 'accounts' })
    await flushPromises()
    expect(wrapper.findAllComponents(SelectStub)).toHaveLength(2)
  })

  it('诊断入口保持键盘可达，并在全屏模式提供明确退出操作', async () => {
    const regular = mountHeader()
    await flushPromises()
    await regular.get('[data-testid="ops-diagnostic-toggle"]').trigger('click')
    expect(regular.get('[aria-describedby="ops-diagnosis-popover"]').element.tagName).toBe('BUTTON')
    expect(regular.get('#ops-diagnosis-popover').attributes('role')).toBe('tooltip')

    const fullscreen = mountHeader({ fullscreen: true })
    await flushPromises()
    expect(fullscreen.find('[data-testid="ops-command-bar"]').exists()).toBe(false)
    expect(fullscreen.get('[data-testid="ops-diagnostic-toggle"]').attributes('aria-expanded')).toBe('false')
    await fullscreen.get('button[title="common.close"]').trigger('click')
    expect(fullscreen.emitted('exitFullscreen')).toHaveLength(1)
  })
})
