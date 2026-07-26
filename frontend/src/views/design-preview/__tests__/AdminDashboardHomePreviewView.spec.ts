import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AdminDashboardHomePreviewView from '../AdminDashboardHomePreviewView.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' },
    }),
  }
})

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    props: ['data', 'options'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
  Line: {
    props: ['data', 'options'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

const mountPreview = () => mount(AdminDashboardHomePreviewView)

const findButton = (wrapper: ReturnType<typeof mount>, label: string) => {
  const button = wrapper.findAll('button').find(candidate => candidate.text().trim() === label)
  if (!button) throw new Error(`Button not found: ${label}`)
  return button
}

describe('AdminDashboardHomePreviewView', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/design-preview/admin-dashboard-home')
  })

  it('renders a complete local dashboard preview without an application shell', () => {
    const wrapper = mountPreview()

    expect(wrapper.text()).toContain('首页风格预览')
    expect(wrapper.text()).toContain('本页不连接后台接口，也不会改动正式管理页')
    expect(wrapper.text()).toContain('1,284')
    expect(wrapper.text()).toContain('1,106 个已启用 · 248 个本周活跃')
    expect(wrapper.text()).toContain('较上周同期')
    expect(wrapper.text()).not.toContain('个正在使用')
    expect(wrapper.text()).toContain('58 当前健康 · 2 异常')
    const previewAccountHealth = wrapper.findAll('.metric-trend')[1]!
    expect(previewAccountHealth.attributes('aria-label')).toContain('预览数据')
    expect(previewAccountHealth.attributes('aria-label')).toContain('58/64 = 90.6%')
    expect(previewAccountHealth.attributes('aria-label')).toContain('健康账号排除有效限流、过载、临时冷却与到期自动暂停')
    expect(previewAccountHealth.attributes('aria-label')).toContain('账号配额窗口不纳入本指标')
    expect(previewAccountHealth.attributes('aria-label')).toContain('状态可能重叠且不作相加')
    const previewRequestComparison = wrapper.findAll('.metric-trend')[2]!
    expect(previewRequestComparison.text()).toContain('较昨日同期')
    expect(previewRequestComparison.attributes('aria-label')).toContain('预览数据')
    expect(previewRequestComparison.attributes('aria-label')).toContain('较昨日同期增长 12.4%')
    const previewNewUserComparison = wrapper.findAll('.metric-trend')[3]!
    expect(previewNewUserComparison.text()).toContain('较昨日同期')
    expect(previewNewUserComparison.attributes('aria-label')).toContain('预览数据')
    expect(previewNewUserComparison.attributes('aria-label')).toContain('较昨日同期增长 8.2%')
    expect(wrapper.text()).toContain('总用户：8,642人')
    expect(wrapper.text()).toContain('38.42M')
    expect(wrapper.text()).toContain('Claude 4 Sonnet')
    expect(wrapper.text()).toContain('北极星工作室')
  })

  it('lets reviewers inspect loading, empty, error and dark states', async () => {
    const wrapper = mountPreview()

    await findButton(wrapper, '加载中').trigger('click')
    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(true)

    await findButton(wrapper, '空数据').trigger('click')
    expect(wrapper.text()).toContain('还没有可展示的用量数据')

    await findButton(wrapper, '异常状态').trigger('click')
    expect(wrapper.text()).toContain('概览加载失败')

    await wrapper.find('button[aria-label="切换到深色模式"]').trigger('click')
    expect(wrapper.classes()).toContain('is-dark')
  })

  it('supports direct state links through a local query parameter', () => {
    window.history.replaceState({}, '', '/design-preview/admin-dashboard-home?state=empty')
    const wrapper = mountPreview()

    expect(wrapper.text()).toContain('还没有可展示的用量数据')
  })
})
