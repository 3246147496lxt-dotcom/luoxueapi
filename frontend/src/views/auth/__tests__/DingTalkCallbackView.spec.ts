import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import DingTalkCallbackView from '../DingTalkCallbackView.vue'

const replace = vi.fn()
const showSuccess = vi.fn()
const showError = vi.fn()
const exchangePendingOAuthCompletion = vi.fn()
const authUser = {
  current: null as { role: 'admin' | 'user' } | null
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
  useRouter: () => ({ replace })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      te: () => false
    })
  }
})

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    user: authUser.current,
    setToken: vi.fn(),
    setPendingAuthSession: vi.fn(),
    clearPendingAuthSession: vi.fn()
  }),
  useAppStore: () => ({
    showSuccess,
    showError
  })
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn()
  }
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    exchangePendingOAuthCompletion: (...args: unknown[]) =>
      exchangePendingOAuthCompletion(...args)
  }
})

describe('DingTalkCallbackView OAuth binding redirect', () => {
  beforeEach(() => {
    replace.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    exchangePendingOAuthCompletion.mockReset()
    authUser.current = { role: 'user' }
    window.location.hash = ''
    localStorage.clear()
    sessionStorage.clear()
  })

  it('returns a bind completion without an explicit redirect to canonical user settings', async () => {
    exchangePendingOAuthCompletion.mockResolvedValue({})

    mount(DingTalkCallbackView, {
      global: {
        stubs: {
          AuthLayout: { template: '<div><slot /></div>' },
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
          transition: false
        }
      }
    })

    await flushPromises()

    expect(showSuccess).toHaveBeenCalledWith('profile.authBindings.bindSuccess')
    expect(replace).toHaveBeenCalledWith(
      '/dashboard?account_settings=account&account_settings_detail=connections'
    )
  })
})
