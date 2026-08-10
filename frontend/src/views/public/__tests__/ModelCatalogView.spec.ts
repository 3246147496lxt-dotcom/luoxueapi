import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ModelCatalogView from '../ModelCatalogView.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import type { PublicModelCatalogItem, PublicModelCatalogResponse } from '@/api/catalog'

const testState = vi.hoisted(() => ({
  getCatalog: vi.fn(),
  copyToClipboard: vi.fn(),
  authStore: {
    isAuthenticated: false,
    isAdmin: false
  },
  appStore: {
    cachedPublicSettings: {
      site_name: '落雪API',
      doc_url: '/tutorial-docs/',
      registration_enabled: true,
      available_channels_enabled: true,
      public_model_catalog_enabled: true
    },
    siteName: '落雪API',
    docUrl: ''
  }
}))

const messages: Record<string, string> = {
  'modelCatalog.title': '模型广场',
  'modelCatalog.description': '公开查看模型',
  'modelCatalog.workspaceTitle': '模型中心',
  'modelCatalog.workspaceDescription': '查看当前可用模型',
  'modelCatalog.publicPriceNote': '公开标准价格说明',
  'modelCatalog.modelCount': '{count} 个模型',
  'modelCatalog.pricingUpdatedAt': '价格更新于 {time}',
  'modelCatalog.searchLabel': '搜索模型',
  'modelCatalog.searchPlaceholder': '搜索模型名称、模型 ID 或能力',
  'modelCatalog.filters.ariaLabel': '模型筛选',
  'modelCatalog.filters.provider': '厂商',
  'modelCatalog.filters.category': '类型',
  'modelCatalog.filters.all': '全部',
  'modelCatalog.resultsTitle': '公开模型',
  'modelCatalog.resultCount': '显示 {count} 个结果',
  'modelCatalog.loading': '正在加载模型',
  'modelCatalog.featured': '推荐',
  'modelCatalog.copy': '复制',
  'modelCatalog.copied': '已复制',
  'modelCatalog.copySuccess': '模型 ID 已复制',
  'modelCatalog.copyModelAria': '复制模型 ID：{model}',
  'modelCatalog.contextWindow': '上下文窗口',
  'modelCatalog.maxOutput': '最大输出',
  'modelCatalog.capabilities': '模型能力',
  'modelCatalog.providers.openai': 'OpenAI',
  'modelCatalog.providers.anthropic': 'Anthropic',
  'modelCatalog.categories.chat': '文本对话',
  'modelCatalog.categories.reasoning': '推理',
  'modelCatalog.capabilityLabels.vision': '视觉理解',
  'modelCatalog.capabilityLabels.reasoning': '深度推理',
  'modelCatalog.pricing.publicLabel': '公开标准价',
  'modelCatalog.pricing.details': '价格详情',
  'modelCatalog.pricing.dialogTitle': '公开价格详情',
  'modelCatalog.pricing.dialogDescription': '价格说明',
  'modelCatalog.pricing.billingMode': '计费方式',
  'modelCatalog.pricing.billingModes.token': '按 Token 计费',
  'modelCatalog.pricing.input': '输入',
  'modelCatalog.pricing.output': '输出',
  'modelCatalog.pricing.cacheWrite': '缓存写入',
  'modelCatalog.pricing.cacheWrite1h': '缓存写入（1 小时）',
  'modelCatalog.pricing.cacheRead': '缓存读取',
  'modelCatalog.pricing.priorityInput': 'Priority 输入',
  'modelCatalog.pricing.priorityOutput': 'Priority 输出',
  'modelCatalog.pricing.priorityCacheWrite': 'Priority 缓存写入',
  'modelCatalog.pricing.priorityCacheRead': 'Priority 缓存读取',
  'modelCatalog.pricing.imageInput': '图像输入',
  'modelCatalog.pricing.imageOutput': '图像输出',
  'modelCatalog.pricing.request': '单次请求',
  'modelCatalog.pricing.image': '单张图片',
  'modelCatalog.pricing.perMillionTokens': '/ 百万 Token',
  'modelCatalog.pricing.perRequest': '/ 次',
  'modelCatalog.pricing.perImage': '/ 张',
  'modelCatalog.pricing.intervalTitle': '区间价格',
  'modelCatalog.pricing.longContext': '长上下文价格',
  'modelCatalog.pricing.peakRate': '高峰时段 {start}–{end}（{timezone}）按 {multiplier} 倍计费。',
  'modelCatalog.pricing.unknown': '暂未提供价格',
  'modelCatalog.noResults.title': '没有匹配的模型',
  'modelCatalog.noResults.description': '换一个关键词',
  'modelCatalog.noResults.clear': '清除筛选',
  'modelCatalog.empty.title': '暂未上架公开模型',
  'modelCatalog.empty.description': '发布后显示',
  'modelCatalog.error.title': '模型目录加载失败',
  'modelCatalog.error.description': '请稍后重试',
  'modelCatalog.error.retry': '重新加载',
  'modelCatalog.unavailable.title': '模型广场暂未开放',
  'modelCatalog.unavailable.description': '当前站点尚未启用',
  'modelCatalog.cta.title': '进入控制台完成接入',
  'modelCatalog.cta.description': '创建 API 密钥',
  'modelCatalog.cta.register': '注册并使用',
  'modelCatalog.cta.login': '登录控制台',
  'modelCatalog.cta.dashboard': '进入控制台',
  'modelCatalog.cta.availableChannels': '查看我的可用渠道',
  'modelCatalog.cta.tutorial': '查看使用教程',
  'modelCatalog.meta.title': '模型广场',
  'modelCatalog.meta.description': '模型广场说明',
  'common.close': '关闭'
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const locale = ref('zh')
  return {
    ...actual,
    useI18n: () => ({
      locale,
      te: (key: string) => key in messages,
      t: (key: string, params?: Record<string, string | number>) => {
        const value = messages[key] ?? key
        return Object.entries(params ?? {}).reduce(
          (result, [name, replacement]) => result.replace(`{${name}}`, String(replacement)),
          value
        )
      }
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => testState.authStore,
  useAppStore: () => testState.appStore
}))

vi.mock('@/api/catalog', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/catalog')>()
  return {
    ...actual,
    getPublicModelCatalog: (...args: unknown[]) => testState.getCatalog(...args)
  }
})

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: testState.copyToClipboard
  })
}))

