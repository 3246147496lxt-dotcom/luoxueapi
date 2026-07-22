import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const dashboardSurfaceFiles = [
  'src/views/user/DashboardView.vue',
  'src/components/user/dashboard/UserDashboardStats.vue',
  'src/components/user/dashboard/UserDashboardCharts.vue',
  'src/components/user/dashboard/UserDashboardApiInfo.vue',
  'src/components/user/dashboard/UserDashboardSupportPanels.vue',
]

const dashboardSources = dashboardSurfaceFiles.map((file) => ({
  file,
  source: readFileSync(resolve(process.cwd(), file), 'utf8'),
}))

describe('user dashboard surface theme contract', () => {
  it.each(dashboardSources)('$file uses the canonical Snow Clay surface tokens', ({ source }) => {
    expect(source).toContain('border: 1px solid var(--lx-clay-border);')
    expect(source).toContain('border-radius: var(--lx-clay-radius-surface);')
    expect(source).toContain('background: var(--lx-clay-surface);')
    expect(source).toContain('box-shadow: var(--lx-clay-shadow-form);')
    expect(source).not.toContain('rgb(30 41 59)')
    expect(source).not.toContain('rgb(51 65 85 / 0.86)')
  })

  it('does not use white chart borders that flare in dark mode', () => {
    const charts = dashboardSources.find(({ file }) => file.endsWith('UserDashboardCharts.vue'))

    expect(charts?.source).toContain("borderColor: 'transparent'")
    expect(charts?.source).toContain("pointBorderColor: '#5b7cfa'")
    expect(charts?.source).not.toContain("borderColor: '#ffffff'")
    expect(charts?.source).not.toContain("pointBorderColor: '#ffffff'")
  })
})
