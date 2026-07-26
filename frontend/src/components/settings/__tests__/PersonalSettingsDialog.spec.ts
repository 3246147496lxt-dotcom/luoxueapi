import { DOMWrapper, flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { PublicSettings, User } from '@/types'

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

import PersonalSettingsDialog from '../PersonalSettingsDialog.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import {
  getVisibleFocusableElements,
  registerModalLayer,
  unregisterModalLayer,
} from '@/utils/modalStack'

const wrappers = new Set<VueWrapper>()
const hosts = new Set<HTMLElement>()

const user: User = {
  id: 7,
  username: 'Riley Quinn',
  email: 'riley@example.com',
  role: 'user',
  balance: 24.5,
  frozen_balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: null,
  balance_notify_enabled: true,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-07-01T00:00:00Z',
  updated_at: '2026-07-01T00:00:00Z',
}

function bodyElement(testId: string) {
  const element = document.body.querySelector<HTMLElement>(
    `[data-testid="${testId}"]`,
  )
  if (!element) throw new Error(`Missing body element: ${testId}`)
  return new DOMWrapper(element)
}

async function mountDialog(initialPath: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const authStore = useAuthStore()
  authStore.user = { ...user }
  const appStore = useAppStore()

  const refreshUser = vi
    .spyOn(authStore, 'refreshUser')
    .mockResolvedValue(authStore.user)
  const fetchPublicSettings = vi
    .spyOn(appStore, 'fetchPublicSettings')
    .mockResolvedValue(null)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', component: { template: '<div />' } },
      { path: '/admin/dashboard', component: { template: '<div />' } },
    ],
  })
  await router.push(initialPath)
  await router.isReady()

  const host = document.createElement('div')
  host.className = 'app-layout'
  host.inert = false
  document.body.appendChild(host)
  hosts.add(host)

  const accountTrigger = document.createElement('button')
  accountTrigger.type = 'button'
  accountTrigger.className = 'sidebar-account-trigger'
  accountTrigger.textContent = 'Account'
  host.appendChild(accountTrigger)

  const wrapper = mount(PersonalSettingsDialog, {
    attachTo: host,
    global: {
      plugins: [pinia, router],
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
        PersonalSettingsGeneralPanel: {
          template: '<div data-testid="general-panel-stub" />',
        },
        PersonalSettingsAccountPanel: {
          props: ['detail'],
          emits: ['open-detail', 'retry'],
          template: `
            <div data-testid="account-panel-stub" :data-detail="detail">
              <button
                data-testid="open-profile-stub"
                @click="$emit('open-detail', 'profile')"
              />
            </div>
          `,
        },
        PersonalSettingsSecurityPanel: {
          props: ['detail'],
          emits: ['open-detail'],
          template: `
            <div data-testid="security-panel-stub" :data-detail="detail">
              <button
                data-testid="open-password-stub"
                @click="$emit('open-detail', 'password')"
              />
            </div>
          `,
        },
        PersonalSettingsNotificationsPanel: {
          props: ['detail'],
          emits: ['open-detail', 'retry'],
          template: '<div data-testid="notifications-panel-stub" :data-detail="detail" />',
        },
        WalletSubscriptionSettingsPanel: {
          emits: ['navigate'],
          template: '<div data-testid="billing-panel-stub" />',
        },
      },
    },
  })
  wrappers.add(wrapper)
  await flushPromises()

  return {
    wrapper,
    router,
    authStore,
    appStore,
    refreshUser,
    fetchPublicSettings,
    host,
    accountTrigger,
  }
}

beforeEach(() => {
  document.body.style.overflow = ''
  const visibleRect = {
    bottom: 20,
    height: 20,
    left: 0,
    right: 20,
    top: 0,
    width: 20,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  } as DOMRect
  const visibleRects = [visibleRect] as unknown as DOMRectList
  vi.spyOn(Element.prototype, 'getClientRects').mockImplementation(function (this: Element) {
    return (this as HTMLElement).dataset.noLayout === 'true'
      ? [] as unknown as DOMRectList
      : visibleRects
  })
})

