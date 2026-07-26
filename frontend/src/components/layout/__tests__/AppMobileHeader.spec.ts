import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/composables/usePageContext', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    usePageContext: () => ({ pageTitle: ref('Dashboard') }),
  }
})

import AppMobileHeader from '../AppMobileHeader.vue'
import { useAppStore } from '@/stores'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppMobileHeader.vue')
const componentSource = readFileSync(componentPath, 'utf8')

function mountHeader() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const wrapper = mount(AppMobileHeader, {
    global: {
      plugins: [pinia],
      stubs: {
        AppBrand: {
          props: ['placement', 'collapsed'],
          template:
            '<a data-testid="mobile-brand" :data-placement="placement" :data-collapsed="String(collapsed)" />',
        },
        Icon: {
          props: ['name', 'size'],
          template: '<svg :data-icon="name" :data-size="size" />',
        },
      },
    },
  })

  return { wrapper, appStore: useAppStore() }
}

describe('AppMobileHeader', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.clear()
  })

  it('is a compact mobile-only shell with the brand and current page title', () => {
    const { wrapper } = mountHeader()
    const header = wrapper.get('[data-testid="app-mobile-header"]')

    expect(header.classes()).toEqual(expect.arrayContaining(['app-mobile-header', 'lg:hidden']))
    expect(wrapper.get('[data-testid="mobile-brand"]').attributes()).toMatchObject({
      'data-placement': 'header',
      'data-collapsed': 'false',
    })
    expect(wrapper.get('[data-testid="mobile-header-page-title"]').text()).toBe('Dashboard')
    expect(componentSource).toContain('height: var(--app-shell-top-offset);')
    expect(componentSource).not.toContain('81px')
    expect(componentSource).not.toContain('5.0625rem')
  })

  it('opens the sidebar expanded and keeps aria state synchronized', async () => {
    const { wrapper, appStore } = mountHeader()
    const menu = wrapper.get('[data-testid="mobile-header-menu"]')

    appStore.setSidebarCollapsed(true)
    expect(menu.attributes('aria-controls')).toBe('app-sidebar')
    expect(menu.attributes('aria-expanded')).toBe('false')
    expect(menu.attributes('aria-label')).toBe('nav.openNavigation')

    await menu.trigger('click')

    expect(appStore.mobileOpen).toBe(true)
    expect(appStore.sidebarCollapsed).toBe(false)
    expect(menu.attributes('aria-expanded')).toBe('true')
    expect(menu.attributes('aria-label')).toBe('nav.closeNavigation')

    await menu.trigger('click')

    expect(appStore.mobileOpen).toBe(false)
    expect(appStore.sidebarCollapsed).toBe(false)
    expect(menu.attributes('aria-expanded')).toBe('false')
  })
})
