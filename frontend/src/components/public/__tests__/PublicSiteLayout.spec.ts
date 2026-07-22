import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import PublicSiteLayout from '../PublicSiteLayout.vue'

const testState = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: false,
    isAdmin: false
  },
  appStore: {
    cachedPublicSettings: {
      site_name: '落雪API',
      site_logo: '',
      doc_url: '/tutorial-docs/',
      public_model_catalog_enabled: false
    },
    siteName: '落雪API',
    siteLogo: '',
    docUrl: '',
    backendModeEnabled: false
  }
}))

const messages: Record<string, string> = {
  'home.nav.ariaLabel': '首页导航',
  'home.nav.capabilities': '产品能力',
  'home.nav.steps': '接入步骤',
  'home.nav.providers': '模型状态',
  'home.nav.faq': '常见问题',
  'home.nav.tutorial': '使用教程',
  'home.nav.openMenu': '打开菜单',
  'home.nav.closeMenu': '关闭菜单',
  'home.switchToLight': '切换到浅色模式',
  'home.switchToDark': '切换到深色模式',
  'home.dashboard': '控制台',
  'home.login': '登录',
  'home.footer.allRightsReserved': '保留所有权利。',
  'home.footer.ariaLabel': '页脚导航',
  'home.footer.tutorial': '使用教程',
  'home.footer.apiDocs': 'API 文档',
  'home.footer.channelStatus': '渠道状态',
  'modelCatalog.navLabel': '模型广场'
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
  useAuthStore: () => testState.authStore,
  useAppStore: () => testState.appStore
}))

const RouterLinkStub = defineComponent({
  inheritAttrs: false,
  props: {
    to: {
      type: [String, Object],
      required: true
    }
  },
  template: '<a v-bind="$attrs" :data-to="typeof to === \'string\' ? to : JSON.stringify(to)"><slot /></a>'
})

function mountLayout(page: 'home' | 'models' = 'home') {
  return mount(PublicSiteLayout, {
    props: { page },
    slots: { default: '<main data-testid="content">Content</main>' },
    global: {
      stubs: {
        LocaleSwitcher: true,
        Icon: true,
        RouterLink: RouterLinkStub
      }
    }
  })
}

beforeEach(() => {
  testState.authStore.isAuthenticated = false
  testState.authStore.isAdmin = false
  testState.appStore.cachedPublicSettings.site_name = '落雪API'
  testState.appStore.siteName = '落雪API'
  testState.appStore.cachedPublicSettings.site_logo = ''
  testState.appStore.siteLogo = ''
  testState.appStore.cachedPublicSettings.doc_url = '/tutorial-docs/'
  testState.appStore.docUrl = ''
  testState.appStore.cachedPublicSettings.public_model_catalog_enabled = false
  testState.appStore.backendModeEnabled = false
  document.documentElement.classList.remove('dark')
  localStorage.clear()
  Object.defineProperty(window, 'innerWidth', { value: 1280, writable: true, configurable: true })
})

