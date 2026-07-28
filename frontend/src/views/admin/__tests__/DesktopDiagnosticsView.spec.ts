import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  download: vi.fn(),
}))
const appStore = vi.hoisted(() => ({ showError: vi.fn() }))

vi.mock('@/api/admin', () => ({
  adminAPI: { desktopDiagnostics: api },
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/composables/usePersistedPageSize', () => ({ getPersistedPageSize: () => 20 }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key}:${JSON.stringify(params)}` : key
      ),
    }),
  }
})

import DesktopDiagnosticsView from '../DesktopDiagnosticsView.vue'

const metadata = {
  id: 'f7f73bea-35a6-4fc7-b3f3-69731589bd5d',
  device_id: '2a030340-e67a-49e3-bf54-91ddf0bcb607',
  user_id: 7,
  app_version: '0.1.0',
  platform: 'macos',
  architecture: 'arm64',
  os_version: '15.5',
  gateway_status: 'running',
  codex_config_status: 'managed',
  request_sample_count: 3,
  request_error_count: 1,
  created_at: '2026-07-28T09:00:00Z',
  expires_at: '2099-08-04T09:00:00Z',
}

const detail = {
  ...metadata,
  diagnostic: {
    app_version: '0.1.0',
    platform: 'macos',
    architecture: 'arm64',
    os_version: '15.5',
    gateway: { status: 'running', port: 11430, takeover_enabled: true },
    codex: { config_status: 'managed', installations: [] },
    route: { group_id: 12, model: 'gpt-5.4', available_route_count: 2 },
    requests: {
      sample_count: 3,
      success_count: 2,
      error_count: 1,
      average_duration_ms: 1234,
      average_first_token_ms: 456,
      recent: [],
    },
  },
}

function mountView() {
  return mount(DesktopDiagnosticsView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        TablePageLayout: {
          template: '<div><slot name="header"/><slot name="table"/><slot name="pagination"/></div>',
        },
        AdminPageHeader: {
          template: '<header><slot name="meta"/><slot name="secondary-actions"/></header>',
        },
        DataTable: {
          props: ['data'],
          template: '<div><div v-for="row in data" :key="row.id" data-testid="diagnostic-row"><slot name="cell-owner" :row="row"/><slot name="cell-actions" :row="row"/></div><slot v-if="!data.length" name="empty"/></div>',
        },
        Pagination: true,
        BaseDialog: {
          props: ['show'],
          emits: ['close'],
          template: '<section v-if="show" data-testid="detail-dialog"><slot/><slot name="footer"/></section>',
        },
        Icon: true,
      },
    },
  })
}

describe('DesktopDiagnosticsView', () => {
  beforeEach(() => {
    api.list.mockReset()
    api.get.mockReset()
    api.download.mockReset()
    appStore.showError.mockReset()
    api.list.mockResolvedValue({ items: [metadata], total: 1, page: 1, page_size: 20, pages: 1 })
    api.get.mockResolvedValue(detail)
  })

  it('loads only metadata, then fetches the audited detail on explicit view', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(api.list).toHaveBeenCalledWith({ page: 1, page_size: 20 })
    expect(api.get).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="diagnostic-row"]').exists()).toBe(true)

    await wrapper.get('button[aria-label="admin.desktopDiagnostics.actions.view"]').trigger('click')
    await flushPromises()

    expect(api.get).toHaveBeenCalledWith(metadata.id)
    expect(wrapper.get('[data-testid="detail-dialog"]').text()).toContain('gpt-5.4')
    expect(wrapper.get('pre').text()).not.toContain('Authorization')
  })

  it('shows the empty retention-aware state when no active bundles remain', async () => {
    api.list.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.desktopDiagnostics.emptyTitle')
    expect(wrapper.text()).toContain('admin.desktopDiagnostics.emptyDescription')
  })
})
