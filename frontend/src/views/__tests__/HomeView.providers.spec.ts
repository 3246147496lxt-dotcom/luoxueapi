import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import HomeView from '../HomeView.vue'

const testState = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as null | { email: string },
    checkAuth: vi.fn()
  },
  appStore: {
    cachedPublicSettings: {
      home_content: '',
      site_name: '落雪API',
      site_logo: '',
      site_subtitle: '你的全模型 API 中转站',
      doc_url: 'https://docs.example.com/tutorial-docs/',
      api_base_url: 'https://api.example.com',
      registration_enabled: true
    },
    siteName: '落雪API',
    siteLogo: '',
    docUrl: '',
    apiBaseUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn()
  },
  clipboard: {
    copyToClipboard: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'home.nav.ariaLabel': '首页导航',
  'home.hero.status': '全球模型直连',
  'home.hero.title': '专用于生产环境的，统一大模型网关',
  'home.hero.register': '注册并开始',
  'home.hero.login': '登录控制台',
  'home.hero.createKey': '获取 API Key',
  'home.codeExample.tabs.curl': 'cURL',
  'home.codeExample.tabs.python': 'Python',
  'home.codeExample.description': '接口地址来自当前站点配置；API 密钥和模型名以控制台显示为准。',
  'home.codeExample.copy': '复制',
  'home.codeExample.copied': '已复制',
  'home.codeExample.copyFailed': '复制失败，请手动复制',
  'home.codeExample.copyAria': '复制 {language} 示例代码',
  'home.codeExample.copiedAria': '{language} 示例代码已复制',
  'home.providers.title': '模型支持情况',
  'home.providers.description': 'GPT 已支持，其他模型暂不支持。',
  'home.providers.supported': '已支持',
  'home.providers.unsupported': '暂不支持',
  'home.providers.note': '具体可用型号和倍率以控制台实时列表为准。',
  'home.providers.claude': 'Claude',
  'home.providers.gpt': 'GPT',
  'home.providers.gemini': 'Gemini',
  'home.providers.antigravity': 'Antigravity',
  'home.cta.button': '开始接入 GPT'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => {
        const message = messages[key] ?? key
        return Object.entries(params ?? {}).reduce(
          (result, [name, value]) => result.replace(`{${name}}`, value),
          message
        )
      }
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => testState.authStore,
  useAppStore: () => testState.appStore
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: testState.clipboard.copyToClipboard
  })
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

function mountHomeView() {
  return mount(HomeView, {
    global: {
      stubs: {
        Icon: true,
        LocaleSwitcher: true,
        PlatformIcon: true,
        RouterLink: RouterLinkStub
      }
    }
  })
}

beforeEach(() => {
  testState.authStore.isAuthenticated = false
  testState.authStore.isAdmin = false
  testState.authStore.user = null
  testState.authStore.checkAuth.mockReset()
  Object.assign(testState.appStore.cachedPublicSettings, {
    home_content: '',
    site_name: '落雪API',
    site_logo: '',
    site_subtitle: '你的全模型 API 中转站',
    doc_url: 'https://docs.example.com/tutorial-docs/',
    api_base_url: 'https://api.example.com',
    registration_enabled: true
  })
  testState.appStore.publicSettingsLoaded = true
  testState.appStore.fetchPublicSettings.mockReset()
  testState.clipboard.copyToClipboard.mockReset()
  testState.clipboard.copyToClipboard.mockResolvedValue(true)
  document.documentElement.classList.remove('dark')
})

