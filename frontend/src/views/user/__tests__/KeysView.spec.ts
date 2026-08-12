import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

const {
  listKeys,
  toggleStatus,
  updateKey,
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
  updateKey: vi.fn(),
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
  'common.create': 'Create',
  'common.delete': 'Delete',
  'common.edit': 'Edit',
  'common.loading': 'Loading',
  'common.name': 'Name',
  'common.notAvailable': 'Not available',
  'common.refresh': 'Refresh',
  'common.status': 'Status',
  'keys.apiKey': 'API Key',
  'keys.cardDescription': 'API key description',
  'keys.clickToChangeGroup': 'Change group',
  'keys.compactList': 'Compact List',
  'keys.comfortableList': 'Comfortable List',
  'keys.copyToClipboard': 'Copy',
  'keys.copied': 'Copied',
  'keys.createFirstKey': 'Create your first API key.',
  'keys.createKey': 'Create API Key',
  'keys.created': 'Created',
  'keys.currentConcurrency': 'Current Concurrency',
  'keys.detailSettings': 'Detail Fields',
  'keys.detailTitle': 'Key Details',
  'keys.disable': 'Disable',
  'keys.enable': 'Enable',
  'keys.expiresAt': 'Expires',
  'keys.group': 'Group',
  'keys.hideKey': 'Hide key',
  'keys.id': 'ID',
  'keys.importToCcSwitch': 'Import to CCS',
  'keys.ipRestriction': 'IP restriction',
  'keys.keyDisabledSuccess': 'Key disabled',
  'keys.lastUsedAt': 'Last Used',
  'keys.lastUsedIP': 'Last Used IP',
  'keys.noExpiration': 'Never',
  'keys.noGroup': 'No group',
  'keys.noKeysYet': 'No API keys yet',
  'keys.quota': 'Quota',
  'keys.quotaUsage': 'Quota usage',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.revealKey': 'Reveal key',
  'keys.searchPlaceholder': 'Search name or key...',
  'keys.sortConcurrencyAsc': 'Concurrency lowest',
  'keys.sortConcurrencyDesc': 'Concurrency highest',
  'keys.sortCreatedAsc': 'Created oldest',
  'keys.sortCreatedDesc': 'Created newest',
  'keys.sortExpirationAsc': 'Expiration earliest',
  'keys.sortExpirationDesc': 'Expiration latest',
  'keys.sortIdAsc': 'ID ascending',
  'keys.sortIdDesc': 'ID descending',
  'keys.sortLastUsedAsc': 'Last used oldest',
  'keys.sortLastUsedDesc': 'Last used newest',
  'keys.sortNameAsc': 'Name A-Z',
  'keys.sortNameDesc': 'Name Z-A',
  'keys.sortStatusAsc': 'Status ascending',
  'keys.sortStatusDesc': 'Status descending',
  'keys.status.active': 'Enabled',
  'keys.status.expired': 'Expired',
  'keys.status.inactive': 'Inactive',
  'keys.status.quota_exhausted': 'Quota exhausted',
  'keys.title': 'API Keys',
  'keys.today': 'Today',
  'keys.total': 'Last 30d',
  'keys.unlimitedQuota': 'Unlimited',
  'keys.usage': 'Usage',
  'keys.viewDetailsAndActions': 'View Details and Actions',
  'keys.workspaceAssignedGroup': 'Assigned Group',
  'keys.workspaceAvailableQuota': 'Available Quota',
  'keys.workspaceCardThirtyDayUsage': '30-Day Billing Total',
  'keys.workspaceCardTodayUsage': 'Today Billing Total',
  'keys.workspaceKeyInfo': 'Key Information',
  'keys.workspacePageOf': 'Page {page} of {total}',
  'keys.workspaceQuotaProgress': 'Quota Usage Progress',
  'keys.serviceTierEnableTitle': 'Enable Fast mode?',
  'keys.serviceTierEnableConfirmMessage': 'Priority costs more',
  'keys.serviceTierEnableConfirm': 'Enable Fast mode',
  'keys.serviceTierEnabledSuccess': 'Fast mode enabled',
  'keys.serviceTierDisabledSuccess': 'Fast mode disabled',
  'keys.serviceTierUpdateFailed': 'Fast mode update failed',
}

