import { describe, expect, it } from 'vitest'

import enAdminAccounts from '../locales/en/admin/accounts'
import enAdminChannels from '../locales/en/admin/channels'
import enAdminOps from '../locales/en/admin/ops'
import enAdminOverview from '../locales/en/admin/overview'
import enAdminResources from '../locales/en/admin/resources'
import enAdminSettings from '../locales/en/admin/settings'
import enCommon from '../locales/en/common'
import enDashboard from '../locales/en/dashboard'
import enLanding from '../locales/en/landing'
import enMisc from '../locales/en/misc'
import zhAdminAccounts from '../locales/zh/admin/accounts'
import zhAdminChannels from '../locales/zh/admin/channels'
import zhAdminOps from '../locales/zh/admin/ops'
import zhAdminOverview from '../locales/zh/admin/overview'
import zhAdminResources from '../locales/zh/admin/resources'
import zhAdminSettings from '../locales/zh/admin/settings'
import zhCommon from '../locales/zh/common'
import zhDashboard from '../locales/zh/dashboard'
import zhLanding from '../locales/zh/landing'
import zhMisc from '../locales/zh/misc'

// locales/{zh,en}/index.ts 与 admin/index.ts 使用对象展开聚合各域模块，
// 展开模块之间若出现同名顶层键会静默覆盖。本测试将该风险固化为显式失败。
type Modules = Record<string, Record<string, unknown>>

function collisions(modules: Modules): string[] {
  const seen = new Map<string, string>()
  const out: string[] = []
  for (const [name, mod] of Object.entries(modules)) {
    for (const key of Object.keys(mod)) {
      const prev = seen.get(key)
      if (prev) {
        out.push(`"${key}" in both ${prev} and ${name}`)
      } else {
        seen.set(key, name)
      }
    }
  }
  return out
}

const roots: Record<string, Modules> = {
  zh: { landing: zhLanding, common: zhCommon, dashboard: zhDashboard, misc: zhMisc },
  en: { landing: enLanding, common: enCommon, dashboard: enDashboard, misc: enMisc }
}

const admins: Record<string, Modules> = {
  zh: {
    overview: zhAdminOverview,
    channels: zhAdminChannels,
    accounts: zhAdminAccounts,
    resources: zhAdminResources,
    ops: zhAdminOps,
    settings: zhAdminSettings
  },
  en: {
    overview: enAdminOverview,
    channels: enAdminChannels,
    accounts: enAdminAccounts,
    resources: enAdminResources,
    ops: enAdminOps,
    settings: enAdminSettings
  }
}

describe.each(Object.keys(roots))('locale %s spread assembly', (locale) => {
  it('root modules have no overlapping top-level keys', () => {
    expect(collisions(roots[locale])).toEqual([])
  })

  it('root modules do not shadow the explicit "admin" namespace', () => {
    for (const [name, mod] of Object.entries(roots[locale])) {
      expect(Object.keys(mod), `module ${name} must not define "admin"`).not.toContain('admin')
    }
  })

  it('admin modules have no overlapping top-level keys', () => {
    expect(collisions(admins[locale])).toEqual([])
  })
})

describe('shared navigation copy', () => {
  it('uses the same profile settings terminology as the profile page title', () => {
    expect(zhCommon.nav.profile).toBe('个人设置')
    expect(zhCommon.nav.profile).toBe(zhDashboard.profile.title)
    expect(enCommon.nav.profile).toBe('Profile Settings')
    expect(enCommon.nav.profile).toBe(enDashboard.profile.title)
  })
})

const keysWorkspaceLocalePaths = [
  'workspaceSnowCreditsUnit',
  'workspaceUnassignedGroup',
  'workspaceThirtyDayShort',
  'workspaceResetInInline',
  'workspaceResetInSheet',
  'workspaceIpConfigured',
  'workspaceCreatedInline',
  'workspaceLastUsedInline',
  'workspaceLastIpInline',
  'workspaceCreatedSheet',
  'workspaceLastUsedSheet',
  'workspaceLastIpSheet',
  'workspaceStatus.active',
  'workspaceStatus.inactive',
  'workspaceStatus.quota_exhausted',
  'workspaceStatus.expired',
] as const

function valueAtPath(source: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((value, segment) => {
    if (!value || typeof value !== 'object') return undefined
    return (value as Record<string, unknown>)[segment]
  }, source)
}

describe('API key Superdesign workspace copy', () => {
  it('keeps the exact Chinese Scheme B labels', () => {
    expect(zhDashboard.keys).toMatchObject({
      workspaceSnowCreditsUnit: 'SNOW CREDITS',
      workspaceUnassignedGroup: '未分配分组',
      workspaceThirtyDayShort: '30D',
      workspaceRateHeading: '额度限制（雪花额度）',
      workspaceResetInInline: 'RESET IN: {time}',
      workspaceResetInSheet: '{time} 后重置',
      workspaceIpConfigured: '{count} 项配置',
      workspaceCreatedInline: '创建于',
      workspaceLastUsedInline: '最后使用',
      workspaceLastIpInline: '最近使用 IP',
      workspaceCreatedSheet: '创建时间',
      workspaceLastUsedSheet: '上次使用',
      workspaceLastIpSheet: '最后 IP',
      workspaceStatus: {
        active: 'ACTIVE',
        inactive: 'INACTIVE',
        quota_exhausted: 'QUOTA EXHAUSTED',
        expired: 'EXPIRED',
      },
    })
  })

  it('keeps every Scheme B workspace key available in both locales', () => {
    const enKeys = enDashboard.keys as Record<string, unknown>
    const zhKeys = zhDashboard.keys as Record<string, unknown>

    for (const path of keysWorkspaceLocalePaths) {
      expect(valueAtPath(enKeys, path), `missing en keys.${path}`).toBeTypeOf('string')
      expect(valueAtPath(zhKeys, path), `missing zh keys.${path}`).toBeTypeOf('string')
    }
  })
})
