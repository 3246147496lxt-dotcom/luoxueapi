import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { adaptQuotaOverview } from '@/api/quotaOverview'
import { useQuotaViewer } from './useQuotaViewer'

const desktop = vi.hoisted(() => ({
  getViewerSnapshot: vi.fn(),
  refreshQuotaOverview: vi.fn(),
  disconnectQuotaAccount: vi.fn(),
  cancelQuotaPairing: vi.fn(),
  openQuotaPairingPage: vi.fn(),
  pollQuotaPairing: vi.fn(),
  startQuotaPairing: vi.fn()
}))

vi.mock('@/lib/desktop', () => ({
  getViewerSnapshot: desktop.getViewerSnapshot,
  refreshQuotaOverview: desktop.refreshQuotaOverview,
  disconnectQuotaAccount: desktop.disconnectQuotaAccount,
  cancelQuotaPairing: desktop.cancelQuotaPairing,
  openQuotaPairingPage: desktop.openQuotaPairingPage,
  pollQuotaPairing: desktop.pollQuotaPairing,
  startQuotaPairing: desktop.startQuotaPairing
}))

const overviewFixture = {
  schema_version: 1,
  generated_at: '2026-07-30T03:00:00Z',
  as_of: '2026-07-30T03:00:00Z',
  fresh_until: '2026-07-30T03:05:00Z',
  display_timezone: 'Asia/Shanghai',
  freshness: 'fresh',
  account: {
    display_label: 'pu***@example.com',
    data_scope: 'all_enabled_api_keys',
    quota_state: 'all_resources_available'
  },
  wallet: {
    unit: 'snow_credit',
    state: 'available',
    available: '128.6400000000',
    reserved: '0.0000000000',
    today_spend: '1.1600000000',
    month_spend: '10.7400000000',
    balance_billed_key_count: 1
  },
  subscriptions: []
}

function mountViewer() {
  let viewer!: ReturnType<typeof useQuotaViewer>
  const wrapper = mount(
    defineComponent({
      setup() {
        viewer = useQuotaViewer()
        return () => h('div')
      }
    })
  )
  return { viewer, wrapper }
}

describe('quota viewer runtime state', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    desktop.cancelQuotaPairing.mockResolvedValue(undefined)
    desktop.disconnectQuotaAccount.mockResolvedValue(undefined)
  })

  it('keeps an in-memory last-good snapshot visibly stale when a refresh has no data', async () => {
    desktop.getViewerSnapshot.mockResolvedValue({
      status: 'ready',
      overview: overviewFixture,
      source: 'cache',
      fetched_at: overviewFixture.generated_at,
      error_code: null,
      message: null
    })
    desktop.refreshQuotaOverview.mockResolvedValue({
      status: 'unavailable',
      overview: null,
      source: null,
      fetched_at: null,
      error_code: 'QUOTA_SERVICE_UNAVAILABLE',
      message: 'temporarily unavailable'
    })
    const { viewer, wrapper } = mountViewer()

    await viewer.boot()
    await vi.waitFor(() => expect(viewer.status.value).toBe('stale'))

    expect(viewer.overview.value?.todaySpend).toBe('1.1600000000')
    expect(viewer.dataStatus.value).toBe('stale')
    wrapper.unmount()
  })

  it('clears sensitive UI state before persistent disconnect cleanup finishes', async () => {
    let finishDisconnect!: () => void
    desktop.disconnectQuotaAccount.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          finishDisconnect = resolve
        })
    )
    const { viewer, wrapper } = mountViewer()
    viewer.status.value = 'ready'
    viewer.overview.value = adaptQuotaOverview(overviewFixture)

    const disconnecting = viewer.disconnect()

    expect(viewer.status.value).toBe('disconnected')
    expect(viewer.overview.value).toBeNull()
    finishDisconnect()
    await disconnecting
    wrapper.unmount()
  })
})
