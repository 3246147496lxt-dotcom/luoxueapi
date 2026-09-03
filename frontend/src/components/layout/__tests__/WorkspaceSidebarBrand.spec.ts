import { computed, reactive } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const summaryState = vi.hoisted(() => ({
  subscriptionsLoaded: true,
  activeSubscriptionCount: 1,
  primarySubscription: {
    id: 1,
    name: 'GPT-订阅会员',
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

import WorkspaceSidebarBrand from '../WorkspaceSidebarBrand.vue'
import { useAppStore } from '@/stores/app'

function mountBrand(homePath = '/dashboard') {
  const pinia = createPinia()
  setActivePinia(pinia)
  useAppStore().siteName = '落雪API'

  return mount(WorkspaceSidebarBrand, {
    props: { homePath },
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

describe('WorkspaceSidebarBrand', () => {
  beforeEach(() => {
    summaryState.subscriptionsLoaded = true
    summaryState.activeSubscriptionCount = 1
    summaryState.primarySubscription = {
      id: 1,
      name: 'GPT-订阅会员',
      expiresAt: null,
    }
  })

  it('keeps the Workspace name fixed and binds the real plan', () => {
    const wrapper = mountBrand('/chat')

    expect(wrapper.get('a').attributes('href')).toBe('/chat')
    expect(wrapper.get('[data-testid="workspace-sidebar-brand-name"]').text()).toBe('落雪AI')
    expect(wrapper.get('[data-testid="workspace-sidebar-plan"]').text()).toBe('GPT-订阅会员')
    expect(wrapper.get('a').attributes('aria-label')).toBe('落雪AI · GPT-订阅会员')
    expect(wrapper.text()).not.toContain('落雪API')
  })

  it('keeps pending subscription copy out of the product title', () => {
    summaryState.subscriptionsLoaded = false
    summaryState.activeSubscriptionCount = 0
    summaryState.primarySubscription = null

    const wrapper = mountBrand()
    expect(wrapper.find('[data-testid="workspace-sidebar-plan"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('accountDock.subscriptionLoading')
    expect(wrapper.get('a').attributes('aria-label')).toBe('落雪AI')
  })

  it('shows Free only after the subscription state confirms there is no membership', () => {
    summaryState.subscriptionsLoaded = true
    summaryState.activeSubscriptionCount = 0
    summaryState.primarySubscription = null

    const wrapper = mountBrand()
    expect(wrapper.get('[data-testid="workspace-sidebar-plan"]').text()).toBe('accountDock.free')
    expect(wrapper.get('a').attributes('aria-label')).toBe('落雪AI · accountDock.free')
  })
})
