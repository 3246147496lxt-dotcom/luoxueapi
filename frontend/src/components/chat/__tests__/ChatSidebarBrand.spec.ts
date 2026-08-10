import { computed, reactive } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const summaryState = vi.hoisted(() => ({
  subscriptionsLoaded: true,
  activeSubscriptionCount: 1,
  primarySubscription: {
    id: 1,
    name: 'Pro',
    expiresAt: null,
  } as { id: number; name: string | null; expiresAt: string | null } | null,
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, values?: Record<string, unknown>) => (
        values ? `${key}:${JSON.stringify(values)}` : key
      ),
    }),
  }
})

vi.mock('@/stores/userProfile', () => ({
  useUserProfileStore: () => reactive({
    subscriptionsLoaded: computed(() => summaryState.subscriptionsLoaded),
    activeSubscriptionCount: computed(() => summaryState.activeSubscriptionCount),
    primarySubscription: computed(() => summaryState.primarySubscription),
  }),
}))

import ChatSidebarBrand from '../ChatSidebarBrand.vue'
import { useAppStore } from '@/stores/app'

function mountBrand(collapsed = false, controls = '') {
  const pinia = createPinia()
  setActivePinia(pinia)
  const appStore = useAppStore()
  appStore.siteName = '落雪AI'

  return mount(ChatSidebarBrand, {
    props: { collapsed, controls },
    global: {
      plugins: [pinia],
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('ChatSidebarBrand', () => {
  beforeEach(() => {
    summaryState.subscriptionsLoaded = true
    summaryState.activeSubscriptionCount = 1
    summaryState.primarySubscription = { id: 1, name: 'Pro', expiresAt: null }
  })

  it('uses the configured text brand and the real subscription name without an image logo', () => {
    const wrapper = mountBrand()

    expect(wrapper.text()).toContain('落雪AI')
    expect(wrapper.text()).toContain('Pro')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.get('a').attributes('aria-label')).toBe('落雪AI · Pro')
    expect(wrapper.get('.chat-sidebar-brand__copy').findAll(':scope > *').map((node) => node.text()))
      .toEqual(['落雪AI', 'Pro'])
  })

  it('turns the collapsed brand mark into an accessible sidebar opener', async () => {
    const wrapper = mountBrand(true, 'chat-sidebar-id')
    const trigger = wrapper.get('button')

    expect(wrapper.get('.chat-sidebar-brand__monogram').text()).toBe('落雪')
    expect(wrapper.find('.chat-sidebar-brand__copy').exists()).toBe(false)
    expect(wrapper.find('a').exists()).toBe(false)
    expect(trigger.attributes('aria-label')).toBe('chat.actions.openSidebar')
    expect(trigger.attributes('aria-controls')).toBe('chat-sidebar-id')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.attributes('aria-describedby')).toBe(wrapper.get('[role="tooltip"]').attributes('id'))
    expect(wrapper.get('[role="tooltip"]').text()).toBe('chat.actions.openSidebar')

    await trigger.trigger('click')
    expect(wrapper.emitted('expand')).toEqual([[]])
  })

  it('shows a top-bar plan only for a confirmed active subscription', () => {
    summaryState.subscriptionsLoaded = false
    const loading = mountBrand()
    expect(loading.get('.chat-sidebar-brand__copy').findAll(':scope > *')).toHaveLength(1)
    expect(loading.text()).not.toContain('accountDock.subscriptionLoading')
    expect(loading.get('a').attributes('aria-label')).toBe('落雪AI')
    loading.unmount()

    summaryState.subscriptionsLoaded = true
    summaryState.primarySubscription = { id: 1, name: null, expiresAt: null }
    const unnamed = mountBrand()
    expect(unnamed.text()).toContain('accountDock.activeSubscriptions:{"count":1}')
    expect(unnamed.get('a').attributes('aria-label'))
      .toBe('落雪AI · accountDock.activeSubscriptions:{"count":1}')
    unnamed.unmount()

    summaryState.activeSubscriptionCount = 0
    summaryState.primarySubscription = null
    const free = mountBrand()
    expect(free.get('.chat-sidebar-brand__copy').findAll(':scope > *')).toHaveLength(1)
    expect(free.text()).not.toContain('accountDock.free')
    expect(free.text()).not.toContain('accountDock.payAsYouGo')
    expect(free.get('a').attributes('aria-label')).toBe('落雪AI')
  })
})
