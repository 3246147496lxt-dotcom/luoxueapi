import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

const {
  listKeys,
  toggleStatus,
  getPublicSettings,
  getDashboardApiKeysUsage,
  getAvailableGroups,
  getUserGroupRates,
  showError,
  showSuccess,
  copyToClipboard,
  isCurrentStep,
  nextStep,
} = vi.hoisted(() => ({
  listKeys: vi.fn(),
  toggleStatus: vi.fn(),
  getPublicSettings: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailableGroups: vi.fn(),
  getUserGroupRates: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  copyToClipboard: vi.fn(),
  isCurrentStep: vi.fn(),
  nextStep: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.actions': 'Actions',
  'common.delete': 'Delete',
  'common.edit': 'Edit',
  'common.filter': 'Filter',
  'common.loading': 'Loading',
  'common.name': 'Name',
  'common.refresh': 'Refresh',
  'common.status': 'Status',
  'keys.apiKey': 'API Key',
  'keys.cardDescription': 'API key description',
  'keys.columnAlwaysVisible': 'Always visible',
  'keys.columnSettings': 'Column Settings',
  'keys.copyToClipboard': 'Copy',
  'keys.copied': 'Copied',
  'keys.createKey': 'Create API Key',
  'keys.created': 'Created',
  'keys.currentConcurrency': 'Current Concurrency',
  'keys.detailSettings': 'Detail Fields',
  'keys.disable': 'Disable',
  'keys.enable': 'Enable',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.hideActions': 'Hide Actions',
  'keys.hideKey': 'Hide key',
  'keys.id': 'ID',
  'keys.importToCcSwitch': 'Import to CCS',
  'keys.ipRestriction': 'IP restriction',
  'keys.lastUsedAt': 'Last Used',
  'keys.lastUsedIP': 'Last Used IP',
  'keys.noExpiration': 'Never',
  'keys.noGroup': 'No group',
  'keys.noIpRestriction': 'No restrictions',
  'keys.noKeysYet': 'No API keys yet',
  'keys.noRateLimit': 'Not set',
  'keys.quota': 'Quota',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.rateLimitUsage': 'Rate limit usage',
  'keys.revealKey': 'Reveal key',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.showActions': 'Show Actions',
  'keys.sortConcurrency': 'Concurrency',
  'keys.sortCreatedAsc': 'Created oldest',
  'keys.sortCreatedDesc': 'Created newest',
  'keys.sortExpiration': 'Expiration',
  'keys.sortLastUsed': 'Last used',
  'keys.sortNameAsc': 'Name A-Z',
  'keys.sortNameDesc': 'Name Z-A',
  'keys.sortStatus': 'Status',
  'keys.status.active': 'Enabled',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.title': 'API Keys',
  'keys.today': 'Today',
  'keys.total': 'Last 30d',
  'keys.unlimitedQuota': 'Unlimited',
  'keys.usage': 'Usage',
  'keys.useKey': 'Use key',
}

vi.mock('@/api', () => ({
  keysAPI: {
    list: listKeys,
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    toggleStatus,
  },
  authAPI: { getPublicSettings },
  usageAPI: { getDashboardApiKeysUsage },
  userGroupsAPI: {
    getAvailable: getAvailableGroups,
    getUserGroupRates,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({ isCurrentStep, nextStep }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const createApiKey = (overrides: Partial<ApiKey> = {}): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test-key',
  name: 'test-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 10,
  quota_used: 2,
  expires_at: null,
  created_at: '2026-06-27T00:00:00Z',
  updated_at: '2026-06-27T00:00:00Z',
  current_concurrency: 3,
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

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"></select>',
}

const SearchInputStub = {
  name: 'SearchInput',
  props: ['modelValue'],
  emits: ['update:modelValue', 'search'],
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['page', 'total', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: '<button data-test="page-size-50" @click="$emit(\'update:pageSize\', 50)">50</button>',
}

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name">{{ name }}</span>',
}

const EmptyStateStub = {
  name: 'EmptyState',
  props: ['title', 'description'],
  template: '<div data-test="empty-state"><span>{{ title }}</span><span>{{ description }}</span><slot name="action" /></div>',
}

const mountView = async (): Promise<VueWrapper> => {
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Pagination: PaginationStub,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: EmptyStateStub,
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

const getButtonByText = (wrapper: VueWrapper, text: string) => {
  const button = wrapper.findAll('button').find((item) => item.text().includes(text))
  if (!button) throw new Error(`Button not found: ${text}`)
  return button
}

describe('user KeysView responsive key layout', () => {
  beforeEach(() => {
    localStorage.clear()
    for (const mock of [
      listKeys,
      toggleStatus,
      getPublicSettings,
      getDashboardApiKeysUsage,
      getAvailableGroups,
      getUserGroupRates,
      showError,
      showSuccess,
      copyToClipboard,
      isCurrentStep,
      nextStep,
    ]) mock.mockReset()

    listKeys.mockResolvedValue({
      items: [createApiKey()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1,
    })
    toggleStatus.mockResolvedValue(undefined)
    getPublicSettings.mockResolvedValue({})
    getDashboardApiKeysUsage.mockResolvedValue({
      stats: { 1: { api_key_id: 1, today_actual_cost: 0.25, total_actual_cost: 1.5 } },
    })
    getAvailableGroups.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    copyToClipboard.mockResolvedValue(true)
    isCurrentStep.mockReturnValue(false)
  })

  it('renders the desktop table and mobile card branches from the same key data', async () => {
    const wrapper = await mountView()

    expect(wrapper.find('[data-test="api-key-table"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-key-table-row-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-key-card-1"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="key-table-status-switch-1"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[data-test="key-status-switch-1"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.text()).toContain('0.2500')
    expect(wrapper.text()).toContain('1.5000')
    expect(wrapper.text()).toContain('2.00/10.00')
    expect(wrapper.findAll('[data-testid="credit-amount"]').length).toBeGreaterThanOrEqual(8)
    expect(wrapper.text()).not.toContain('$')
  })

  it('keeps mobile secondary filters collapsed behind an accessible 44px control', async () => {
    const wrapper = await mountView()
    const toggle = wrapper.get('[data-test="key-mobile-filter-toggle"]')
    const secondaryFilters = wrapper.get('[data-test="key-mobile-secondary-filters"]')

    expect(toggle.text()).toContain('Filter')
    expect(toggle.classes()).toContain('min-h-11')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('key-mobile-secondary-filters')
    expect(secondaryFilters.attributes('id')).toBe('key-mobile-secondary-filters')
    expect(secondaryFilters.classes()).toContain('hidden')
    expect(secondaryFilters.classes()).toContain('md:contents')
    expect(wrapper.findAllComponents({ name: 'Select' })).toHaveLength(3)

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(secondaryFilters.classes()).not.toContain('hidden')
    expect(secondaryFilters.classes()).toContain('grid')
    expect(secondaryFilters.classes()).toContain('md:contents')
  })

  it('keeps exactly one responsive create-key CTA for the empty state', async () => {
    listKeys.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0,
    })
    const wrapper = await mountView()

    const headerCta = wrapper.get('[data-test="key-create-header"]')
    const emptyCta = wrapper.get('[data-test="key-create-empty"]')

    expect(wrapper.get('[data-test="empty-state"]').text()).toContain('No API keys yet')
    expect(headerCta.classes()).toContain('key-header-create--empty')
    expect(emptyCta.classes()).toContain('key-empty-create')
    expect(emptyCta.classes()).toContain('min-h-11')
  })

  it('keeps low-frequency fields hidden and can reveal them from detail settings', async () => {
    const wrapper = await mountView()

    expect(wrapper.text()).not.toContain('Rate limit usage')
    await wrapper.get('[data-test="key-detail-settings"]').trigger('click')
    await getButtonByText(wrapper, 'Rate Limit').trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('Rate limit usage')
    expect(localStorage.getItem('api-key-hidden-columns')).toBe(
      JSON.stringify(['id', 'last_used_at', 'last_used_ip'])
    )
  })

  it('keeps status, usage, key, and group always visible in detail settings', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="key-detail-settings"]').trigger('click')
    const menuText = wrapper.get('[data-test="key-detail-menu"]').text()
    expect(menuText).toContain('ID')
    expect(menuText).toContain('Rate Limit')
    expect(menuText).not.toContain('Actions')
    expect(menuText).not.toContain('API Key')
    expect(menuText).not.toContain('Usage')
  })

  it('collapses and restores the complete action area', async () => {
    const wrapper = await mountView()

    expect(wrapper.find('[data-test="key-actions-1"]').exists()).toBe(true)
    await wrapper.get('[data-test="key-actions-toggle"]').trigger('click')
    expect(wrapper.find('[data-test="key-actions-1"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="key-actions-toggle"]').attributes('aria-expanded')).toBe('false')
  })

  it('toggles an active key once and reloads the card data', async () => {
    const wrapper = await mountView()
    listKeys.mockClear()

    await wrapper.get('[data-test="key-status-switch-1"]').trigger('click')
    await flushPromises()

    expect(toggleStatus).toHaveBeenCalledTimes(1)
    expect(toggleStatus).toHaveBeenCalledWith(1, 'inactive')
    expect(showSuccess).toHaveBeenCalledWith('keys.keyDisabledSuccess')
    expect(listKeys).toHaveBeenCalledTimes(1)
  })

  it('copies the selected key through the existing clipboard flow', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="key-copy-1"]').trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenCalledWith('sk-test-key', 'Copied')
  })

  it('keeps filters, page size, and explicit sort selection in server requests', async () => {
    getAvailableGroups.mockResolvedValue([{ id: 42, name: 'OpenAI' }])
    const wrapper = await mountView()

    await wrapper.get('[data-test="page-size-50"]').trigger('click')
    await flushPromises()

    const search = wrapper.findComponent({ name: 'SearchInput' })
    await search.vm.$emit('update:modelValue', 'target')
    await search.vm.$emit('search')
    await flushPromises()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    await selects[0].vm.$emit('update:modelValue', 42)
    await flushPromises()
    await selects[1].vm.$emit('update:modelValue', 'active')
    await flushPromises()

    listKeys.mockClear()
    await selects[2].vm.$emit('update:modelValue', 'current_concurrency:desc')
    await flushPromises()

    expect(listKeys).toHaveBeenLastCalledWith(
      1,
      50,
      {
        search: 'target',
        status: 'active',
        group_id: 42,
        sort_by: 'current_concurrency',
        sort_order: 'desc',
      },
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
  })
})
