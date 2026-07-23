import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createMemoryHistory, createRouter, type LocationQueryRaw, type Router } from 'vue-router'

import ProxiesView from '../ProxiesView.vue'

const {
  listProxies,
  getAllWithCount,
  getById,
  proxyRequests
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  getAllWithCount: vi.fn(),
  getById: vi.fn(),
  proxyRequests: [] as Array<Record<string, unknown>>
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      getAllWithCount,
      getById,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      batchCreate: vi.fn(),
      batchDelete: vi.fn(),
      testProxy: vi.fn(),
      checkProxyQuality: vi.fn(),
      getProxyAccounts: vi.fn(),
      exportData: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const stubs = {
  AppLayout: { template: '<div><slot /></div>' },
  TablePageLayout: {
    template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
  },
  DataTable: {
    name: 'DataTable',
    props: ['mobilePrimaryKey', 'mobileVisibleKeys'],
    template: '<div data-testid="data-table-stub"><slot name="cell-actions" :row="{ id: 7 }" /></div>'
  },
  Pagination: true,
  BaseDialog: true,
  ConfirmDialog: true,
  EmptyState: true,
  ImportDataModal: true,
  Select: true,
  ProxyAdBanner: true,
  PlatformTypeBadge: true,
  Icon: true
}

async function mountView(query: LocationQueryRaw): Promise<{ wrapper: VueWrapper; router: Router }> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/admin/proxies', component: { template: '<div />' } }]
  })
  await router.push({ path: '/admin/proxies', query })
  await router.isReady()

  const wrapper = mount(ProxiesView, {
    global: {
      plugins: [router],
      stubs
    }
  })
  await flushPromises()
  return { wrapper, router }
}

async function settleHistoryNavigation(): Promise<void> {
  await Promise.resolve()
  await flushPromises()
}

