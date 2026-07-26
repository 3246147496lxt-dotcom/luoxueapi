import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UsersView from '../UsersView.vue'

const {
  listUsers,
  getAllGroups,
  getAllGroupsIncludingInactive,
  getBatchUsersUsage,
  listEnabledDefinitions,
  getBatchUserAttributes,
  routerPush
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  getAllGroups: vi.fn(),
  getAllGroupsIncludingInactive: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  listEnabledDefinitions: vi.fn(),
  getBatchUserAttributes: vi.fn(),
  routerPush: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      toggleStatus: vi.fn(),
      delete: vi.fn()
    },
    groups: {
      getAll: getAllGroups,
      getAllIncludingInactive: getAllGroupsIncludingInactive
    },
    dashboard: {
      getBatchUsersUsage
    },
    userAttributes: {
      listEnabledDefinitions,
      getBatchUserAttributes
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
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

const createAdminUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  id: 42,
  username: 'scoped-user',
  email: 'scoped@example.com',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  last_active_at: '2026-04-16T02:00:00Z',
  last_used_at: '2026-04-17T02:00:00Z',
  current_concurrency: 0,
  ...overrides
})

const DataTableStub = {
  props: {
    columns: { type: Array, default: () => [] },
    data: { type: Array, default: () => [] },
    mobilePrimaryKey: { type: String, default: '' },
    mobileVisibleKeys: { type: Array, default: () => [] }
  },
  emits: ['sort'],
  template: `
    <div
      data-test="data-table"
      :data-mobile-primary-key="mobilePrimaryKey"
      :data-mobile-visible-keys="mobileVisibleKeys.join(',')"
    >
      <div data-test="columns">{{ columns.map(col => col.key).join(',') }}</div>
      <div data-test="row-order">{{ data.map(row => row.email).join(',') }}</div>
      <button data-test="sort-last-used" @click="$emit('sort', 'last_used_at', 'desc')">sort</button>
      <template v-for="col in columns" :key="col.key">
        <slot :name="'header-' + col.key" :column="col" />
      </template>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-last_used_at" :value="row.last_used_at" :row="row" />
        <slot name="cell-actions" :value="row.actions" :row="row" />
      </div>
    </div>
  `
}

const mountUsersView = () => mount(UsersView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      AdminPageHeader: true,
      TablePageLayout: {
        template: '<div><slot name="header" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
      },
      DataTable: DataTableStub,
      Pagination: true,
      ConfirmDialog: true,
      EmptyState: true,
      GroupBadge: true,
      Select: true,
      UserAttributesConfigModal: true,
      UserConcurrencyCell: true,
      UserCreateModal: true,
      UserEditModal: true,
      UserPlatformQuotaModal: true,
      UserApiKeysModal: true,
      UserAllowedGroupsModal: true,
      UserBalanceModal: true,
      UserBalanceHistoryModal: true,
      GroupReplaceModal: true,
      Icon: true,
      Teleport: true
    }
  }
})

