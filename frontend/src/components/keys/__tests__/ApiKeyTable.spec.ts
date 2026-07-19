import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { ApiKey } from '@/types'
import ApiKeyTable from '../ApiKeyTable.vue'

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.delete': 'Delete',
  'common.edit': 'Edit',
  'common.name': 'Name',
  'common.status': 'Status',
  'keys.apiKey': 'API Key',
  'keys.blacklistCount': 'Blocklist {count}',
  'keys.clickToChangeGroup': 'Change group',
  'keys.copyToClipboard': 'Copy',
  'keys.copied': 'Copied',
  'keys.created': 'Created',
  'keys.currentConcurrency': 'Current concurrency',
  'keys.disable': 'Disable',
  'keys.enable': 'Enable',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.hideKey': 'Hide key',
  'keys.id': 'ID',
  'keys.importToCcSwitch': 'Import to CCS',
  'keys.ipRestriction': 'IP restriction',
  'keys.lastUsedAt': 'Last used',
  'keys.lastUsedIP': 'Last used IP',
  'keys.noExpiration': 'Never',
  'keys.noGroup': 'No group',
  'keys.noIpRestriction': 'No restrictions',
  'keys.noRateLimit': 'Not set',
  'keys.quotaUsage': 'Quota usage',
  'keys.rateLimitColumn': 'Rate limit',
  'keys.resetUsage': 'Reset',
  'keys.revealKey': 'Reveal key',
  'keys.status.active': 'Enabled',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.today': 'Today',
  'keys.total': 'Last 30d',
  'keys.unlimitedQuota': 'Unlimited',
  'keys.useKey': 'Use key',
  'keys.whitelistCount': 'Allowlist {count}',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        let value = messages[key] ?? key
        for (const [name, replacement] of Object.entries(params ?? {})) {
          value = value.replace(`{${name}}`, String(replacement))
        }
        return value
      },
    }),
  }
})

const createApiKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 12,
  user_id: 1,
  key: 'sk-desktop-super-secret-key',
  name: 'Desktop key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 10,
  quota_used: 2.5,
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
  ...overrides,
})

const mountTable = (props: Record<string, unknown> = {}) =>
  mount(ApiKeyTable, {
    props: {
      apiKeys: [createApiKey()],
      usageStats: {
        12: {
          api_key_id: 12,
          today_actual_cost: 0.75,
          total_actual_cost: 3.25,
        },
      },
      userGroupRates: {},
      visibleColumns: ['current_concurrency', 'expires_at', 'created_at'],
      sortBy: 'created_at',
      sortOrder: 'desc',
      ...props,
    },
    global: {
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name">{{ name }}</span>',
        },
        GroupBadge: true,
      },
    },
  })

describe('ApiKeyTable', () => {
  it('integrates status, usage, quota, and progress in the desktop row', () => {
    const wrapper = mountTable()

    expect(wrapper.find('[data-test="api-key-table-row-12"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Enabled')
    expect(wrapper.text()).toContain('Today $0.7500')
    expect(wrapper.text()).toContain('Last 30d $3.2500')
    expect(wrapper.text()).toContain('$2.50')
    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.get('[data-test="key-table-status-switch-12"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[data-test="key-table-quota-progress-12"]').attributes('aria-valuenow')).toBe('25')
  })

  it('emits status changes and reflects the updating lock', async () => {
    const wrapper = mountTable()
    const switchButton = wrapper.get('[data-test="key-table-status-switch-12"]')

    await switchButton.trigger('click')
    expect(wrapper.emitted('toggle-status')?.[0]?.[0]).toMatchObject({ id: 12 })

    await wrapper.setProps({ statusUpdatingIds: [12] })
    expect(switchButton.attributes('disabled')).toBeDefined()
    expect(switchButton.attributes('aria-busy')).toBe('true')
  })

  it('keeps the key masked until reveal and emits copy for the same row', async () => {
    const wrapper = mountTable()

    expect(wrapper.text()).not.toContain('sk-desktop-super-secret-key')
    await wrapper.get('[data-test="key-table-reveal-12"]').trigger('click')
    expect(wrapper.text()).toContain('sk-desktop-super-secret-key')

    await wrapper.get('[data-test="key-table-copy-12"]').trigger('click')
    expect(wrapper.emitted('copy-key')?.[0]?.[0]).toMatchObject({ id: 12 })
  })

  it('preserves row actions and group changes as business events', async () => {
    const wrapper = mountTable()

    const groupButton = wrapper.findAll('button').find((button) => button.attributes('title') === 'Change group')
    expect(groupButton).toBeDefined()
    await groupButton!.trigger('click')

    for (const [label, event] of [
      ['Use key', 'use-key'],
      ['Import to CCS', 'import-ccs'],
      ['Edit', 'edit'],
      ['Delete', 'delete'],
    ] as const) {
      const button = wrapper.findAll('button').find((candidate) =>
        candidate.text().includes(label) || candidate.attributes('aria-label') === label
      )
      expect(button, `${label} action should be rendered`).toBeDefined()
      await button!.trigger('click')
      expect(wrapper.emitted(event)?.[0]?.[0]).toMatchObject({ id: 12 })
    }

    expect(wrapper.emitted('change-group')?.[0]?.[0]).toMatchObject({ id: 12 })
    expect(wrapper.emitted('change-group')?.[0]?.[1]).toBeInstanceOf(MouseEvent)
  })

  it('emits explicit sort direction and reverses the active column', async () => {
    const wrapper = mountTable()
    const nameHeader = wrapper.get('button[aria-label="Name"]')

    await nameHeader.trigger('click')
    expect(wrapper.emitted('sort')?.[0]).toEqual(['name', 'asc'])

    await wrapper.setProps({ sortBy: 'name', sortOrder: 'asc' })
    await nameHeader.trigger('click')
    expect(wrapper.emitted('sort')?.[1]).toEqual(['name', 'desc'])
  })

  it('hides CCS import and the entire sticky action column when configured', async () => {
    const wrapper = mountTable({ showCcsImport: false })

    expect(wrapper.text()).not.toContain('Import to CCS')
    expect(wrapper.text()).toContain('Use key')

    await wrapper.setProps({ showActions: false })
    expect(wrapper.text()).not.toContain('Actions')
    expect(wrapper.text()).not.toContain('Use key')
    expect(wrapper.text()).not.toContain('Edit')
    expect(wrapper.text()).not.toContain('Delete')
  })
})
