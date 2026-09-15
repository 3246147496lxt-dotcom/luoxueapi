import { defineComponent, ref } from 'vue'
import { flushPromises, mount, type DOMWrapper, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ModelCatalogView from '../ModelCatalogView.vue'
import catalogMessages from '@/i18n/locales/zh/landing'
import type {
  PublicModelCatalogItem,
  PublicModelCatalogPricing,
  PublicModelCatalogResponse,
} from '@/api/catalog'

const testState = vi.hoisted(() => ({
  getCatalog: vi.fn(),
  authStore: { isAuthenticated: false, isAdmin: false },
  appStore: {
    cachedPublicSettings: {
      site_name: '落雪API',
      doc_url: '/tutorial-docs/',
      registration_enabled: true,
      available_channels_enabled: true,
      public_model_catalog_enabled: true,
    },
    siteName: '落雪API',
    docUrl: '',
  },
}))

function message(key: string): string | undefined {
  let value: unknown = catalogMessages
  for (const part of key.split('.')) {
    if (!value || typeof value !== 'object') return undefined
    value = (value as Record<string, unknown>)[part]
  }
  return typeof value === 'string' ? value : undefined
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const locale = ref('zh')
  return {
    ...actual,
    useI18n: () => ({
      locale,
      te: (key: string) => message(key) !== undefined,
      t: (key: string, params?: Record<string, string | number>) => (
        (message(key) ?? key).replace(/\{(\w+)\}/g, (placeholder, name: string) => (
          params?.[name] === undefined ? placeholder : String(params[name])
        ))
      ),
    }),
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => testState.authStore,
  useAppStore: () => testState.appStore,
}))

vi.mock('@/api/catalog', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/api/catalog')>(),
  getPublicModelCatalog: (...args: unknown[]) => testState.getCatalog(...args),
}))

const PublicLayoutStub = defineComponent({
  props: ['page'],
  template: '<div data-testid="public-site-layout" :data-page="page"><slot /></div>',
})
const AppLayoutStub = defineComponent({
  template: '<div data-testid="app-layout"><slot /></div>',
})
const mountedWrappers = new Set<VueWrapper>()

function pricing(overrides: Partial<PublicModelCatalogPricing> = {}): PublicModelCatalogPricing {
  return {
    label: '公开标准价',
    billing_mode: 'token',
    currency: 'USD',
    unit: 'per_token',
    input_price: 0.0000002,
    output_price: 0.0000012,
    cache_write_price: null,
    cache_write_1h_price: null,
    cache_read_price: 0.00000002,
    priority_input_price: null,
    priority_output_price: null,
    priority_cache_write_price: null,
    priority_cache_read_price: null,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    peak_rate: { enabled: false, start: '', end: '', multiplier: 1 },
    ...overrides,
  }
}

function officialPricing(overrides: Partial<PublicModelCatalogPricing> = {}) {
  return pricing({
    input_price: 0.000005,
    output_price: 0.00003,
    cache_read_price: 0.0000005,
    ...overrides,
  })
}

function model(overrides: Partial<PublicModelCatalogItem> = {}): PublicModelCatalogItem {
  return {
    slug: 'gpt-5-5-welfare',
    model: 'gpt-5.5',
    display_name: 'GPT-5.5',
    summary: '适合编码与复杂任务。',
    provider: 'openai',
    logo_key: 'openai',
    category: 'chat',
    tags: ['coding'],
    capabilities: ['reasoning'],
    context_window: 1_000_000,
    max_output_tokens: 128_000,
    featured: false,
    public_group: { id: 101, name: 'GPT 福利组', platform: 'openai', rate_multiplier: 0.04 },
    rate_multiplier: 0.04,
    pricing: pricing(),
    official_pricing: officialPricing(),
    ...overrides,
  }
}

function comparisonModels(): PublicModelCatalogItem[] {
  return [
    model(),
    model({
      slug: 'gpt-5-5-pro',
      public_group: { id: 102, name: 'GPT Pro 组', platform: 'openai', rate_multiplier: 0.2 },
      rate_multiplier: 0.2,
      pricing: pricing({ input_price: 0.000001, output_price: 0.000006, cache_read_price: 0.0000001 }),
    }),
    model({
      slug: 'claude-sonnet',
      model: 'claude-sonnet-4',
      display_name: 'Claude Sonnet 4',
      provider: 'anthropic',
      logo_key: 'anthropic',
      public_group: { id: 103, name: 'Claude 专线', platform: 'anthropic', rate_multiplier: 0.6 },
      rate_multiplier: 0.6,
      pricing: pricing({ input_price: 0.0000018, output_price: 0.000009, cache_read_price: 0.00000018 }),
      official_pricing: officialPricing({ input_price: 0.000003, output_price: 0.000015, cache_read_price: 0.0000003 }),
    }),
  ]
}

