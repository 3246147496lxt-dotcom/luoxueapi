import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import type { ApiKey, PublicSettings } from '@/types'
import { formatDateTime } from '@/utils/format'
import ApiKeyInspector from '../ApiKeyInspector.vue'

const messages: Record<string, string> = {
  'common.available': 'Available',
  'common.close': 'Close',
  'common.delete': 'Delete',
  'common.edit': 'Edit',
  'common.none': 'None',
  'common.notAvailable': 'N/A',
  'common.reset': 'Reset',
  'common.total': 'Total',
  'common.unlimited': 'Unlimited',
  'keys.apiKey': 'API Key',
  'keys.clickToChangeGroup': 'Change group',
  'keys.copyToClipboard': 'Copy',
  'keys.copied': 'Copied',
  'keys.created': 'Created',
  'keys.currentConcurrency': 'Current concurrency',
  'keys.detailSettings': 'Details',
  'keys.disable': 'Disable',
  'keys.enable': 'Enable',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.hideKey': 'Hide key',
  'keys.id': 'ID',
  'keys.importToCcSwitch': 'Import to CCS',
  'keys.editKey': 'Edit API Key',
  'keys.deleteKey': 'Delete API Key',
  'keys.ipBlacklist': 'IP blocklist',
  'keys.ipRestriction': 'IP restrictions',
  'keys.ipRestrictionEnabled': 'Configured',
  'keys.ipWhitelist': 'IP allowlist',
  'keys.lastUsedAt': 'Last used',
  'keys.lastUsedIP': 'Last used IP',
  'keys.noExpiration': 'Never',
  'keys.noGroup': 'No group',
  'keys.noIpRestriction': 'No restrictions',
  'keys.noRateLimit': 'Not set',
  'keys.quota': 'Quota',
  'keys.quotaUsage': 'Quota usage',
  'keys.quotaUsed': 'Used',
  'keys.rateLimitColumn': 'Limit',
  'keys.rateLimitSection': 'Rate limits',
  'keys.rateLimitUsage': 'Rate limit usage',
  'keys.reset': 'Reset',
  'keys.resetNow': 'Now',
  'keys.resetUsage': 'Reset usage',
  'keys.resetsIn': 'Resets in {time}',
  'keys.revealKey': 'Reveal key',
  'keys.status.active': 'Enabled',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.today': 'Today',
  'keys.total': 'Last 30d',
  'keys.unlimitedQuota': 'Unlimited',
  'keys.usage': 'Usage',
  'keys.useKey': 'Use key',
  'keys.workspaceTokenLabel': 'API Key Token',
  'keys.workspaceEndpointLabel': 'API Endpoint',
  'keys.workspaceEndpointEmpty': 'Hidden when not configured',
  'keys.workspaceQuotaHeading': 'Quota Progress',
  'keys.workspaceCurrentAvailable': 'Currently Available',
  'keys.workspaceTotalLimit': 'Total Limit',
  'keys.workspaceSnowCreditsUnit': 'SNOW CREDITS',
  'keys.workspaceTodayUsage': 'Today Total',
  'keys.workspaceThirtyDayUsage': 'Last 30 Days Total',
  'keys.workspaceResetQuota': 'Reset Used Quota',
  'keys.workspaceRateHeading': 'Rate Limits (Snow credits)',
  'keys.workspaceResetAll': 'Reset All',
  'keys.workspaceResetInInline': 'RESET IN: {time}',
  'keys.workspaceRate5hInline': '5h Limit (5-hour window)',
  'keys.workspaceRate1dInline': '1d Limit (daily)',
  'keys.workspaceRate7dInline': '7d Limit (weekly)',
  'keys.workspaceSecurityHeading': 'Security Controls and History',
  'keys.workspaceConcurrencyLimit': 'Concurrency Limit',
  'keys.workspaceIpConfigured': '{count} items configured',
  'keys.workspaceCreatedInline': 'Created on',
  'keys.workspaceLastUsedInline': 'Last used',
  'keys.workspaceLastIpInline': 'Recent IP',
  'keys.workspaceImportCcs': 'Import CCS',
  'keys.workspaceCoreAuthHeading': 'Core Status and Authentication',
  'keys.workspaceCurrentStatus': 'Current Status',
  'keys.workspaceStatus.active': 'ACTIVE',
  'keys.workspaceStatus.expired': 'EXPIRED',
  'keys.workspaceStatus.inactive': 'INACTIVE',
  'keys.workspaceStatus.quota_exhausted': 'QUOTA EXHAUSTED',
  'keys.workspaceSheetTokenLabel': 'Key Token (API KEY TOKEN)',
  'keys.workspaceSheetEndpointLabel': 'Endpoint (API ENDPOINT)',
  'keys.workspaceSheetQuotaHeading': 'Quota and Usage Overview',
  'keys.workspaceSheetCurrentAvailable': 'Available Snow Credits',
  'keys.workspaceSheetTotalLimit': 'Total Usage Limit',
  'keys.workspaceSheetTodayUsage': 'Today Billing',
  'keys.workspaceSheetThirtyDayUsage': 'Last 30 Days Billing',
  'keys.workspaceSheetRateHeading': 'Rate Limits and Windows',
  'keys.workspaceResetWindow': 'Reset Window',
  'keys.workspaceResetInSheet': '{time} 后重置',
  'keys.workspaceRate5hSheet': '5-hour Limit',
  'keys.workspaceRate1dSheet': 'Daily Limit (24H)',
  'keys.workspaceRate7dSheet': '7-day Limit (168H)',
  'keys.workspaceSheetSecurityHeading': 'Security and History Audit',
  'keys.workspaceSheetConcurrencyLimit': 'Concurrency Window Limit',
  'keys.workspaceSheetExpiration': 'Expiration Date',
  'keys.workspaceCreatedSheet': 'Creation time',
  'keys.workspaceLastUsedSheet': 'Previous use',
  'keys.workspaceLastIpSheet': 'Last IP',
  'keys.serviceTierPreferenceLabel': 'Fast mode (Priority)',
  'keys.serviceTierPriority': 'Priority',
  'keys.serviceTierStandard': 'Standard',
  'keys.serviceTierPreferenceHint': 'Verified upstreams only',
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

const IconStub = {
  name: 'Icon',
  props: ['name'],
  template: '<span :data-icon="name">{{ name }}</span>',
}

const GroupBadgeStub = {
  name: 'GroupBadge',
  props: ['name', 'platform', 'subscriptionType', 'rateMultiplier', 'userRateMultiplier'],
  template: '<span data-test="group-badge">{{ name }}</span>',
}

const EndpointPopoverStub = {
  name: 'EndpointPopover',
  props: ['apiBaseUrl', 'customEndpoints'],
  template: `
    <div data-test="endpoint-popover-stub">
      <span>{{ apiBaseUrl }}</span>
      <span v-for="endpoint in customEndpoints" :key="endpoint.endpoint">{{ endpoint.endpoint }}</span>
    </div>
  `,
}

function makeKey(overrides: Partial<ApiKey> = {}): ApiKey {
  return {
    id: 7,
    user_id: 1,
    key: 'sk-inspector-super-secret-key',
    name: 'Production key',
    group_id: 12,
    status: 'active',
    ip_whitelist: ['192.0.2.10', '10.0.0.0/8'],
    ip_blacklist: ['198.51.100.9'],
    last_used_at: '2026-07-18T08:30:00Z',
    last_used_ip: '203.0.113.42',
    quota: 100,
    quota_used: 25,
    expires_at: '2026-08-19T00:00:00Z',
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-19T00:00:00Z',
    current_concurrency: 4,
    rate_limit_5h: 10,
    rate_limit_1d: 20,
    rate_limit_7d: 0,
    usage_5h: 2.5,
    usage_1d: 8,
    usage_7d: 0.75,
    window_5h_start: '2026-07-19T00:00:00Z',
    window_1d_start: '2026-07-19T00:00:00Z',
    window_7d_start: '2026-07-19T00:00:00Z',
    reset_5h_at: '2026-07-20T00:00:00Z',
    reset_1d_at: '2026-07-21T00:00:00Z',
    reset_7d_at: null,
    group: {
      id: 12,
      name: 'Codex Plus',
      platform: 'openai',
      subscription_type: 'standard',
      rate_multiplier: 1,
    } as ApiKey['group'],
    ...overrides,
  }
}

function makeSettings(overrides: Partial<PublicSettings> = {}): PublicSettings {
  return {
    api_base_url: '',
    custom_endpoints: [],
    hide_ccs_import_button: false,
    ...overrides,
  } as PublicSettings
}

function mountInspector(
  apiKey = makeKey(),
  props: Record<string, unknown> = {},
  attachTo?: Element,
) {
  return mount(ApiKeyInspector, {
    attachTo,
    props: {
      apiKey,
      usage: {
        api_key_id: apiKey.id,
        today_actual_cost: 1.25,
        total_actual_cost: 12.75,
      },
      userGroupRate: 0.8,
      now: new Date('2026-07-19T00:00:00Z'),
      ...props,
    },
    global: {
      stubs: {
        Icon: IconStub,
        GroupBadge: GroupBadgeStub,
        EndpointPopover: EndpointPopoverStub,
      },
    },
  })
}

function creditValues(element: { findAll: (selector: string) => Array<{ text: () => string }> }) {
  return element.findAll('[data-testid="credit-amount-value"]').map((value) => value.text())
}

function inspectorDateTime(value: string | Date) {
  return formatDateTime(value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }, 'sv-SE')
}