afterEach(() => {
  for (const wrapper of wrappers) wrapper.unmount()
  wrappers.clear()
  for (const host of hosts) host.remove()
  hosts.clear()
  document.body.style.overflow = ''
  vi.restoreAllMocks()
})

describe('PersonalSettingsDialog', () => {
  it('opens General with no data request and loads each later section only on first visit', async () => {
    const {
      router,
      refreshUser,
      fetchPublicSettings,
    } = await mountDialog('/dashboard?account_settings=general')

    expect(bodyElement('general-panel-stub').exists()).toBe(true)
    expect(document.body.querySelector('#personal-settings-dialog-title')?.textContent?.trim())
      .toBe('personalSettings.dialog.sections.general')
    expect(document.body.querySelector('[data-testid="account-panel-stub"]')).toBeNull()
    expect(refreshUser).not.toHaveBeenCalled()
    expect(fetchPublicSettings).not.toHaveBeenCalled()

    await new DOMWrapper(
      document.body.querySelector<HTMLElement>('[data-settings-section="account"]')!,
    ).trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.account_settings).toBe('account')
    expect(document.body.querySelector('#personal-settings-dialog-title')?.textContent?.trim())
      .toBe('personalSettings.dialog.sections.account')
    expect(refreshUser).toHaveBeenCalledOnce()
    expect(fetchPublicSettings).toHaveBeenCalledOnce()
    expect(bodyElement('account-panel-stub').exists()).toBe(true)

    await new DOMWrapper(
      document.body.querySelector<HTMLElement>('[data-settings-section="general"]')!,
    ).trigger('click')
    await new DOMWrapper(
      document.body.querySelector<HTMLElement>('[data-settings-section="account"]')!,
    ).trigger('click')
    await flushPromises()

    expect(refreshUser).toHaveBeenCalledOnce()
    expect(fetchPublicSettings).toHaveBeenCalledOnce()
  })

  it('uses replace-style details and returns to the current section in-place', async () => {
    const { router } = await mountDialog('/dashboard?account_settings=account')

    await bodyElement('open-profile-stub').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query).toMatchObject({
      account_settings: 'account',
      account_settings_detail: 'profile',
    })

    await bodyElement('personal-settings-back').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.account_settings).toBe('account')
    expect(router.currentRoute.value.query.account_settings_detail).toBeUndefined()
  })

  it('supports arrow-key category navigation', async () => {
    const { router } = await mountDialog('/dashboard?account_settings=general')
    const generalTab = document.body.querySelector<HTMLElement>(
      '[data-settings-section="general"]',
    )!

    generalTab.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'ArrowRight',
      bubbles: true,
    }))
    await flushPromises()

    expect(router.currentRoute.value.query.account_settings).toBe('account')
  })

  it('cycles focus from the dialog container and excludes mobile-layout hidden controls', async () => {
    await mountDialog('/dashboard?account_settings=general')
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-testid="personal-settings-dialog"]',
    )!
    const desktopClose = document.body.querySelector<HTMLElement>(
      '[data-testid="personal-settings-close-desktop"]',
    )!
    const mobileClose = document.body.querySelector<HTMLElement>(
      '[data-testid="personal-settings-close-mobile"]',
    )!
    // jsdom does not evaluate the component's responsive stylesheet. Mirror
    // the mobile state where the sidebar close control is not rendered.
    desktopClose.style.display = 'none'
    mobileClose.style.display = 'inline-flex'
    const focusable = getVisibleFocusableElements(dialog)
    const first = focusable[0]
    const last = focusable.at(-1)

    expect(first).toBeDefined()
    expect(last).toBeDefined()
    expect(window.getComputedStyle(desktopClose).display).toBe('none')
    expect(focusable).not.toContain(desktopClose)
    expect(focusable).toContain(mobileClose)

    last?.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      bubbles: true,
      cancelable: true,
    }))
    expect(document.activeElement).toBe(first)

    first?.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))
    expect(document.activeElement).toBe(last)

    dialog.focus()
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true,
    }))
    expect(document.activeElement).toBe(last)
  })

  it('scopes secondary detail state to its active category', async () => {
    const { router } = await mountDialog('/dashboard?account_settings=account')

    await bodyElement('open-profile-stub').trigger('click')
    await flushPromises()
    expect(bodyElement('account-panel-stub').attributes('data-detail')).toBe('profile')

    await new DOMWrapper(
      document.body.querySelector<HTMLElement>('[data-settings-section="security"]')!,
    ).trigger('click')
    await flushPromises()

    expect(bodyElement('account-panel-stub').attributes('data-detail')).toBeUndefined()
    expect(bodyElement('security-panel-stub').attributes('data-detail')).toBeUndefined()

    await bodyElement('open-password-stub').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.query.account_settings_detail).toBe('password')
    expect(bodyElement('security-panel-stub').attributes('data-detail')).toBe('password')
    expect(bodyElement('account-panel-stub').attributes('data-detail')).toBeUndefined()
  })

  it('does not mount billing contents until capability loading succeeds and supports retry', async () => {
    const {
      appStore,
      fetchPublicSettings,
    } = await mountDialog('/dashboard?account_settings=billing')
    await flushPromises()

    expect(fetchPublicSettings).toHaveBeenCalledOnce()
    expect(bodyElement('personal-settings-billing-error').attributes('role')).toBe('alert')
    expect(document.body.querySelector('[data-testid="billing-panel-stub"]')).toBeNull()

    const settings = {
      payment_enabled: true,
    } as PublicSettings
    fetchPublicSettings.mockImplementationOnce(async () => {
      appStore.cachedPublicSettings = settings
      appStore.publicSettingsLoaded = true
      return settings
    })
    await bodyElement('personal-settings-billing-error').get('button').trigger('click')
    await flushPromises()

    expect(fetchPublicSettings).toHaveBeenCalledTimes(2)
    expect(bodyElement('billing-panel-stub').exists()).toBe(true)
    expect(document.body.querySelector('[data-testid="personal-settings-billing-error"]')).toBeNull()
  })

  it('locks the host and lets only the top modal respond to Escape', async () => {
    const { router, host } = await mountDialog(
      '/dashboard?foo=kept&account_settings=general#anchor',
    )

    expect(host.inert).toBe(true)
    expect(host.getAttribute('aria-hidden')).toBe('true')
    expect(document.body.style.overflow).toBe('hidden')

    const childToken = Symbol('child-modal')
    registerModalLayer(childToken)
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
    }))
    await flushPromises()
    expect(router.currentRoute.value.query.account_settings).toBe('general')

    unregisterModalLayer(childToken)
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
    }))
    await flushPromises()
    await new Promise((resolve) => setTimeout(resolve, 220))

    expect(router.currentRoute.value.path).toBe('/dashboard')
    expect(router.currentRoute.value.query).toEqual({ foo: 'kept' })
    expect(router.currentRoute.value.hash).toBe('#anchor')
    expect(document.body.querySelector('[data-testid="personal-settings-dialog"]')).toBeNull()
    expect(host.inert).toBe(false)
    expect(host.hasAttribute('aria-hidden')).toBe(false)
    expect(document.body.style.overflow).toBe('')
  })

  it('does not restore focus to body and falls back to the account trigger', async () => {
    expect(document.activeElement).toBe(document.body)
    const {
      accountTrigger,
    } = await mountDialog('/dashboard?account_settings=general')

    expect(document.activeElement).toBe(
      document.body.querySelector('[data-testid="personal-settings-dialog"]'),
    )

    await bodyElement('personal-settings-close-desktop').trigger('click')
    await flushPromises()
    await new Promise((resolve) => setTimeout(resolve, 220))

    expect(document.activeElement).toBe(accountTrigger)
  })
})
