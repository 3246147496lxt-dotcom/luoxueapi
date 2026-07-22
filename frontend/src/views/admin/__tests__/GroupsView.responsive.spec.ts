import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminGroup } from '@/types'
import GroupsView from '../GroupsView.vue'

vi.mock('vue-router', () => ({
  onBeforeRouteLeave: vi.fn(),
  onBeforeRouteUpdate: vi.fn(),
  useRoute: () => ({ name: 'AdminGroups', params: {}, query: {} }),
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

const {
  listGroups,
  getAllGroups,
  getModelsListCandidates,
  getUsageSummary,
  getCapacitySummary,
  listAccounts,
  showError,
  showSuccess,
  isCurrentStep,
  nextStep,
} = vi.hoisted(() => ({
  listGroups: vi.fn(),
  getAllGroups: vi.fn(),
  getModelsListCandidates: vi.fn(),
  getUsageSummary: vi.fn(),
  getCapacitySummary: vi.fn(),
  listAccounts: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  isCurrentStep: vi.fn(),
  nextStep: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.filter': 'Filter',
  'common.more': 'More',
  'common.refresh': 'Refresh',
  'common.edit': 'Edit',
  'common.delete': 'Delete',
  'admin.groups.columnSettings': 'Column Settings',
  'admin.groups.sortOrder': 'Sort Order',
  'admin.groups.createGroup': 'Create Group',
  'admin.groups.rateMultipliers': 'Rate Multipliers',
  'admin.groups.rpmOverrides': 'RPM Overrides',
  'admin.groups.columns.name': 'Name',
  'admin.groups.columns.id': 'ID',
  'admin.groups.columns.platform': 'Platform',
  'admin.groups.columns.billingType': 'Billing Type',
  'admin.groups.columns.rateMultiplier': 'Rate Multiplier',
  'admin.groups.columns.type': 'Type',
  'admin.groups.columns.accounts': 'Accounts',
  'admin.groups.columns.capacity': 'Capacity',
  'admin.groups.columns.usage': 'Usage',
  'admin.groups.columns.status': 'Status',
  'admin.groups.columns.actions': 'Actions',
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: listGroups,
      getAll: getAllGroups,
      getModelsListCandidates,
      getUsageSummary,
      getCapacitySummary,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      updateSortOrder: vi.fn(),
    },
    accounts: {
      list: listAccounts,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({ isCurrentStep, nextStep }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => messages[key] ?? key }),
  }
})

const createGroup = (overrides: Partial<AdminGroup> = {}): AdminGroup => ({
  id: 1,
  name: 'Core Anthropic',
  description: null,
  platform: 'anthropic',
  rate_multiplier: 1,
  rpm_limit: 0,
  is_exclusive: false,
  status: 'active',
  subscription_type: 'standard',
  daily_limit_usd: null,
  weekly_limit_usd: null,
  monthly_limit_usd: null,
  allow_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  claude_code_only: false,
  fallback_group_id: null,
  fallback_group_id_on_invalid_request: null,
  allow_messages_dispatch: false,
  default_mapped_model: '',
  messages_dispatch_model_config: undefined,
  require_oauth_only: false,
  require_privacy_set: false,
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
  model_routing: null,
  model_routing_enabled: false,
  mcp_xml_inject: true,
  supported_model_scopes: [],
  account_count: 3,
  active_account_count: 2,
  rate_limited_account_count: 1,
  models_list_config: undefined,
  sort_order: 10,
  ...overrides,
})

const TablePageLayoutStub = {
  template: `
    <div>
      <slot name="header" />
      <slot name="filters" />
      <slot name="table" />
      <slot name="pagination" />
    </div>
  `,
}

const DataTableStub = {
  props: {
    columns: { type: Array, default: () => [] },
    data: { type: Array, default: () => [] },
    mobilePrimaryKey: { type: String, default: '' },
    mobileVisibleKeys: { type: Array, default: () => [] },
  },
  emits: ['sort'],
  template: `
    <div
      data-test="groups-data-table"
      :data-mobile-primary-key="mobilePrimaryKey"
      :data-mobile-visible-keys="mobileVisibleKeys.join(',')"
    >
      <div data-test="columns">{{ columns.map((column) => column.key).join(',') }}</div>
      <div v-for="row in data" :key="row.id" data-test="group-row-actions">
        <slot name="cell-actions" :row="row" />
      </div>
      <slot v-if="data.length === 0" name="empty" />
    </div>
  `,
}

const EmptyStateStub = {
  props: ['title', 'description', 'actionText'],
  emits: ['action'],
  template: `
    <div
      data-test="groups-empty-state"
      :data-has-action="Boolean(actionText)"
    >
      {{ title }}
    </div>
  `,
}

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder'],
  emits: ['update:modelValue', 'change'],
  template: '<select :value="modelValue"><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>',
}

const mountView = async () => {
  const wrapper = mount(GroupsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        AdminPageHeader: true,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
        ConfirmDialog: true,
        EmptyState: EmptyStateStub,
        Select: SelectStub,
        PlatformIcon: true,
        Icon: { props: ['name'], template: '<span data-test="icon">{{ name }}</span>' },
        GroupCapacityBadge: true,
        GroupRateMultipliersModal: true,
        GroupRPMOverridesModal: true,
        VueDraggable: { template: '<div><slot /></div>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('admin GroupsView responsive list contract', () => {
  beforeEach(() => {
    localStorage.clear()

    listGroups.mockReset()
    getAllGroups.mockReset()
    getModelsListCandidates.mockReset()
    getUsageSummary.mockReset()
    getCapacitySummary.mockReset()
    listAccounts.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    isCurrentStep.mockReset()
    nextStep.mockReset()

    listGroups.mockResolvedValue({
      items: [createGroup()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    getAllGroups.mockResolvedValue([])
    getModelsListCandidates.mockResolvedValue([])
    getUsageSummary.mockResolvedValue([])
    getCapacitySummary.mockResolvedValue([])
    listAccounts.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    isCurrentStep.mockReturnValue(false)
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('keeps search, refresh, and the single primary action visible while filters disclose below desktop', async () => {
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="groups-search"]').classes()).toContain('min-h-11')
    expect(wrapper.findAll('[data-test="groups-create"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="groups-create"]').classes()).toContain('min-h-11')
    expect(wrapper.get('[data-test="groups-refresh"]').classes()).toEqual(
      expect.arrayContaining(['min-h-11', 'min-w-11']),
    )

    const toggle = wrapper.get('[data-test="groups-mobile-filter-toggle"]')
    const panel = wrapper.get('[data-test="groups-mobile-secondary-filters"]')
    expect(toggle.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11', 'lg:hidden']))
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('groups-mobile-secondary-filters')
    expect(panel.classes()).toEqual(expect.arrayContaining(['hidden', 'lg:flex']))

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(panel.classes()).not.toContain('hidden')
    expect(panel.classes()).toEqual(expect.arrayContaining(['grid', 'sm:grid-cols-3', 'lg:flex']))
    expect(panel.findAll('select')).toHaveLength(3)
  })

  it('moves column settings and sorting into the mobile more menu while retaining desktop controls', async () => {
    const wrapper = await mountView()

    const moreToggle = wrapper.get('[data-test="groups-mobile-more-toggle"]')
    expect(moreToggle.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11']))
    expect(moreToggle.element.parentElement?.classList.contains('lg:hidden')).toBe(true)
    expect(wrapper.get('button[title="Column Settings"]').element.parentElement?.classList.contains('lg:block')).toBe(true)
    expect(wrapper.get('button[title="Sort Order"]').classes()).toEqual(
      expect.arrayContaining(['hidden', 'lg:inline-flex']),
    )

    await moreToggle.trigger('click')

    const menu = wrapper.get('[data-test="groups-mobile-tools-menu"]')
    expect(menu.text()).toContain('Column Settings')
    expect(menu.text()).toContain('Sort Order')
  })

  it('configures compact mobile group cards without changing the desktop column collection', async () => {
    const wrapper = await mountView()
    const table = wrapper.get('[data-test="groups-data-table"]')

    expect(table.attributes('data-mobile-primary-key')).toBe('name')
    expect(table.attributes('data-mobile-visible-keys')).toBe(
      'platform,billing_type,account_count,capacity,status',
    )
    expect(wrapper.get('[data-test="columns"]').text().split(',')).toEqual([
      'name',
      'platform',
      'billing_type',
      'rate_multiplier',
      'is_exclusive',
      'account_count',
      'capacity',
      'usage',
      'status',
      'actions',
    ])
  })

  it('keeps only edit and more upfront on mobile and progressively reveals secondary row actions', async () => {
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="groups-row-edit"]').classes()).toEqual(
      expect.arrayContaining(['min-h-11', 'min-w-11']),
    )
    expect(wrapper.get('[data-test="groups-row-more"]').classes()).toEqual(
      expect.arrayContaining(['min-h-11', 'min-w-11', 'lg:hidden']),
    )
    expect(wrapper.get('[data-test="groups-row-rate-desktop"]').classes()).toEqual(
      expect.arrayContaining(['hidden', 'lg:flex']),
    )
    expect(wrapper.get('[data-test="groups-row-rpm-desktop"]').classes()).toEqual(
      expect.arrayContaining(['hidden', 'lg:flex']),
    )
    expect(wrapper.get('[data-test="groups-row-delete-desktop"]').classes()).toEqual(
      expect.arrayContaining(['hidden', 'lg:flex']),
    )
    expect(wrapper.find('[data-test="groups-row-secondary-actions"]').exists()).toBe(false)

    await wrapper.get('[data-test="groups-row-more"]').trigger('click')

    const secondaryActions = wrapper.get('[data-test="groups-row-secondary-actions"]')
    expect(secondaryActions.classes()).toContain('lg:hidden')
    expect(secondaryActions.findAll('button')).toHaveLength(3)
    secondaryActions.findAll('button').forEach((button) => {
      expect(button.classes()).toContain('min-h-11')
    })
  })

  it('keeps the empty state explanatory without duplicating the create action', async () => {
    listGroups.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })

    const wrapper = await mountView()

    expect(wrapper.findAll('[data-test="groups-create"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="groups-empty-state"]').attributes('data-has-action')).toBe('false')
  })
})
