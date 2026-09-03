import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { AdminGroup } from '@/types'
import GroupsView from '../GroupsView.vue'

const {
  createGroupRequest,
  getGroupById,
  getAllGroups,
  getModelsListCandidates,
  updateGroupRequest,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  createGroupRequest: vi.fn(),
  getGroupById: vi.fn(),
  getAllGroups: vi.fn(),
  getModelsListCandidates: vi.fn(),
  updateGroupRequest: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      create: createGroupRequest,
      getById: getGroupById,
      getAllIncludingInactive: getAllGroups,
      getModelsListCandidates,
      update: updateGroupRequest,
      list: vi.fn(),
      getUsageSummary: vi.fn(),
      getCapacitySummary: vi.fn(),
      delete: vi.fn(),
      updateSortOrder: vi.fn(),
    },
    accounts: {
      getById: vi.fn(),
      list: vi.fn().mockResolvedValue({ items: [] }),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn().mockReturnValue(false),
    nextStep: vi.fn(),
  }),
}))

const messages: Record<string, string> = {
  'admin.groups.editGroup': 'Edit Group',
  'admin.groups.editor.editDescription': 'Maintain group configuration',
  'admin.groups.editor.backToGroups': 'Back to groups',
  'admin.groups.editor.navigationLabel': 'Group sections',
  'admin.groups.editor.loading': 'Loading group configuration',
  'admin.groups.editor.saved': 'No unsaved changes',
  'admin.groups.editor.dirty': 'Unsaved changes',
  'admin.groups.editor.leaveDirtyConfirm': 'Leave with unsaved changes?',
  'admin.groups.editor.retry': 'Reload',
  'admin.groups.editor.notFound': 'This group does not exist or has been deleted.',
  'admin.groups.editor.forbidden': 'You do not have permission to view or edit this group.',
  'admin.groups.editor.networkError': 'Network unavailable.',
  'admin.groups.editor.serverError': 'Service unavailable.',
  'admin.groups.editor.loadFailed': 'Could not load group.',
  'admin.groups.editor.rateMultiplierInvalid': 'Invalid rate multiplier.',
  'admin.groups.editor.sections.general.label': 'General',
  'admin.groups.editor.sections.general.description': 'Base rules',
  'admin.groups.editor.sections.access.label': 'Access',
  'admin.groups.editor.sections.access.description': 'Quotas',
  'admin.groups.editor.sections.models.label': 'Models',
  'admin.groups.editor.sections.models.description': 'Model list',
  'admin.groups.editor.sections.pricing.label': 'Pricing',
  'admin.groups.editor.sections.pricing.description': 'Rates',
  'admin.groups.editor.sections.routing.label': 'Routing',
  'admin.groups.editor.sections.routing.description': 'Fallbacks',
  'admin.groups.editor.sections.advanced.label': 'Advanced',
  'admin.groups.editor.sections.advanced.description': 'Account routing',
  'admin.groups.platforms.anthropic': 'Anthropic',
  'admin.groups.groupUpdated': 'Group updated',
  'admin.groups.failedToUpdate': 'Update failed',
  'admin.groups.nameRequired': 'Enter a group name',
  'admin.groups.form.name': 'Name',
  'admin.groups.form.description': 'Description',
  'admin.groups.form.platform': 'Platform',
  'admin.groups.form.rateMultiplier': 'Rate multiplier',
  'admin.groups.form.rpmLimit': 'RPM limit',
  'admin.groups.form.status': 'Status',
  'admin.groups.form.exclusive': 'Exclusive',
  'common.cancel': 'Cancel',
  'common.update': 'Update',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => messages[key] ?? key }),
  }
})