describe('admin ProxiesView precise deep links', () => {
  beforeEach(() => {
    localStorage.clear()
    proxyRequests.length = 0
    for (const mock of [listProxies, getAllWithCount, getById]) mock.mockReset()
    listProxies.mockImplementation(async (_page, _pageSize, filters) => {
      proxyRequests.push({ ...filters })
      return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
    })
    getAllWithCount.mockResolvedValue([])
    getById.mockResolvedValue({ id: 27, name: 'must-not-be-fetched' })
  })

  it('initializes protocol, status, health and focus with one #id list request', async () => {
    const { wrapper, router } = await mountView({
      protocol: 'socks5',
      status: 'active',
      health: 'suspected_restricted',
      focus_id: '27',
      open: 'health'
    })

    expect(proxyRequests).toEqual([
      expect.objectContaining({
        protocol: 'socks5',
        status: 'active',
        health: 'suspected_restricted',
        search: '#27'
      })
    ])
    expect(getById).not.toHaveBeenCalled()
    expect(router.currentRoute.value.query).toEqual({
      protocol: 'socks5',
      status: 'active',
      health: 'suspected_restricted',
      focus_id: '27',
      open: 'health'
    })
    wrapper.unmount()
  })

  it('removes illegal managed parameters, preserves unrelated query state and loads once', async () => {
    const { wrapper, router } = await mountView({
      protocol: 'ftp',
      status: 'broken',
      health: 'maybe',
      focus_id: '-1',
      open: 'edit',
      source: 'ops'
    })

    expect(router.currentRoute.value.query).toEqual({ source: 'ops' })
    expect(proxyRequests).toEqual([
      expect.objectContaining({
        protocol: undefined,
        status: undefined,
        health: undefined,
        search: undefined
      })
    ])
    expect(getById).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('syncs filters and focus through browser back/forward with one list request per navigation', async () => {
    const initialQuery = {
      protocol: 'http',
      status: 'active',
      health: 'healthy',
      focus_id: '5',
      open: 'health'
    }
    const nextQuery = {
      protocol: 'https',
      status: 'expired',
      health: 'failed',
      search: 'edge-pool'
    }
    const { wrapper, router } = await mountView(initialQuery)

    await router.push({ path: '/admin/proxies', query: nextQuery })
    await flushPromises()
    expect(proxyRequests).toHaveLength(2)
    expect(proxyRequests[1]).toEqual(expect.objectContaining({
      protocol: 'https',
      status: 'expired',
      health: 'failed',
      search: 'edge-pool'
    }))

    router.back()
    await settleHistoryNavigation()
    expect(router.currentRoute.value.query).toEqual(initialQuery)
    expect(proxyRequests).toHaveLength(3)
    expect(proxyRequests[2]).toEqual(expect.objectContaining({
      protocol: 'http',
      status: 'active',
      health: 'healthy',
      search: '#5'
    }))

    router.forward()
    await settleHistoryNavigation()
    expect(router.currentRoute.value.query).toEqual(nextQuery)
    expect(proxyRequests).toHaveLength(4)
    expect(proxyRequests[3]).toEqual(expect.objectContaining({
      protocol: 'https',
      status: 'expired',
      health: 'failed',
      search: 'edge-pool'
    }))
    expect(getById).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps search and the primary action visible while progressively disclosing mobile controls', async () => {
    const { wrapper } = await mountView({})
    const toolbar = wrapper.get('.proxy-toolbar')

    expect(toolbar.find('input[type="text"]').exists()).toBe(true)
    expect(toolbar.findAll('.btn-primary')).toHaveLength(1)
    expect(toolbar.get('[data-testid="proxy-create-action"]').classes()).toContain('proxy-touch-target')

    const filterToggle = toolbar.get('[data-testid="proxy-filter-toggle"]')
    expect(filterToggle.attributes('aria-expanded')).toBe('false')
    expect(filterToggle.attributes('aria-controls')).toBe('proxy-secondary-filters')
    expect(filterToggle.classes()).toContain('proxy-touch-target')
    expect(toolbar.get('#proxy-secondary-filters').classes()).not.toContain('proxy-secondary-filters--open')

    await filterToggle.trigger('click')
    expect(filterToggle.attributes('aria-expanded')).toBe('true')
    expect(toolbar.get('#proxy-secondary-filters').classes()).toContain('proxy-secondary-filters--open')

    const moreToggle = toolbar.get('[data-testid="proxy-more-actions-toggle"]')
    expect(moreToggle.attributes('aria-expanded')).toBe('false')
    expect(moreToggle.attributes('aria-controls')).toBe('proxy-secondary-actions')
    expect(moreToggle.classes()).toContain('proxy-touch-target')
    expect(toolbar.get('#proxy-secondary-actions').classes()).not.toContain('proxy-secondary-actions--open')

    await moreToggle.trigger('click')
    expect(moreToggle.attributes('aria-expanded')).toBe('true')
    expect(toolbar.get('#proxy-secondary-actions').classes()).toContain('proxy-secondary-actions--open')
    wrapper.unmount()
  })

  it('configures a concise proxy card without changing the desktop column set', async () => {
    const { wrapper } = await mountView({})
    const table = wrapper.findComponent({ name: 'DataTable' })

    expect(table.props('mobilePrimaryKey')).toBe('name')
    expect(table.props('mobileVisibleKeys')).toEqual([
      'name',
      'protocol',
      'address',
      'account_count',
      'status'
    ])
    wrapper.unmount()
  })

  it('keeps one direct row action and expands secondary actions in the mobile card flow', async () => {
    const { wrapper } = await mountView({})
    const rowActions = wrapper.get('[data-testid="proxy-row-actions-7"]')
    const moreToggle = rowActions.get('[data-testid="proxy-row-more-7"]')
    const secondaryActions = rowActions.get('#proxy-row-secondary-actions-7')

    expect(rowActions.findAll('.proxy-row-edit-action')).toHaveLength(1)
    expect(rowActions.findAll('.proxy-row-more-toggle')).toHaveLength(1)
    expect(secondaryActions.findAll('button')).toHaveLength(3)
    expect(moreToggle.attributes('aria-expanded')).toBe('false')
    expect(moreToggle.attributes('aria-controls')).toBe('proxy-row-secondary-actions-7')
    expect(secondaryActions.classes()).not.toContain('proxy-row-secondary-actions--open')

    await moreToggle.trigger('click')
    expect(moreToggle.attributes('aria-expanded')).toBe('true')
    expect(secondaryActions.classes()).toContain('proxy-row-secondary-actions--open')
    wrapper.unmount()
  })
})
