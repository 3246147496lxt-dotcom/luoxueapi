import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import SkillsView from '../SkillsView.vue'

const {
  list,
  getConfig,
  updateConfig,
  publish,
  archive,
  push,
  showSuccess,
  showError,
  refreshPublicSettingsAfterMutation,
  useStepUp,
} = vi.hoisted(() => ({
  list: vi.fn(),
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  publish: vi.fn(),
  archive: vi.fn(),
  push: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  refreshPublicSettingsAfterMutation: vi.fn(),
  useStepUp: vi.fn(),
}))

vi.mock('@/api/admin/skills', () => ({
  default: { list, getConfig, updateConfig, publish, archive },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
    refreshPublicSettingsAfterMutation,
    backendModeEnabled: false,
  }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp,
  isStepUpCancelled: () => false,
  isStepUpBlocked: () => false,
  stepUpBlockReason: () => '',
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
    }),
  }
})

const version = {
  id: 11,
  skill_id: 1,
  version: '1.0.0',
  changelog: 'Initial release',
  manifest_name: 'api-doc-writer',
  manifest_description: 'Writes API docs',
  sha256: 'a'.repeat(64),
  byte_size: 100,
  unpacked_size: 200,
  file_count: 1,
  file_manifest: [],
  validation_report: { valid: true, errors: [], warnings: [] },
  status: 'available' as const,
  yanked_at: null,
  created_at: '2026-08-01T00:00:00Z',
  download_count: 0,
}

const draft = {
  id: 1,
  slug: 'api-doc-writer',
  display_name: 'API 文档生成器',
  summary: '生成接入文档',
  description: '读取接口定义并生成文档。',
  category: '文档与数据',
  tags: ['Codex'],
  icon: '',
  example_prompts: [],
  risk_notes: '',
  status: 'draft' as const,
  featured: true,
  sort_order: 1,
  current_version_id: 11,
  current_version: version,
  download_count: 0,
  published_at: null,
  archived_at: null,
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-01T00:00:00Z',
  versions: [version],
}

const SlotLayout = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', [
      slots.header?.(),
      slots.actions?.(),
      slots.filters?.(),
      slots.table?.(),
      slots.pagination?.(),
      slots.default?.(),
    ])
  },
})

const HeaderStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('header', [slots.meta?.(), slots['secondary-actions']?.(), slots['primary-actions']?.()])
  },
})

const DataTableStub = defineComponent({
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () => h('div', { 'data-testid': 'skill-table' },
      (props.data as Array<Record<string, unknown>>).map((row) => h('div', { class: 'skill-row' }, [
        slots['cell-skill']?.({ row }),
        slots['cell-version']?.({ row }),
        slots['cell-actions']?.({ row }),
      ])))
  },
})

const ConfirmDialogStub = defineComponent({
  props: { show: Boolean, title: String, message: String },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('button', { class: 'confirm-action', onClick: () => emit('confirm') }, props.title)
      : null
  },
})

const SelectStub = defineComponent({
  props: { modelValue: [String, Boolean], options: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup() { return () => h('div') },
})

const ToggleStub = defineComponent({
  props: { modelValue: Boolean, disabled: Boolean },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('button', {
      'data-testid': 'market-toggle',
      'data-value': String(props.modelValue),
      disabled: props.disabled,
      onClick: () => emit('update:modelValue', !props.modelValue),
    })
  },
})

function mountView() {
  return mount(SkillsView, {
    global: {
      stubs: {
        AppLayout: SlotLayout,
        TablePageLayout: SlotLayout,
        AdminPageHeader: HeaderStub,
        DataTable: DataTableStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: SelectStub,
        Toggle: ToggleStub,
        EmptyState: true,
        Pagination: true,
        SkillStatusBadge: true,
        Icon: true,
      },
    },
  })
}

describe('admin SkillsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue({ items: [draft], total: 1, page: 1, page_size: 20 })
    getConfig.mockResolvedValue({ enabled: false })
    updateConfig.mockResolvedValue({ enabled: true })
    refreshPublicSettingsAfterMutation.mockResolvedValue(null)
    publish.mockResolvedValue({ ...draft, status: 'published' })
    archive.mockResolvedValue({ ...draft, status: 'archived' })
  })

  it('loads Skills and the public market gate together', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(list).toHaveBeenCalledWith(expect.objectContaining({
      status: 'all',
      featured: 'all',
      page: 1,
      page_size: 20,
    }))
    expect(getConfig).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('API 文档生成器')
  })

  it('publishes only after an explicit confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[title="admin.skills.publish"]').trigger('click')
    expect(publish).not.toHaveBeenCalled()
    await wrapper.get('.confirm-action').trigger('click')
    await flushPromises()

    expect(publish).toHaveBeenCalledWith(1, 11)
    expect(useStepUp).not.toHaveBeenCalled()
    expect(showSuccess).toHaveBeenCalledWith('admin.skills.publishSuccess')
  })

  it('updates the public marketplace directly without a confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="market-toggle"]').trigger('click')
    expect(updateConfig).toHaveBeenCalledWith(true)
    expect(wrapper.find('.confirm-action').exists()).toBe(false)
    await flushPromises()

    expect(wrapper.get('[data-testid="market-toggle"]').attributes('data-value')).toBe('true')
    expect(useStepUp).not.toHaveBeenCalled()
    expect(refreshPublicSettingsAfterMutation).toHaveBeenCalledOnce()
    expect(showSuccess).toHaveBeenCalledWith('admin.skills.marketplace.enabledSuccess')
  })

  it('restores the marketplace toggle when the direct update fails', async () => {
    updateConfig.mockRejectedValueOnce(new Error('network error'))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="market-toggle"]').trigger('click')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith(true)
    expect(wrapper.get('[data-testid="market-toggle"]').attributes('data-value')).toBe('false')
    expect(showError).toHaveBeenCalledWith('admin.skills.marketplace.updateFailed')
  })

  it('archives only after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button[title="admin.skills.archive"]').trigger('click')
    await wrapper.get('.confirm-action').trigger('click')
    await flushPromises()

    expect(archive).toHaveBeenCalledWith(1)
    expect(useStepUp).not.toHaveBeenCalled()
  })
})
