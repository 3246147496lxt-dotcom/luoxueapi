import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import SkillEditorView from '../SkillEditorView.vue'

const {
  getById,
  create,
  update,
  uploadVersion,
  publish,
  activateVersion,
  yankVersion,
  archive,
  push,
  replace,
  showSuccess,
  showError,
  routeParams,
} = vi.hoisted(() => ({
  getById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  uploadVersion: vi.fn(),
  publish: vi.fn(),
  activateVersion: vi.fn(),
  yankVersion: vi.fn(),
  archive: vi.fn(),
  push: vi.fn(),
  replace: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  routeParams: { id: '1' as string | undefined },
}))

vi.mock('@/api/admin/skills', () => ({
  default: {
    getById,
    create,
    update,
    uploadVersion,
    publish,
    activateVersion,
    yankVersion,
    archive,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (error: { message?: string }, fallback: string) => error?.message || fallback,
}))

vi.mock('@/composables/useStepUp', () => ({
  useStepUp: () => ({
    visible: { value: false },
    blockedReason: { value: '' },
    run: (action: () => Promise<unknown>) => action(),
    onVerified: vi.fn(),
    onCancel: vi.fn(),
  }),
  isStepUpCancelled: () => false,
  isStepUpBlocked: () => false,
  stepUpBlockReason: () => '',
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: routeParams }),
  useRouter: () => ({ push, replace }),
  onBeforeRouteLeave: vi.fn(),
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

const availableVersion = {
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
  file_manifest: [{ path: 'SKILL.md', byte_size: 100, sha256: 'b'.repeat(64) }],
  validation_report: { valid: true, errors: [], warnings: [] },
  status: 'available' as const,
  yanked_at: null,
  created_at: '2026-08-01T00:00:00Z',
  download_count: 0,
}

const draftSkill = {
  id: 1,
  slug: 'api-doc-writer',
  display_name: 'API 文档生成器',
  summary: '生成接入文档',
  description: '读取接口定义并生成文档。',
  category: '文档与数据',
  tags: ['Codex'],
  icon: '',
  example_prompts: ['为这个接口生成文档'],
  risk_notes: '只读取当前项目文件，不访问网络。',
  status: 'draft' as const,
  featured: false,
  sort_order: 1,
  current_version_id: null,
  current_version: null,
  download_count: 0,
  published_at: null,
  archived_at: null,
  created_at: '2026-08-01T00:00:00Z',
  updated_at: '2026-08-01T00:00:00Z',
  versions: [availableVersion],
}

const activeVersion = { ...availableVersion, status: 'active' as const }
const secondVersion = { ...availableVersion, id: 12, version: '1.1.0', status: 'available' as const }
const publishedSkill = {
  ...draftSkill,
  status: 'published' as const,
  current_version_id: 11,
  current_version: activeVersion,
  published_at: '2026-08-02T00:00:00Z',
  versions: [activeVersion, secondVersion],
}

const SlotLayout = defineComponent({
  setup(_props, { slots }) { return () => h('div', slots.default?.()) },
})

const HeaderStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('header', [slots.meta?.(), slots['secondary-actions']?.(), slots['primary-actions']?.()])
  },
})

const ConfirmDialogStub = defineComponent({
  props: { show: Boolean, title: String },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('button', { class: 'confirm-action', onClick: () => emit('confirm') }, props.title)
      : null
  },
})

const PackageUploaderStub = defineComponent({
  name: 'SkillPackageUploader',
  emits: ['upload'],
  setup(_props, { emit, expose }) {
    expose({ reset: vi.fn() })
    return () => h('button', {
      'data-testid': 'emit-upload',
      onClick: () => emit('upload', {
        file: new File(['zip'], 'skill.zip', { type: 'application/zip' }),
        version: '1.2.0',
        changelog: 'New release',
      }),
    })
  },
})

const ReportStub = defineComponent({
  props: { report: Object },
  setup(props) {
    return () => h('div', { 'data-testid': 'validation-report' },
      ((props.report as { errors?: Array<{ message: string }> } | null)?.errors ?? [])
        .map((issue) => issue.message))
  },
})

function mountView() {
  return mount(SkillEditorView, {
    global: {
      stubs: {
        AppLayout: SlotLayout,
        AdminPageHeader: HeaderStub,
        ConfirmDialog: ConfirmDialogStub,
        SkillPackageUploader: PackageUploaderStub,
        SkillValidationReportPanel: ReportStub,
        SkillPreviewCard: true,
        SkillStatusBadge: true,
        TotpStepUpDialog: true,
        Toggle: true,
        Icon: true,
      },
    },
  })
}

describe('admin SkillEditorView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    routeParams.id = '1'
    getById.mockResolvedValue(draftSkill)
    create.mockResolvedValue(draftSkill)
    update.mockResolvedValue(draftSkill)
    uploadVersion.mockResolvedValue({ ...availableVersion, id: 13, version: '1.2.0' })
    publish.mockResolvedValue(publishedSkill)
    activateVersion.mockResolvedValue({
      ...publishedSkill,
      current_version_id: 12,
      current_version: { ...secondVersion, status: 'active' },
    })
    yankVersion.mockResolvedValue(publishedSkill)
    archive.mockResolvedValue({ ...draftSkill, status: 'archived' })
  })

  it('updates market metadata as a saved draft', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('#skill-summary').setValue('生成清晰、可执行的接入文档')
    await wrapper.get('[data-testid="save-skill-draft"]').trigger('click')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(1, expect.objectContaining({
      slug: 'api-doc-writer',
      summary: '生成清晰、可执行的接入文档',
    }))
  })

  it('publishes a valid immutable version only after confirmation', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="publish-skill"]').trigger('click')
    expect(publish).not.toHaveBeenCalled()
    await wrapper.get('.confirm-action').trigger('click')
    await flushPromises()

    expect(publish).toHaveBeenCalledWith(1, 11)
  })

  it('renders the rejected ZIP validation report from serialized error metadata', async () => {
    uploadVersion.mockRejectedValue({
      message: 'Skill archive validation failed',
      metadata: {
        validation_report: JSON.stringify({
          valid: false,
          errors: [{ code: 'PATH_TRAVERSAL', message: '文件路径不能包含 ..', path: '../secret' }],
          warnings: [],
        }),
      },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-testid="emit-upload"]').trigger('click')
    await flushPromises()

    expect(uploadVersion).toHaveBeenCalledWith(1, expect.objectContaining({ version: '1.2.0' }))
    expect(wrapper.get('[data-testid="validation-report"]').text()).toContain('文件路径不能包含 ..')
    expect(wrapper.text()).toContain('Skill archive validation failed')
  })

  it('switches a published Skill to another checked version after confirmation', async () => {
    getById.mockResolvedValue(publishedSkill)
    const wrapper = mountView()
    await flushPromises()

    const activateButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.editor.versions.activate'),
    )
    expect(activateButton).toBeTruthy()
    await activateButton!.trigger('click')
    await wrapper.get('.confirm-action').trigger('click')
    await flushPromises()

    expect(activateVersion).toHaveBeenCalledWith(1, 12)
  })
})
