import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const activeDashboardFiles = [
  'src/views/user/DashboardView.vue',
  'src/components/user/dashboard/DashboardNotificationPopover.vue',
  'src/components/user/dashboard/UserDashboardStats.vue',
  'src/components/user/dashboard/UserDashboardCharts.vue',
  'src/components/user/dashboard/UserDashboardInsights.vue',
]

const dashboardSources = activeDashboardFiles.map(file => ({
  file,
  source: readFileSync(resolve(process.cwd(), file), 'utf8'),
}))

describe('dashboard dark CSS selector contract', () => {
  it.each(dashboardSources)('$file never scopes a local selector after :global(html.dark)', ({ source }) => {
    expect(source).not.toMatch(/:global\(html\.dark\)\s+/)
  })

  it('keeps intentional dark overrides inside a complete global selector', () => {
    const notifications = dashboardSources.find(item => item.file.endsWith('DashboardNotificationPopover.vue'))?.source || ''
    const insights = dashboardSources.find(item => item.file.endsWith('UserDashboardInsights.vue'))?.source || ''

    expect(notifications).toContain(':global(html.dark .dashboard-notifications .dashboard-notifications__panel)')
    expect(insights).toContain(':global(html.dark .dashboard-insights .dashboard-channel-select select)')
  })

  it('lets the recharge action switch themes through semantic tokens', () => {
    const stats = dashboardSources.find(item => item.file.endsWith('UserDashboardStats.vue'))?.source || ''

    expect(stats).toContain('var(--workspace-work-accent-soft)')
    expect(stats).toContain('var(--workspace-work-accent-soft-hover)')
    expect(stats).not.toContain(':global(html.dark .dashboard-metric-grid')
  })

  it('contains the OpenAI inversion inside the icon selector', () => {
    const insights = dashboardSources.find(item => item.file.endsWith('UserDashboardInsights.vue'))?.source || ''

    expect(insights).toContain(':global(html.dark .dashboard-insights .dashboard-model-row__icon--openai .model-icon)')
    expect(insights).toContain('filter: invert(1)')
  })
})
