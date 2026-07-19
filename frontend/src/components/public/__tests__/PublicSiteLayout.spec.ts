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
  testState.appStore.cachedPublicSettings.public_model_catalog_enabled = false
  testState.appStore.backendModeEnabled = false
  document.documentElement.classList.remove('dark')
  localStorage.clear()
})

describe('PublicSiteLayout', () => {
  it('uses cross-page home anchors and hides the catalog entry while disabled', () => {
    const wrapper = mountLayout()
    const hrefs = wrapper.findAll('.public-site-desktop-nav > a')
      .map((link) => link.attributes('href'))

    expect(hrefs).toContain('/home#capabilities')
    expect(hrefs).toContain('/home#steps')
    expect(wrapper.find('[data-to="/models.html"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="content"]').text()).toBe('Content')
  })

  it('shows the catalog in desktop, mobile, and footer navigation when enabled', async () => {
    testState.appStore.cachedPublicSettings.public_model_catalog_enabled = true
    const wrapper = mountLayout()

    expect(wrapper.findAll('[data-to="/models.html"]')).toHaveLength(2)
    await wrapper.get('.public-site-mobile-menu-button').trigger('click')
    expect(wrapper.findAll('[data-to="/models.html"]')).toHaveLength(3)
    expect(wrapper.get('.public-site-mobile-menu-button').attributes('aria-expanded')).toBe('true')
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
    const themeButton = wrapper.findAll('.public-site-icon-button')[0]

    await themeButton.trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(wrapper.classes()).toContain('public-site-page--dark')
  })
})
