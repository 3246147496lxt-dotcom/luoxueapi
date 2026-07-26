import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AccountsView from '../AccountsView.vue'
import {
  getAccountUsageHealthSnapshot,
  invalidateAccountUsageHealthSnapshot,
  publishAccountUsage
} from '@/composables/useAccountUsageHealth'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getAllProxies,
  getAllGroups,
  getUsage,
  getAccountById,
  setSchedulable,
  routerPush
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  getUsage: vi.fn(),
  getAccountById: vi.fn(),
  setSchedulable: vi.fn(),
  routerPush: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getUsage,
      getById: getAccountById,
      getUpstreamBillingProbeSettings: vi.fn().mockResolvedValue({
        enabled: false,
        interval_minutes: 30
      }),
      setSchedulable,
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn()
    },
    proxies: {
      getAll: getAllProxies,
      getAllWithCount: getAllProxies
    },
    groups: {
      getAll: getAllGroups
    }
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
  useAuthStore: () => ({
    token: 'test-token',
    isSimpleMode: false
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({
    replace: vi.fn().mockResolvedValue(undefined),
    push: routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const DataTableStub = {
  name: 'DataTableStub',
  props: ['columns', 'data', 'selectedRowKey'],
  emits: ['rowClick', 'sort'],
  template: `
    <div data-test="workbench-table" :data-selected-row-key="selectedRowKey ?? ''">
      <div
        v-for="row in data"
        :key="row.id"
        :data-test="'account-row-' + row.id"
        :data-row-id="row.id"
        tabindex="0"
        @click="$emit('rowClick', row)"
      >
        <slot name="cell-select" :row="row" />
        <slot name="cell-name" :row="row" :value="row.name" />
        <slot name="cell-usage" :row="row" />
        <slot name="cell-status" :row="row" />
        <slot name="cell-groups" :row="row" />
        <slot name="cell-actions" :row="row" />
        <span :data-test="'account-error-' + row.id">{{ row.error_message }}</span>
      </div>
    </div>
  `
}

const AccountInspectorStub = {
  name: 'AccountInspectorStub',
  props: ['account', 'todayStats', 'proxyTelemetry', 'mode'],
  emits: [
    'close',
    'test',
    'stats',
    'edit',
    'more',
    'revertFallback',
    'toggleSchedulable',
    'viewProxy'
  ],
  methods: {
    focus: vi.fn()
  },
  template: `
    <div
      data-test="workbench-inspector"
      :data-account-id="account?.id ?? ''"
      :data-mode="mode"
    >
      <span data-test="inspector-account-name">{{ account?.name ?? 'none' }}</span>
      <button v-if="account" data-test="inspector-close" @click="$emit('close')">close</button>
      <button v-if="account" data-test="inspector-test" @click="$emit('test', account)">test</button>
      <button v-if="account" data-test="inspector-stats" @click="$emit('stats', account)">stats</button>
      <button v-if="account" data-test="inspector-edit" @click="$emit('edit', account)">edit</button>
      <button
        v-if="account && (proxyTelemetry?.id ?? account.proxy_id)"
        data-test="inspector-view-proxy"
        @click="$emit('viewProxy', proxyTelemetry?.id ?? account.proxy_id)"
      >
        proxy
      </button>
    </div>
  `
}

const AccountTableActionsStub = {
  emits: ['refresh', 'create'],
  template: `
    <div>
      <button data-test="manual-refresh" @click="$emit('refresh')">refresh</button>
      <slot name="after" />
    </div>
  `
}

const AccountUsageCellStub = {
  props: ['displayMode'],
  template: '<div class="account-usage-summary" :data-display-mode="displayMode"></div>'
}

const modalStub = (testId: string) => ({
  props: ['show', 'account'],
  emits: ['close', 'completed'],
  template: `
    <div data-test="${testId}" :data-show="String(show)" :data-account-id="account?.id ?? ''">
      <button data-test="${testId}-completed" @click="$emit('completed')">completed</button>
    </div>
  `
})

const mountedWrappers: Array<{ unmount: () => void }> = []

function makeAccount(name: string, updatedAt: string, id = 1) {
  return {
    id,
    name,
    platform: 'openai',
    type: 'oauth',
    credentials: {},
    extra: {},
    proxy_id: null as number | null,
    concurrency: 4,
    priority: 10,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-07-01T00:00:00Z',
    updated_at: updatedAt,
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    groups: []
  }
}

function responseFor(...accounts: ReturnType<typeof makeAccount>[]) {
  return {
    items: accounts,
    total: accounts.length,
    page: 1,
    page_size: 20,
    pages: 1
  }
}

function mountView() {
  const wrapper = mount(AccountsView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        AdminPageHeader: true,
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /></div>'
        },
        DataTable: DataTableStub,
        AccountInspector: AccountInspectorStub,
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: AccountTableActionsStub,
        AccountTableFilters: { template: '<div></div>' },
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: modalStub('test-modal'),
        AccountStatsModal: modalStub('stats-modal'),
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: modalStub('edit-modal'),
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: AccountUsageCellStub,
        UpstreamBillingRateCell: true,
        HelpTooltip: true,
        Icon: true,
        Toggle: true
      }
    }
  })
  mountedWrappers.push(wrapper)
  return wrapper
}