function catalogResponse(items: PublicModelCatalogItem[]): PublicModelCatalogResponse {
  return { items, server_timezone: 'Asia/Shanghai', pricing_updated_at: '2026-09-15T03:00:00Z' }
}

async function mountCatalog(initialPath = '/models', embedded = true) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/models.html', component: ModelCatalogView },
      { path: '/models', component: ModelCatalogView },
    ],
  })
  await router.push(initialPath)
  await router.isReady()
  const wrapper = mount(ModelCatalogView, {
    props: { embedded },
    global: {
      plugins: [router],
      stubs: {
        PublicSiteLayout: PublicLayoutStub,
        AppLayout: AppLayoutStub,
        ModelIcon: true,
        Icon: true,
        RouterLink: true,
      },
    },
  })
  mountedWrappers.add(wrapper)
  await flushPromises()
  return { wrapper, router }
}

function unmountCatalog(wrapper: VueWrapper) {
  wrapper.unmount()
  mountedWrappers.delete(wrapper)
}

function group(wrapper: VueWrapper, id: number) {
  const card = wrapper.find(`.group-card[data-group-id="${id}"]`)
  expect(card.exists(), `catalog should contain group ${id}`).toBe(true)
  return card
}

function filterButton(wrapper: VueWrapper, filter: 'platform' | 'group' | 'rate', label: string) {
  const button = wrapper.get(`[data-filter="${filter}"]`).findAll('button')
    .find((candidate) => candidate.text().includes(label))
  expect(button, `${filter} filter should contain ${label}`).toBeDefined()
  return button!
}

function amounts(root: DOMWrapper<Element>, selector: string): number[] {
  return root.findAll(selector).flatMap((cell) => (
    Array.from(cell.text().matchAll(/\$\s*([\d,]+(?:\.\d+)?)/g), (match) => Number(match[1].replaceAll(',', '')))
  ))
}

beforeEach(() => {
  testState.getCatalog.mockReset()
  testState.authStore.isAuthenticated = false
  testState.authStore.isAdmin = false
  document.head.querySelectorAll('meta[property^="og:"], link[rel="canonical"]')
    .forEach((node) => node.remove())
})

afterEach(() => {
  for (const wrapper of mountedWrappers) wrapper.unmount()
  mountedWrappers.clear()
})

