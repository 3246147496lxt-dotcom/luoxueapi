import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AuthLayout from '../AuthLayout.vue'

const authLayoutSource = readFileSync(
  resolve(process.cwd(), 'src/components/layout/AuthLayout.vue'),
  'utf8'
)

const testState = vi.hoisted(() => ({
  appStore: {
    siteName: '落雪API',
    siteLogo: '',
    cachedPublicSettings: {
      site_name: '落雪API',
      site_logo: '',
      site_subtitle: 'Subscription to API Conversion Platform',
      site_subtitle_customized: false
    },
    fetchPublicSettings: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'home.switchToLight': '切换到浅色模式',
  'home.switchToDark': '切换到深色模式',
  'auth.siteSubtitle': '订阅转 API 转换平台',
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

function mountLayout(variant?: 'default' | 'snow') {
  return mount(AuthLayout, {
    props: variant ? { variant } : {},
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
  testState.appStore.siteName = '落雪API'
  testState.appStore.siteLogo = ''
  testState.appStore.cachedPublicSettings.site_name = '落雪API'
  testState.appStore.cachedPublicSettings.site_logo = ''
  testState.appStore.cachedPublicSettings.site_subtitle =
    'Subscription to API Conversion Platform'
  testState.appStore.cachedPublicSettings.site_subtitle_customized = false
  document.documentElement.classList.remove('dark')
  localStorage.clear()
  localStorage.setItem('theme', 'light')
})

describe('AuthLayout', () => {
  it('keeps both toolbar SVG controls on the shared theme-blue token', () => {
    const toolbarRule = authLayoutSource.match(
      /\.auth-layout--snow \.auth-toolbar :deep\(\.locale-switcher-trigger\),[\s\S]*?\n\}/
    )?.[0]
    const toolbarHoverRule = authLayoutSource.match(
      /\.auth-layout--snow \.auth-toolbar :deep\(\.locale-switcher-trigger:hover\),[\s\S]*?\n\}/
    )?.[0]

    expect(toolbarRule).toContain('color: var(--auth-blue);')
    expect(toolbarHoverRule).toContain('color: var(--auth-blue);')
    expect(authLayoutSource).not.toContain('--auth-blue-deep')
    expect(authLayoutSource).toContain('useThemePreference')
    expect(authLayoutSource).not.toContain("localStorage.setItem('theme'")
    expect(authLayoutSource).not.toContain('document.documentElement.classList.toggle')
  })

  it('renders the Snow Clay composition by default without replacing slotted auth content', () => {
    const wrapper = mountLayout()

    expect(wrapper.classes()).toContain('auth-layout--snow')
    expect(wrapper.get('.auth-snow-title').text()).toBe(
      '订阅转 API 转换平台'
    )
    expect(wrapper.find('.auth-snow-media').exists()).toBe(false)
    expect(wrapper.findAll('.auth-feature-rail li').map((item) => item.text())).toEqual([
      '仪表盘',
      'API 密钥',
      '使用记录'
    ])
    expect(wrapper.get('[data-testid="auth-content"]').text()).toBe('登录')
    expect(wrapper.get('[data-testid="auth-footer"]').text()).toBe('注册')
    expect(wrapper.get('.auth-brand-mark img').attributes('src')).toBe(
      '/logo.png'
    )
    expect(wrapper.get('.auth-brand-name').text()).toBe('落雪API')
    expect(wrapper.get('.auth-brand-api').text()).toBe('API')
    const localeSwitcher = wrapper.get('locale-switcher-stub')
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
    expect(testState.appStore.fetchPublicSettings).toHaveBeenCalledOnce()
  })

  it('prefers an administrator-configured site logo over the canonical default', () => {
    testState.appStore.siteLogo = '/brand/custom-site-logo.svg'

    const wrapper = mountLayout()

    expect(wrapper.get('.auth-brand-mark img').attributes('src')).toBe(
      '/brand/custom-site-logo.svg'
    )
  })

  it('uses the same cached public brand priority as the home page', () => {
    testState.appStore.cachedPublicSettings.site_name = '缓存品牌'
    testState.appStore.cachedPublicSettings.site_logo = '/brand/cached-logo.svg'
    testState.appStore.siteName = '旧站点名'
    testState.appStore.siteLogo = '/brand/store-logo.svg'

    const wrapper = mountLayout()

    expect(wrapper.get('.auth-brand-name').text()).toBe('缓存品牌')
    expect(wrapper.find('.auth-brand-api').exists()).toBe(false)
    expect(wrapper.get('.auth-brand-mark img').attributes('src')).toBe(
      '/brand/cached-logo.svg'
    )
  })

  it('keeps an administrator-defined site subtitle unchanged even when it matches the default', () => {
    testState.appStore.cachedPublicSettings.site_subtitle =
      'Subscription to API Conversion Platform'
    testState.appStore.cachedPublicSettings.site_subtitle_customized = true

    const wrapper = mountLayout()

    expect(wrapper.get('.auth-snow-title').text()).toBe(
      'Subscription to API Conversion Platform'
    )
  })

  it('retains an explicit legacy escape hatch without making it the route default', () => {
    const wrapper = mountLayout('default')

    expect(wrapper.classes()).not.toContain('auth-layout--snow')
    expect(wrapper.find('.auth-snow-stage').exists()).toBe(false)
    expect(wrapper.find('.auth-toolbar').exists()).toBe(false)
    expect(wrapper.find('.auth-tool-button').exists()).toBe(false)
  })

  it('persists theme changes through the shared preference controller', async () => {
    const wrapper = mountLayout()
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