const makeGroup = (overrides: Partial<AdminGroup> = {}): AdminGroup => ({
  id: 2,
  name: 'Primary group',
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
  allow_batch_image_generation: false,
  image_rate_independent: false,
  image_rate_multiplier: 1,
  batch_image_discount_multiplier: 1,
  batch_image_hold_multiplier: 1,
  image_price_1k: null,
  image_price_2k: null,
  image_price_4k: null,
  video_rate_independent: false,
  video_rate_multiplier: 1,
  video_price_480p: null,
  video_price_720p: null,
  video_price_1080p: null,
  web_search_price_per_call: null,
  peak_rate_enabled: false,
  peak_start: '00:00',
  peak_end: '00:00',
  peak_rate_multiplier: 1,
  profit_control_enabled: false,
  profit_min_margin: 0,
  profit_safety_buffer: 0,
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
  account_count: 1,
  active_account_count: 1,
  rate_limited_account_count: 0,
  models_list_config: undefined,
  sort_order: 10,
  ...overrides,
})

const SelectStub = {
  props: ['modelValue', 'options', 'disabled'],
  emits: ['update:modelValue', 'change'],
  template: '<select :value="modelValue" :disabled="disabled"><option v-for="option in options" :key="String(option.value)" :value="option.value">{{ option.label }}</option></select>',
}

async function mountEditor(path = '/admin/groups/2/edit?section=general') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin/groups', name: 'AdminGroups', component: GroupsView },
      { path: '/admin/groups/new', name: 'AdminGroupCreate', component: GroupsView },
      { path: '/admin/groups/:id/edit', name: 'AdminGroupEdit', component: GroupsView },
    ],
  })
  await router.push(path)

  const wrapper = mount({ template: '<RouterView />' }, {
    global: {
      plugins: [router],
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        AdminPageHeader: true,
        TablePageLayout: { template: '<div><slot name="header" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: true,
        Pagination: true,
        BaseDialog: true,
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        PlatformIcon: true,
        Icon: { props: ['name'], template: '<span aria-hidden="true">{{ name }}</span>' },
        GroupCapacityBadge: true,
        GroupRateMultipliersModal: true,
        GroupRPMOverridesModal: true,
        VueDraggable: { template: '<div><slot /></div>' },
      },
    },
  })
  await flushPromises()
  await flushPromises()
  return { router, wrapper }
}

