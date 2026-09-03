import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const viewerMocks = vi.hoisted(() => ({
  refresh: vi.fn().mockResolvedValue(undefined)
}))

const desktopMocks = vi.hoisted(() => ({
  hideTrayPopover: vi.fn().mockResolvedValue(undefined),
  openMainPanel: vi.fn().mockResolvedValue(undefined),
  setTrayDetailPanelOpen: vi.fn().mockResolvedValue(false),
  onTrayPopoverShown: vi.fn().mockResolvedValue(vi.fn()),
  onTrayPanelLayoutChanged: vi.fn().mockResolvedValue(vi.fn())
}))

vi.mock('@/composables/useQuotaViewer', async () => {
  const { computed, ref } = await import('vue')
  const { demoOverview } = await import('@/data/demo')

  return {
    useQuotaViewer: () => ({
      status: ref('ready'),
      overview: ref(demoOverview),
      pairing: ref(null),
      refreshing: ref(false),
      errorCode: ref(null),
      errorMessage: ref(null),
      dataStatus: computed(() => 'ready'),
      boot: vi.fn().mockResolvedValue(undefined),
      refresh: viewerMocks.refresh,
      connect: vi.fn().mockResolvedValue(undefined),
      disconnect: vi.fn().mockResolvedValue(undefined),
      reopenPairing: vi.fn().mockResolvedValue(undefined),
      cancelConnection: vi.fn().mockResolvedValue(undefined)
    })
  }
})

vi.mock('@/lib/desktop', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/desktop')>()

  return {
    ...actual,
    hasTauriRuntime: () => true,
    hideTrayPopover: desktopMocks.hideTrayPopover,
    onTrayPopoverShown: desktopMocks.onTrayPopoverShown,
    onTrayPanelLayoutChanged: desktopMocks.onTrayPanelLayoutChanged,
    openMainPanel: desktopMocks.openMainPanel,
    setTrayDetailPanelOpen: desktopMocks.setTrayDetailPanelOpen
  }
})

describe('menu bar App surface', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-29T06:00:00Z'))
    for (const mock of Object.values(desktopMocks)) mock.mockClear()
    viewerMocks.refresh.mockClear()
    window.history.replaceState({}, '', '/?surface=tray')
    document.documentElement.dataset.runtime = 'tauri'
    document.documentElement.dataset.surface = 'tray'
  })

  afterEach(() => {
    vi.useRealTimers()
    window.history.replaceState({}, '', '/')
    delete document.documentElement.dataset.runtime
    delete document.documentElement.dataset.surface
  })

  it('keeps the previous wallet and circular membership popover', async () => {
    const { default: App } = await import('./App.vue')
    const wrapper = mount(App)
    await flushPromises()

    expect(wrapper.text()).toContain('账户余额')
    expect(wrapper.text()).toContain('今日消费')
    expect(wrapper.text()).toContain('本月消费')
    expect(wrapper.get('.quota-ring__copy strong').text()).toBe('32%')
    expect(wrapper.get('.quota-ring__copy').text()).toBe('32%')
    expect(wrapper.get('.membership-period-label').text()).toBe('周剩余：')
    expect(wrapper.text()).toContain('本周期剩余积分')
    expect(wrapper.text()).not.toContain('本周期请求')
    expect(wrapper.find('.floating-quota-widget').exists()).toBe(false)
    wrapper.unmount()
  })

  it('expands membership details to the right and collapses first on Escape', async () => {
    const { default: App } = await import('./App.vue')
    const wrapper = mount(App)
    await flushPromises()

    await wrapper.get('.membership-card').trigger('click')
    await flushPromises()

    expect(wrapper.get('.tray-popover-shell').classes()).toContain(
      'tray-popover-shell--detail'
    )
    expect(wrapper.get('.membership-card').attributes('aria-expanded')).toBe(
      'true'
    )
    expect(wrapper.get('.tray-detail-panel').text()).toContain(
      '本周期 Token 消耗'
    )
    expect(desktopMocks.setTrayDetailPanelOpen).toHaveBeenLastCalledWith(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()

    expect(wrapper.find('.tray-detail-panel').exists()).toBe(false)
    expect(desktopMocks.hideTrayPopover).not.toHaveBeenCalled()
    expect(desktopMocks.setTrayDetailPanelOpen).toHaveBeenLastCalledWith(false)
    wrapper.unmount()
  })

  it('refreshes, opens the main panel, and dismisses with Escape', async () => {
    const { default: App } = await import('./App.vue')
    const wrapper = mount(App)
    await flushPromises()

    await wrapper.get('[aria-label="刷新积分"]').trigger('click')
    await wrapper.get('.tray-open-main').trigger('click')
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()

    expect(viewerMocks.refresh).toHaveBeenCalledTimes(1)
    expect(desktopMocks.openMainPanel).toHaveBeenCalledTimes(1)
    expect(desktopMocks.hideTrayPopover).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
