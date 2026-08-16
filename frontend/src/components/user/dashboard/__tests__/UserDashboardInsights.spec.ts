import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import UserDashboardInsights from '../UserDashboardInsights.vue'

const { getDashboardModels, listChannelMonitors } = vi.hoisted(() => ({
  getDashboardModels: vi.fn(),
  listChannelMonitors: vi.fn(),
}))

vi.mock('@/api/usage', () => ({
  usageAPI: { getDashboardModels },
}))

vi.mock('@/api/channelMonitor', () => ({
  channelMonitorUserAPI: { list: listChannelMonitors },
}))

vi.mock('@/components/user/MonitorDetailDialog.vue', () => ({
  default: { name: 'MonitorDetailDialog', template: '<div data-testid="monitor-dialog" />' },
}))

vi.mock('@/components/common/ModelIcon.vue', () => ({
  default: { name: 'ModelIcon', template: '<span class="model-icon" />' },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string) => ({
      'dashboard.workspace.insightsSection': 'Dashboard 分析模块',
      'dashboard.workspace.modelRanking': '模型使用排行',
      'dashboard.workspace.model': '模型',
      'dashboard.workspace.requestCount': '请求次数',
      'dashboard.workspace.tokens': 'Token',
      'dashboard.workspace.creditsUsed': '积分消耗',
      'dashboard.workspace.usageShare': '使用占比',
      'dashboard.workspace.viewAll': '查看全部',
      'dashboard.workspace.channelStatus': '渠道状态',
      'dashboard.workspace.last24Hours': '近24小时',
      'dashboard.workspace.channelSelector': '选择渠道',
      'dashboard.workspace.noChannels': '暂无渠道',
      'dashboard.workspace.noModelUsage': '暂无模型使用记录',
      'dashboard.workspace.loadingModels': '加载模型用量',
      'dashboard.workspace.viewDetails': '查看详情',
      'dashboard.workspace.modelRankingLoadError': '模型排行加载失败',
      'dashboard.workspace.channelLoadError': '渠道状态加载失败',
      'channelStatus.detailTitle': '渠道详情',
      'common.loading': '加载中',
    }[key] ?? key),
  }),
}))

function mountInsights() {
  return mount(UserDashboardInsights, {
    props: { period: 'today' },
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('UserDashboardInsights', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getDashboardModels.mockResolvedValue({
      models: [
        { model: 'Claude Sonnet', requests: 2, total_tokens: 200, actual_cost: 30 },
        { model: 'GPT-5.6', requests: 4, total_tokens: 400, actual_cost: 70 },
      ],
    })
    listChannelMonitors.mockResolvedValue({
      items: [{
        id: 7,
        name: 'GPT primary',
        provider: 'openai',
        group_name: 'default',
        primary_model: 'gpt-5.6',
        primary_status: 'operational',
        primary_latency_ms: 300,
        primary_ping_latency_ms: 40,
        availability_7d: 99,
        extra_models: [],
        timeline: [
          { status: 'operational', latency_ms: 300, ping_latency_ms: 40, checked_at: new Date().toISOString() },
          { status: 'degraded', latency_ms: 500, ping_latency_ms: 60, checked_at: new Date(Date.now() - 60_000).toISOString() },
        ],
      }],
    })
  })

  it('sorts models by actual spend and derives a shared-period usage share', async () => {
    const wrapper = mountInsights()
    await flushPromises()

    const rows = wrapper.findAll('[role="row"]')
    expect(rows).toHaveLength(3)
    expect(rows[1].text()).toContain('GPT-5.6')
    expect(rows[1].text()).toContain('70.0%')
    expect(rows[2].text()).toContain('Claude Sonnet')
    expect(rows[2].text()).toContain('30.0%')
    expect(getDashboardModels).toHaveBeenCalledWith(expect.objectContaining({ model_source: 'requested' }))
  })

  it('uses all models in the selected period as the usage-share denominator', async () => {
    getDashboardModels.mockResolvedValueOnce({
      models: [
        { model: 'A', requests: 1, total_tokens: 10, actual_cost: 50 },
        { model: 'B', requests: 1, total_tokens: 10, actual_cost: 20 },
        { model: 'C', requests: 1, total_tokens: 10, actual_cost: 10 },
        { model: 'D', requests: 1, total_tokens: 10, actual_cost: 8 },
        { model: 'E', requests: 1, total_tokens: 10, actual_cost: 7 },
        { model: 'F', requests: 1, total_tokens: 10, actual_cost: 5 },
      ],
    })

    const wrapper = mountInsights()
    await flushPromises()

    expect(wrapper.findAll('[role="row"]')).toHaveLength(6)
    expect(wrapper.findAll('[role="row"]')[1].text()).toContain('50.0%')
  })

  it('builds the provider selector from real monitor data and shows no-data-safe health', async () => {
    const wrapper = mountInsights()
    await flushPromises()

    expect(wrapper.get('.dashboard-channel-card__title-group h2').text()).toBe('渠道状态')
    expect(wrapper.get('.dashboard-channel-select select').text()).toContain('GPT')
    expect(wrapper.get('.dashboard-channel-card__value').text()).toBe('100.0%')
    expect(wrapper.get('.dashboard-channel-card__details').attributes('disabled')).toBeUndefined()

    listChannelMonitors.mockResolvedValueOnce({ items: [] })
    const emptyWrapper = mountInsights()
    await flushPromises()
    expect(emptyWrapper.get('.dashboard-channel-card__value').text()).toBe('--%')
    expect(emptyWrapper.get('.dashboard-channel-card__details').attributes('disabled')).toBeDefined()
  })
})
