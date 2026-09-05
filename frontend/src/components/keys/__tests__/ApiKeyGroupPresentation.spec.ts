import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { ApiKey, Group, GroupPlatform } from '@/types'
import ApiKeySummaryCard from '../ApiKeySummaryCard.vue'
import ApiKeyWorkspaceList from '../ApiKeyWorkspaceList.vue'
import KeysLucideIcon from '../KeysLucideIcon.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const createGroup = (platform: GroupPlatform): Group => ({
  id: 11,
  name: `${platform}-group`,
  platform,
} as Group)

const createApiKey = (platform?: GroupPlatform): ApiKey => ({
  id: 7,
  user_id: 1,
  key: 'sk-super-secret-key',
  name: 'Codex key',
  group_id: platform ? 11 : null,
  group: platform ? createGroup(platform) : undefined,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 10,
  quota_used: 5,
  expires_at: null,
  created_at: '2026-07-19T00:00:00Z',
  updated_at: '2026-07-19T00:00:00Z',
  current_concurrency: 2,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
})

const mountWorkspace = (platform?: GroupPlatform) => mount(ApiKeyWorkspaceList, {
  props: {
    apiKeys: [createApiKey(platform)],
    usageStats: {},
    userGroupRates: {},
    selectedKeyId: 7,
  },
})

const mountSummary = (platform?: GroupPlatform) => mount(ApiKeySummaryCard, {
  props: {
    apiKey: createApiKey(platform),
  },
})

describe('API key group presentation', () => {
  it.each<GroupPlatform>(['openai', 'anthropic', 'gemini', 'antigravity', 'grok', 'zhipu', 'deepseek'])(
    'uses the %s platform icon and tone on desktop and mobile',
    (platform) => {
      const workspace = mountWorkspace(platform)
      const summary = mountSummary(platform)

      const workspaceButton = workspace.get('.workspace-group-button--assigned')
      expect(workspaceButton.classes()).toContain(`workspace-group-button--${platform}`)
      expect(workspaceButton.attributes('data-platform')).toBe(platform)
      expect(workspace.getComponent(PlatformIcon).props()).toMatchObject({ platform, size: 'xs' })
      expect(workspace.getComponent(PlatformIcon).classes()).not.toContain('sr-only')

      const summaryButton = summary.get('.api-key-summary-card__group--assigned')
      expect(summaryButton.classes()).toContain(`api-key-summary-card__group--${platform}`)
      expect(summaryButton.attributes('data-platform')).toBe(platform)
      expect(summary.getComponent(PlatformIcon).props()).toMatchObject({ platform, size: 'md' })
      expect(summary.getComponent(PlatformIcon).classes()).not.toContain('sr-only')

      expect(workspace.findAllComponents(KeysLucideIcon).map((icon) => icon.props('name')))
        .not.toContain('sparkles')
      expect(summary.findAllComponents(KeysLucideIcon).map((icon) => icon.props('name')))
        .not.toContain('sparkles')

      workspace.unmount()
      summary.unmount()
    },
  )

  it('keeps unassigned groups neutral and without a provider icon', () => {
    const workspace = mountWorkspace()
    const summary = mountSummary()

    expect(workspace.get('.workspace-group-button--unassigned').attributes('data-platform'))
      .toBeUndefined()
    expect(summary.get('.api-key-summary-card__group--unassigned').attributes('data-platform'))
      .toBeUndefined()
    expect(workspace.findComponent(PlatformIcon).exists()).toBe(false)
    expect(summary.findComponent(PlatformIcon).exists()).toBe(false)
  })

  it('shows the Zhipu identity for legacy OpenAI-compatible GLM groups', () => {
    const apiKey = createApiKey('openai')
    apiKey.group!.name = '智谱GLM-5.3'

    const workspace = mount(ApiKeyWorkspaceList, {
      props: {
        apiKeys: [apiKey],
        usageStats: {},
        userGroupRates: {},
        selectedKeyId: 7,
      },
    })
    const summary = mount(ApiKeySummaryCard, { props: { apiKey } })

    const workspaceButton = workspace.get('.workspace-group-button--assigned')
    expect(workspaceButton.classes()).toContain('workspace-group-button--zhipu')
    expect(workspaceButton.attributes('data-platform')).toBe('openai')
    expect(workspace.getComponent(PlatformIcon).props('platform')).toBe('zhipu')

    const summaryButton = summary.get('.api-key-summary-card__group--assigned')
    expect(summaryButton.classes()).toContain('api-key-summary-card__group--zhipu')
    expect(summaryButton.attributes('data-platform')).toBe('openai')
    expect(summary.getComponent(PlatformIcon).props('platform')).toBe('zhipu')

    workspace.unmount()
    summary.unmount()
  })
})
