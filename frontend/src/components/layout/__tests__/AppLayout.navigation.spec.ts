import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/composables/useOnboardingTour', () => ({
  useOnboardingTour: () => ({ replayTour: vi.fn() }),
}))

import AppLayout from '../AppLayout.vue'
import { useAppStore } from '@/stores'

describe('AppLayout navigation structure', () => {
  afterEach(() => {
    document.body.classList.remove('admin-home-clay-portals')
  })

  it('places the global header before the floating sidebar and content shell', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia],
        stubs: {
          AppHeader: { template: '<header data-testid="header-stub" />' },
          AppSidebar: { template: '<aside data-testid="sidebar-stub" />' },
        },
      },
      slots: { default: '<div data-testid="page-content" />' },
    })

    const rootChildren = Array.from(wrapper.element.children)
    expect(rootChildren).toHaveLength(3)
    expect(rootChildren[0]).toBe(wrapper.get('[data-testid="header-stub"]').element)
    expect(rootChildren[1]).toBe(wrapper.get('[data-testid="sidebar-stub"]').element)
    expect(rootChildren[2]).toBe(wrapper.get('[data-testid="app-main-shell"]').element)
    expect(wrapper.classes()).toContain('pt-[81px]')
    expect(wrapper.findAll('[data-testid="header-stub"]')).toHaveLength(1)
    expect(wrapper.findAll('[data-testid="sidebar-stub"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="page-content"]').exists()).toBe(true)
  })

  it('keeps the content offset synchronized with the sidebar width', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const appStore = useAppStore()
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia],
        stubs: {
          AppHeader: true,
          AppSidebar: true,
        },
      },
    })
    const mainShell = wrapper.get('[data-testid="app-main-shell"]')

    expect(mainShell.classes()).toContain('min-h-[calc(100vh-81px)]')
    expect(mainShell.classes()).toEqual(
      expect.arrayContaining([
        'lg:ml-[184px]',
        'min-[1025px]:ml-[196px]',
        'min-[1281px]:ml-[208px]',
      ]),
    )
    expect(mainShell.classes()).not.toContain('lg:ml-[68px]')

    appStore.setSidebarCollapsed(true)
    await nextTick()

    expect(mainShell.classes()).toContain('lg:ml-[68px]')
    expect(mainShell.classes()).not.toContain('lg:ml-[184px]')

    appStore.setSidebarCollapsed(false)
    await nextTick()

    expect(mainShell.classes()).toEqual(
      expect.arrayContaining([
        'lg:ml-[184px]',
        'min-[1025px]:ml-[196px]',
        'min-[1281px]:ml-[208px]',
      ]),
    )
  })

  it('keeps content theming opt-in without changing the shared sidebar variant', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(AppLayout, {
      global: {
        plugins: [pinia],
        stubs: {
          AppHeader: {
            template: '<header data-testid="header-original" />',
          },
          AppSidebar: { template: '<aside data-testid="sidebar-original" />' },
        },
      },
    })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="header-original"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)

    await wrapper.setProps({ variant: 'home-clay' })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="header-original"]').attributes('data-variant')).toBeUndefined()
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.documentElement.classList.contains('app-layout--home-clay')).toBe(false)
    expect(document.body.classList.contains('app-layout--home-clay')).toBe(false)
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(true)

    await wrapper.setProps({ variant: 'default' })

    expect(wrapper.classes()).toContain('app-layout--snow-shell')
    expect(wrapper.get('[data-testid="app-main-shell"]').classes()).not.toContain('app-layout--home-clay')
    expect(wrapper.get('[data-testid="header-original"]').attributes('data-variant')).toBeUndefined()
    expect(wrapper.get('[data-testid="sidebar-original"]').attributes('variant')).toBeUndefined()
    expect(document.body.classList.contains('admin-home-clay-portals')).toBe(false)
  })
})
