import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import ModelCatalogView from '../ModelCatalogView.vue'

const {
  list,
  candidates,
  create,
  getById,
  update,
  publish,
  unpublish,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  list: vi.fn(),
  candidates: vi.fn(),
  create: vi.fn(),
  getById: vi.fn(),
  update: vi.fn(),
  publish: vi.fn(),
  unpublish: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelCatalog: { list, candidates, create, getById, update, publish, unpublish },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
      locale: { value: 'zh-CN' },
    }),
  }
})

const pricing = {
  label: '公开标准价',
  billing_mode: 'token',
  currency: 'USD',
  unit: '1M tokens',
  input_price: 1,
  output_price: 5,
  cache_write_price: null,
  cache_read_price: null,
  image_input_price: null,
  image_output_price: null,
  per_request_price: null,
  intervals: [],
  peak_rate: { enabled: false, start: null, end: null, multiplier: null },
}

const draft = {
  id: 1,
  slug: 'anthropic-claude-sonnet-4-5',
  model: 'claude-sonnet-4-5',
  platform: 'anthropic',
  metadata_model_id: 'claude-sonnet-4-5',
  display_name_zh: 'Claude Sonnet 4.5',
  display_name_en: 'Claude Sonnet 4.5',
  summary_zh: '适合代码与日常任务',
  summary_en: '',
  provider: 'Anthropic',
  logo_key: 'claude',
  category: 'code',
  tags: ['代码'],
  capabilities: ['vision', 'tools'],
  context_window: 200000,
  max_output_tokens: 64000,
  public_group_id: 3,
  public_group: {
    id: 3,
    name: '标准分组',
    platform: 'anthropic',
    rate_multiplier: 1,
    peak_rate_enabled: false,
    peak_start: null,
    peak_end: null,
    peak_rate_multiplier: null,
  },
  featured: false,
  sort_order: 10,
  status: 'draft' as const,
  pricing,
  validation: { valid: true, errors: [] },
  published_at: null,
  created_at: '2026-07-18T00:00:00Z',
  updated_at: '2026-07-18T00:00:00Z',
}

const candidate = {
  model: 'gpt-5',
  platform: 'openai',
  provider: 'OpenAI',
  logo_key: 'openai',
  category: 'reasoning',
  context_window: 400000,
  max_output_tokens: 128000,
  capabilities: ['reasoning', 'tools'],
  group_options: [
    {
      id: 8,
      name: 'OpenAI 标准',
      platform: 'openai',
      channel_id: 4,
      channel_name: 'OpenAI 主渠道',
      rate_multiplier: 1,
      peak_rate_enabled: false,
      peak_start: null,
      peak_end: null,
      peak_rate_multiplier: null,
      pricing,
    },
  ],
}

const SlotLayout = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', [slots.actions?.(), slots.filters?.(), slots.table?.(), slots.default?.()])
  },
})

const DataTableStub = defineComponent({
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots }) {
    return () => h('div', { 'data-testid': 'table' }, (props.data as Array<Record<string, unknown>>).map((row) =>
      h('div', { class: 'data-row' }, [
        slots['cell-model']?.({ row, value: row.model }),
        slots['cell-actions']?.({ row, value: null }),
      ]),
    ))
  },
})

const BaseDialogStub = defineComponent({
  props: { show: Boolean, title: String },
  setup(props, { slots }) {
    return () => props.show
      ? h('section', { class: 'dialog', 'data-title': props.title }, [slots.default?.(), slots.footer?.()])
      : null
  },
})

const SelectStub = defineComponent({
  props: { modelValue: [String, Number, Boolean], options: { type: Array, default: () => [] } },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit }) {
    return () => h('select', {
      value: props.modelValue,
      onChange: (event: Event) => {
        const value = (event.target as HTMLSelectElement).value
        emit('update:modelValue', value)
        emit('change', value)
      },
    }, (props.options as Array<Record<string, unknown>>).map((option) =>
      h('option', { value: option.value as string }, String(option.label)),
    ))
  },
})

function mountView() {
  return mount(ModelCatalogView, {
    global: {
      stubs: {
        AppLayout: SlotLayout,
        TablePageLayout: SlotLayout,
        DataTable: DataTableStub,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        ModelIcon: true,
        Select: SelectStub,
        Toggle: true,
        Icon: true,
      },
    },
  })
}

describe('admin ModelCatalogView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    list.mockResolvedValue({ items: [draft], total: 1 })
    candidates.mockResolvedValue({ items: [candidate], total: 1 })
    getById.mockResolvedValue(draft)
    create.mockResolvedValue({ ...draft, id: 2, model: candidate.model, platform: candidate.platform })
    update.mockResolvedValue(draft)
    publish.mockResolvedValue({ ...draft, status: 'published' })
    unpublish.mockResolvedValue(draft)
  })

  it('loads catalog entries and publishes a valid draft', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(list).toHaveBeenCalledWith({ status: 'all', search: '' })
    const publishButton = wrapper.find(`button[title="admin.modelCatalog.publish"]`)
    expect(publishButton.exists()).toBe(true)

    await publishButton.trigger('click')
    await flushPromises()

    expect(publish).toHaveBeenCalledWith(1)
    expect(showSuccess).toHaveBeenCalled()
  })

  it('creates an unpublished draft from an eligible candidate', async () => {
    const wrapper = mountView()
    await flushPromises()

    const discoverButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.modelCatalog.discoverCandidates'),
    )
    expect(discoverButton).toBeTruthy()
    await discoverButton!.trigger('click')
    await flushPromises()

    const createButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.modelCatalog.createDraft'),
    )
    expect(createButton).toBeTruthy()
    await createButton!.trigger('click')
    await flushPromises()

    expect(create).toHaveBeenCalledWith(expect.objectContaining({
      model: 'gpt-5',
      platform: 'openai',
      public_group_id: null,
    }))
    expect(getById).toHaveBeenCalledWith(2)
  })
})
