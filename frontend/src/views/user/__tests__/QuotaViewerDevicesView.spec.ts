import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QuotaViewerDevicesView from '@/views/user/QuotaViewerDevicesView.vue'

const api = vi.hoisted(() => ({
  listDevices: vi.fn(),
  revokeDevice: vi.fn(),
}))

vi.mock('@/api/quotaViewer', () => ({ quotaViewerAPI: api }))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string, params?: Record<string, string>) => {
        if (params?.name) return `${key}:${params.name}`
        if (params?.time) return `${key}:${params.time}`
        if (params?.version) return `${key}:${params.version}`
        return key
      },
    }),
  }
})

const activeDevice = {
  id: 'device-1',
  client_id: 'luoxue-quota-viewer',
  scope: 'quota:read',
  name: 'Office Mac',
  platform: 'macos',
  architecture: 'arm64',
  os_version: '15.5',
  app_version: '1.0.0',
  status: 'active',
  approved_at: '2026-07-29T09:00:00Z',
  last_seen_at: '2026-07-30T09:00:00Z',
}

function mountView() {
  return mount(QuotaViewerDevicesView, {
    global: {
      stubs: {
        AppLayout: { template: '<main><slot /></main>' },
        AdminPageHeader: { template: '<header><slot name="secondary-actions" /></header>' },
        Icon: true,
        ConfirmDialog: {
          props: ['show'],
          emits: ['confirm', 'cancel'],
          template: '<button v-if="show" data-testid="confirm-revoke" @click="$emit(\'confirm\')">confirm</button>',
        },
      },
    },
  })
}

describe('QuotaViewerDevicesView', () => {
  beforeEach(() => {
    api.listDevices.mockReset()
    api.revokeDevice.mockReset()
    api.listDevices.mockResolvedValue([activeDevice])
  })

  it('loads authorized quota viewer devices', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(api.listDevices).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('Office Mac')
    expect(wrapper.get('.quota-viewer-device-status').text()).toBe('active')
  })

  it('requires confirmation and keeps the revoked device visible', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.quota-viewer-device__revoke').trigger('click')
    api.revokeDevice.mockResolvedValue({ ...activeDevice, status: 'revoked' })
    await wrapper.get('[data-testid="confirm-revoke"]').trigger('click')
    await flushPromises()

    expect(api.revokeDevice).toHaveBeenCalledWith('device-1')
    expect(wrapper.text()).toContain('Office Mac')
    expect(wrapper.get('.quota-viewer-device-status').text()).toBe('revoked')
    expect(wrapper.find('.quota-viewer-device__revoke').exists()).toBe(false)
  })
})