describe('HomeView clay composition', () => {
  it('keeps the hero visual self-contained without decorative image assets', () => {
    const wrapper = mountHomeView()

    expect(wrapper.find('.status-badge').exists()).toBe(false)
    expect(wrapper.find('.status-dot').exists()).toBe(true)
    expect(wrapper.find('[data-testid="hero-snowflake"]').exists()).toBe(false)
    expect(wrapper.find('.hero-stage img').exists()).toBe(false)
    expect(wrapper.find('.hero-visual-frame').exists()).toBe(false)
    expect(wrapper.get('.bezier-svg').attributes('viewBox')).toBe('0 0 1320 340')
    expect(wrapper.findAll('.route-base')).toHaveLength(7)
    expect(wrapper.find('.reference-background-wash').exists()).toBe(true)
    expect(wrapper.find('.blue-mesh').exists()).toBe(true)
    expect(wrapper.find('.pixel-grid').exists()).toBe(true)
  })

  it('renders only the full-screen hero on the default route', () => {
    const wrapper = mountHomeView()

    expect(wrapper.findAll('h1')).toHaveLength(1)
    expect(wrapper.findAll('main > section')).toHaveLength(1)
    const hero = wrapper.get('main > section')
    expect(hero.classes()).toContain('reference-hero-shell')
    expect(hero.attributes('id')).toBe('steps')
    for (const selector of ['.fact-section', '#capabilities', '#providers', '#faq', '.final-cta-section']) {
      expect(wrapper.find(selector).exists()).toBe(false)
    }
  })
})

describe('HomeView primary actions', () => {
  it.each([
    { registration: true, authenticated: false, path: '/register', label: '获取 API Key' },
    { registration: false, authenticated: false, path: '/login', label: '获取 API Key' },
    { registration: true, authenticated: true, path: '/keys', label: '获取 API Key' }
  ])('uses the correct CTA for the current account state', ({ registration, authenticated, path, label }) => {
    testState.appStore.cachedPublicSettings.registration_enabled = registration
    testState.authStore.isAuthenticated = authenticated
    testState.authStore.user = authenticated ? { email: 'user@example.com' } : null

    const wrapper = mountHomeView()
    const heroCta = wrapper.get('[data-testid="hero-primary-cta"]')

    expect(heroCta.attributes('data-to')).toBe(path)
    expect(heroCta.text()).toContain(label)
    expect(heroCta.get('icon-stub').attributes('name')).toBe('key')
  })

  it('does not repeat app-level authentication or settings initialization', () => {
    testState.appStore.publicSettingsLoaded = false
    mountHomeView()

    expect(testState.authStore.checkAuth).not.toHaveBeenCalled()
    expect(testState.appStore.fetchPublicSettings).not.toHaveBeenCalled()
  })
})

describe('HomeView API endpoint control', () => {
  it.each([
    ['https://api.example.com', 'https://api.example.com/v1'],
    ['https://api.example.com/', 'https://api.example.com/v1'],
    ['https://api.example.com/v1', 'https://api.example.com/v1'],
    ['https://api.example.com/v1/', 'https://api.example.com/v1'],
    ['  https://api.example.com/proxy/v1/  ', 'https://api.example.com/proxy/v1']
  ])('normalizes the configured API base URL', (configuredBase, expectedEndpoint) => {
    testState.appStore.cachedPublicSettings.api_base_url = configuredBase
    const wrapper = mountHomeView()
    const endpoint = wrapper.get('.api-url').text()

    expect(endpoint).toBe(expectedEndpoint)
    expect(endpoint).not.toContain('/v1/v1')
  })

  it('copies the API endpoint and exposes success feedback', async () => {
    const wrapper = mountHomeView()
    const copyButton = wrapper.get('[data-testid="copy-api-url-button"]')

    expect(copyButton.attributes('aria-label')).toBe('复制 API 地址')

    await copyButton.trigger('click')

    expect(testState.clipboard.copyToClipboard).toHaveBeenCalledWith(
      'https://api.example.com/v1',
      'API 地址已复制'
    )
  })
})

describe('HomeView custom content compatibility', () => {
  it('treats whitespace-only custom content as empty', () => {
    testState.appStore.cachedPublicSettings.home_content = '   '
    const wrapper = mountHomeView()

    expect(wrapper.find('.home-page').exists()).toBe(true)
  })

  it('renders a trimmed URL in an iframe without the default page', () => {
    testState.appStore.cachedPublicSettings.home_content = '  https://example.com/custom-home  '
    const wrapper = mountHomeView()

    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/custom-home')
    expect(wrapper.find('.home-page').exists()).toBe(false)
  })

  it('renders administrator HTML without an iframe or the default page', () => {
    testState.appStore.cachedPublicSettings.home_content = '<div data-testid="custom-home">Custom</div>'
    const wrapper = mountHomeView()

    expect(wrapper.get('[data-testid="custom-home"]').text()).toBe('Custom')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('.home-page').exists()).toBe(false)
  })
})