function definitionValue(
  element: { findAll: (selector: string) => Array<{ element: Element; text: () => string }> },
  label: string,
) {
  const term = element.findAll('dt').find((candidate) => candidate.text() === label)
  if (!term) throw new Error(`Definition term not found: ${label}`)
  return term.element.nextElementSibling?.textContent?.trim() ?? ''
}

function definitionLabels(
  element: { findAll: (selector: string) => Array<{ text: () => string }> },
) {
  return element.findAll('dt').map((term) => term.text())
}

describe('ApiKeyInspector', () => {
  it('renders the complete quota, usage, rate-limit, audit, and IP detail contract', () => {
    const apiKey = makeKey()
    const wrapper = mountInspector(apiKey)

    expect(wrapper.get('[data-test="api-key-inspector"]').attributes()).toMatchObject({
      'data-key-id': '7',
      'data-mode': 'inline',
      role: 'region',
    })
    expect(wrapper.text()).toContain('Production key')
    expect(wrapper.text()).toContain('KEY ID: #7')
    expect(wrapper.text()).toContain('Enabled')
    expect(wrapper.find('[data-test="group-badge"]').exists()).toBe(false)
    const serviceTierSwitch = wrapper.get('[data-test="key-inspector-service-tier-switch-7"]')
    expect(serviceTierSwitch.attributes('aria-checked')).toBe('false')
    expect(wrapper.get('[data-test="api-key-inspector-service-tier"]').text()).toContain('Standard')

    const quota = wrapper.get('[data-test="api-key-inspector-quota"]')
    expect(creditValues(quota)).toEqual(['75.00', '100.00', '1.25', '12.75'])
    expect(quota.text()).toContain('SNOW CREDITS')
    expect(quota.findAll('[data-testid="snowflake-credit-icon"]')).toHaveLength(4)
    expect(quota.find('[role="progressbar"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="key-inspector-reset-quota-7"]').attributes('disabled')).toBeUndefined()

    const usage = wrapper.get('[data-test="api-key-inspector-usage"]')
    expect(creditValues(usage)).toEqual(['1.25', '12.75'])

    const rateSection = wrapper.get('[data-test="api-key-inspector-rate-limits"]')
    expect(rateSection.text()).toContain('Rate Limits (Snow credits)')
    expect(rateSection.text()).not.toContain('USD')

    const rateItems = rateSection.findAll('.api-key-inspector__rate-item')
    expect(rateItems).toHaveLength(3)
    expect(rateItems[0].text()).toContain('5h')
    expect(creditValues(rateItems[0])).toEqual(['2.50', '10.00'])
    expect(rateItems[0].get('[role="progressbar"]').attributes('aria-valuenow')).toBe('25')
    expect(rateItems[0].get('.api-key-inspector__reset-time').text()).toBe('RESET IN: 1d 0h')
    expect(rateItems[1].text()).toContain('1d')
    expect(creditValues(rateItems[1])).toEqual(['8.00', '20.00'])
    expect(rateItems[1].get('[role="progressbar"]').attributes('aria-valuenow')).toBe('40')
    expect(rateItems[1].get('.api-key-inspector__reset-time').text()).toBe('RESET IN: 2d 0h')
    expect(rateItems[2].text()).toContain('7d')
    expect(creditValues(rateItems[2])).toEqual(['0.75'])
    expect(rateItems[2].text()).toContain('Not set')
    expect(rateItems[2].find('.api-key-inspector__reset-time').exists()).toBe(false)
    expect(rateItems[2].text()).not.toContain('N/A')
    expect(rateItems[2].find('[role="progressbar"]').exists()).toBe(false)

    const audit = wrapper.get('[data-test="api-key-inspector-audit"]')
    expect(definitionLabels(audit)).toEqual([
      'Expires',
      'Concurrency Limit',
      'IP allowlist',
      'IP blocklist',
      'Created on',
      'Last used',
      'Recent IP',
    ])
    expect(definitionValue(audit, 'Concurrency Limit')).toBe('Unlimited')
    expect(audit.text()).not.toContain('4 / Unlimited')
    expect(audit.text()).toContain(inspectorDateTime(apiKey.expires_at!))
    expect(audit.text()).toContain(inspectorDateTime(apiKey.created_at))
    expect(audit.text()).toContain(inspectorDateTime(apiKey.last_used_at!))
    expect(audit.text()).toContain('203.0.113.42')

    const ipRules = wrapper.get('[data-test="api-key-inspector-ip-rules"]')
    expect(definitionValue(ipRules, 'IP allowlist')).toContain('2 items configured')
    expect(definitionValue(ipRules, 'IP blocklist')).toContain('1 items configured')
    for (const entry of ['192.0.2.10', '10.0.0.0/8', '198.51.100.9']) {
      expect(ipRules.text()).toContain(entry)
    }
    expect(wrapper.text()).not.toContain('$')
  })

  it('keeps the full key out of the DOM by default and remasks when the selected ID changes', async () => {
    const first = makeKey()
    const wrapper = mountInspector(first)

    expect(wrapper.html()).not.toContain(first.key)
    expect(wrapper.text()).toContain('sk-ins...-key')

    await wrapper.get('[data-test="key-inspector-reveal-7"]').trigger('click')
    expect(wrapper.text()).toContain(first.key)

    await wrapper.get('[data-test="key-inspector-copy-7"]').trigger('click')
    expect(wrapper.emitted('copy-key')?.[0]?.[0]).toMatchObject({ id: 7, key: first.key })

    const second = makeKey({
      id: 8,
      key: 'sk-second-private-value',
      name: 'Second key',
    })
    await wrapper.setProps({ apiKey: second })

    expect(wrapper.html()).not.toContain(second.key)
    expect(wrapper.text()).not.toContain(first.key)
    expect(wrapper.text()).toContain('sk-sec...alue')
    expect(wrapper.find('[data-test="key-inspector-reveal-7"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="key-inspector-reveal-8"]').exists()).toBe(true)
  })

  it('shows configured endpoints only and obeys both CCS visibility controls', async () => {
    const wrapper = mountInspector(makeKey(), {
      publicSettings: makeSettings(),
    })

    expect(wrapper.find('[data-test="api-key-inspector-endpoints"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('Import CCS')

    await wrapper.setProps({
      publicSettings: makeSettings({
        api_base_url: 'https://api.example.invalid/v1',
        custom_endpoints: [
          {
            name: 'Messages',
            endpoint: 'https://messages.example.invalid/v1',
            description: 'Messages endpoint',
          },
          { name: 'Empty', endpoint: '', description: '' },
        ],
      }),
    })

    const endpoints = wrapper.getComponent(EndpointPopoverStub)
    expect(endpoints.props('apiBaseUrl')).toBe('https://api.example.invalid/v1')
    expect(endpoints.props('customEndpoints')).toEqual([
      {
        name: 'Messages',
        endpoint: 'https://messages.example.invalid/v1',
        description: 'Messages endpoint',
      },
    ])

    await wrapper.setProps({ showCcsImport: false })
    expect(wrapper.text()).not.toContain('Import CCS')

    await wrapper.setProps({
      showCcsImport: true,
      publicSettings: makeSettings({ hide_ccs_import_button: true }),
    })
    expect(wrapper.text()).not.toContain('Import CCS')
    expect(wrapper.find('[data-test="api-key-inspector-endpoints"]').exists()).toBe(false)
  })

  it('emits sheet status, reset, and footer commands for the selected key', async () => {
    const wrapper = mountInspector(makeKey(), { mode: 'sheet' })
    const audit = wrapper.get('[data-test="api-key-inspector-audit"]')
    const rateItems = wrapper.get('[data-test="api-key-inspector-rate-limits"]')
      .findAll('.api-key-inspector__rate-item')

    expect(wrapper.get('.api-key-inspector__sheet-status-control strong').text()).toBe('ACTIVE')
    expect(definitionLabels(audit)).toEqual([
      'Concurrency Window Limit',
      'Expiration Date',
      'IP allowlist',
      'IP blocklist',
      'Creation time',
      'Previous use',
      'Last IP',
    ])
    expect(definitionValue(audit, 'Concurrency Window Limit')).toBe('Unlimited')
    expect(audit.text()).not.toContain('4 / Unlimited')
    expect(rateItems[0].get('.api-key-inspector__reset-time').text()).toBe('1d 0h 后重置')
    expect(rateItems[2].find('.api-key-inspector__reset-time').exists()).toBe(false)

    await wrapper.get('[data-test="key-inspector-status-switch-7"]').trigger('click')
    await wrapper.get('[data-test="key-inspector-service-tier-switch-7"]').trigger('click')
    await wrapper.get('[data-test="key-inspector-reset-quota-7"]').trigger('click')
    await wrapper.get('[data-test="key-inspector-reset-rate-limit-7"]').trigger('click')

    for (const [label, event] of [
      ['Use key', 'use-key'],
      ['Import CCS', 'import-ccs'],
      ['Edit API Key', 'edit'],
      ['Delete API Key', 'delete'],
    ] as const) {
      const button = wrapper.get('[data-test="api-key-inspector-actions"]')
        .findAll('button')
        .find((candidate) => candidate.text().includes(label))
      expect(button, `${label} command should exist`).toBeDefined()
      await button!.trigger('click')
      expect(wrapper.emitted(event)?.[0]?.[0]).toMatchObject({ id: 7 })
    }

    expect(wrapper.emitted('toggle-status')?.[0]?.[0]).toMatchObject({ id: 7 })
    expect(wrapper.emitted('toggle-service-tier')?.[0]?.[0]).toMatchObject({ id: 7 })
    expect(wrapper.emitted('change-group')).toBeUndefined()
    expect(wrapper.emitted('reset-quota')?.[0]?.[0]).toMatchObject({ id: 7 })
    expect(wrapper.emitted('reset-rate-limit')?.[0]?.[0]).toMatchObject({ id: 7 })
  })

  it('renders explicit unlimited and empty states and locks an updating status switch', async () => {
    const wrapper = mountInspector(makeKey({
      status: 'quota_exhausted',
      quota: 0,
      quota_used: 0,
      ip_whitelist: [],
      ip_blacklist: [],
      expires_at: null,
      last_used_at: null,
      last_used_ip: null,
      rate_limit_5h: 0,
      rate_limit_1d: 0,
      rate_limit_7d: 0,
      usage_5h: 0,
      usage_1d: 0,
      usage_7d: 0,
      reset_5h_at: null,
      reset_1d_at: null,
      reset_7d_at: null,
    }), {
      usage: undefined,
      statusUpdating: true,
      mode: 'sheet',
    })

    expect(wrapper.text()).toContain('QUOTA EXHAUSTED')
    expect(wrapper.text()).toContain('Unlimited')
    expect(wrapper.text()).toContain('Never')
    expect(wrapper.get('[data-test="api-key-inspector-quota"]').find('[role="progressbar"]').exists()).toBe(false)
    expect(wrapper.findAll('.api-key-inspector__rate-item [role="progressbar"]')).toHaveLength(0)
    expect(wrapper.get('[data-test="key-inspector-reset-quota-7"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-test="key-inspector-reset-rate-limit-7"]').attributes('disabled')).toBeDefined()

    const audit = wrapper.get('[data-test="api-key-inspector-audit"]')
    const ipRules = wrapper.get('[data-test="api-key-inspector-ip-rules"]')
    expect(definitionValue(audit, 'Concurrency Window Limit')).toBe('Unlimited')
    expect(definitionValue(ipRules, 'IP allowlist')).toBe('0 items configured')
    expect(definitionValue(ipRules, 'IP blocklist')).toBe('0 items configured')
    expect(ipRules.find('code').exists()).toBe(false)
    expect(ipRules.text()).not.toContain('Example:')
    expect(wrapper.findAll('.api-key-inspector__reset-time')).toHaveLength(0)

    const statusSwitch = wrapper.get('[data-test="key-inspector-status-switch-7"]')
    expect(statusSwitch.attributes('aria-checked')).toBe('false')
    expect(statusSwitch.attributes('aria-busy')).toBe('true')
    expect(statusSwitch.attributes('disabled')).toBeDefined()
    await statusSwitch.trigger('click')
    expect(wrapper.emitted('toggle-status')).toBeUndefined()
  })

  it('obeys optional detail-field visibility preferences', () => {
    const wrapper = mountInspector(makeKey(), {
      visibleColumns: ['name', 'status', 'usage', 'group', 'key', 'actions', 'expires_at'],
    })

    expect(wrapper.text()).toContain('KEY ID: #7')
    expect(wrapper.get('[data-test="api-key-inspector-audit"]').text()).not.toContain('Current concurrency')
    expect(wrapper.find('[data-test="api-key-inspector-rate-limits"]').exists()).toBe(false)

    const audit = wrapper.get('[data-test="api-key-inspector-audit"]')
    expect(audit.text()).toContain('Expires')
    expect(audit.text()).not.toContain('Created on')
    expect(audit.text()).not.toContain('Last used')
    expect(audit.text()).not.toContain('Recent IP')
  })

  it('hides Fast mode for non-OpenAI groups and reflects a priority preference', () => {
    const nonOpenAi = makeKey({
      group: { ...makeKey().group!, platform: 'anthropic' },
    })
    const hiddenWrapper = mountInspector(nonOpenAi)
    expect(hiddenWrapper.find('[data-test="api-key-inspector-service-tier"]').exists()).toBe(false)

    const priorityWrapper = mountInspector(makeKey({ service_tier_preference: 'priority' }))
    const prioritySwitch = priorityWrapper.get('[data-test="key-inspector-service-tier-switch-7"]')
    expect(prioritySwitch.attributes('aria-checked')).toBe('true')
    expect(priorityWrapper.get('[data-test="api-key-inspector-service-tier"]').text()).toContain('Priority')
  })

  it('uses a non-dialog sheet region without duplicate header controls', async () => {
    const host = document.createElement('div')
    document.body.appendChild(host)
    const wrapper = mountInspector(makeKey(), { mode: 'sheet' }, host)
    const inspector = wrapper.get('[data-test="api-key-inspector"]')

    expect(inspector.attributes('role')).toBe('region')
    expect(inspector.attributes('aria-label')).toBe('Production key')
    expect(inspector.attributes('aria-modal')).toBeUndefined()
    expect(inspector.attributes('aria-labelledby')).toBeUndefined()
    expect(wrapper.find('.api-key-inspector__header').exists()).toBe(false)
    expect(wrapper.find('[data-test="api-key-inspector-close"]').exists()).toBe(false)

    ;(wrapper.vm as unknown as { focus: () => void }).focus()
    expect(document.activeElement).toBe(inspector.element)

    await inspector.trigger('keydown', { key: 'Escape' })
    expect(wrapper.emitted('close')).toBeUndefined()

    wrapper.unmount()
    host.remove()
  })
})
