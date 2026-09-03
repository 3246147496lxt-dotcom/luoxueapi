import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import SkillImportsView from '../SkillImportsView.vue'

const {
  route,
  listSources,
  listSchedules,
  listRuns,
  getRun,
  listItems,
  listEvents,
  publishRun,
  createRun,
  showSuccess,
  showError,
} = vi.hoisted(() => ({
  route: { query: {} as Record<string, string> },
  listSources: vi.fn(),
  listSchedules: vi.fn(),
  listRuns: vi.fn(),
  getRun: vi.fn(),
  listItems: vi.fn(),
  listEvents: vi.fn(),
  publishRun: vi.fn(),
  createRun: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/skillImport', () => ({
  default: {
    listSources,
    listSchedules,
    listRuns,
    getRun,
    listItems,
    listEvents,
    publishRun,
    createRun,
    uploadRun: vi.fn(),
    cancelRun: vi.fn(),
    retryFailed: vi.fn(),
    createSource: vi.fn(),
    updateSource: vi.fn(),
    createSchedule: vi.fn(),
    updateSchedule: vi.fn(),
    runSchedule: vi.fn(),
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showSuccess, showError }),
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (_error: unknown, fallback: string) => fallback,
}))

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key,
      locale: { value: 'en' },
    }),
  }
})

const source = {
  id: 1,
  name: 'skills.sh All Time',
  adapter: 'skills_sh',
  namespace: 'skills.sh',
  base_url: 'https://skills.sh',
  source_config: {},
  catalog_priority: 10,
  enabled: true,
  created_at: '2026-08-12T00:00:00Z',
  updated_at: '2026-08-12T00:00:00Z',
}

function counts() {
  return {
    requested: 500,
    discovered: 20,
    prepared: 5,
    created: 5,
    updated: 0,
    unchanged: 0,
    skipped: 0,
    blocked: 0,
    failed: 0,
    published: 0,
  }
}

function run(status = 'discovering') {
  return {
    id: 7,
    source_id: 1,
    trigger_type: 'manual',
    mode: 'auto_publish',
    status,
    request_config: {},
    counts: counts(),
    attempt_count: 0,
    created_at: '2026-08-12T00:00:00Z',
    updated_at: '2026-08-12T00:00:00Z',
  }
}

const item = {
  id: 41,
  run_id: 7,
  stable_key: { source_id: 1, namespace: 'skills.sh', external_id: 'owner/repo/skill' },
  rank: 1,
  market_slug: 'find-skills',
  status: 'ready',
  upstream_name: 'find-skills',
  origin_url: 'https://skills.sh/owner/repo/find-skills',
  desired_skill: {
    slug: 'find-skills',
    display_name: 'Find Skills',
    summary: 'Find a Skill',
    category: 'Productivity',
  },
  license_unverified: false,
  excluded_files: [],
  warnings: [],
  attempt_count: 0,
  created_at: '2026-08-12T00:00:00Z',
  updated_at: '2026-08-12T00:00:00Z',
}

const SlotStub = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', [
      slots.meta?.(),
      slots['secondary-actions']?.(),
      slots['primary-actions']?.(),
      slots.default?.(),
    ])
  },
})

const ConfirmStub = defineComponent({
  props: { show: Boolean },
  emits: ['confirm', 'cancel'],
  setup(props, { emit }) {
    return () => props.show
      ? h('button', { 'data-testid': 'confirm-run-action', onClick: () => emit('confirm') }, 'confirm')
      : null
  },
})

function mountView() {
  return mount(SkillImportsView, {
    global: {
      stubs: {
        AppLayout: SlotStub,
        AdminPageHeader: SlotStub,
        SkillMarketNav: true,
        ConfirmDialog: ConfirmStub,
        Icon: true,
      },
    },
  })
}

