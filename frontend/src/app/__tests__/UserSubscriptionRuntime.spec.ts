import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const runtimeHarness = vi.hoisted(() => ({
  initialize: vi.fn<() => Promise<void>>(),
  clear: vi.fn<() => void>(),
  setSession: null as null | ((user: { id: number } | null, authenticated: boolean) => void),
}))

vi.mock('@/stores/auth', async () => {
  const { ref } = await import('vue')
  const user = ref<{ id: number } | null>(null)
  const authenticated = ref(false)

  runtimeHarness.setSession = (nextUser, nextAuthenticated) => {
    user.value = nextUser
    authenticated.value = nextAuthenticated
  }

  return {
    useAuthStore: () => ({
      get user() {
        return user.value
      },
      get isAuthenticated() {
        return authenticated.value
      },
    }),
  }
})

vi.mock('@/stores/userProfile', () => ({
  useUserProfileStore: () => ({
    initialize: runtimeHarness.initialize,
    clear: runtimeHarness.clear,
  }),
}))

import UserSubscriptionRuntime from '../UserSubscriptionRuntime.vue'

describe('UserSubscriptionRuntime', () => {
  beforeEach(() => {
    runtimeHarness.initialize.mockReset().mockResolvedValue()
    runtimeHarness.clear.mockReset()
    runtimeHarness.setSession?.(null, false)
  })

  afterEach(() => {
    runtimeHarness.setSession?.(null, false)
  })

  it('initializes once per authenticated principal and clears on logout', () => {
    const wrapper = mount(UserSubscriptionRuntime)
    expect(runtimeHarness.clear).toHaveBeenCalled()
    runtimeHarness.clear.mockClear()

    runtimeHarness.setSession?.({ id: 7 }, true)
    expect(runtimeHarness.initialize).toHaveBeenCalledTimes(1)

    runtimeHarness.setSession?.({ id: 7 }, true)
    expect(runtimeHarness.initialize).toHaveBeenCalledTimes(1)

    runtimeHarness.setSession?.({ id: 9 }, true)
    expect(runtimeHarness.initialize).toHaveBeenCalledTimes(2)

    runtimeHarness.setSession?.(null, false)
    expect(runtimeHarness.clear).toHaveBeenCalled()
    wrapper.unmount()
  })
})