const PublicLayoutStub = defineComponent({
  props: ['page'],
  template: '<div data-testid="public-site-layout" :data-page="page"><slot /></div>'
})

const AppLayoutStub = defineComponent({
  template: '<div data-testid="app-layout"><slot /></div>'
})

function model(overrides: Partial<PublicModelCatalogItem> = {}): PublicModelCatalogItem {
  return {
    slug: 'gpt-4o',
    model: 'gpt-4o',
    display_name: 'GPT-4o',
    summary: '兼顾速度与质量的多模态模型。',
    provider: 'openai',
    logo_key: 'openai',
    category: 'chat',
    tags: ['vision'],
    capabilities: ['vision'],
    context_window: 128_000,
    max_output_tokens: 16_384,
    featured: true,
    pricing: {
      label: '公开标准价',
      billing_mode: 'token',
      currency: 'CREDIT',
      unit: 'per_token',
      input_price: 0,
      output_price: 0.00001,
      cache_write_price: null,
      cache_write_1h_price: null,
      cache_read_price: 0.000001,
      priority_input_price: null,
      priority_output_price: null,
      priority_cache_write_price: null,
      priority_cache_read_price: null,
      image_input_price: null,
      image_output_price: null,
      per_request_price: null,
      intervals: [],
      peak_rate: {
        enabled: true,
        start: '18:00',
        end: '23:00',
        multiplier: 1.5
      }
    },
    ...overrides
  }
}

