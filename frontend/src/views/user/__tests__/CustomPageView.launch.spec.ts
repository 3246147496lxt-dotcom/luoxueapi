import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  requestLaunch: vi.fn(),
  showError: vi.fn(),
  menuItems: [{
    id: 'secure-page',
    label: 'Secure page',
    icon_svg: '',
    url: 'https://external.example/page',
    auth_mode: 'exchange_code' as const,
    visibility: 'user' as const,
    sort_order: 0,
  }],
}))

vi.mock('@/api/customPages', () => ({
  requestCustomPageLaunch: mocks.requestLaunch,
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    cachedPublicSettings: { custom_menu_items: mocks.menuItems },
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
    showError: mocks.showError,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isAdmin: false }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: 'secure-page' } }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: ref('zh-CN'),
    }),
  }
})

import CustomPageView from '../CustomPageView.vue'

describe('CustomPageView exchange-code launch', () => {
  beforeEach(() => {
    mocks.requestLaunch.mockReset()
    mocks.showError.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('requests independent launch URLs for iframe and new tab', async () => {
    mocks.requestLaunch
      .mockResolvedValueOnce({
        launch_url: 'https://external.example/page?s2a_client_id=client&s2a_launch_code=iframe-once',
        auth_mode: 'exchange_code',
        expires_in: 60,
      })
      .mockResolvedValueOnce({
        launch_url: 'https://external.example/page?s2a_client_id=client&s2a_launch_code=tab-once',
        auth_mode: 'exchange_code',
        expires_in: 60,
      })

    const popupDocument = document.implementation.createHTMLDocument('pending')
    const replace = vi.fn()
    const close = vi.fn()
    const popup = {
      opener: window,
      document: popupDocument,
      location: { replace },
      close,
    }
    vi.spyOn(window, 'open').mockReturnValue(popup as unknown as Window)

    const wrapper = mount(CustomPageView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
        },
      },
    })
    await flushPromises()

    expect(mocks.requestLaunch).toHaveBeenNthCalledWith(
      1,
      'secure-page',
      expect.objectContaining({ ui_mode: 'embedded', lang: 'zh-CN' }),
    )
    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('src')).toContain('s2a_launch_code=iframe-once')
    expect(iframe.attributes('referrerpolicy')).toBe('no-referrer')

    await wrapper.get('.custom-open-fab').trigger('click')
    await flushPromises()

    expect(mocks.requestLaunch).toHaveBeenNthCalledWith(
      2,
      'secure-page',
      expect.objectContaining({ ui_mode: 'new_tab', lang: 'zh-CN' }),
    )
    expect(replace).toHaveBeenCalledWith(
      expect.stringContaining('s2a_launch_code=tab-once'),
    )
    expect(popup.opener).toBeNull()
    expect(close).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
