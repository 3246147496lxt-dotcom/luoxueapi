import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { ApiKey } from '@/types'
import ApiKeyCard from '../ApiKeyCard.vue'

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
  'keys.quota': 'Quota',
  'keys.quotaUsage': 'Quota usage',
  'keys.rateLimitUsage': 'Rate limit usage',
  'keys.resetNow': 'Now',
  'keys.resetUsage': 'Reset',
  'keys.resetsIn': 'Resets in {time}',
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
  id: 7,
  user_id: 1,
  key: 'sk-super-secret-key',
  name: 'Codex key',
  group_id: null,
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
  ...overrides,
})

const mountCard = (apiKey = createApiKey(), props: Record<string, unknown> = {}) =>
  mount(ApiKeyCard, {
    props: {
      apiKey,
      usage: {
        api_key_id: apiKey.id,
        today_actual_cost: 1.25,
        total_actual_cost: 4.75,
      },
      visibleColumns: ['id', 'current_concurrency', 'rate_limit', 'expires_at', 'created_at'],
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

describe('ApiKeyCard', () => {
  it('shows status, actual costs, and a clamped quota progress bar', () => {
    const wrapper = mountCard()

    expect(wrapper.text()).toContain('Enabled')
    expect(wrapper.text()).toContain('$1.2500')
    expect(wrapper.text()).toContain('$4.7500')
    expect(wrapper.text()).toContain('$5.00/$10.00')
    expect(wrapper.get('[data-test="key-status-switch-7"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[data-test="key-quota-progress-7"]').attributes('aria-valuenow')).toBe('50')
  })

  it('renders each non-active status distinctly and keeps the switch off', () => {
    for (const [status, label] of [
      ['inactive', 'Inactive'],
      ['quota_exhausted', 'Quota exhausted'],
      ['expired', 'Expired'],
    ] as const) {
      const wrapper = mountCard(createApiKey({ status }))
      expect(wrapper.text()).toContain(label)
      expect(wrapper.get('[role="switch"]').attributes('aria-checked')).toBe('false')
      wrapper.unmount()
    }
  })

  it('shows unlimited quota without a fake progress bar', () => {
    const wrapper = mountCard(createApiKey({ quota: 0, quota_used: 0 }))

    expect(wrapper.text()).toContain('Unlimited')
    expect(wrapper.find('[data-test="key-quota-progress-7"]').exists()).toBe(false)
  })

  it('emits one status event and disables the switch while updating', async () => {
    const wrapper = mountCard(createApiKey(), { statusUpdating: true })
    const toggle = wrapper.get('[role="switch"]')

    expect(toggle.attributes('disabled')).toBeDefined()
    expect(toggle.attributes('aria-busy')).toBe('true')
    await toggle.trigger('click')
    expect(wrapper.emitted('toggle-status')).toBeUndefined()

    await wrapper.setProps({ statusUpdating: false })
    await toggle.trigger('click')
    expect(wrapper.emitted('toggle-status')).toHaveLength(1)
  })

  it('reveals, masks, and emits copy without rendering the full key initially', async () => {
    const wrapper = mountCard()

    expect(wrapper.text()).not.toContain('sk-super-secret-key')
    await wrapper.get('[data-test="key-reveal-7"]').trigger('click')
    expect(wrapper.text()).toContain('sk-super-secret-key')
    await wrapper.get('[data-test="key-copy-7"]').trigger('click')
    expect(wrapper.emitted('copy-key')?.[0]?.[0]).toMatchObject({ id: 7 })
  })

  it('visualizes rate-limit windows and emits reset', async () => {
    const wrapper = mountCard(createApiKey({
      rate_limit_5h: 5,
      usage_5h: 2.5,
      reset_5h_at: '2026-07-20T00:00:00Z',
    }), { now: new Date('2026-07-19T00:00:00Z') })

    expect(wrapper.text()).toContain('$2.50 / $5.00')
    expect(wrapper.find('[role="progressbar"][aria-label="5h Rate limit usage"]').exists()).toBe(true)
    const reset = wrapper.findAll('button').find((button) => button.text().includes('Reset'))
    expect(reset).toBeDefined()
    await reset!.trigger('click')
    expect(wrapper.emitted('reset-rate-limit')).toHaveLength(1)
  })

  it('keeps all business actions and can collapse their container', async () => {
    const wrapper = mountCard()

    expect(wrapper.text()).toContain('Use key')
    expect(wrapper.text()).toContain('Import to CCS')
    expect(wrapper.text()).toContain('Edit')
    expect(wrapper.text()).toContain('Delete')

    const actionButtons = wrapper.get('[data-test="key-actions-7"]').findAll('button')
    for (const button of actionButtons) await button.trigger('click')
    expect(wrapper.emitted('use-key')).toHaveLength(1)
    expect(wrapper.emitted('import-ccs')).toHaveLength(1)
    expect(wrapper.emitted('edit')).toHaveLength(1)
    expect(wrapper.emitted('delete')).toHaveLength(1)

    await wrapper.setProps({ showActions: false })
    expect(wrapper.find('[data-test="key-actions-7"]').exists()).toBe(false)
  })

  it('hides CCS import when disabled by public settings', () => {
    const wrapper = mountCard(createApiKey(), { showCcsImport: false })
    expect(wrapper.text()).not.toContain('Import to CCS')
  })
})
