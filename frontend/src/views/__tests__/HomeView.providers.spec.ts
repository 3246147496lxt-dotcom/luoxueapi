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
  'home.hero.status': 'GPT 已支持 · 其他模型暂不支持',
  'home.hero.title': '稳定接入 GPT API，按量计费',
  'home.hero.register': '注册并开始',
  'home.hero.login': '登录控制台',
  'home.hero.createKey': '创建 API 密钥',
  'home.codeExample.tabs.curl': 'cURL',
  'home.codeExample.tabs.python': 'Python',
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

describe('HomeView provider availability', () => {
  it('uses the matching Lucide fact icons from the tutorial design', () => {
    const wrapper = mountHomeView()
    const icons = wrapper.findAll('.fact-rail icon-stub')

    expect(icons.map((icon) => icon.attributes('name'))).toEqual([
      'lucideCheckCircle',
      'lucideBarChart3',
      'lucideShieldCheck'
    ])
    expect(icons.every((icon) => icon.attributes('size') === 'lg')).toBe(true)
    expect(icons.every((icon) => icon.attributes('strokewidth') === '2')).toBe(true)
  })

  it('shows GPT as the only supported provider', () => {
    const wrapper = mountHomeView()
    const supportedProviders = wrapper.findAll('[data-provider-status="supported"]')

    expect(supportedProviders).toHaveLength(1)
    expect(supportedProviders[0].attributes('data-provider')).toBe('gpt')
    expect(supportedProviders[0].text()).toContain('已支持')
  })

  it.each(['claude', 'gemini', 'antigravity'])('marks %s as readable but unsupported', (provider) => {
    const wrapper = mountHomeView()
    const providerCard = wrapper.get(`[data-provider="${provider}"]`)

    expect(providerCard.attributes('data-provider-status')).toBe('unsupported')
    expect(providerCard.text()).toContain('暂不支持')
    expect(providerCard.attributes('aria-label')).toContain('暂不支持')
    expect(providerCard.classes().some((className) => className.startsWith('opacity-'))).toBe(false)
  })
})

describe('HomeView clay composition', () => {
  it('omits the hero status and decorative image while keeping product imagery', () => {
    const wrapper = mountHomeView()
    const dashboard = wrapper.get('.dashboard-figure img')

    expect(wrapper.find('.status-badge').exists()).toBe(false)
    expect(wrapper.find('.status-dot').exists()).toBe(false)
    expect(wrapper.find('[data-testid="hero-snowflake"]').exists()).toBe(false)
    expect(wrapper.find('.hero-stage img').exists()).toBe(false)
    expect(dashboard.attributes()).toMatchObject({
      src: '/brand/home-dashboard.webp',
      width: '1600',
      height: '757',
      loading: 'lazy'
    })
  })

  it('keeps one page heading and all navigation anchor targets labeled', () => {
    const wrapper = mountHomeView()

    expect(wrapper.findAll('h1')).toHaveLength(1)
    for (const id of ['capabilities', 'steps', 'providers', 'faq']) {
      const section = wrapper.get(`#${id}`)
      const headingId = section.attributes('aria-labelledby')

      expect(headingId).toBeTruthy()
      expect(wrapper.find(`#${headingId}`).exists()).toBe(true)
    }
  })
})