function catalogResponse(items: PublicModelCatalogItem[]): PublicModelCatalogResponse {
  return {
    items,
    server_timezone: 'Asia/Shanghai',
    pricing_updated_at: '2026-07-18T13:00:00Z'
  }
}

async function mountCatalog(initialPath = '/models.html', props: { embedded?: boolean } = {}) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/models.html', component: ModelCatalogView },
      { path: '/models', component: ModelCatalogView }
    ]
  })
  await router.push(initialPath)
  await router.isReady()
  const wrapper = mount(ModelCatalogView, {
    props,
    global: {
      plugins: [router],
      stubs: {
        PublicSiteLayout: PublicLayoutStub,
        AppLayout: AppLayoutStub,
        ModelIcon: true,
        Icon: true,
        RouterLink: true
      }
    }
  })
  await flushPromises()
  return { wrapper, router }
}

beforeEach(() => {
  testState.getCatalog.mockReset()
  testState.copyToClipboard.mockReset()
  testState.copyToClipboard.mockResolvedValue(true)
  testState.authStore.isAuthenticated = false
  testState.authStore.isAdmin = false
  testState.appStore.cachedPublicSettings.registration_enabled = true
  document.head.querySelectorAll('meta[property^="og:"], link[rel="canonical"]').forEach((node) => node.remove())
})