describe('admin SkillImportsView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.clearAllMocks()
    route.query = {}
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'visible' })
    listSources.mockResolvedValue({ items: [source], total: 1, page: 1, page_size: 200, pages: 1 })
    listSchedules.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 200, pages: 1 })
    listRuns.mockResolvedValue({ items: [run()], total: 1, page: 1, page_size: 100, pages: 1 })
    getRun.mockResolvedValue(run())
    listItems.mockResolvedValue({ items: [item], total: 1, page: 1, page_size: 200, pages: 1 })
    listEvents.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 200, pages: 1 })
    publishRun.mockResolvedValue(run('succeeded'))
    createRun.mockResolvedValue(run('queued'))
  })

  it('loads the queue and polls active runs every two seconds', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Find Skills')
    expect(listRuns).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(2_000)
    await flushPromises()
    expect(listRuns).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('pauses polling while the page is hidden', async () => {
    const wrapper = mountView()
    await flushPromises()
    Object.defineProperty(document, 'visibilityState', { configurable: true, value: 'hidden' })
    document.dispatchEvent(new Event('visibilitychange'))

    await vi.advanceTimersByTimeAsync(30_000)
    expect(listRuns).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('publishes ready items only after confirmation', async () => {
    listRuns.mockResolvedValue({ items: [run('awaiting_review')], total: 1, page: 1, page_size: 100, pages: 1 })
    getRun.mockResolvedValue(run('awaiting_review'))
    const wrapper = mountView()
    await flushPromises()

    const publishButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.publishEligible'))
    expect(publishButton).toBeTruthy()
    await publishButton!.trigger('click')
    expect(publishRun).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="confirm-run-action"]').trigger('click')
    await flushPromises()
    expect(publishRun).toHaveBeenCalledWith(7, { item_ids: [] }, expect.stringContaining('skill-import-publish'))
    wrapper.unmount()
  })

  it('publishes the full prepared cohort even when every item is unchanged', async () => {
    const unchangedRun = run('awaiting_review')
    unchangedRun.counts.prepared = 5
    unchangedRun.counts.unchanged = 5
    listRuns.mockResolvedValue({ items: [unchangedRun], total: 1, page: 1, page_size: 50, pages: 1 })
    getRun.mockResolvedValue(unchangedRun)
    const wrapper = mountView()
    await flushPromises()

    const publishButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.publishEligible'))
    expect(publishButton).toBeTruthy()
    await publishButton!.trigger('click')
    await wrapper.get('[data-testid="confirm-run-action"]').trigger('click')
    await flushPromises()

    expect(publishRun).toHaveBeenCalledWith(7, { item_ids: [] }, expect.stringContaining('skill-import-publish'))
    wrapper.unmount()
  })

  it('renders a non-link placeholder when an item has no origin URL', async () => {
    listItems.mockResolvedValue({
      items: [{ ...item, origin_url: '' }],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1,
    })
    const wrapper = mountView()
    await flushPromises()

    const originCell = wrapper.get('.skill-import-items-table tbody td:nth-child(4)')
    expect(originCell.find('a').exists()).toBe(false)
    expect(originCell.text()).toContain('admin.skills.imports.detail.originUnavailable')
    wrapper.unmount()
  })

  it('rejects manifest files larger than 5 MiB before upload', async () => {
    const wrapper = mountView()
    await flushPromises()
    const uploadButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.uploadManifest'))
    await uploadButton!.trigger('click')
    const input = wrapper.get('input[type="file"]')
    const oversized = new File([new Uint8Array(5 * 1024 * 1024 + 1)], 'skills.zip', {
      type: 'application/zip',
    })
    Object.defineProperty(input.element, 'files', { configurable: true, value: [oversized] })
    await input.trigger('change')

    expect(showError).toHaveBeenCalledWith('admin.skills.imports.errors.fileTooLarge')
    expect((input.element as HTMLInputElement).value).toBe('')
    wrapper.unmount()
  })

  it('offers cancellation after preparation while a run awaits review', async () => {
    listRuns.mockResolvedValue({ items: [run('awaiting_review')], total: 1, page: 1, page_size: 50, pages: 1 })
    getRun.mockResolvedValue(run('awaiting_review'))
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('button').some((button) =>
      button.text().includes('admin.skills.imports.actions.cancelRun'))).toBe(true)
    wrapper.unmount()
  })

  it('allows an empty queued run to be cancelled', async () => {
    const queued = run('queued')
    queued.counts.prepared = 0
    listRuns.mockResolvedValue({ items: [queued], total: 1, page: 1, page_size: 50, pages: 1 })
    getRun.mockResolvedValue(queued)
    const wrapper = mountView()
    await flushPromises()

    const cancelButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.cancelRun'))
    expect(cancelButton).toBeTruthy()
    expect(cancelButton!.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('offers recovery for a failed run even when no item failure was counted', async () => {
    const failed = run('failed')
    failed.counts.failed = 0
    listRuns.mockResolvedValue({ items: [failed], total: 1, page: 1, page_size: 50, pages: 1 })
    getRun.mockResolvedValue(failed)
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('button').some((button) =>
      button.text().includes('admin.skills.imports.actions.retryFailed'))).toBe(true)
    wrapper.unmount()
  })

  it('keeps the manual-run safety gate explicit and enabled by default', async () => {
    const wrapper = mountView()
    await flushPromises()
    const newRun = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.newRun'))
    await newRun!.trigger('click')

    const mode = wrapper.findAll('select').find((select) => select.find('option[value="review"]').exists())
    await mode!.setValue('review')
    const safeGate = wrapper.get('[aria-label="admin.skills.imports.fields.safeGate"]')
    expect(safeGate.attributes('aria-checked')).toBe('true')
    await safeGate.trigger('click')
    expect(safeGate.attributes('aria-checked')).toBe('false')

    await wrapper.get('.skill-import-composer__form').trigger('submit')
    await flushPromises()
    expect(createRun).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'review',
        run_config: expect.objectContaining({ safe_gate: false }),
      }),
      expect.stringContaining('skill-import-run'),
    )
    wrapper.unmount()
  })

  it('loads another run page instead of limiting the queue to the first 50', async () => {
    listRuns.mockImplementation(async (params: { page: number; page_size: number }) => ({
      items: [run()],
      total: 75,
      page: params.page,
      page_size: params.page_size,
      pages: 2,
    }))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.skill-import-queue-pagination .pagination-button--next').trigger('click')
    await flushPromises()
    expect(listRuns).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2, page_size: 50 }))
    wrapper.unmount()
  })

  it('creates schedule drafts disabled at 03:00 Asia/Shanghai', async () => {
    route.query = { tab: 'schedules' }
    const wrapper = mountView()
    await flushPromises()

    const newSchedule = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.skills.imports.actions.newSchedule'))
    expect(newSchedule).toBeTruthy()
    await newSchedule!.trigger('click')

    expect(wrapper.get('input.skill-import-mono-input').element).toHaveProperty('value', '0 3 * * *')
    expect(wrapper.findAll('input').some((input) => input.element.value === 'Asia/Shanghai')).toBe(true)
    expect(wrapper.get('[aria-label="admin.skills.imports.fields.scheduleEnabled"]').attributes('aria-checked')).toBe('false')
    expect(wrapper.get('[aria-label="admin.skills.imports.fields.requireAllValid"]').attributes('aria-checked')).toBe('false')
    expect(wrapper.get('[aria-label="admin.skills.imports.fields.allowLicenseUnverified"]').attributes('aria-checked')).toBe('true')
    expect(wrapper.get('[aria-label="admin.skills.imports.fields.safeGate"]').attributes('aria-checked')).toBe('true')
    wrapper.unmount()
  })
})
