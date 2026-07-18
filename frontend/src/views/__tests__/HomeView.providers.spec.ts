import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import HomeView from '../HomeView.vue'

const testState = vi.hoisted(() => ({
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null,
    checkAuth: vi.fn()
  },
  appStore: {
    cachedPublicSettings: {
      home_content: '',
      site_name: '落雪API',
      site_logo: '',
      site_subtitle: '测试首页',
      doc_url: ''
    },
    siteName: '落雪API',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn()
  }
}))

const messages: Record<string, string> = {
  'home.providers.title': 'AI 模型支持情况',
  'home.providers.description': 'GPT 已支持，其他模型暂未开放',
  'home.providers.supported': '已支持',
  'home.providers.unsupported': '暂不支持',
  'home.providers.soon': '即将推出',
  'home.providers.claude': 'Claude',
  'home.providers.gpt': 'GPT',
  'home.providers.gemini': 'Gemini',
  'home.providers.antigravity': 'Antigravity',
  'home.providers.more': '更多'
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

function mountHomeView() {
  return mount(HomeView, {
    global: {
      stubs: {
        Icon: true,
        LocaleSwitcher: true,
        RouterLink: {
          template: '<a><slot /></a>'
        }
      }
    }
  })
}

describe('HomeView provider availability', () => {
  it('shows GPT as the only supported provider', () => {
    const wrapper = mountHomeView()
    const supportedProviders = wrapper.findAll('[data-provider-status="supported"]')

    expect(supportedProviders).toHaveLength(1)
    expect(supportedProviders[0].attributes('data-provider')).toBe('gpt')
    expect(supportedProviders[0].text()).toContain('已支持')
  })

  it.each(['claude', 'gemini', 'antigravity'])('marks %s as unsupported', (provider) => {
    const wrapper = mountHomeView()
    const providerCard = wrapper.get(`[data-provider="${provider}"]`)

    expect(providerCard.attributes('data-provider-status')).toBe('unsupported')
    expect(providerCard.text()).toContain('暂不支持')
    expect(providerCard.classes()).toContain('opacity-60')
  })
})
