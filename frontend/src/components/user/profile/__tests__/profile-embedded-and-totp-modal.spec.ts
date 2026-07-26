import { nextTick } from 'vue'
import { mount, shallowMount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import TotpDisableDialog from '@/components/user/profile/TotpDisableDialog.vue'
import TotpSetupModal from '@/components/user/profile/TotpSetupModal.vue'

const mocks = vi.hoisted(() => ({
  getStatus: vi.fn(),
  getVerificationMethod: vi.fn(),
  sendVerifyCode: vi.fn(),
  initiateSetup: vi.fn(),
  enable: vi.fn(),
  disable: vi.fn(),
  updateProfile: vi.fn(),
  toggleNotifyEmail: vi.fn(),
  sendNotifyEmailCode: vi.fn(),
  verifyNotifyEmail: vi.fn(),
  removeNotifyEmail: vi.fn(),
  getProfile: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
  authState: {
    user: null as unknown,
  },
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => mocks.authState,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError,
  }),
}))

vi.mock('@/api', () => ({
  totpAPI: {
    getStatus: mocks.getStatus,
    getVerificationMethod: mocks.getVerificationMethod,
    sendVerifyCode: mocks.sendVerifyCode,
    initiateSetup: mocks.initiateSetup,
    enable: mocks.enable,
    disable: mocks.disable,
  },
  userAPI: {
    updateProfile: mocks.updateProfile,
    toggleNotifyEmail: mocks.toggleNotifyEmail,
    sendNotifyEmailCode: mocks.sendNotifyEmailCode,
    verifyNotifyEmail: mocks.verifyNotifyEmail,
    removeNotifyEmail: mocks.removeNotifyEmail,
    getProfile: mocks.getProfile,
  },
}))

vi.mock('qrcode', () => ({
  default: {
    toDataURL: vi.fn().mockResolvedValue('data:image/png;base64,qr'),
  },
}))

const mountedWrappers: VueWrapper[] = []

async function flushAsyncWork() {
  await Promise.resolve()
  await Promise.resolve()
  await nextTick()
}

function mountAttached<T extends typeof TotpSetupModal | typeof TotpDisableDialog>(component: T) {
  const host = document.createElement('div')
  host.dataset.testModalHost = ''
  document.body.appendChild(host)
  const wrapper = mount(component, { attachTo: host })
  mountedWrappers.push(wrapper)
  return wrapper
}

describe('profile embedded cards', () => {
  beforeEach(() => {
    mocks.getStatus.mockReset()
    mocks.updateProfile.mockReset()
    mocks.getStatus.mockResolvedValue({
      feature_enabled: true,
      enabled: false,
      enabled_at: null,
    })
    mocks.updateProfile.mockResolvedValue({
      balance_notify_enabled: true,
      balance_notify_extra_emails: [],
    })
  })

  it('keeps the default TOTP card shell and removes it only when embedded', async () => {
    const regular = shallowMount(ProfileTotpCard)
    await flushAsyncWork()

    expect(regular.classes()).toContain('card')
    expect(regular.get('h2').text()).toBe('profile.totp.title')
    expect(regular.find('.px-6.py-6').exists()).toBe(true)

    const embedded = shallowMount(ProfileTotpCard, {
      props: { embedded: true },
    })
    await flushAsyncWork()

    expect(embedded.classes()).not.toContain('card')
    expect(embedded.find('h2').exists()).toBe(false)
    expect(embedded.find('.px-6.py-6').exists()).toBe(false)

    await embedded.get('button.btn-primary').trigger('click')
    expect(embedded.findComponent(TotpSetupModal).exists()).toBe(true)
  })

  it('keeps balance notification behavior in the padding-free embedded form', async () => {
    const baseProps = {
      enabled: false,
      threshold: null,
      extraEmails: [],
      systemDefaultThreshold: 10,
      userEmail: 'riley@example.com',
    }
    const regular = shallowMount(ProfileBalanceNotifyCard, { props: baseProps })

    expect(regular.classes()).toContain('card')
    expect(regular.get('h2').text()).toBe('profile.balanceNotify.title')
    expect(regular.find('.px-6.py-6').exists()).toBe(true)

    const embedded = shallowMount(ProfileBalanceNotifyCard, {
      props: {
        ...baseProps,
        embedded: true,
      },
    })

    expect(embedded.classes()).not.toContain('card')
    expect(embedded.find('h2').exists()).toBe(false)
    expect(embedded.find('.px-6.py-6').exists()).toBe(false)

    await embedded.get('input[type="checkbox"]').setValue(true)
    await flushAsyncWork()

    expect(mocks.updateProfile).toHaveBeenCalledWith({
      balance_notify_enabled: true,
    })
  })
})

