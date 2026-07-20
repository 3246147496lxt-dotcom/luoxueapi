import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AuthLayout from '../AuthLayout.vue'

const testState = vi.hoisted(() => ({
  appStore: {
    siteName: '落雪API',
    siteLogo: '',
    cachedPublicSettings: {
      site_subtitle: 'Subscription to API Conversion Platform'
    },
    fetchPublicSettings: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'home.switchToLight': '切换到浅色模式',
  'home.switchToDark': '切换到深色模式',
  'nav.dashboard': '仪表盘',
  'nav.apiKeys': 'API 密钥',
  'nav.usage': '使用记录',
  'nav.availableChannels': '可用渠道',
  'nav.subscriptions': '订阅'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key
    })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => testState.appStore
}))

function mountLayout(variant: 'default' | 'snow' = 'default') {
  return mount(AuthLayout, {
    props: { variant },
    slots: {
      default: '<h1 data-testid="auth-content">登录</h1>',
      footer: '<p data-testid="auth-footer">注册</p>'
    },
    global: {
      stubs: {
        LocaleSwitcher: true,
        Icon: true
      }
    }
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  document.documentElement.classList.remove('dark')
  localStorage.clear()
})

describe('AuthLayout', () => {
  it('renders the snow login composition without replacing slotted auth content', () => {
    const wrapper = mountLayout('snow')

    expect(wrapper.classes()).toContain('auth-layout--snow')
    expect(wrapper.get('.auth-snow-title').text()).toBe(
      'Subscription to API Conversion Platform'
    )
    expect(wrapper.get('.auth-snow-media img').attributes('src')).toBe(
      '/brand/luoxue-snowflake-3d.png'
    )
    expect(wrapper.findAll('.auth-feature-rail li').map((item) => item.text())).toEqual([
      '仪表盘',
      'API 密钥',
      '使用记录'
    ])
    expect(wrapper.get('[data-testid="auth-content"]').text()).toBe('登录')
    expect(wrapper.get('[data-testid="auth-footer"]').text()).toBe('注册')
    const localeSwitcher = wrapper.get('locale-switcher-stub')
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
    expect(testState.appStore.fetchPublicSettings).toHaveBeenCalledOnce()
  })

  it('keeps the shared default auth layout free of snow-only controls', () => {
    const wrapper = mountLayout()

    expect(wrapper.classes()).not.toContain('auth-layout--snow')
    expect(wrapper.find('.auth-snow-stage').exists()).toBe(false)
    expect(wrapper.find('.auth-toolbar').exists()).toBe(false)
    expect(wrapper.find('.auth-tool-button').exists()).toBe(false)
  })

  it('persists theme changes and applies the local dark-theme surface class', async () => {
    const wrapper = mountLayout('snow')
    const themeButton = wrapper.get('.auth-tool-button')

    expect(themeButton.attributes('aria-label')).toBe('切换到深色模式')
    expect(wrapper.classes()).not.toContain('auth-layout--dark')

    await themeButton.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(wrapper.classes()).toContain('auth-layout--dark')
    expect(themeButton.attributes('aria-label')).toBe('切换到浅色模式')
  })
})
