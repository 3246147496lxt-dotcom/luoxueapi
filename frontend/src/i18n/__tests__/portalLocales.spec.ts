import { describe, expect, it } from 'vitest'
import enPortal from '../locales/en/portal'
import zhPortal from '../locales/zh/portal'

describe('portal locale boundary', () => {
  it.each([
    ['en', enPortal],
    ['zh', zhPortal],
  ] as const)('keeps shared usage labels but excludes admin-only domains for %s', (_locale, messages) => {
    expect(messages.admin.dashboard.timeRange).toBeTruthy()
    expect(messages.admin.groups.rateLabel).toBeTruthy()
    expect(messages.admin.users.columnSettings).toBeTruthy()
    expect(messages.admin.usage.workspace.summaryLabel).toBeTruthy()
    expect(messages.admin.redeem.userPrefix).toBeTruthy()

    expect('settings' in messages.admin).toBe(false)
    expect('accounts' in messages.admin).toBe(false)
    expect('ops' in messages.admin).toBe(false)
  })
})
