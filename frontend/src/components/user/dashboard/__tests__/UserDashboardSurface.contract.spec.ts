import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const activeDashboardFiles = [
  'src/components/user/dashboard/UserDashboardStats.vue',
  'src/components/user/dashboard/UserDashboardCharts.vue',
  'src/components/user/dashboard/UserDashboardInsights.vue',
]

const dashboardSources = activeDashboardFiles.map((file) => ({
  file,
  source: readFileSync(resolve(process.cwd(), file), 'utf8'),
}))
const workspaceTokens = readFileSync(
  resolve(process.cwd(), 'src/styles/luoxue-clay-tokens.css'),
  'utf8',
)

describe('user dashboard workspace surface contract', () => {
  it.each(dashboardSources)('$file keeps neutral semantic cards across light and dark themes', ({ source }) => {
    expect(source).toContain('border: 1px solid var(--workspace-dashboard-card-border);')
    expect(source).toContain('border-radius: 24px;')
    expect(source).toContain('background: var(--workspace-card-surface);')
    expect(source).toContain('box-shadow: var(--workspace-dashboard-card-shadow);')
    expect(source).not.toMatch(/--workspace-work-(?:canvas|surface|border|divider)/)
    expect(source).not.toMatch(/(?:linear-gradient|radial-gradient|drop-shadow\()/i)
    expect(source).not.toMatch(/font-size:\s*(?:\d|clamp\()/)
    expect(source).not.toContain('font-family:')
  })

  it('implements the approved four stable summaries without dashboard-detail actions', () => {
    const stats = dashboardSources.find(item => item.file.endsWith('UserDashboardStats.vue'))?.source || ''

    expect(stats).toContain('dashboard.workspace.accountBalance')
    expect(stats).toContain('dashboard.workspace.planQuota')
    expect(stats).toContain('dashboard.workspace.cumulativeTokens')
    expect(stats).toContain('dashboard.workspace.cumulativeSpend')
    expect(stats).toContain('dashboard-quota-track__value--healthy')
    expect(stats).toContain('dashboard-metric-card__action')
    expect(stats).toContain('to="/purchase"')
    expect(stats).not.toContain('<Icon')
    expect(stats).not.toContain('today_actual_cost')
  })

  it('keeps one period selector and the approved orange-to-purple chart hierarchy', () => {
    const charts = dashboardSources.find(item => item.file.endsWith('UserDashboardCharts.vue'))?.source || ''

    expect(charts).not.toContain('<select')
    expect(charts).toContain('dashboard-period-select__trigger')
    expect(charts).toContain('dashboard-period-select__menu')
    expect(charts).toContain("'chevronDown'")
    expect(charts).toContain("'chevronUp'")
    expect(charts).toContain("orange: '#FF8A00'")
    expect(charts).toContain("token: '#8B5CF6'")
    expect(charts).toContain("request: '#A78BFA'")
    expect(charts).toContain('pointRadius: 0')
    expect(charts).toContain("id: 'dashboardHoverGuide'")
    expect(charts).toContain('maxTicksLimit: 4')
    expect(charts).toContain('count: 3')
  })

  it('keeps the lower insight cards on real model and channel hooks', () => {
    const insights = dashboardSources.find(item => item.file.endsWith('UserDashboardInsights.vue'))?.source || ''

    expect(insights).toContain('usageAPI.getDashboardModels({')
    expect(insights).toContain('channelMonitorUserAPI.list({ signal: controller.signal })')
    expect(insights).toContain('modelRanking')
    expect(insights).toContain('channelStatus')
    expect(insights).toContain('grid-template-columns: minmax(0, 34%) 14% 20% 14% 18%;')
    expect(insights).toContain("selectedProvider.value = providerOptions.value.includes('openai')")
    expect(insights).toContain("channelHealth.value.rate == null ? '--%'")
    expect(insights).not.toContain('Best Selling Products')
    expect(insights).not.toContain('Repeat Customer Rate')
  })

  it('keeps approved typography and Workspace values in the canonical token source', () => {
    const dashboard = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(dashboard).not.toContain('font-family:')
    expect(dashboard).toContain('font-size: calc(var(--workspace-type-page-title-size) + 0.125rem);')
    expect(dashboard).toContain('font-weight: 700;')
    expect(dashboard).not.toContain('letter-spacing: -0.025em;')
    expect(dashboard).not.toMatch(/font-size:\s*(?:\d|clamp\()/)
    expect(workspaceTokens).toContain('--workspace-radius-work-card: 14px;')
    expect(workspaceTokens).toContain('--workspace-type-page-title-size: 28px;')
    expect(workspaceTokens).toContain('--workspace-type-navigation-size: 14px;')
    expect(workspaceTokens).toContain('--workspace-type-secondary-size: 12px;')
    expect(workspaceTokens).toContain('--workspace-light-work-accent: #7c3aed;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-card-border: #f1f5f9;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-text-strong: #0f172a;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-text-heading: #475569;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-text-muted: #64748b;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-text-subtle: #94a3b8;')
    expect(workspaceTokens).toContain('--workspace-light-dashboard-period-text: #334155;')
    expect(workspaceTokens).toContain('--workspace-dark-dashboard-card-shadow: none;')
    expect(dashboard).toContain('background: var(--workspace-canvas);')
  })

  it('removes the generic admin and chat content from the live dashboard request flow', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(source).not.toContain('getDashboardModels')
    expect(source).not.toContain('useChatStore')
    expect(source).not.toContain('chatStore.hydrate')
    expect(source).not.toContain('UserDashboardRecentConversations')
    expect(source).not.toContain('UserDashboardQuickActions')
    expect(source).not.toContain('UserDashboardApiInfo')
    expect(source).not.toContain('UserDashboardAccountInfo')
    expect(source).not.toContain('BaseDialog')
    expect(source).toContain('DashboardNotificationPopover')
  })

  it('uses real profile, subscription quota and trend APIs with a passive initial mount', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(source).toContain('useUserProfileStore')
    expect(source).not.toContain('useSubscriptionStore')
    expect(source).toContain('getSubscriptionsProgress()')
    expect(source).toContain('getDashboardStats()')
    expect(source).toContain('getDashboardTrend({')
    expect(source).toContain("const usagePeriod = ref<DashboardUsagePeriod>('today')")
    expect(source).toContain('void loadDashboard(false)')
    expect(source).toContain('await loadDashboard(true)')
    expect(source).not.toContain('authStore.refreshUser()')
  })
})