describe('HomeView primary actions', () => {
  it.each([
    { registration: true, authenticated: false, path: '/register', label: '注册并开始' },
    { registration: false, authenticated: false, path: '/login', label: '登录控制台' },
    { registration: true, authenticated: true, path: '/keys', label: '创建 API 密钥' }
  ])('uses the correct CTA for the current account state', ({ registration, authenticated, path, label }) => {
    testState.appStore.cachedPublicSettings.registration_enabled = registration
    testState.authStore.isAuthenticated = authenticated
    testState.authStore.user = authenticated ? { email: 'user@example.com' } : null

    const wrapper = mountHomeView()
    const heroCta = wrapper.get('[data-testid="hero-primary-cta"]')
    const finalCta = wrapper.get('[data-testid="final-primary-cta"]')

    expect(heroCta.attributes('data-to')).toBe(path)
    expect(heroCta.text()).toContain(label)
    expect(heroCta.get('icon-stub').attributes('name')).toBe('lucideSparkles')
    expect(finalCta.attributes('data-to')).toBe(path)
  })

  it('does not repeat app-level authentication or settings initialization', () => {
    testState.appStore.publicSettingsLoaded = false
    mountHomeView()

    expect(testState.authStore.checkAuth).not.toHaveBeenCalled()
    expect(testState.appStore.fetchPublicSettings).not.toHaveBeenCalled()
  })
})

describe('HomeView code examples', () => {
  it('keeps both tabs connected to one focusable code panel', () => {
    const wrapper = mountHomeView()
    const panel = wrapper.get('#code-example-panel')

    expect(wrapper.get('#code-tab-curl').attributes('aria-controls')).toBe('code-example-panel')
    expect(wrapper.get('#code-tab-python').attributes('aria-controls')).toBe('code-example-panel')
    expect(panel.attributes('role')).toBe('tabpanel')
    expect(panel.attributes('tabindex')).toBe('0')
    expect(panel.attributes('aria-labelledby')).toBe('code-tab-curl')
  })

  it.each([
    ['https://api.example.com', 'https://api.example.com/v1/chat/completions'],
    ['https://api.example.com/', 'https://api.example.com/v1/chat/completions'],
    ['https://api.example.com/v1', 'https://api.example.com/v1/chat/completions'],
    ['https://api.example.com/v1/', 'https://api.example.com/v1/chat/completions'],
    ['  https://api.example.com/proxy/v1/  ', 'https://api.example.com/proxy/v1/chat/completions']
  ])('normalizes the configured API base URL', (configuredBase, expectedEndpoint) => {
    testState.appStore.cachedPublicSettings.api_base_url = configuredBase
    const wrapper = mountHomeView()
    const snippet = wrapper.get('[data-testid="active-code-example"]').text()

    expect(snippet).toContain(expectedEndpoint)
    expect(snippet).toContain('<YOUR_API_KEY>')
    expect(snippet).toContain('<YOUR_MODEL>')
    expect(snippet).not.toContain('/v1/v1')
  })

  it('copies the active example and shows success feedback', async () => {
    const wrapper = mountHomeView()
    const snippet = wrapper.get('[data-testid="active-code-example"]').text()
    const copyButton = wrapper.get('[data-testid="copy-code-button"]')

    expect(copyButton.get('icon-stub').attributes()).toMatchObject({
      name: 'lucideCopy',
      size: 'xs'
    })

    await copyButton.trigger('click')

    expect(testState.clipboard.copyToClipboard).toHaveBeenCalledWith(snippet, '已复制')
    expect(copyButton.text()).toContain('已复制')
  })

  it('supports arrow, Home, and End keys in the code tab list', async () => {
    const wrapper = mountHomeView()
    const curlTab = wrapper.get('#code-tab-curl')

    await curlTab.trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.get('#code-tab-python').attributes('aria-selected')).toBe('true')
    expect(wrapper.get('#code-example-panel').attributes('aria-labelledby')).toBe('code-tab-python')
    expect(wrapper.get('[data-testid="active-code-example"]').text()).toContain('from openai import OpenAI')

    await wrapper.get('#code-tab-python').trigger('keydown', { key: 'Home' })
    expect(wrapper.get('#code-tab-curl').attributes('aria-selected')).toBe('true')

    await wrapper.get('#code-tab-curl').trigger('keydown', { key: 'End' })
    expect(wrapper.get('#code-tab-python').attributes('aria-selected')).toBe('true')

    await wrapper.get('#code-tab-python').trigger('keydown', { key: 'ArrowLeft' })
    expect(wrapper.get('#code-tab-curl').attributes('aria-selected')).toBe('true')
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