function setViewport(width: number) {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: width
  })
}

describe('admin AccountsView workbench', () => {
  beforeEach(() => {
    localStorage.clear()
    Object.defineProperty(window, 'innerWidth', {
      configurable: true,
      value: 1440
    })

    for (const mock of [
      listAccounts,
      listWithEtag,
      getBatchTodayStats,
      getAllProxies,
      getAllGroups,
      getUsage,
      getAccountById,
      setSchedulable,
      routerPush
    ]) {
      mock.mockReset()
    }

    listWithEtag.mockResolvedValue({ notModified: true, etag: null, data: null })
    getBatchTodayStats.mockResolvedValue({
      stats: {
        '1': { requests: 5, tokens: 200, cost: 0.1, standard_cost: 0.2, user_cost: 0.3 }
      }
    })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
    setSchedulable.mockResolvedValue({ schedulable: false })
    routerPush.mockResolvedValue(undefined)
    getAccountById.mockResolvedValue(makeAccount('refreshed-after-test', '2026-07-01T00:02:00Z'))
  })

  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) {
      wrapper.unmount()
    }
    document.body.style.overflow = ''
    document.body.replaceChildren()
  })

  it('auto-selects the first wide-screen row and keeps selection by ID after refresh', async () => {
    listAccounts
      .mockResolvedValueOnce(responseFor(
        makeAccount('old-name', '2026-07-01T00:00:00Z'),
        makeAccount('second-name', '2026-07-01T00:00:00Z', 2)
      ))
      .mockResolvedValueOnce(responseFor(
        makeAccount('fresh-name', '2026-07-01T00:01:00Z'),
        makeAccount('second-name', '2026-07-01T00:01:00Z', 2)
      ))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('old-name')
    expect(wrapper.get('[data-test="workbench-table"]').attributes('data-selected-row-key')).toBe('1')

    await wrapper.get('[data-test="manual-refresh"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('fresh-name')
    expect(wrapper.get('[data-test="workbench-table"]').attributes('data-selected-row-key')).toBe('1')
    expect(getUsage).not.toHaveBeenCalled()
  })

  it('does not reopen the wide inspector after it is closed and manually refreshed', async () => {
    listAccounts
      .mockResolvedValueOnce(responseFor(makeAccount('old-name', '2026-07-01T00:00:00Z')))
      .mockResolvedValueOnce(responseFor(makeAccount('fresh-name', '2026-07-01T00:01:00Z')))

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('old-name')
    await wrapper.get('[data-test="inspector-close"]').trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('none')

    await wrapper.get('[data-test="manual-refresh"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="account-row-1"]').text()).toContain('fresh-name')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('none')
    expect(wrapper.get('[data-test="workbench-table"]').attributes('data-selected-row-key')).toBe('')
  })

  it('does not select the row when its checkbox is clicked', async () => {
    listAccounts.mockResolvedValue(responseFor(makeAccount('row-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="inspector-close"]').trigger('click')
    await wrapper.get('[data-test="account-row-1"] input[type="checkbox"]').trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('none')
    expect(wrapper.get('[data-test="workbench-table"]').attributes('data-selected-row-key')).toBe('')
  })

  it('opens from status, capacity, and quota summary areas', async () => {
    listAccounts.mockResolvedValue(responseFor(makeAccount('row-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="inspector-close"]').trigger('click')
    await wrapper.get('[data-test="account-row-1"] .account-status-cell').trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('row-account')

    await wrapper.get('[data-test="inspector-close"]').trigger('click')
    await wrapper.get('[data-test="account-row-1"] .account-quota-capacity__capacity').trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('row-account')

    await wrapper.get('[data-test="inspector-close"]').trigger('click')
    await wrapper.get('[data-test="account-row-1"] .account-usage-summary').trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('row-account')
  })

  it('closes the inline inspector with Escape from the selected desktop row', async () => {
    listAccounts.mockResolvedValue(responseFor(makeAccount('keyboard-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-1"]')
    const rowElement = row.element as HTMLElement
    rowElement.focus()
    await row.trigger('click')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('keyboard-account')

    await row.trigger('keydown', { key: 'Escape' })
    await flushPromises()

    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('none')
    expect(wrapper.get('[data-test="workbench-table"]').attributes('data-selected-row-key')).toBe('')
    expect(document.activeElement).toBe(rowElement)
  })

  it('wires the three inspector commands to the existing modals', async () => {
    listAccounts.mockResolvedValue(responseFor(makeAccount('action-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="account-row-1"]').trigger('click')

    await wrapper.get('[data-test="inspector-test"]').trigger('click')
    expect(wrapper.get('[data-test="test-modal"]').attributes()).toMatchObject({
      'data-show': 'true',
      'data-account-id': '1'
    })

    await wrapper.get('[data-test="inspector-stats"]').trigger('click')
    expect(wrapper.get('[data-test="stats-modal"]').attributes()).toMatchObject({
      'data-show': 'true',
      'data-account-id': '1'
    })

    await wrapper.get('[data-test="inspector-edit"]').trigger('click')
    expect(wrapper.get('[data-test="edit-modal"]').attributes()).toMatchObject({
      'data-show': 'true',
      'data-account-id': '1'
    })
  })

  it('opens proxy details from the wide inline inspector', async () => {
    listAccounts.mockResolvedValue(responseFor({
      ...makeAccount('proxied-account', '2026-07-01T00:00:00Z'),
      proxy_id: 9
    }))
    getAllProxies.mockResolvedValue([{
      id: 9,
      name: 'Tokyo egress',
      status: 'active',
      username: 'must-not-cross-inspector-boundary',
      password: 'must-not-cross-inspector-boundary'
    }])

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.getComponent(AccountInspectorStub).props('proxyTelemetry')).toEqual({
      id: 9,
      name: 'Tokyo egress',
      status: 'active'
    })
    await wrapper.get('[data-test="inspector-view-proxy"]').trigger('click')

    expect(routerPush).toHaveBeenCalledTimes(1)
    expect(routerPush).toHaveBeenCalledWith({
      name: 'AdminProxies',
      query: {
        focus_id: '9',
        open: 'health'
      }
    })
  })

  it('opens proxy details from the overlay inspector', async () => {
    setViewport(1200)
    listAccounts.mockResolvedValue(responseFor({
      ...makeAccount('proxied-account', '2026-07-01T00:00:00Z'),
      proxy_id: 9
    }))
    getAllProxies.mockResolvedValue([{
      id: 9,
      name: 'Tokyo egress',
      status: 'active'
    }])

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="account-row-1"]').trigger('click')
    await flushPromises()

    const proxyButton = document.body.querySelector<HTMLButtonElement>(
      '[data-test="inspector-view-proxy"]'
    )
    expect(proxyButton).not.toBeNull()
    proxyButton?.click()
    await flushPromises()

    expect(routerPush).toHaveBeenCalledTimes(1)
    expect(routerPush).toHaveBeenCalledWith({
      name: 'AdminProxies',
      query: {
        focus_id: '9',
        open: 'health'
      }
    })
  })

  it('refreshes the tested account immediately after a connection test completes', async () => {
    listAccounts.mockResolvedValue(
      responseFor(makeAccount('before-test', '2026-07-01T00:00:00Z'))
    )
    getAccountById.mockResolvedValue({
      ...makeAccount('after-test', '2026-07-01T00:02:00Z'),
      status: 'error',
      schedulable: false,
      error_message: 'Access forbidden (403)'
    })
    getUsage.mockResolvedValue({
      updated_at: '2026-07-01T00:02:01Z',
      five_hour: null,
      seven_day: null,
      seven_day_sonnet: null
    })
    publishAccountUsage(1, {
      is_forbidden: true,
      forbidden_type: 'validation',
      error_code: 'forbidden'
    } as any, { authoritative: true })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="inspector-test"]').trigger('click')
    await wrapper.get('[data-test="test-modal-completed"]').trigger('click')
    await flushPromises()

    expect(getAccountById).toHaveBeenCalledWith(1)
    expect(getUsage).toHaveBeenCalledTimes(1)
    expect(getUsage).toHaveBeenCalledWith(1, 'active', true)
    expect(wrapper.get('[data-test="account-row-1"]').text()).toContain('after-test')
    expect(wrapper.get('[data-test="inspector-account-name"]').text()).toBe('after-test')
    expect(wrapper.get('[data-test="account-error-1"]').text()).toBe('Access forbidden (403)')
    expect(getAccountUsageHealthSnapshot(1)?.health.isForbidden).toBe(false)
    invalidateAccountUsageHealthSnapshot(1)
  })

  it('uses the tablet drawer, restores body scrolling, and returns focus to the selected row', async () => {
    setViewport(1200)
    listAccounts.mockResolvedValue(responseFor(makeAccount('tablet-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()

    const row = wrapper.get('[data-test="account-row-1"]')
    await row.trigger('click')
    await flushPromises()

    expect(document.body.querySelector('[data-testid="account-inspector-overlay"]')).not.toBeNull()
    expect(document.body.querySelector('[data-test="workbench-inspector"]')?.getAttribute('data-mode')).toBe('drawer')
    expect(document.body.style.overflow).toBe('hidden')

    const closeButton = document.body.querySelector<HTMLButtonElement>('[data-test="inspector-close"]')
    closeButton?.click()
    await flushPromises()

    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(row.element)
  })

  it('uses the bottom sheet below the mobile breakpoint', async () => {
    setViewport(390)
    listAccounts.mockResolvedValue(responseFor(makeAccount('mobile-account', '2026-07-01T00:00:00Z')))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="account-row-1"]').trigger('click')
    await flushPromises()

    expect(
      document.body
        .querySelector('.account-inspector-overlay--sheet [data-test="workbench-inspector"]')
        ?.getAttribute('data-mode')
    ).toBe('sheet')
  })
})
