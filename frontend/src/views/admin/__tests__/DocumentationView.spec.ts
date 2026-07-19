import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import DocumentationView from '../DocumentationView.vue'
import ImageUpload from '@/components/common/ImageUpload.vue'

const {
  get,
  saveDraft,
  publish,
  revisions,
  restoreRevision,
  uploadAsset,
  showSuccess,
  showError,
  showWarning,
} = vi.hoisted(() => ({
  get: vi.fn(),
  saveDraft: vi.fn(),
  publish: vi.fn(),
  revisions: vi.fn(),
  restoreRevision: vi.fn(),
  uploadAsset: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    documentation: { get, saveDraft, publish, revisions, restoreRevision, uploadAsset },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError, showWarning }),
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
    }),
  }
})

const content = {
  schema_version: 1 as const,
  site_config: { brand_name: '落雪API', support_contact: '客服' },
  tutorials: [
    {
      id: 'quick-start',
      tab_label: '新手教程',
      icon: 'key',
      icon_svg: '<svg viewBox="0 0 24 24"><path d="M2 2h20v20H2z"/></svg>',
      description: '完成首次接入',
      steps: [{ title: '创建密钥', description: '打开密钥页面并创建。' }],
    },
  ],
}

const SlotLayout = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const BaseDialogStub = defineComponent({
  props: { show: Boolean, title: String },
  setup(props, { slots }) {
    return () => props.show
      ? h('section', { class: 'base-dialog-stub', 'data-title': props.title }, [slots.default?.(), slots.footer?.()])
      : null
  },
})

const ConfirmDialogStub = defineComponent({
  props: { show: Boolean, title: String, message: String },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('button', { class: 'confirm-dialog-stub', onClick: () => emit('confirm') }, props.title)
      : null
  },
})

function mountView() {
  return mount(DocumentationView, {
    global: {
      stubs: {
        AppLayout: SlotLayout,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        DocumentationPreview: true,
        Icon: true,
      },
    },
  })
}

describe('admin DocumentationView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    get.mockResolvedValue({
      draft: { content, updated_at: '2026-07-19T00:00:00Z' },
      published: { content, version: 1, published_at: '2026-07-18T00:00:00Z' },
    })
    saveDraft.mockResolvedValue({ content, updated_at: '2026-07-19T01:00:00Z' })
    publish.mockResolvedValue({ draft: { content }, published: { content, version: 2 } })
    revisions.mockResolvedValue({ items: [{ id: 8, version: 8, published_at: '2026-07-17T00:00:00Z' }] })
    restoreRevision.mockResolvedValue({ content })
    uploadAsset.mockResolvedValue({ url: '/documentation/assets/image.png' })
  })

  it('loads structured content, tracks edits, and saves a draft', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(get).toHaveBeenCalledOnce()
    const labelInput = wrapper.get('#tutorial-label-quick-start')
    await labelInput.setValue('快速开始')
    expect(wrapper.text()).toContain('admin.documentation.unsavedChanges')

    const saveButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.saveDraft'),
    )
    expect(saveButton).toBeTruthy()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(saveDraft).toHaveBeenCalledWith(expect.objectContaining({
      schema_version: 1,
      site_config: { brand_name: '落雪API', support_contact: '客服' },
      tutorials: [expect.objectContaining({ tab_label: '快速开始' })],
    }))
    expect(showSuccess).toHaveBeenCalledWith('admin.documentation.saveSuccess')
  })

  it('sanitizes custom SVG icons, saves them, and removes them back to the fallback icon', async () => {
    const wrapper = mountView()
    await flushPromises()

    const uploader = wrapper.getComponent(ImageUpload)
    expect(uploader.props('modelValue')).toContain('<svg')

    uploader.vm.$emit(
      'update:modelValue',
      '<svg viewBox="0 0 24 24" onload="alert(1)"><script>alert(1)</script><circle cx="12" cy="12" r="8"/></svg>',
    )
    await flushPromises()

    const saveButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.saveDraft'),
    )
    await saveButton!.trigger('click')
    await flushPromises()

    const savedIcon = saveDraft.mock.calls.at(-1)?.[0].tutorials[0].icon_svg
    expect(savedIcon).toContain('<circle')
    expect(savedIcon).not.toContain('onload')
    expect(savedIcon).not.toContain('<script')

    uploader.vm.$emit('update:modelValue', '')
    await flushPromises()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(saveDraft.mock.calls.at(-1)?.[0].tutorials[0]).not.toHaveProperty('icon_svg')
    expect(saveDraft.mock.calls.at(-1)?.[0].tutorials[0].icon).toBe('key')
  })

  it('validates and publishes the current draft through confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    const publishButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.publish'),
    )
    expect(publishButton).toBeTruthy()
    await publishButton!.trigger('click')
    await wrapper.get('.confirm-dialog-stub').trigger('click')
    await flushPromises()

    expect(publish).toHaveBeenCalledOnce()
    expect(showSuccess).toHaveBeenCalledWith('admin.documentation.publishSuccess')
  })

  it('blocks publication when a section ID does not match the backend contract', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('#tutorial-id-quick-start').setValue('1_Invalid')
    const publishButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.publish'),
    )
    await publishButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.documentation.validation.categoryIdInvalid')
    expect(wrapper.find('.confirm-dialog-stub').exists()).toBe(false)
    expect(publish).not.toHaveBeenCalled()
  })

  it('keeps the editor mounted while the section ID is edited continuously', async () => {
    const wrapper = mountView()
    await flushPromises()

    const idInput = wrapper.get('#tutorial-id-quick-start')
    await idInput.setValue('quick')
    await wrapper.get('#tutorial-id-quick').setValue('quick-start-updated')

    expect(wrapper.get('#tutorial-id-quick-start-updated').element).toHaveProperty('value', 'quick-start-updated')
    expect(wrapper.text()).toContain('admin.documentation.categoryDetails')
    expect(wrapper.text()).toContain('admin.documentation.steps')
  })

  it('surfaces incomplete optional code and image sections before publication', async () => {
    const wrapper = mountView()
    await flushPromises()

    const addCodeButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.addCode'),
    )
    const addImageButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.addImage'),
    )
    await addCodeButton!.trigger('click')
    await addImageButton!.trigger('click')

    const publishButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.publish'),
    )
    await publishButton!.trigger('click')

    expect(wrapper.text()).toContain('admin.documentation.validation.codeRequired')
    expect(wrapper.text()).toContain('admin.documentation.validation.imageUrlRequired')
    expect(wrapper.text()).toContain('admin.documentation.validation.imageAltRequired')
    expect(publish).not.toHaveBeenCalled()
  })

  it('loads publication history and restores a revision to the draft', async () => {
    const wrapper = mountView()
    await flushPromises()

    const historyButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.history'),
    )
    await historyButton!.trigger('click')
    await flushPromises()

    expect(revisions).toHaveBeenCalledOnce()
    const restoreButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.documentation.restoreToDraft'),
    )
    expect(restoreButton).toBeTruthy()
    await restoreButton!.trigger('click')
    await wrapper.get('.confirm-dialog-stub').trigger('click')
    await flushPromises()

    expect(restoreRevision).toHaveBeenCalledWith(8)
    expect(showSuccess).toHaveBeenCalledWith('admin.documentation.restoreSuccess')
  })
})
