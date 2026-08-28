import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountInspector from '../AccountInspector.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Account, Proxy, WindowStats } from '@/types'

const { queryQuotaMock } = vi.hoisted(() => ({
  queryQuotaMock: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      te: () => true,
      t: (key: string) => key
    }
  },
  getLocale: () => 'en-US'
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    cachedPublicSettings: null
  })
}))

vi.mock('@/components/account/AccountUsageCell.vue', () => {
  let instanceSequence = 0
  return {
    default: {
      name: 'AccountUsageCell',
      props: ['account', 'todayStats', 'todayStatsLoading', 'manualRefreshToken', 'displayMode'],
      setup(_props: unknown, { expose }: { expose: (value: unknown) => void }) {
        expose({ queryQuota: queryQuotaMock })
      },
      data: () => ({ instanceId: ++instanceSequence }),
      template: `
        <div
          data-testid="account-inspector-quota-cell"
          :data-account-id="account.id"
          :data-display-mode="displayMode"
          :data-instance-id="instanceId"
        />
      `
    }
  }
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 41,
    name: 'primary-openai',
    notes: 'Prefer this route during office hours',
    platform: 'openai',
    type: 'oauth',
    credentials: {
      auth_mode: 'personal_access_token',
      plan_type: 'plus'
    },
    extra: {},
    proxy_id: null,
    concurrency: 4,
    priority: 20,
    rate_multiplier: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  }
}

function makeProxy(overrides: Partial<Proxy> = {}): Proxy {
  return {
    id: 9,
    name: 'Tokyo egress',
    protocol: 'socks5',
    host: 'proxy.internal.example',
    port: 1080,
    username: 'relay-user',
    password: 'must-never-render',
    status: 'active',
    ip_address: '203.0.113.42',
    country: 'Japan',
    country_code: 'JP',
    latency_ms: 83,
    latency_status: 'success',
    expires_at: null,
    fallback_mode: 'none',
    backup_proxy_id: null,
    expiry_warn_days: 7,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: '2026-07-01T00:00:00Z',
    ...overrides
  }
}

const todayStats: WindowStats = {
  requests: 128,
  tokens: 4096,
  cost: 1.25,
  standard_cost: 1.5,
  user_cost: 1.8
}