describe('ModelCatalogView', () => {
  it('uses the public shell by default and the Work shell for its embedded variant', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([]))

    const publicCatalog = await mountCatalog()
    expect(publicCatalog.wrapper.get('[data-testid="public-site-layout"]').attributes('data-page')).toBe('models')
    expect(publicCatalog.wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    publicCatalog.wrapper.unmount()

    const workCatalog = await mountCatalog('/models', { embedded: true })
    const workLayout = workCatalog.wrapper.get('[data-testid="app-layout"]')
    expect(workCatalog.wrapper.find('[data-testid="public-site-layout"]').exists()).toBe(false)
    expect(workLayout.classes()).toContain('model-catalog-page--embedded')
    expect(workLayout.get('#catalog-title').text()).toBe('模型中心')
    expect(workLayout.text()).toContain('查看当前可用模型')
    expect(workLayout.text()).not.toContain('公开查看模型')
    expect(document.head.querySelector('link[rel="canonical"]')).toBeNull()
  })

  it('renders effective public prices as Snow credits and preserves a real zero price', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([
      model(),
      model({
        slug: 'claude-sonnet',
        model: 'claude-sonnet-4',
        display_name: 'Claude Sonnet 4',
        provider: 'anthropic',
        category: 'reasoning',
        tags: ['reasoning'],
        capabilities: ['reasoning'],
        pricing: {
          ...model().pricing,
          input_price: 0.000003,
          output_price: 0.000015,
          peak_rate: { enabled: false, start: '', end: '', multiplier: 1 }
        }
      })
    ]))

    const { wrapper } = await mountCatalog()

    expect(wrapper.findAll('.catalog-card')).toHaveLength(2)
    expect(wrapper.text()).toContain('GPT-4o')
    expect(wrapper.findAllComponents(CreditAmount).map((amount) => amount.props('value'))).toEqual([
      '0',
      '10',
      '3',
      '15'
    ])
    expect(wrapper.findAll('[data-testid="snowflake-credit-icon"]')).toHaveLength(4)
    expect(wrapper.text()).not.toMatch(/[$¥]/)
    expect(wrapper.text()).toContain('128K')
  })

  it('keeps an explicitly USD-denominated catalog price in dollars', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([
      model({
        pricing: {
          ...model().pricing,
          currency: 'USD'
        }
      })
    ]))

    const { wrapper } = await mountCatalog()

    expect(wrapper.text()).toContain('$0')
    expect(wrapper.text()).toContain('$10')
    expect(wrapper.findComponent(CreditAmount).exists()).toBe(false)
  })

  it('filters by provider and writes filter state into the URL', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([
      model(),
      model({
        slug: 'claude-sonnet',
        model: 'claude-sonnet-4',
        display_name: 'Claude Sonnet 4',
        provider: 'anthropic',
        category: 'reasoning'
      })
    ]))
    const { wrapper, router } = await mountCatalog('/models.html?q=gpt')

    expect(wrapper.findAll('.catalog-card')).toHaveLength(1)
    await wrapper.get('#catalog-search').setValue('')
    const providerButton = wrapper.findAll('.catalog-filter-options button')
      .find((button) => button.text() === 'Anthropic')
    expect(providerButton).toBeTruthy()
    await providerButton!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.provider).toBe('anthropic')
    expect(router.currentRoute.value.query.q).toBeUndefined()
    expect(wrapper.findAll('.catalog-card')).toHaveLength(1)
    expect(wrapper.text()).toContain('Claude Sonnet 4')
  })

  it('copies the exact model ID and opens complete pricing details', async () => {
    const pricedModel = model()
    pricedModel.pricing.cache_write_1h_price = 0.000004
    pricedModel.pricing.priority_input_price = 0.000008
    pricedModel.pricing.priority_output_price = 0.00002
    pricedModel.pricing.intervals = [{
      min_tokens: 128_000,
      max_tokens: null,
      tier_label: 'long_context',
      input_price: 0.00001,
      output_price: 0.00003,
      cache_write_price: 0.000012,
      cache_write_1h_price: 0.000016,
      cache_read_price: 0.000002,
      per_request_price: null
    }]
    testState.getCatalog.mockResolvedValue(catalogResponse([pricedModel]))
    const { wrapper } = await mountCatalog()

    await wrapper.get('.catalog-copy-button').trigger('click')
    expect(testState.copyToClipboard).toHaveBeenCalledWith('gpt-4o', '模型 ID 已复制')
    expect(wrapper.get('.catalog-copy-button').text()).toContain('已复制')

    await wrapper.get('.catalog-pricing-heading button').trigger('click')
    await flushPromises()
    const dialog = wrapper.get('dialog')
    expect(dialog.attributes('open')).toBeDefined()
    expect(dialog.text()).toContain('缓存读取')
    expect(dialog.find('[data-testid="snowflake-credit-icon"]').exists()).toBe(true)
    expect(dialog.text()).not.toContain('$1')
    expect(dialog.text()).toContain('缓存写入（1 小时）')
    expect(dialog.text()).toContain('Priority 输入')
    expect(dialog.text()).toContain('长上下文价格')
    expect(dialog.text()).toContain('(128K, ∞] Token')
    expect(dialog.text()).toContain('18:00–23:00')
    expect(dialog.text()).toContain('Asia/Shanghai')
  })

  it('distinguishes a disabled catalog from a transient load error', async () => {
    testState.getCatalog.mockRejectedValue({ status: 404 })
    const unavailable = await mountCatalog()
    expect(unavailable.wrapper.text()).toContain('模型广场暂未开放')
    expect(unavailable.wrapper.find('.catalog-secondary-button').exists()).toBe(false)
    unavailable.wrapper.unmount()

    testState.getCatalog.mockRejectedValue({ status: 503 })
    const failed = await mountCatalog()
    expect(failed.wrapper.text()).toContain('模型目录加载失败')
    expect(failed.wrapper.get('.catalog-secondary-button').text()).toContain('重新加载')
  })

  it('installs canonical and Open Graph metadata and restores the document on unmount', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([]))
    const { wrapper } = await mountCatalog()

    expect(document.head.querySelector('link[rel="canonical"]')?.getAttribute('href')).toContain('/models.html')
    expect(document.head.querySelector('meta[property="og:title"]')?.getAttribute('content')).toBe('模型广场 - 落雪API')
    expect(document.head.querySelector('meta[property="og:description"]')?.getAttribute('content')).toBe('模型广场说明')

    wrapper.unmount()
    expect(document.head.querySelector('link[rel="canonical"]')).toBeNull()
    expect(document.head.querySelector('meta[property="og:title"]')).toBeNull()
  })
})