describe('admin UsersView', () => {
  beforeEach(() => {
    vi.useRealTimers()
    localStorage.clear()

    listUsers.mockReset()
    getAllGroups.mockReset()
    getAllGroupsIncludingInactive.mockReset()
    getBatchUsersUsage.mockReset()
    listEnabledDefinitions.mockReset()
    getBatchUserAttributes.mockReset()
    routerPush.mockReset()

    listUsers.mockResolvedValue({
      items: [createAdminUser()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
    getAllGroupsIncludingInactive.mockResolvedValue([])
    getBatchUsersUsage.mockResolvedValue({ stats: {} })
    listEnabledDefinitions.mockResolvedValue([])
    getBatchUserAttributes.mockResolvedValue({ values: {} })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows active, used, and created activity columns in order and requests last_used_at sort', async () => {
    const wrapper = mountUsersView()

    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text()
    const visibleColumns = columns.split(',')
    expect(visibleColumns.slice(-4, -1)).toEqual(['last_active_at', 'last_used_at', 'created_at'])
    expect(visibleColumns).not.toContain('last_login_at')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('clears usage current-page sort when switching to last_used_at server sort', async () => {
    vi.useFakeTimers()
    localStorage.setItem('user-column-settings-version', '3')
    localStorage.setItem(
      'user-hidden-columns',
      JSON.stringify([
        'notes',
        'groups',
        'subscriptions',
        'concurrency',
        'usage_anthropic',
        'usage_openai',
        'usage_gemini',
        'usage_antigravity',
        'balance_platform_quota'
      ])
    )

    listUsers.mockResolvedValue({
      items: [
        createAdminUser({ id: 1, email: 'last-used-first@example.com' }),
        createAdminUser({ id: 2, email: 'usage-first@example.com' })
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getBatchUsersUsage.mockResolvedValue({
      stats: {
        1: { user_id: 1, today_actual_cost: 1, total_actual_cost: 1, by_platform: [] },
        2: { user_id: 2, today_actual_cost: 9, total_actual_cost: 9, by_platform: [] }
      }
    })

    const wrapper = mountUsersView()

    await flushPromises()
    await vi.advanceTimersByTimeAsync(50)
    await flushPromises()

    expect(wrapper.get('[data-test="row-order"]').text()).toBe('last-used-first@example.com,usage-first@example.com')

    await wrapper.get('[data-test="usage-sort-trigger-usage"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="usage-sort-usage-today"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="row-order"]').text()).toBe('usage-first@example.com,last-used-first@example.com')
    expect(localStorage.getItem('admin-users-usage-sort')).toContain('"key":"usage"')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(localStorage.getItem('admin-users-usage-sort')).toBeNull()
    expect(wrapper.get('[data-test="row-order"]').text()).toBe('last-used-first@example.com,usage-first@example.com')
    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('keeps search and the single primary action visible while progressively disclosing secondary controls below desktop', async () => {
    const wrapper = mountUsersView()
    await flushPromises()

    expect(wrapper.get('[data-test="users-search"]').classes()).toContain('min-h-11')
    expect(wrapper.findAll('[data-test="users-create"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="users-create"]').classes()).toContain('min-h-11')

    const toggle = wrapper.get('[data-test="users-mobile-filter-toggle"]')
    const panel = wrapper.get('[data-test="users-mobile-secondary-filters"]')
    expect(toggle.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11', 'lg:hidden']))
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('users-mobile-secondary-filters')
    expect(panel.attributes('id')).toBe('users-mobile-secondary-filters')
    expect(panel.classes()).toEqual(expect.arrayContaining(['hidden', 'lg:contents']))

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(panel.classes()).not.toContain('hidden')
    expect(panel.classes()).toEqual(expect.arrayContaining(['flex', 'lg:contents']))
  })

  it('configures a compact mobile card without changing the desktop column collection', async () => {
    const wrapper = mountUsersView()
    await flushPromises()

    const table = wrapper.get('[data-test="data-table"]')
    expect(table.attributes('data-mobile-primary-key')).toBe('email')
    expect(table.attributes('data-mobile-visible-keys')).toBe(
      'email,role,groups,status,balance'
    )

    const visibleColumns = wrapper.get('[data-test="columns"]').text().split(',')
    expect(visibleColumns).toEqual(
      expect.arrayContaining(['email', 'role', 'balance', 'status', 'actions'])
    )
  })

  it('opens the selected user chat history from the more-actions menu', async () => {
    const wrapper = mountUsersView()
    await flushPromises()

    await wrapper.get('.action-menu-trigger').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="user-chat-history"]').trigger('click')

    expect(routerPush).toHaveBeenCalledWith({
      name: 'AdminUserChatHistory',
      params: { userId: '42' }
    })
  })

  it('retains persisted built-in and dynamic attribute filters inside the responsive panel', async () => {
    localStorage.setItem('user-visible-filters', JSON.stringify(['role', 'attr_7']))
    localStorage.setItem(
      'user-filter-values',
      JSON.stringify({ role: 'admin', attributes: { 7: 'priority' } })
    )
    listEnabledDefinitions.mockResolvedValue([
      {
        id: 7,
        name: 'Customer tier',
        type: 'text',
        enabled: true,
        options: []
      }
    ])

    const wrapper = mountUsersView()
    await flushPromises()

    const panel = wrapper.get('[data-test="users-mobile-secondary-filters"]')
    expect(panel.classes()).toContain('lg:contents')
    expect(wrapper.findAll('select-stub')).toHaveLength(1)
    expect(wrapper.get('input[placeholder="Customer tier"]').element).toHaveProperty(
      'value',
      'priority'
    )
    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        role: 'admin',
        attributes: { 7: 'priority' }
      }),
      expect.any(Object)
    )
  })
})
