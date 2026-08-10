import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const dashboardSurfaceFiles = [
  'src/components/user/dashboard/UserDashboardStats.vue',
  'src/components/user/dashboard/UserDashboardCharts.vue',
  'src/components/user/dashboard/UserDashboardApiInfo.vue',
  'src/components/user/dashboard/UserDashboardRecentConversations.vue',
  'src/components/user/dashboard/UserDashboardQuickActions.vue',
  'src/components/user/dashboard/UserDashboardAccountInfo.vue',
]

const dashboardSources = dashboardSurfaceFiles.map((file) => ({
  file,
  source: readFileSync(resolve(process.cwd(), file), 'utf8'),
}))
const workspaceTokens = readFileSync(
  resolve(process.cwd(), 'src/styles/luoxue-clay-tokens.css'),
  'utf8',
)

describe('user dashboard workspace surface contract', () => {
  it.each(dashboardSources)('$file keeps neutral cards dominant while consuming shared Workspace surfaces', ({ source }) => {
    expect(source).toContain('border: 1px solid var(--workspace-border);')
    expect(source).toContain('border-radius: var(--workspace-radius-work-card);')
    expect(source).toContain('background: var(--workspace-card-surface);')
    expect(source).toContain('var(--workspace-work-accent')
    expect(source).not.toMatch(/--workspace-work-(?:canvas|surface|border|divider)/)
    expect(source).not.toMatch(/(?:linear-gradient|radial-gradient|drop-shadow\()/i)
    expect(source).not.toMatch(/font-size:\s*(?:\d|clamp\()/)
    expect(source).not.toMatch(/font-weight:\s*(?:450|550|650|680)\b/)
    expect(source).not.toContain('font-family:')
  })

  it('uses the approved restrained accent roles instead of recoloring whole cards', () => {
    const stats = dashboardSources.find(item => item.file.endsWith('UserDashboardStats.vue'))?.source || ''
    const charts = dashboardSources.find(item => item.file.endsWith('UserDashboardCharts.vue'))?.source || ''
    const quickActions = dashboardSources.find(item => item.file.endsWith('UserDashboardQuickActions.vue'))?.source || ''
    const apiInfo = dashboardSources.find(item => item.file.endsWith('UserDashboardApiInfo.vue'))?.source || ''
    const accountInfo = dashboardSources.find(item => item.file.endsWith('UserDashboardAccountInfo.vue'))?.source || ''
    const conversations = dashboardSources.find(item => item.file.endsWith('UserDashboardRecentConversations.vue'))?.source || ''

    expect(stats).toContain('dashboard-metric-card__icon-well')
    expect(stats).toContain('var(--workspace-work-accent-deep)')
    expect(charts).toContain('var(--workspace-work-accent)')
    expect(charts).toContain('background: var(--workspace-work-chart-secondary);')
    expect(quickActions).toContain('dashboard-action__icon')
    expect(apiInfo).toContain("'is-primary': endpoint.kind === 'primary'")
    expect(accountInfo).toContain('background: var(--workspace-work-accent-deep);')
    expect(conversations).toContain('background: var(--workspace-work-accent-soft);')
  })

  it('keeps the approved Work values and typography in the canonical token source', () => {
    const dashboard = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(dashboard).not.toContain('font-family:')
    expect(dashboard).toContain('font-size: var(--workspace-type-page-title-size);')
    expect(dashboard).toContain('font-weight: var(--workspace-type-page-title-weight);')
    expect(dashboard).not.toMatch(/font-size:\s*(?:\d|clamp\()/)
    expect(dashboard).not.toMatch(/font-weight:\s*(?:450|550|650|680)\b/)
    expect(workspaceTokens).toContain('--workspace-radius-work-card: 14px;')
    expect(workspaceTokens).toContain(
      '--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;',
    )
    expect(workspaceTokens).toContain('--workspace-type-page-title-size: 28px;')
    expect(workspaceTokens).toContain('--workspace-type-page-title-weight: 600;')
    expect(workspaceTokens).toContain('--workspace-type-navigation-size: 14px;')
    expect(workspaceTokens).toContain('--workspace-type-navigation-weight: 500;')
    expect(workspaceTokens).toContain('--workspace-type-body-size: 14px;')
    expect(workspaceTokens).toContain('--workspace-type-body-weight: 400;')
    expect(workspaceTokens).toContain('--workspace-type-secondary-size: 12px;')
    expect(workspaceTokens).toContain('--workspace-type-secondary-weight: 400;')
    expect(workspaceTokens).toContain('--workspace-type-numeric-size: 32px;')
    expect(workspaceTokens).toContain('--workspace-type-numeric-weight: 600;')
    expect(workspaceTokens).toContain('--workspace-light-work-text: #0d0d0d;')
    expect(workspaceTokens).toContain('--workspace-light-work-text-secondary: #5d5d5d;')
    expect(workspaceTokens).toContain('--workspace-light-work-accent: #7c3aed;')
    expect(workspaceTokens).toContain('--workspace-light-work-accent-deep: #5b21b6;')
    expect(workspaceTokens).toContain('--workspace-light-work-chart-secondary: #a78bfa;')
    expect(dashboard).toContain('background: var(--workspace-canvas);')
    expect(dashboard).not.toMatch(/--workspace-work-(?:canvas|surface|border|divider)/)
    expect(workspaceTokens).toContain(
      '--workspace-work-text: var(--workspace-text);',
    )
    expect(workspaceTokens).not.toMatch(/--workspace-work-(?:canvas|surface|border|divider):/)
  })

  it('keeps infrastructure monitoring and channel status out of the dashboard render and request flow', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(source).not.toContain('channelMonitor')
    expect(source).not.toContain('UserDashboardSupportPanels')
    expect(source).not.toContain('loadMonitors')
    expect(source).not.toContain('monitorEnabled')
    expect(source).not.toContain('availability_7d')
    expect(source).not.toContain('primary_latency_ms')
  })

  it('uses the shared profile and chat stores for real user workspace data', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(source).toContain("useUserProfileStore")
    expect(source).not.toContain("useSubscriptionStore")
    expect(source).not.toContain("useAccountSummary")
    expect(source).toContain("useChatStore")
    expect(source).toContain('chatStore.hydrate(userId)')
    expect(source).toContain('chatStore.syncHistory()')
    expect(source).toContain('chatStore.loadConversationPage(true)')
  })

  it('keeps route mounting passive and only refreshes the shared profile on user action', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/user/DashboardView.vue'), 'utf8')

    expect(source).toContain('void loadDashboard(false)')
    expect(source).toContain('await loadDashboard(true)')
    expect(source).toContain('userProfileStore.refreshProfile()')
    expect(source).not.toContain('authStore.refreshUser()')
    expect(source).not.toContain('subscriptionStore.fetchActiveSubscriptions(')
  })

  it('keeps quick actions focused on user tasks, including balance management', () => {
    const source = readFileSync(
      resolve(process.cwd(), 'src/components/user/dashboard/UserDashboardQuickActions.vue'),
      'utf8',
    )

    expect(source).toContain("to: '/purchase'")
    expect(source).toContain("labelKey: 'dashboard.workspace.actions.balance'")
  })
})
