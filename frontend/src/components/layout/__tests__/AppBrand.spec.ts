import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import type { User } from '@/types'

import AppBrand from '../AppBrand.vue'
import { useAppStore, useAuthStore } from '@/stores'

let pinia: Pinia

function createUser(role: User['role']): User {
  return {
    id: 7,
    username: 'brand-user',
    email: 'brand@example.com',
    role,
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-07-18T00:00:00Z',
    updated_at: '2026-07-18T00:00:00Z',
  }
}

function mountBrand(
  props: {
    placement?: 'header' | 'sidebar'
    collapsed?: boolean
    wordmark?: boolean
    wordmarkOnly?: boolean
  } = {},
): VueWrapper {
  return mount(AppBrand, {
    props: {
      placement: props.placement ?? 'sidebar',
      collapsed: props.collapsed ?? false,
      wordmark: props.wordmark ?? false,
      wordmarkOnly: props.wordmarkOnly ?? false,
    },
    global: {
      plugins: [pinia],
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="String(to)"><slot /></a>',
        },
      },
    },
  })
}

describe('AppBrand', () => {
  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    useAuthStore().user = createUser('user')
  })

  afterEach(() => {
    document.querySelector('meta[name="app-entry"]')?.remove()
  })

  it('renders the canonical default logo as a logo-only link', () => {
    const appStore = useAppStore()
    appStore.siteName = '落雪API'
    appStore.siteLogo = ''
    const wrapper = mountBrand()
    const brand = wrapper.get('[data-testid="sidebar-brand"]')

    expect(brand.attributes('aria-label')).toBe('落雪API')
    expect(brand.text()).toBe('')
    expect(wrapper.get('[data-testid="sidebar-brand-logo"] img').attributes('src'))
      .toBe('/logo.png')
    expect(wrapper.get('[data-testid="sidebar-brand-logo"] img').classes())
      .toContain('app-brand-logo-image-default')
  })

  it('renders a configured logo with the configured accessible name', () => {
    const appStore = useAppStore()
    appStore.siteName = 'Snow Console'
    appStore.siteLogo = '/brand/custom.svg'
    const wrapper = mountBrand({ placement: 'header' })

    const brand = wrapper.get('[data-testid="header-brand"]')
    expect(brand.attributes('aria-label')).toBe('Snow Console')
    expect(brand.get('img').attributes('src')).toBe('/brand/custom.svg')
    expect(brand.get('img').classes()).not.toContain('app-brand-logo-image-default')
    expect(brand.text()).toBe('')
  })

  it('renders the configured site name as an opt-in wordmark', async () => {
    const appStore = useAppStore()
    appStore.siteName = 'Luoxue AI Workspace'
    const wrapper = mountBrand({ wordmark: true })

    expect(wrapper.get('[data-testid="sidebar-brand-wordmark"]').text())
      .toBe('Luoxue AI Workspace')
    expect(wrapper.get('[data-testid="sidebar-brand"]').classes())
      .toContain('app-brand--wordmark')
    expect(wrapper.find('[data-testid="sidebar-brand-logo"]').exists()).toBe(true)

    await wrapper.setProps({ collapsed: true })

    expect(wrapper.find('[data-testid="sidebar-brand-wordmark"]').exists()).toBe(false)
  })

  it('renders a pure text Work wordmark while preserving a collapsed fallback logo', async () => {
    const appStore = useAppStore()
    appStore.siteName = '落雪 AI Workspace'
    const wrapper = mountBrand({ wordmark: true, wordmarkOnly: true })

    expect(wrapper.get('[data-testid="sidebar-brand-wordmark"]').text())
      .toBe('落雪 AI Workspace')
    expect(wrapper.get('[data-testid="sidebar-brand"]').classes())
      .toContain('app-brand--wordmark-only')
    expect(wrapper.find('[data-testid="sidebar-brand-logo"]').exists()).toBe(false)
    expect(wrapper.find('img').exists()).toBe(false)

    await wrapper.setProps({ collapsed: true })

    expect(wrapper.find('[data-testid="sidebar-brand-wordmark"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="sidebar-brand-logo"] img').attributes('src'))
      .toBe('/logo.png')
  })

  it('does not replace an uploaded data-image logo with the canonical default', async () => {
    const appStore = useAppStore()
    appStore.siteName = '落雪API'
    appStore.siteLogo = 'data:image/svg+xml;base64,PHN2Zy8+'
    const wrapper = mountBrand()
    const image = wrapper.get('[data-testid="sidebar-brand-logo"] img')

    expect(image.attributes('src')).toBe('data:image/svg+xml;base64,PHN2Zy8+')
    expect(image.classes()).not.toContain('app-brand-logo-image-default')

    appStore.siteLogo = 'data:image/png;base64,iVBORw0KGgo='
    await wrapper.vm.$nextTick()

    expect(image.attributes('src')).toBe('data:image/png;base64,iVBORw0KGgo=')
    expect(image.classes()).not.toContain('app-brand-logo-image-default')
  })

  it('keeps the logo inside the active frontend entry regardless of role', async () => {
    const authStore = useAuthStore()
    const wrapper = mountBrand()

    expect(wrapper.get('[data-testid="sidebar-brand"]').attributes('href')).toBe('/dashboard')

    authStore.user = createUser('admin')
    await wrapper.vm.$nextTick()

    expect(wrapper.get('[data-testid="sidebar-brand"]').attributes('href'))
      .toBe('/dashboard')

    wrapper.unmount()
    const entryMeta = document.createElement('meta')
    entryMeta.name = 'app-entry'
    entryMeta.content = 'admin'
    document.head.append(entryMeta)

    expect(mountBrand().get('[data-testid="sidebar-brand"]').attributes('href'))
      .toBe('/admin/dashboard')
  })

  it('keeps the accessible logo link intact while collapsed', async () => {
    const wrapper = mountBrand({ collapsed: true })
    const brand = wrapper.get('[data-testid="sidebar-brand"]')

    expect(brand.classes()).toContain('app-brand--collapsed')
    expect(brand.attributes('aria-label')).toBe('落雪API')
    expect(brand.get('img').attributes('alt')).toBe('')

    await wrapper.setProps({ collapsed: false })

    expect(brand.classes()).not.toContain('app-brand--collapsed')
    expect(brand.attributes('aria-label')).toBe('落雪API')
  })

  it('rejects unsafe configured logo URLs', async () => {
    const appStore = useAppStore()
    appStore.siteName = 'Snow Console'
    appStore.siteLogo = 'javascript:alert(1)'
    const wrapper = mountBrand()
    const image = wrapper.get('[data-testid="sidebar-brand-logo"] img')

    expect(image.attributes('src')).toBe('/logo.png')

    appStore.siteLogo = '//example.com/tracker.svg'
    await wrapper.vm.$nextTick()

    expect(image.attributes('src')).toBe('/logo.png')
  })
})