describe('admin group editor workspace', () => {
  beforeEach(() => {
    getGroupById.mockReset().mockResolvedValue(makeGroup())
    getAllGroups.mockReset().mockResolvedValue([makeGroup()])
    getModelsListCandidates.mockReset().mockResolvedValue([])
    updateGroupRequest.mockReset().mockImplementation(async (_id, payload) => makeGroup(payload))
    createGroupRequest.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    vi.spyOn(window, 'scrollTo').mockImplementation(() => undefined)
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation(callback => {
      callback(0)
      return 1
    })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders only the active section while preserving state and avoiding reloads', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true)
    const { router, wrapper } = await mountEditor()

    expect(wrapper.findAll('[data-test="group-editor-active-section"]')).toHaveLength(1)
    expect(wrapper.get('[data-test="group-editor-active-section"]').attributes('id')).toBe('group-editor-section-general')
    await wrapper.get('#edit-group-name').setValue('Renamed group')

    await router.push('/admin/groups/2/edit?section=pricing')
    await flushPromises()
    expect(confirm).not.toHaveBeenCalled()
    expect(wrapper.find('#edit-group-name').exists()).toBe(false)
    expect(wrapper.get('[data-test="group-editor-active-section"]').attributes('id')).toBe('group-editor-section-pricing')

    await router.back()
    await flushPromises()
    expect(wrapper.get<HTMLInputElement>('#edit-group-name').element.value).toBe('Renamed group')
    expect(getGroupById).toHaveBeenCalledTimes(1)
    expect(getModelsListCandidates).toHaveBeenCalledTimes(1)
  })

  it('protects dirty navigation and keeps updates in the current section', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const { router, wrapper } = await mountEditor()
    await wrapper.get('#edit-group-name').setValue('Changed group')

    const unload = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(unload)
    expect(unload.defaultPrevented).toBe(true)

    await router.push('/admin/groups')
    expect(router.currentRoute.value.name).toBe('AdminGroupEdit')
    expect(confirm).toHaveBeenCalledTimes(1)

    await router.push('/admin/groups/2/edit?section=pricing')
    await flushPromises()
    await wrapper.get('#edit-group-form').trigger('submit')
    await flushPromises()

    expect(updateGroupRequest).toHaveBeenCalledTimes(1)
    expect(router.currentRoute.value.name).toBe('AdminGroupEdit')
    expect(router.currentRoute.value.query.section).toBe('pricing')
    expect(wrapper.text()).toContain('No unsaved changes')
  })

  it('replaces a completed create route with the new edit workspace', async () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    createGroupRequest.mockResolvedValueOnce(makeGroup({ id: 7, name: 'New group' }))
    getGroupById.mockImplementationOnce(async id => makeGroup({ id, name: 'New group' }))
    const { router, wrapper } = await mountEditor('/admin/groups/new?section=general')

    await wrapper.get('#create-group-name').setValue('New group')
    await wrapper.get('#create-group-form').trigger('submit')
    await flushPromises()
    await flushPromises()

    expect(createGroupRequest).toHaveBeenCalledTimes(1)
    expect(router.currentRoute.value.name).toBe('AdminGroupEdit')
    expect(router.currentRoute.value.params.id).toBe('7')
    expect(router.currentRoute.value.query.section).toBe('general')
    expect(confirm).not.toHaveBeenCalled()
  })

  it('hydrates profit percentages and submits backend decimal values', async () => {
    getGroupById.mockResolvedValueOnce(makeGroup({
      profit_control_enabled: true,
      profit_min_margin: 0.3,
      profit_safety_buffer: 0.025,
    }))
    const { wrapper } = await mountEditor('/admin/groups/2/edit?section=pricing')

    expect(wrapper.get<HTMLInputElement>('[data-test="profit-control-enabled"]').element.checked).toBe(true)
    expect(wrapper.get<HTMLInputElement>('[data-test="profit-min-margin"]').element.value).toBe('30')
    expect(wrapper.get<HTMLInputElement>('[data-test="profit-safety-buffer"]').element.value).toBe('2.5')

    await wrapper.get('[data-test="profit-min-margin"]').setValue('35.5')
    await wrapper.get('[data-test="profit-safety-buffer"]').setValue('4.5')
    await wrapper.get('#edit-group-form').trigger('submit')
    await flushPromises()

    expect(updateGroupRequest).toHaveBeenCalledWith(
      2,
      expect.objectContaining({
        profit_control_enabled: true,
        profit_min_margin: 0.355,
        profit_safety_buffer: 0.045,
      }),
    )
  })

  it('blocks an invalid profit threshold before the admin API call', async () => {
    getGroupById.mockResolvedValueOnce(makeGroup({
      profit_control_enabled: true,
      profit_min_margin: 0.6,
      profit_safety_buffer: 0.4,
    }))
    const { wrapper } = await mountEditor('/admin/groups/2/edit?section=pricing')

    await wrapper.get('#edit-group-form').trigger('submit')
    await flushPromises()

    expect(updateGroupRequest).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.groups.profitControl.sumTooHigh')
  })

  it.each([
    [403, 'You do not have permission to view or edit this group.', false],
    [404, 'This group does not exist or has been deleted.', false],
    [0, 'Network unavailable.', true],
    [503, 'Service unavailable.', true],
  ])('distinguishes load status %s', async (status, message, retryable) => {
    getGroupById.mockRejectedValueOnce({ status })
    const { wrapper } = await mountEditor()

    const alert = wrapper.get('[data-test="group-editor-load-error"]')
    expect(alert.text()).toContain(message)
    expect(alert.find('button.btn-primary').exists()).toBe(retryable)
  })
})