describe('nested TOTP modal protocol', () => {
  beforeEach(() => {
    mocks.getVerificationMethod.mockReset()
    mocks.getVerificationMethod.mockResolvedValue({ method: 'password' })
    mocks.showSuccess.mockReset()
    mocks.showError.mockReset()
    document.body.style.overflow = 'auto'
  })

  afterEach(() => {
    mountedWrappers.splice(0).reverse().forEach(wrapper => wrapper.unmount())
    document.body.querySelectorAll('[data-test-modal-host]').forEach(element => element.remove())
    document.body.style.overflow = ''
  })

  it('exposes dialog semantics, traps focus, and restores the trigger', async () => {
    const trigger = document.createElement('button')
    trigger.dataset.testModalHost = ''
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mountAttached(TotpSetupModal)
    await flushAsyncWork()

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    expect(dialog).not.toBeNull()
    expect(dialog?.getAttribute('aria-modal')).toBe('true')
    expect(dialog?.getAttribute('aria-labelledby')).toBeTruthy()
    expect(dialog?.getAttribute('aria-describedby')).toBeTruthy()
    expect(document.activeElement).toBe(dialog)
    expect(document.body.style.overflow).toBe('hidden')

    const focusable = Array.from(dialog!.querySelectorAll<HTMLElement>(
      'button:not([disabled]), input:not([disabled])',
    ))
    const first = focusable[0]
    const last = focusable[focusable.length - 1]

    dialog?.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true }))
    expect(document.activeElement).toBe(first)

    first.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
    }))
    expect(document.activeElement).toBe(last)

    wrapper.unmount()
    mountedWrappers.splice(mountedWrappers.indexOf(wrapper), 1)

    expect(document.activeElement).toBe(trigger)
    expect(document.body.style.overflow).toBe('auto')
  })

  it('lets only the topmost nested modal handle Escape and reference-counts scroll lock', async () => {
    const setupWrapper = mountAttached(TotpSetupModal)
    await flushAsyncWork()
    const setupDialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    const disableWrapper = mountAttached(TotpDisableDialog)
    await flushAsyncWork()

    const alertDialog = document.body.querySelector<HTMLElement>('[role="alertdialog"]')
    expect(alertDialog?.getAttribute('aria-modal')).toBe('true')
    expect(alertDialog?.getAttribute('aria-labelledby')).toBeTruthy()
    expect(alertDialog?.getAttribute('aria-describedby')).toBeTruthy()
    expect(setupDialog?.inert).toBe(true)
    expect(setupDialog?.getAttribute('aria-hidden')).toBe('true')
    expect(document.body.style.overflow).toBe('hidden')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))

    expect(disableWrapper.emitted('close')).toHaveLength(1)
    expect(setupWrapper.emitted('close')).toBeUndefined()

    disableWrapper.unmount()
    mountedWrappers.splice(mountedWrappers.indexOf(disableWrapper), 1)
    expect(setupDialog?.inert).toBe(false)
    expect(setupDialog?.hasAttribute('aria-hidden')).toBe(false)
    expect(document.body.style.overflow).toBe('hidden')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(setupWrapper.emitted('close')).toHaveLength(1)

    setupWrapper.unmount()
    mountedWrappers.splice(mountedWrappers.indexOf(setupWrapper), 1)
    expect(document.body.style.overflow).toBe('auto')
  })
})