describe('AccountInspector', () => {
  it('keeps the default section order and rebuilds quota state per account', async () => {
    const wrapper = mount(AccountInspector, {
      props: {
        account: makeAccount({ current_concurrency: 2, concurrency: 4 }),
        todayStats,
        todayStatsLoading: true,
        usageRefreshToken: 7
      }
    })

    const sections = wrapper.findAll('.account-inspector__content > .account-inspector__section')
    expect(sections.map(section => section.attributes('data-testid'))).toEqual([
      'account-inspector-quota',
      'account-inspector-usage',
      'account-inspector-policies',
      'account-inspector-groups-proxy'
    ])
    expect(sections.at(-2)?.attributes('data-testid')).toBe('account-inspector-policies')
    expect(sections.at(-1)?.attributes('data-testid')).toBe('account-inspector-groups-proxy')

    const quotaCell = wrapper.getComponent({ name: 'AccountUsageCell' })
    const firstQuotaInstance = quotaCell.attributes('data-instance-id')
    expect(quotaCell.props('account')).toMatchObject({ id: 41 })
    expect(quotaCell.props('todayStats')).toEqual(todayStats)
    expect(quotaCell.props('todayStatsLoading')).toBe(true)
    expect(quotaCell.props('manualRefreshToken')).toBe(7)
    expect(quotaCell.props('displayMode')).toBe('overview')
    expect(wrapper.get('[data-testid="account-inspector-concurrency"]').text())
      .toContain('admin.accounts.workbench.concurrencySlots')

    const content = wrapper.get('.account-inspector__content').element as HTMLElement
    content.scrollTop = 240
    await wrapper.setProps({ account: makeAccount({ id: 42 }) })
    await wrapper.vm.$nextTick()

    const nextQuotaCell = wrapper.getComponent({ name: 'AccountUsageCell' })
    expect(nextQuotaCell.props('account')).toMatchObject({ id: 42 })
    expect(nextQuotaCell.attributes('data-instance-id')).not.toBe(firstQuotaInstance)
    expect(content.scrollTop).toBe(0)
  })

  it('keeps scheduling priority, hides billing rate, and exposes quota recovery actions', async () => {
    queryQuotaMock.mockReset()
    queryQuotaMock.mockResolvedValue(undefined)
    const wrapper = mount(AccountInspector, {
      props: {
        account: makeAccount({
          priority: 20,
          rate_multiplier: 1.37
        }),
        todayStats
      }
    })

    const policies = wrapper.get('[data-testid="account-inspector-policies"]')
    expect(policies.text()).toContain('admin.accounts.workbench.priority')
    expect(policies.text()).toContain('20')
    expect(policies.text()).not.toContain('admin.accounts.workbench.billingRate')
    expect(policies.text()).not.toContain('1.37x')

    expect(wrapper.get('[data-testid="account-inspector-query-quota"]').text())
      .toContain('admin.accounts.usageWindow.activeQuery')
    expect(wrapper.get('[data-testid="openai-quota-reset-count"]').text())
      .toContain('admin.accounts.openaiQuotaReset.count')
    expect(wrapper.get('[data-testid="openai-quota-reset-button"]').attributes('disabled'))
      .toBeDefined()

    await wrapper.get('[data-testid="account-inspector-query-quota"]').trigger('click')
    expect(queryQuotaMock).toHaveBeenCalledTimes(1)
  })

  it('keeps account cost in USD and renders user cost as points', () => {
    const wrapper = mount(AccountInspector, {
      props: { account: makeAccount(), todayStats }
    })

    const usage = wrapper.get('[data-testid="account-inspector-usage"]')
    expect(usage.text()).toContain('$1.25')
    expect(usage.get('[data-testid="credit-amount-value"]').text()).toBe('1.80')
    expect(usage.text()).not.toContain('$1.80')
  })

  it('renders real account groups with GroupBadge', () => {
    const wrapper = mount(AccountInspector, {
      props: {
        account: makeAccount({
          groups: [
            {
              id: 12,
              name: 'Codex Plus',
              platform: 'openai',
              subscription_type: 'standard',
              rate_multiplier: 1
            }
          ] as Account['groups']
        }),
        todayStats
      }
    })

    const badge = wrapper.getComponent(GroupBadge)
    expect(badge.props()).toMatchObject({
      name: 'Codex Plus',
      platform: 'openai',
      subscriptionType: 'standard',
      rateMultiplier: 1,
      showRate: false
    })
  })

  it('emits only the proxy ID from a compact proxy summary and hides proxy internals', async () => {
    const account = makeAccount({ proxy_id: 9 })
    const wrapper = mount(AccountInspector, {
      props: {
        account,
        proxyTelemetry: makeProxy({ expires_at: '2030-08-09T10:11:12Z' }),
        todayStats
      }
    })

    const section = wrapper.get('[data-testid="account-inspector-groups-proxy"]')
    const proxyButton = wrapper.get('[data-testid="account-inspector-view-proxy"]')
    expect(proxyButton.text()).toContain('Tokyo egress')
    expect(proxyButton.text()).toContain('admin.accounts.workbench.proxyStatus.active')
    for (const hiddenValue of [
      'socks5',
      'proxy.internal.example',
      '1080',
      '203.0.113.42',
      'Japan',
      'JP',
      '83 ms',
      '2030-08-09',
      'relay-user',
      'must-never-render'
    ]) {
      expect(section.text()).not.toContain(hiddenValue)
      expect(section.html()).not.toContain(hiddenValue)
    }

    await proxyButton.trigger('click')
    expect(wrapper.emitted('viewProxy')).toEqual([[9]])
    expect(wrapper.emitted('revertFallback')).toBeUndefined()
  })

  it('shows only an allowlisted account email and prefers the parent email for shadow accounts', async () => {
    const secrets = {
      access_token: 'access-token-must-never-render',
      refresh_token: 'refresh-token-must-never-render',
      api_key: 'api-key-must-never-render',
      client_secret: 'client-secret-must-never-render'
    }
    const wrapper = mount(AccountInspector, {
      props: {
        account: makeAccount({
          credentials: {
            email: 'owner@example.com',
            client_email: 'fallback@example.com',
            ...secrets
          },
          extra: {
            email: 'extra@example.com',
            api_key: secrets.api_key
          }
        }),
        todayStats
      }
    })

    const email = wrapper.get('[data-testid="account-inspector-email"]')
    expect(email.text()).toBe('owner@example.com')
    expect(email.attributes('title')).toBe('owner@example.com')
    for (const secret of Object.values(secrets)) {
      expect(wrapper.text()).not.toContain(secret)
      expect(wrapper.html()).not.toContain(secret)
    }

    await wrapper.setProps({
      account: makeAccount({
        id: 42,
        parent_account_id: 41,
        quota_dimension: 'spark',
        parent_email: 'parent@example.com',
        credentials: {
          email: 'shadow-fallback@example.com',
          ...secrets
        }
      })
    })

    expect(wrapper.get('[data-testid="account-inspector-email"]').text())
      .toBe('parent@example.com')
  })

  it('shows the direct-route copy without a proxy action when no proxy is assigned', () => {
    const wrapper = mount(AccountInspector, {
      props: {
        account: makeAccount({
          proxy_id: null,
          proxy: undefined
        }),
        proxyTelemetry: null
      }
    })

    const section = wrapper.get('[data-testid="account-inspector-groups-proxy"]')
    expect(section.text()).toContain('admin.accounts.workbench.directRoute')
    expect(wrapper.find('[data-testid="account-inspector-view-proxy"]').exists()).toBe(false)
  })

  it('hides recovery for a healthy account and shows it only for real anomalies', async () => {
    const wrapper = mount(AccountInspector, {
      props: { account: makeAccount(), todayStats }
    })

    expect(wrapper.find('[data-testid="account-inspector-recovery"]').exists()).toBe(false)

    await wrapper.setProps({
      account: makeAccount({
        status: 'error',
        error_message: 'Upstream authentication failed'
      })
    })

    const recovery = wrapper.get('[data-testid="account-inspector-recovery"]')
    expect(recovery.text()).toContain('Upstream authentication failed')
    const sections = wrapper.findAll('.account-inspector__content > .account-inspector__section')
    expect(sections[0].attributes('data-testid')).toBe('account-inspector-recovery')
    expect(sections.map(section => section.attributes('data-testid'))).toEqual([
      'account-inspector-recovery',
      'account-inspector-quota',
      'account-inspector-usage',
      'account-inspector-policies',
      'account-inspector-groups-proxy'
    ])
  })

  it('uses a compact colored platform badge and emits the existing account actions', async () => {
    const account = makeAccount({
      proxy_fallback_origin_id: 3,
      proxy_fallback_origin_name: 'primary-proxy'
    })
    const wrapper = mount(AccountInspector, {
      props: { account, todayStats }
    })

    const badge = wrapper.getComponent(PlatformTypeBadge)
    expect(badge.props('platform')).toBe('openai')
    expect(badge.props('compact')).toBe(true)

    await wrapper.get('[data-testid="account-inspector-test"]').trigger('click')
    await wrapper.get('[data-testid="account-inspector-stats"]').trigger('click')
    await wrapper.get('[data-testid="account-inspector-edit"]').trigger('click')
    await wrapper.get('[data-testid="account-inspector-more"]').trigger('click')
    await wrapper.get('[data-testid="account-inspector-scheduling"]').trigger('click')
    await wrapper.get('[data-testid="account-inspector-revert-fallback"]').trigger('click')

    expect(wrapper.emitted('test')?.[0]?.[0]).toMatchObject({ id: account.id })
    expect(wrapper.emitted('stats')?.[0]?.[0]).toMatchObject({ id: account.id })
    expect(wrapper.emitted('edit')?.[0]?.[0]).toMatchObject({ id: account.id })
    expect(wrapper.emitted('more')?.[0]?.[0]).toMatchObject({ id: account.id })
    expect(wrapper.emitted('toggleSchedulable')?.[0]?.[0]).toMatchObject({ id: account.id })
    expect(wrapper.emitted('revertFallback')?.[0]?.[0]).toMatchObject({ id: account.id })
  })

  it('matches the approved Superdesign quota and footer icon set', () => {
    const wrapper = mount(AccountInspector, {
      props: { account: makeAccount(), todayStats }
    })

    const quotaTitle = wrapper.get(
      '.account-inspector__section--quota .account-inspector__section-title'
    )
    const quotaIcon = quotaTitle.getComponent(Icon)
    expect(quotaIcon.props()).toMatchObject({
      name: 'lucideBarChart3',
      size: 'sm',
      strokeWidth: 2
    })
    expect(quotaIcon.classes()).toContain('text-purple-500')
    expect(quotaTitle.find('.account-inspector__section-icon').exists()).toBe(false)
    expect(quotaIcon.get('path').attributes('d')).toBe(
      'M3 3v18h18m-3-4V9m-5 8V5M8 17v-3'
    )

    const testIcon = wrapper
      .get('[data-testid="account-inspector-test"]')
      .getComponent(Icon)
    expect(testIcon.props()).toMatchObject({
      name: 'lucidePlayCircle',
      size: 'xs',
      strokeWidth: 2
    })
    expect(testIcon.classes()).toContain('account-inspector__command-icon')
    expect(testIcon.classes()).toContain('text-green-500')
    expect(testIcon.attributes('aria-hidden')).toBe('true')
    expect(testIcon.get('path').attributes('d')).toBe(
      'M9 9.003a1 1 0 0 1 1.517-.859l4.997 2.997a1 1 0 0 1 0 1.718l-4.997 2.997A1 1 0 0 1 9 14.996z'
    )
    expect(testIcon.get('circle').attributes()).toMatchObject({
      cx: '12',
      cy: '12',
      r: '10'
    })

    const statsIcon = wrapper
      .get('[data-testid="account-inspector-stats"]')
      .getComponent(Icon)
    expect(statsIcon.props()).toMatchObject({
      name: 'lucideBarChartBig',
      size: 'xs',
      strokeWidth: 2
    })
    expect(statsIcon.classes()).toContain('text-indigo-500')
    expect(statsIcon.attributes('aria-hidden')).toBe('true')
    expect(statsIcon.get('path').attributes('d')).toBe('M3 3v18h18')
    expect(statsIcon.findAll('rect').map(rect => rect.attributes())).toEqual([
      { width: '4', height: '7', x: '7', y: '10', rx: '1' },
      { width: '4', height: '12', x: '15', y: '5', rx: '1' }
    ])

    const editIcon = wrapper
      .get('[data-testid="account-inspector-edit"]')
      .getComponent(Icon)
    expect(editIcon.props()).toMatchObject({
      name: 'lucideEdit3',
      size: 'xs',
      strokeWidth: 2
    })
    expect(editIcon.classes()).toContain('text-gray-500')
    expect(editIcon.attributes('aria-hidden')).toBe('true')
    expect(editIcon.get('path').attributes('d')).toBe(
      'M13 21h8m.174-14.188a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z'
    )
  })

  it('closes with Escape in both inline and overlay modes', async () => {
    for (const mode of ['inline', 'drawer', 'sheet'] as const) {
      const wrapper = mount(AccountInspector, {
        props: { account: makeAccount(), mode }
      })
      await wrapper.get('[data-testid="account-inspector"]').trigger('keydown', { key: 'Escape' })
      expect(wrapper.emitted('close')).toHaveLength(1)
      wrapper.unmount()
    }
  })

  it('keeps reverse tab navigation inside overlay inspectors', async () => {
    const wrapper = mount(AccountInspector, {
      attachTo: document.body,
      props: {
        account: makeAccount(),
        mode: 'drawer'
      }
    })
    const inspector = wrapper.get('[data-testid="account-inspector"]')
    const lastAction = wrapper.get('[data-testid="account-inspector-edit"]')

    ;(inspector.element as HTMLElement).focus()
    await inspector.trigger('keydown', { key: 'Tab', shiftKey: true })

    expect(document.activeElement).toBe(lastAction.element)
    wrapper.unmount()
  })
})
