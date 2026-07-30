import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QuotaViewerAuthorizeView from '@/views/user/QuotaViewerAuthorizeView.vue'

const mocks = vi.hoisted(() => ({
  api: {
    getPairingPreview: vi.fn(),
    approvePairing: vi.fn(),
  },
  route: {
    query: { user_code: 'abcd-efgh' } as Record<string, string | string[] | undefined>,
  },
  routerPush: vi.fn(),
}))

vi.mock('@/api/quotaViewer', () => ({
  quotaViewerAPI: mocks.api,
}))

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ push: mocks.routerPush }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ isAdmin: false }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  const messages: Record<string, string> = {
    'quotaViewerAuthorization.errors.QUOTA_AUTH_RESPONSE_INVALID': '安全校验失败',
    'quotaViewerAuthorization.errors.QUOTA_AUTH_DEVICE_LIMIT': '设备达到上限',
  }
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const validPreview = {
  user_code: 'ABCD-EFGH',
  client_id: 'luoxue-quota-viewer',
  scope: 'quota:read',
  device_name: 'Office Mac',
  platform: 'macos',
  architecture: 'arm64',
  os_version: '15.5',
  app_version: '1.0.0',
  read_only: true,
  expires_at: '2026-07-30T12:00:00Z',
}

const approvedDevice = {
  id: 'device-1',
  client_id: 'luoxue-quota-viewer',
  scope: 'quota:read',
  name: 'Office Mac',
  platform: 'macos',
  architecture: 'arm64',
  status: 'pending',
}

function mountView() {
  return mount(QuotaViewerAuthorizeView, {
    global: {
      stubs: {
        AuthLayout: { template: '<main><slot /></main>' },
        Icon: true,
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('QuotaViewerAuthorizeView', () => {
  beforeEach(() => {
    mocks.route.query = { user_code: 'abcd-efgh' }
    mocks.routerPush.mockReset()
    mocks.api.getPairingPreview.mockReset()
    mocks.api.approvePairing.mockReset()
    mocks.api.getPairingPreview.mockResolvedValue(validPreview)
    mocks.api.approvePairing.mockResolvedValue(approvedDevice)
  })

  it('normalizes the deep-link code and displays a verified read-only request', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(mocks.api.getPairingPreview).toHaveBeenCalledWith('ABCD-EFGH')
    expect(wrapper.text()).toContain('Office Mac')
    expect(wrapper.text()).toContain('quotaViewerAuthorization.scopeRead')
  })

  it.each([
    ['client id', { client_id: 'another-client' }],
    ['scope', { scope: 'managed_keys:write' }],
    ['read-only flag', { read_only: false }],
  ])('rejects a preview with the wrong %s', async (_label, override) => {
    mocks.api.getPairingPreview.mockResolvedValue({ ...validPreview, ...override })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('安全校验失败')
    expect(wrapper.find('.quota-viewer-auth-details').exists()).toBe(false)
    expect(mocks.api.approvePairing).not.toHaveBeenCalled()
  })

  it('validates the approved device and links the success state to device management', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(mocks.api.approvePairing).toHaveBeenCalledWith('ABCD-EFGH')
    expect(wrapper.text()).toContain('quotaViewerAuthorization.successTitle')
    expect(wrapper.get('a').attributes('href')).toBe('/quota-viewer/devices')
  })

  it('does not show success when the approval response has an unsafe status', async () => {
    mocks.api.approvePairing.mockResolvedValue({ ...approvedDevice, status: 'active' })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('安全校验失败')
    expect(wrapper.text()).not.toContain('quotaViewerAuthorization.successTitle')
    expect(wrapper.find('.quota-viewer-auth-details').exists()).toBe(true)
  })

  it('offers device management when the account has reached its device limit', async () => {
    mocks.api.getPairingPreview.mockRejectedValue({
      reason: 'QUOTA_AUTH_DEVICE_LIMIT',
      message: 'raw backend message',
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('设备达到上限')
    expect(wrapper.get('a').attributes('href')).toBe('/quota-viewer/devices')
  })
})