describe('PublicSiteLayout', () => {
  it('uses the shared docs resolver while keeping the tutorial fallback available', () => {
    testState.appStore.cachedPublicSettings.doc_url = '/tutorial-docs/?source=public#old'
    const wrapper = mountLayout()

    expect(wrapper.get('[data-testid="public-api-docs-link"]').attributes('href'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/?source=public#old')
    expect(wrapper.findAll('a').filter((link) => link.text() === '使用教程')
      .every((link) => (
        link.attributes('href') === 'http://127.0.0.1:4179/tutorial-docs/?source=public#quick-start'
      )))
      .toBe(true)

    wrapper.unmount()
    testState.appStore.cachedPublicSettings.doc_url = ''
    const fallbackWrapper = mountLayout()

    expect(fallbackWrapper.find('[data-testid="public-api-docs-link"]').exists()).toBe(false)
    const tutorialLinks = fallbackWrapper.findAll('a').filter((link) => link.text() === '使用教程')
    expect(tutorialLinks.length).toBeGreaterThan(0)
    expect(tutorialLinks.every((link) => (
      link.attributes('href') === 'http://127.0.0.1:4179/tutorial-docs/#quick-start'
    ))).toBe(true)
  })

  it('uses the same Snow Clay shell while preserving each public page identity', () => {
    const homeWrapper = mountLayout('home')

    expect(homeWrapper.classes()).toContain('public-site-page--snow')
    expect(homeWrapper.classes()).toContain('public-site-page--home')
    expect(homeWrapper.classes()).not.toContain('public-site-page--models')
    expect(homeWrapper.get('[data-testid="public-site-brand-logo"]').attributes('src'))
      .toBe('/brand/luoxue-snowpuff-extracted.svg')

    homeWrapper.unmount()
    const modelsWrapper = mountLayout('models')

    expect(modelsWrapper.classes()).toContain('public-site-page--snow')
    expect(modelsWrapper.classes()).toContain('public-site-page--models')
    expect(modelsWrapper.classes()).not.toContain('public-site-page--home')
    expect(modelsWrapper.get('[data-testid="public-site-brand-logo"]').attributes('src'))
      .toBe('/brand/luoxue-snowpuff-extracted.svg')
    expect(modelsWrapper.find('.public-site-footer-mark').exists()).toBe(true)
    const localeSwitcher = modelsWrapper.get('.public-site-desktop-action locale-switcher-stub')
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
    expect(modelsWrapper.get('[data-testid="theme-toggle"] icon-stub').attributes('name'))
      .toBe('lucideMoon')
  })

  it('uses the configured site logo in both the header and footer', () => {
    testState.appStore.cachedPublicSettings.site_logo = '/brand/custom-site-logo.svg'
    testState.appStore.siteLogo = '/brand/store-logo.svg'

    const wrapper = mountLayout()

    expect(wrapper.get('[data-testid="public-site-brand-logo"]').attributes('src'))
      .toBe('/brand/custom-site-logo.svg')
    expect(wrapper.get('.public-site-footer-mark img').attributes('src'))
      .toBe('/brand/custom-site-logo.svg')
  })

  it('renders a trailing API brand token as the ice-blue wordmark accent', () => {
    const wrapper = mountLayout()

    expect(wrapper.get('.public-site-brand-name').text()).toBe('落雪API')
    expect(wrapper.get('.public-site-brand-api').text()).toBe('API')

    testState.appStore.cachedPublicSettings.site_name = '雪落开发者平台'
    const customWrapper = mountLayout()

    expect(customWrapper.get('.public-site-brand-name').text()).toBe('雪落开发者平台')
    expect(customWrapper.find('.public-site-brand-api').exists()).toBe(false)
  })

  it('uses cross-page home anchors and hides the catalog entry while disabled', () => {
    const wrapper = mountLayout()
    const hrefs = wrapper.findAll('.public-site-desktop-nav > a')
      .map((link) => link.attributes('href'))

    expect(hrefs).toContain('/home#capabilities')
    expect(hrefs).toContain('/home#steps')
    expect(hrefs.slice(0, 2)).toEqual(['/home#steps', '/home#capabilities'])
    expect(wrapper.find('[data-to="/models.html"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="content"]').text()).toBe('Content')
    expect(wrapper.get('.public-site-footer nav').attributes('aria-label')).toBe('页脚导航')
  })

  it('shows the catalog in desktop, mobile, and footer navigation when enabled', async () => {
    testState.appStore.cachedPublicSettings.public_model_catalog_enabled = true
    const wrapper = mountLayout()

    expect(wrapper.findAll('[data-to="/models.html"]')).toHaveLength(2)
    await wrapper.get('[data-testid="mobile-menu-toggle"]').trigger('click')
    expect(wrapper.findAll('[data-to="/models.html"]')).toHaveLength(3)
    expect(wrapper.get('[data-testid="mobile-menu-toggle"]').attributes('aria-expanded')).toBe('true')
  })

  it('provides a locale switcher in the compact home menu', async () => {
    const wrapper = mountLayout('home')

    expect(wrapper.find('[data-testid="mobile-locale-switcher"]').exists()).toBe(false)

    await wrapper.get('[data-testid="mobile-menu-toggle"]').trigger('click')

    const localeSwitcher = wrapper.get('[data-testid="mobile-locale-switcher"]')
    expect(localeSwitcher.element.closest('.public-site-mobile-footer')).not.toBeNull()
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
  })

  it('closes an open mobile menu after crossing into the desktop breakpoint', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 768, writable: true, configurable: true })
    const wrapper = mountLayout('home')

    await wrapper.get('[data-testid="mobile-menu-toggle"]').trigger('click')
    expect(wrapper.find('#public-site-mobile-menu').exists()).toBe(true)

    window.innerWidth = 1024
    window.dispatchEvent(new Event('resize'))
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-testid="mobile-menu-toggle"]').attributes('aria-expanded'))
      .toBe('false')
    await vi.waitFor(() => {
      expect(wrapper.find('#public-site-mobile-menu').exists()).toBe(false)
    })
  })

  it('hides every catalog entry in backend mode even when the feature is enabled', () => {
    testState.appStore.cachedPublicSettings.public_model_catalog_enabled = true
    testState.appStore.backendModeEnabled = true

    const wrapper = mountLayout('models')

    expect(wrapper.find('[data-to="/models.html"]').exists()).toBe(false)
  })

  it('keeps the current models page visible and marks its primary navigation entry', () => {
    const wrapper = mountLayout('models')
    const currentLink = wrapper.findAll('[data-to="/models.html"]')
      .find((link) => link.attributes('aria-current') === 'page')

    expect(currentLink).toBeTruthy()
  })

  it('persists theme changes through the shared header', async () => {
    const wrapper = mountLayout()
    const themeButton = wrapper.get('[data-testid="theme-toggle"]')

    const localeSwitcher = wrapper.get('.public-site-desktop-action locale-switcher-stub')
    expect(localeSwitcher.attributes('icon-variant') ?? localeSwitcher.attributes('iconvariant'))
      .toBe('lucide')
    const lightThemeIcon = themeButton.get('icon-stub')
    expect(lightThemeIcon.attributes('name')).toBe('lucideMoon')
    expect(lightThemeIcon.attributes('stroke-width') ?? lightThemeIcon.attributes('strokewidth'))
      .toBe('2')

    await themeButton.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(wrapper.classes()).toContain('public-site-page--dark')
    expect(themeButton.get('icon-stub').attributes('name')).toBe('lucideSun')
  })
})
