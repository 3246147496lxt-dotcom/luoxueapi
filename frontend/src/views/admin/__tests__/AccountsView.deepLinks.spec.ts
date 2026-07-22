import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type LocationQueryRaw, type Router } from 'vue-router'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getAllProxies,
  getAllGroups,
  accountRequests
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  accountRequests: [] as Array<Record<string, unknown>>
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({ enabled: false, interval_minutes: 30 }),
      updateUpstreamBillingProbeSettings: vi.fn(),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      setSchedulable: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token' })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  TablePageLayout: {
    template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
  },
  DataTable: true,
  HelpTooltip: true,
  Toggle: true,
  Pagination: true,
  ConfirmDialog: true,
  TotpStepUpDialog: true,
  AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
  AccountTableFilters: true,
  AccountBulkActionsBar: true,
  AccountActionMenu: true,
  ImportDataModal: true,
  ReAuthAccountModal: true,
  AccountTestModal: true,
  AccountStatsModal: true,
  ScheduledTestsPanel: true,
  SyncFromCrsModal: true,
  TempUnschedStatusModal: true,
  ErrorPassthroughRulesModal: true,
  TLSFingerprintProfilesModal: true,
  CreateAccountModal: true,
  EditAccountModal: true,
  BulkEditAccountModal: true,
  PlatformTypeBadge: true,
  AccountCapacityCell: true,
  AccountStatusIndicator: true,
  AccountTodayStatsCell: true,
  AccountGroupsCell: true,
  AccountUsageCell: true,
  UpstreamBillingRateCell: true,
  Icon: true
}

async function mountView(query: LocationQueryRaw): Promise<{ wrapper: VueWrapper; router: Router }> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/admin/accounts', component: { template: '<div />' } }]
  })
  await router.push({ path: '/admin/accounts', query })
  await router.isReady()

  const wrapper = mount(AccountsView, {
    global: {
      plugins: [router],
      stubs
    }
  })
  await flushPromises()
  return { wrapper, router }
}

describe('admin AccountsView precise deep links', () => {
  beforeEach(() => {
    localStorage.clear()
    accountRequests.length = 0
    for (const mock of [listAccounts, listWithEtag, getBatchTodayStats, getAllProxies, getAllGroups]) {
      mock.mockReset()
    }
    listAccounts.mockImplementation(async (_page, _pageSize, filters) => {
      accountRequests.push({ ...filters })
      return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    })
    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('initializes health, platform and group while account_id takes search priority', async () => {
    const { wrapper } = await mountView({
      health: 'error',
      platform: 'openai',
      group: '7',
      account_id: '42',
      proxy_id: '9',
      search: 'ignored-by-account-focus'
    })

    expect(accountRequests).toHaveLength(1)
    expect(accountRequests[0]).toEqual(expect.objectContaining({
      status: 'error',
      platform: 'openai',
      group: '7',
      search: '#42'
    }))
    wrapper.unmount()
  })

  it.each([
    [{ proxy_id: '13' }, 'proxy:13'],
    [{ search: 'north-pool' }, 'north-pool'],
    [{ group: 'ungrouped' }, '']
  ] as const)('initializes the focused search contract for %o', async (query, expectedSearch) => {
    const { wrapper } = await mountView(query)

    expect(accountRequests).toHaveLength(1)
    expect(accountRequests[0]?.search).toBe(expectedSearch)
    if ('group' in query) expect(accountRequests[0]?.group).toBe('ungrouped')
    wrapper.unmount()
  })

  it('removes illegal managed parameters, preserves unrelated query state and loads once', async () => {
    const { wrapper, router } = await mountView({
      platform: 'invalid-platform',
      health: 'on-fire',
      group: '-2',
      account_id: 'zero',
      proxy_id: '0',
      search: 'fallback-search',
      source: 'ops'
    })

    expect(router.currentRoute.value.query).toEqual({ search: 'fallback-search', source: 'ops' })
    expect(accountRequests).toHaveLength(1)
    expect(accountRequests[0]).toEqual(expect.objectContaining({
      platform: '',
      status: '',
      group: '',
      search: 'fallback-search'
    }))
    wrapper.unmount()
  })

  it('reloads exactly once when browser navigation changes the route filters', async () => {
    const { wrapper, router } = await mountView({ platform: 'openai', health: 'active' })
    expect(accountRequests).toHaveLength(1)

    await router.push({
      path: '/admin/accounts',
      query: { platform: 'gemini', health: 'rate_limited', group: '4', search: 'fallback' }
    })
    await flushPromises()

    expect(accountRequests).toHaveLength(2)
    expect(accountRequests[1]).toEqual(expect.objectContaining({
      platform: 'gemini',
      status: 'rate_limited',
      group: '4',
      search: 'fallback'
    }))
    wrapper.unmount()
  })
})