vi.mock('@/api', () => ({
  keysAPI: {
    list: listKeys,
    create: vi.fn(),
    update: updateKey,
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
  key: 'sk-test-key-one-1234',
  name: 'Primary key',
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

const createKeySet = () => [
  createApiKey(),
  createApiKey({
    id: 2,
    key: 'sk-test-key-two-5678',
    name: 'Secondary key',
    status: 'inactive',
    quota: 20,
    quota_used: 5,
  }),
]

const keyResponse = (items: ApiKey[], page = 1, pageSize = 20) => ({
  items,
  total: items.length,
  page,
  page_size: pageSize,
  pages: Math.max(1, Math.ceil(items.length / pageSize)),
})

const SelectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"></select>',
}

const SearchInputStub = {
  name: 'SearchInput',
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue', 'search'],
  template: '<input :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
}

const PaginationStub = {
  name: 'Pagination',
  props: ['page', 'total', 'pageSize'],
  emits: ['update:page', 'update:pageSize'],
  template: `
    <div data-test="pagination-stub">
      <button data-test="page-2" @click="$emit('update:page', 2)">2</button>
      <button data-test="page-size-50" @click="$emit('update:pageSize', 50)">50</button>
    </div>
  `,
}

const IconStub = {
  name: 'Icon',
  props: ['name'],
  template: '<span :data-icon="name">{{ name }}</span>',
}

const EmptyStateStub = {
  name: 'EmptyState',
  props: ['title', 'description'],
  template: '<div data-test="empty-state"><span>{{ title }}</span><span>{{ description }}</span><slot name="action" /></div>',
}

const ApiKeyInspectorStub = {
  name: 'ApiKeyInspector',
  props: [
    'apiKey',
    'usage',
    'userGroupRate',
    'publicSettings',
    'copied',
    'statusUpdating',
    'serviceTierUpdating',
    'now',
    'showCcsImport',
    'mode',
    'visibleColumns',
  ],
  emits: [
    'copy-key',
    'toggle-status',
    'toggle-service-tier',
    'change-group',
    'reset-quota',
    'reset-rate-limit',
    'use-key',
    'import-ccs',
    'edit',
    'delete',
  ],
  template: `
    <div
      data-test="inspector-stub"
      :data-key-id="apiKey.id"
      :data-key-name="apiKey.name"
      :data-mode="mode"
      :data-service-tier="apiKey.service_tier_preference || 'standard'"
    >
      <button
        type="button"
        data-test="inspector-service-tier-toggle"
        @click="$emit('toggle-service-tier', apiKey)"
      />
    </div>
  `,
}

const ApiKeyDetailSheetStub = {
  name: 'ApiKeyDetailSheet',
  props: ['show', 'title', 'subtitle'],
  emits: ['close'],
  template: `
    <div v-if="show" data-test="detail-sheet-stub">
      <button data-test="detail-sheet-close" @click="$emit('close')">close</button>
      <slot />
    </div>
  `,
}

const ConfirmDialogStub = {
  name: 'ConfirmDialog',
  props: ['show', 'title', 'message', 'confirmText', 'cancelText'],
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-test="confirm-dialog-stub">
      <span data-test="confirm-dialog-title">{{ title }}</span>
      <span data-test="confirm-dialog-message">{{ message }}</span>
      <button data-test="confirm-dialog-confirm" @click="$emit('confirm')">{{ confirmText }}</button>
      <button data-test="confirm-dialog-cancel" @click="$emit('cancel')">{{ cancelText }}</button>
    </div>
  `,
}

const mountedWrappers: VueWrapper[] = []

function installMatchMedia(matches: boolean) {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(() => true),
    })),
  })
}

const mountView = async ({ inlineInspector = true } = {}): Promise<VueWrapper> => {
  installMatchMedia(inlineInspector)
  const wrapper = mount(KeysView, {
    global: {
      stubs: {
        AppLayout: { name: 'AppLayout', props: ['variant'], template: '<div><slot /></div>' },
        Pagination: PaginationStub,
        BaseDialog: true,
        ConfirmDialog: ConfirmDialogStub,
        EmptyState: EmptyStateStub,
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: IconStub,
        ApiKeyInspector: ApiKeyInspectorStub,
        ApiKeyDetailSheet: ApiKeyDetailSheetStub,
        UseKeyModal: true,
        EndpointPopover: true,
        GroupBadge: true,
        GroupOptionItem: true,
        Teleport: true,
      },
    },
  })
  mountedWrappers.push(wrapper)
  await flushPromises()
  await nextTick()
  return wrapper
}

const inlineInspector = (wrapper: VueWrapper) =>
  wrapper.get('[data-test="inspector-stub"][data-mode="inline"]')

describe('user KeysView workspace integration', () => {
  beforeEach(() => {
    localStorage.clear()
    for (const mock of [
      listKeys,
      toggleStatus,
      updateKey,
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

    listKeys.mockResolvedValue(keyResponse(createKeySet()))
    toggleStatus.mockResolvedValue(undefined)
    updateKey.mockResolvedValue(undefined)
    getPublicSettings.mockResolvedValue({})
    getDashboardApiKeysUsage.mockImplementation(async (keyIds: number[]) => ({
      stats: Object.fromEntries(keyIds.map((id) => [String(id), {
        api_key_id: id,
        today_actual_cost: id * 0.25,
        total_actual_cost: id * 1.5,
      }])),
    }))
    getAvailableGroups.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    copyToClipboard.mockResolvedValue(true)
    isCurrentStep.mockReturnValue(false)
  })

  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) wrapper.unmount()
    document.body.classList.remove('api-key-detail-sheet-open')
  })

  it('automatically selects the first desktop row and renders both responsive branches', async () => {
    const wrapper = await mountView()
    const detailPane = wrapper.get('[data-test="key-inline-inspector"]')

    expect(window.matchMedia).toHaveBeenCalledWith('(min-width: 768px)')
    expect(detailPane.classes()).toContain('md:flex')
    expect(detailPane.classes()).not.toContain('xl:flex')
    expect(wrapper.get('[data-test="api-key-workspace-row-1"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-test="api-key-workspace-row-2"]').attributes('aria-selected')).toBe('false')
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('1')
    expect(wrapper.find('[data-test="api-key-summary-card-1"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="api-key-summary-card-2"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('$')
  })

  it('switches the selected row and inspector by pointer and keyboard', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="api-key-workspace-row-2"]').trigger('click')
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('2')
    expect(wrapper.get('[data-test="api-key-workspace-row-2"]').attributes('aria-selected')).toBe('true')

    await wrapper.get('[data-test="api-key-workspace-row-1"]').trigger('keydown', { key: 'Enter' })
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('1')
    expect(wrapper.get('[data-test="api-key-workspace-row-1"]').attributes('aria-selected')).toBe('true')
  })

  it('preserves the selected key by ID when refresh returns new objects or order', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-test="api-key-workspace-row-2"]').trigger('click')

    const refreshed = [
      createApiKey({ id: 2, key: 'sk-test-key-two-5678', name: 'Secondary key refreshed' }),
      createApiKey(),
    ]
    listKeys.mockResolvedValueOnce(keyResponse(refreshed))

    await wrapper.get('[data-test="key-refresh"]').trigger('click')
    await flushPromises()

    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('2')
    expect(inlineInspector(wrapper).attributes('data-key-name')).toBe('Secondary key refreshed')
    expect(wrapper.get('[data-test="api-key-workspace-row-2"]').attributes('aria-selected')).toBe('true')
  })

  it('falls back to the first returned row when the selected key disappears', async () => {
    const wrapper = await mountView()
    await wrapper.get('[data-test="api-key-workspace-row-2"]').trigger('click')

    const replacement = createApiKey({ id: 3, key: 'sk-replacement-key-9012', name: 'Replacement key' })
    listKeys.mockResolvedValueOnce(keyResponse([replacement, createApiKey()]))

    await wrapper.get('[data-test="key-refresh"]').trigger('click')
    await flushPromises()

    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('3')
    expect(wrapper.get('[data-test="api-key-workspace-row-3"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.find('[data-test="detail-sheet-stub"]').exists()).toBe(false)
  })

  it('does not select a row when its status, copy, or group controls are used', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="key-workspace-status-switch-2"]').trigger('click')
    await flushPromises()
    expect(toggleStatus).toHaveBeenCalledWith(2, 'active')
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('1')

    await wrapper.get('[data-test="key-workspace-copy-2"]').trigger('click')
    await flushPromises()
    expect(copyToClipboard).toHaveBeenCalledWith('sk-test-key-two-5678', 'Copied')
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('1')

    const row = wrapper.get('[data-test="api-key-workspace-row-2"]')
    const groupButton = row.findAll('button').find((button) => button.attributes('title') === 'Change group')
    expect(groupButton).toBeDefined()
    await groupButton!.trigger('click')
    expect(inlineInspector(wrapper).attributes('data-key-id')).toBe('1')
  })

  it('opens and closes mobile key details without clearing the selected key', async () => {
    const wrapper = await mountView({ inlineInspector: false })

    expect(wrapper.find('[data-test="detail-sheet-stub"]').exists()).toBe(false)
    await wrapper.get('[data-test="key-summary-open-details-2"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-test="detail-sheet-stub"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="inspector-stub"][data-mode="sheet"]').attributes('data-key-id')).toBe('2')

    await wrapper.get('[data-test="detail-sheet-close"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-test="detail-sheet-stub"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="api-key-workspace-row-2"]').attributes('aria-selected')).toBe('true')
  })

  it('confirms enabling Fast mode before updating an OpenAI key preference', async () => {
    const openAiKey = createApiKey({
      group_id: 42,
      group: {
        id: 42,
        name: 'OpenAI',
        platform: 'openai',
        subscription_type: 'standard',
        rate_multiplier: 1,
      } as ApiKey['group'],
    })
    listKeys.mockResolvedValue(keyResponse([openAiKey]))
    const wrapper = await mountView()

    await inlineInspector(wrapper).get('[data-test="inspector-service-tier-toggle"]').trigger('click')
    expect(updateKey).not.toHaveBeenCalled()

    const dialog = wrapper.get('[data-test="confirm-dialog-stub"]')
    expect(dialog.get('[data-test="confirm-dialog-message"]').text()).toContain('Priority costs more')
    await dialog.get('[data-test="confirm-dialog-confirm"]').trigger('click')
    await flushPromises()

    expect(updateKey).toHaveBeenCalledWith(1, {
      service_tier_preference: 'priority',
    })
    expect(showSuccess).toHaveBeenCalledWith('Fast mode enabled')
  })

  it('leaves Fast mode unchanged when the enable confirmation is cancelled', async () => {
    const openAiKey = createApiKey({
      group_id: 42,
      group: {
        id: 42,
        name: 'OpenAI',
        platform: 'openai',
        subscription_type: 'standard',
        rate_multiplier: 1,
      } as ApiKey['group'],
    })
    listKeys.mockResolvedValue(keyResponse([openAiKey]))
    const wrapper = await mountView()

    await inlineInspector(wrapper).get('[data-test="inspector-service-tier-toggle"]').trigger('click')
    const dialog = wrapper.get('[data-test="confirm-dialog-stub"]')
    await dialog.get('[data-test="confirm-dialog-cancel"]').trigger('click')
    await flushPromises()

    expect(updateKey).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="confirm-dialog-stub"]').exists()).toBe(false)
    expect(inlineInspector(wrapper).attributes('data-service-tier')).toBe('standard')
  })

  it('disables Fast mode directly without a confirmation dialog', async () => {
    const openAiKey = createApiKey({
      service_tier_preference: 'priority',
      group_id: 42,
      group: {
        id: 42,
        name: 'OpenAI',
        platform: 'openai',
        subscription_type: 'standard',
        rate_multiplier: 1,
      } as ApiKey['group'],
    })
    listKeys.mockResolvedValue(keyResponse([openAiKey]))
    const wrapper = await mountView()

    await inlineInspector(wrapper).get('[data-test="inspector-service-tier-toggle"]').trigger('click')
    await flushPromises()

    expect(updateKey).toHaveBeenCalledWith(1, {
      service_tier_preference: 'standard',
    })
    expect(wrapper.find('[data-test="confirm-dialog-stub"]').exists()).toBe(false)
  })

  it('keeps the previous Fast mode value and reports an update failure', async () => {
    const openAiKey = createApiKey({
      group_id: 42,
      group: {
        id: 42,
        name: 'OpenAI',
        platform: 'openai',
        subscription_type: 'standard',
        rate_multiplier: 1,
      } as ApiKey['group'],
    })
    listKeys.mockResolvedValue(keyResponse([openAiKey]))
    updateKey.mockRejectedValueOnce(new Error('network unavailable'))
    const wrapper = await mountView()

    await inlineInspector(wrapper).get('[data-test="inspector-service-tier-toggle"]').trigger('click')
    await wrapper.get('[data-test="confirm-dialog-confirm"]').trigger('click')
    await flushPromises()

    expect(updateKey).toHaveBeenCalledWith(1, {
      service_tier_preference: 'priority',
    })
    expect(showError).toHaveBeenCalledWith('Fast mode update failed')
    expect(showSuccess).not.toHaveBeenCalledWith('Fast mode enabled')
    expect(inlineInspector(wrapper).attributes('data-service-tier')).toBe('standard')
  })

  it('forwards page, page size, filters, and explicit sort to list requests', async () => {
    getAvailableGroups.mockResolvedValue([{ id: 42, name: 'OpenAI' }])
    const wrapper = await mountView()
    listKeys.mockClear()

    await wrapper.get('[data-test="page-2"]').trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(
      2,
      20,
      { sort_by: 'created_at', sort_order: 'desc' },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )

    await wrapper.get('[data-test="page-size-50"]').trigger('click')
    await flushPromises()
    expect(listKeys).toHaveBeenLastCalledWith(
      1,
      50,
      { sort_by: 'created_at', sort_order: 'desc' },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )

    const search = wrapper.findComponent({ name: 'SearchInput' })
    await search.vm.$emit('update:modelValue', 'target')
    await search.vm.$emit('search')
    await flushPromises()

    const selects = wrapper.findAllComponents({ name: 'Select' })
    expect(selects).toHaveLength(3)
    await selects[0].vm.$emit('update:modelValue', 42)
    await flushPromises()
    await selects[1].vm.$emit('update:modelValue', 'active')
    await flushPromises()
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
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    )
  })

  it('keeps responsive create actions for an empty result', async () => {
    listKeys.mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0,
    })
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="empty-state"]').text()).toContain('No API keys yet')
    expect(wrapper.get('[data-test="key-create-header"]').classes()).toContain('key-header-create--empty')
    expect(wrapper.get('[data-test="key-create-empty"]').classes()).toContain('key-empty-create')
  })

  it('continues persisting optional detail-field preferences', async () => {
    const wrapper = await mountView()
    const inspector = wrapper.getComponent(ApiKeyInspectorStub)

    expect(inspector.props('visibleColumns')).toContain('id')
    expect(inspector.props('visibleColumns')).toContain('rate_limit')
    expect(wrapper.get('[data-test="api-key-summary-card-1"]').text()).toContain('ID: #1')

    await wrapper.get('[data-test="key-detail-settings"]').trigger('click')
    const menu = wrapper.get('[data-test="key-detail-menu"]')
    const rateLimitButton = menu.findAll('button').find((button) => button.text().includes('Rate Limit'))
    expect(rateLimitButton).toBeDefined()
    await rateLimitButton!.trigger('click')

    expect(localStorage.getItem('api-key-hidden-columns')).toBe(
      JSON.stringify(['rate_limit']),
    )
    expect(inspector.props('visibleColumns')).not.toContain('rate_limit')
  })
})