describe('ModelCatalogView grouped price comparison', () => {
  it('keeps the authenticated Work shell, visible page heading and public shell separate', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([]))
    const workCatalog = await mountCatalog()
    expect(workCatalog.wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(workCatalog.wrapper.find('[data-testid="public-site-layout"]').exists()).toBe(false)
    expect(workCatalog.wrapper.get('h1').text()).toBe('模型中心')
    expect(document.head.querySelector('link[rel="canonical"]')).toBeNull()
    unmountCatalog(workCatalog.wrapper)

    const publicCatalog = await mountCatalog('/models.html', false)
    expect(publicCatalog.wrapper.get('[data-testid="public-site-layout"]').attributes('data-page')).toBe('models')
    expect(publicCatalog.wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
  })

  it('renders each real group vertically with an eight-column comparison and preserves duplicate model offers', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse(comparisonModels()))
    const { wrapper } = await mountCatalog()
    expect(wrapper.findAll('.group-card')).toHaveLength(3)
    expect(group(wrapper, 101).text()).toContain('GPT 福利组')
    expect(group(wrapper, 102).text()).toContain('GPT Pro 组')
    expect(group(wrapper, 101).get('.price-table tbody').text()).toContain('gpt-5.5')
    expect(group(wrapper, 102).get('.price-table tbody').text()).toContain('gpt-5.5')
    for (const card of wrapper.findAll('.group-card')) {
      const firstHeader = card.get('.price-table thead tr')
      const columns = firstHeader.findAll('th').reduce((sum, cell) => sum + Number(cell.attributes('colspan') || 1), 0)
      expect(columns).toBe(8)
      expect(card.find('button[aria-expanded]').exists()).toBe(false)
    }
    expect(wrapper.find('[data-filter="billing"]').exists()).toBe(false)
    expect(wrapper.find('.catalog-copy-button').exists()).toBe(false)
    expect(wrapper.find('dialog').exists()).toBe(false)
  })

  it('uses the server effective prices directly, without applying the group rate a second time', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([model()]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    expect(amounts(card, 'tbody .discount-cell')).toEqual(expect.arrayContaining([0.2, 1.2, 0.02]))
    expect(amounts(card, 'tbody .official-cell')).toEqual(expect.arrayContaining([5, 30, 0.5]))
    expect(amounts(card, 'tbody .discount-cell')).not.toContain(0.008)
  })

  it('keeps zero prices and missing values distinct and never invents an official price', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      pricing: pricing({ input_price: 0, output_price: null, cache_read_price: null }),
      official_pricing: null,
    })]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    expect(amounts(card, 'tbody .discount-cell')).toEqual([0])
    expect(amounts(card, 'tbody .official-cell')).toEqual([])
    expect(card.get('tbody').text()).toMatch(/—|–|-/)
    expect(card.text()).not.toMatch(/NaN|Infinity/)
  })

  it('renders all groups supplied by the API rather than a hardcoded design inventory', async () => {
    const models = Array.from({ length: 9 }, (_, index) => model({
      slug: `group-offer-${index}`,
      public_group: { id: 200 + index, name: `实际分组 ${index + 1}`, platform: 'openai', rate_multiplier: 0.1 },
      rate_multiplier: 0.1,
    }))
    testState.getCatalog.mockResolvedValue(catalogResponse(models))
    const { wrapper } = await mountCatalog()
    expect(wrapper.findAll('.group-card')).toHaveLength(9)
    for (const item of models) expect(group(wrapper, item.public_group!.id).text()).toContain(item.public_group!.name)
  })

  it('filters by platform while keeping incompatible group and rate choices visible but disabled', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse(comparisonModels()))
    const { wrapper, router } = await mountCatalog()
    await filterButton(wrapper, 'platform', 'OpenAI').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.provider).toBe('openai')
    expect(wrapper.findAll('.group-card')).toHaveLength(2)
    expect(filterButton(wrapper, 'group', 'Claude 专线').attributes('disabled')).toBeDefined()
    expect(filterButton(wrapper, 'rate', '0.6').attributes('disabled')).toBeDefined()
    expect(filterButton(wrapper, 'group', 'GPT 福利组').attributes('disabled')).toBeUndefined()
    expect(filterButton(wrapper, 'group', '全部').attributes('disabled')).toBeUndefined()
    expect(filterButton(wrapper, 'rate', '全部').attributes('disabled')).toBeUndefined()
  })

  it('combines group, rate and search filters and writes their selected values into the URL', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse(comparisonModels()))
    const { wrapper, router } = await mountCatalog()
    await filterButton(wrapper, 'group', 'GPT Pro 组').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.query.group).toBe('102')
    expect(wrapper.findAll('.group-card')).toHaveLength(1)
    expect(group(wrapper, 102).exists()).toBe(true)

    await filterButton(wrapper, 'rate', '0.2').trigger('click')
    await wrapper.get('input[type="search"]').setValue('gpt-5.5')
    await flushPromises()
    expect(router.currentRoute.value.query.rate).toBe('0.2')
    expect(router.currentRoute.value.query.q).toBe('gpt-5.5')
    expect(wrapper.get('input[type="search"]').attributes('placeholder')).toBe('搜索模型名称')
    expect(wrapper.findAll('.group-card')).toHaveLength(1)
  })

  it('restores supported filters from deep links and reacts to later URL navigation', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse(comparisonModels()))
    const { wrapper, router } = await mountCatalog('/models?provider=openai&group=101&rate=0.04&q=GPT-5.5')
    expect(wrapper.findAll('.group-card')).toHaveLength(1)
    expect(group(wrapper, 101).exists()).toBe(true)
    expect((wrapper.get('input[type="search"]').element as HTMLInputElement).value).toBe('GPT-5.5')

    await router.push('/models?provider=anthropic&q=claude')
    await flushPromises()
    expect(wrapper.findAll('.group-card')).toHaveLength(1)
    expect(group(wrapper, 103).exists()).toBe(true)
    expect((wrapper.get('input[type="search"]').element as HTMLInputElement).value).toBe('claude')
  })

  it('clears an empty-result filter state and restores the catalog', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse(comparisonModels()))
    const { wrapper, router } = await mountCatalog('/models?provider=openai&group=101&rate=0.04&q=not-a-model')
    expect(wrapper.findAll('.group-card')).toHaveLength(0)
    expect(wrapper.text()).toContain(catalogMessages.modelCatalog.noResults.title)
    const clear = wrapper.get('.empty-state').findAll('button')
      .find((button) => button.text() === catalogMessages.modelCatalog.table.clear)
    expect(clear).toBeDefined()
    await clear!.trigger('click')
    await flushPromises()
    expect(wrapper.findAll('.group-card')).toHaveLength(3)
    for (const key of ['provider', 'group', 'rate', 'q']) expect(router.currentRoute.value.query[key]).toBeUndefined()
  })

  it('shows base and long-context price tiers on both sides of the comparison', async () => {
    const tier = {
      min_tokens: 272_000, max_tokens: null, tier_label: 'long_context',
      input_price: 0.0000004, output_price: 0.0000018,
      cache_write_price: null, cache_write_1h_price: null, cache_read_price: 0.00000004,
      per_request_price: null,
    }
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      pricing: pricing({ intervals: [tier] }),
      official_pricing: officialPricing({ intervals: [{ ...tier, input_price: 0.00001, output_price: 0.000045, cache_read_price: 0.000001 }] }),
    })]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    expect(card.get('tbody').text()).toMatch(/272\s*K/i)
    expect(amounts(card, 'tbody .discount-cell')).toEqual(expect.arrayContaining([0.2, 1.2, 0.02, 0.4, 1.8, 0.04]))
    expect(amounts(card, 'tbody .official-cell')).toEqual(expect.arrayContaining([5, 30, 0.5, 10, 45, 1]))
  })

  it.each([
    { mode: 'per_request' as const, unitLabel: /次|请求/ },
    { mode: 'image' as const, unitLabel: /张|图片/ },
  ])('keeps $mode prices in their native billing unit', async ({ mode, unitLabel }) => {
    const fixedPricing = pricing({
      billing_mode: mode, unit: 'per_request',
      input_price: null, output_price: null, cache_read_price: null, per_request_price: 0.02,
    })
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      pricing: fixedPricing,
      official_pricing: pricing({ ...fixedPricing, per_request_price: 0.5 }),
    })]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    expect(amounts(card, 'tbody .discount-cell')).toContain(0.02)
    expect(amounts(card, 'tbody .official-cell')).toContain(0.5)
    expect(card.get('thead').text()).toMatch(unitLabel)
    expect(amounts(card, 'tbody .discount-cell')).not.toContain(20_000)
  })

  it('labels the paid default image size separately from an official base quote without a size', async () => {
    const imageTier = {
      min_tokens: 0, max_tokens: null,
      input_price: null, output_price: null,
      cache_write_price: null, cache_write_1h_price: null, cache_read_price: null,
    }
    const imagePricing = pricing({
      billing_mode: 'image', unit: 'per_request',
      input_price: null, output_price: null, cache_read_price: null,
      per_request_price: 0.02,
      intervals: [
        { ...imageTier, tier_label: '1K', per_request_price: 0.01 },
        { ...imageTier, tier_label: '2K', per_request_price: 0.02 },
      ],
    })
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      pricing: imagePricing,
      official_pricing: pricing({ ...imagePricing, per_request_price: 0.5, intervals: [] }),
    })]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    const rows = card.findAll('tbody tr')
    const base = rows[0]
    expect(base.get('.discount-cell .request-context').text()).toBe('默认 2K')
    expect(base.get('.official-cell .request-context').text()).toBe('基础价（规格未提供）')
    expect(amounts(base, '.discount-cell')).toEqual([0.02])
    expect(amounts(base, '.official-cell')).toEqual([0.5])
    expect(rows.slice(1).map(row => row.get('.discount-cell .request-context').text())).toEqual(['1K', '2K'])
    expect(amounts(card, 'tbody tr:not(:first-child) .official-cell')).toEqual([])
  })

  it('marks Priority and image token extras as base quotes with a separate context-tier note', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      pricing: pricing({ priority_input_price: 0.000008, image_input_price: 0.000007 }),
    })]))
    const { wrapper } = await mountCatalog()
    const card = group(wrapper, 101)
    for (const label of ['Priority 基础', '图像基础']) {
      const row = card.findAll('tbody tr').find(candidate => candidate.text().includes(label))
      expect(row).toBeDefined()
      const context = row!.get('.discount-cell .tier-context')
      expect(context.text()).toBe(label)
      expect(context.attributes('title')).toBe('此处为基础 Token 报价，上下文档位另计。')
    }
  })

  it('keeps a legacy model without group metadata visible', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([model({
      public_group: null, rate_multiplier: null, official_pricing: null,
    })]))
    const { wrapper } = await mountCatalog()
    expect(wrapper.findAll('.price-table')).toHaveLength(1)
    expect(wrapper.get('.price-table tbody').text()).toContain('gpt-5.5')
    expect(wrapper.text()).not.toMatch(/NaN|Infinity/)
  })

  it('distinguishes an empty catalog, a disabled catalog and a transient failure with working retry', async () => {
    testState.getCatalog.mockResolvedValueOnce(catalogResponse([]))
    const empty = await mountCatalog()
    expect(empty.wrapper.text()).toContain(catalogMessages.modelCatalog.empty.title)
    unmountCatalog(empty.wrapper)

    testState.getCatalog.mockRejectedValueOnce({ response: { status: 404 } })
    const unavailable = await mountCatalog()
    expect(unavailable.wrapper.text()).toContain(catalogMessages.modelCatalog.unavailable.title)
    expect(unavailable.wrapper.findAll('button').some((button) => button.text() === catalogMessages.modelCatalog.error.retry)).toBe(false)
    unmountCatalog(unavailable.wrapper)

    testState.getCatalog.mockRejectedValueOnce({ status: 503 })
    testState.getCatalog.mockResolvedValueOnce(catalogResponse([model()]))
    const failed = await mountCatalog()
    expect(failed.wrapper.text()).toContain(catalogMessages.modelCatalog.error.title)
    const retry = failed.wrapper.findAll('button').find((button) => button.text() === catalogMessages.modelCatalog.error.retry)
    expect(retry).toBeDefined()
    await retry!.trigger('click')
    await flushPromises()
    expect(failed.wrapper.findAll('.group-card')).toHaveLength(1)
    expect(failed.wrapper.text()).not.toContain(catalogMessages.modelCatalog.error.title)
  })

  it('shows loading until the catalog request completes and cleans up its request on unmount', async () => {
    let resolveCatalog!: (response: PublicModelCatalogResponse) => void
    testState.getCatalog.mockImplementationOnce(() => new Promise<PublicModelCatalogResponse>((resolve) => { resolveCatalog = resolve }))
    const { wrapper } = await mountCatalog()
    expect(wrapper.text()).toContain(catalogMessages.modelCatalog.loading)
    const requestOptions = testState.getCatalog.mock.calls[0][0] as { signal: AbortSignal }
    resolveCatalog(catalogResponse([model()]))
    await flushPromises()
    expect(wrapper.findAll('.group-card')).toHaveLength(1)
    expect(wrapper.text()).not.toContain(catalogMessages.modelCatalog.loading)
    unmountCatalog(wrapper)
    expect(requestOptions.signal.aborted).toBe(true)
  })

  it('installs public canonical and Open Graph metadata and removes its additions on unmount', async () => {
    testState.getCatalog.mockResolvedValue(catalogResponse([]))
    const { wrapper } = await mountCatalog('/models.html', false)
    expect(document.head.querySelector('link[rel="canonical"]')?.getAttribute('href')).toContain('/models.html')
    expect(document.head.querySelector('meta[property="og:title"]')?.getAttribute('content')).toBe(`${catalogMessages.modelCatalog.meta.title} - 落雪API`)
    expect(document.head.querySelector('meta[property="og:description"]')?.getAttribute('content')).toBe(catalogMessages.modelCatalog.meta.description)
    unmountCatalog(wrapper)
    expect(document.head.querySelector('link[rel="canonical"]')).toBeNull()
    expect(document.head.querySelector('meta[property="og:title"]')).toBeNull()
  })
})
