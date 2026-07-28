import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DesktopDevicesView from '@/views/user/DesktopDevicesView.vue'

const api = vi.hoisted(() => ({
  listDevices: vi.fn(),
  renameDevice: vi.fn(),
  revokeDevice: vi.fn(),
}))

vi.mock('@/api/desktop', () => ({ desktopAPI: api }))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'en-US' },
      t: (key: string, params?: Record<string, string>) => params?.name ? `${key}:${params.name}` : key,
    }),
  }
})

const activeDevice = {
  id: 'device-1',
  name: 'Office Mac',
  platform: 'macos',
  architecture: 'aarch64',
  os_version: '15.5',
  app_version: '0.1.0',
  status: 'active',
  approved_at: '2026-07-27T09:00:00Z',
  last_seen_at: '2026-07-28T09:00:00Z',
}

describe('DesktopDevicesView', () => {
  beforeEach(() => {
    api.listDevices.mockReset()
    api.renameDevice.mockReset()
    api.revokeDevice.mockReset()
    api.listDevices.mockResolvedValue([activeDevice])
  })

  function mountView() {
    return mount(DesktopDevicesView, {
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

  it('loads devices and supports inline rename', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(api.listDevices).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('Office Mac')

    await wrapper.get('.desktop-device-card__actions .btn').trigger('click')
    const input = wrapper.get('input')
    await input.setValue('Studio Mac')
    api.renameDevice.mockResolvedValue({ ...activeDevice, name: 'Studio Mac' })
    await wrapper.get('.desktop-device-rename').trigger('submit')
    await flushPromises()

    expect(api.renameDevice).toHaveBeenCalledWith('device-1', 'Studio Mac')
    expect(wrapper.text()).toContain('Studio Mac')
  })

  it('requires confirmation and keeps the revoked device visible', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('.desktop-device-revoke').trigger('click')
    api.revokeDevice.mockResolvedValue({ ...activeDevice, status: 'revoked' })
    await wrapper.get('[data-testid="confirm-revoke"]').trigger('click')
    await flushPromises()

    expect(api.revokeDevice).toHaveBeenCalledWith('device-1')
    expect(wrapper.text()).toContain('Office Mac')
    expect(wrapper.get('.desktop-device-status').text()).toBe('revoked')
    expect(wrapper.find('.desktop-device-revoke').exists()).toBe(false)
  })
})
