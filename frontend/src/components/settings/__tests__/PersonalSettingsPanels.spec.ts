import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, shallowMount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { PublicSettings, User } from '@/types'

const {
  getTotpStatus,
  setLocalePreference,
  setThemePreference,
  updateProfile,
} = vi.hoisted(() => ({
  getTotpStatus: vi.fn(),
  setLocalePreference: vi.fn(),
  setThemePreference: vi.fn(),
  updateProfile: vi.fn(),
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

vi.mock('@/api', () => ({
  totpAPI: {
    getStatus: getTotpStatus,
  },
  userAPI: {
    updateProfile,
  },
}))

vi.mock('@/composables/useThemePreference', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useThemePreference: () => ({
      preference: ref('system'),
      setPreference: setThemePreference,
    }),
  }
})

vi.mock('@/composables/useLocalePreference', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useLocalePreference: () => ({
      preference: ref('auto'),
      setPreference: setLocalePreference,
    }),
  }
})

import PersonalSettingsAccountPanel from '../PersonalSettingsAccountPanel.vue'
import PersonalSettingsGeneralPanel from '../PersonalSettingsGeneralPanel.vue'
import PersonalSettingsNotificationsPanel from '../PersonalSettingsNotificationsPanel.vue'
import PersonalSettingsSecurityPanel from '../PersonalSettingsSecurityPanel.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import SettingsChoiceMenu from '../SettingsChoiceMenu.vue'
import { useAppStore } from '@/stores/app'

const directory = dirname(fileURLToPath(import.meta.url))
const settingsStyles = readFileSync(resolve(directory, '../personal-settings.css'), 'utf8')
const walletPanelSource = readFileSync(
  resolve(directory, '../../layout/WalletSubscriptionSettingsPanel.vue'),
  'utf8',
)
const wrappers = new Set<VueWrapper>()

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

const publicSettings = {
  balance_low_notify_enabled: true,
  balance_low_notify_threshold: 10,
  linuxdo_oauth_enabled: true,
} as PublicSettings

function track<T extends VueWrapper>(wrapper: T): T {
  wrappers.add(wrapper)
  return wrapper
}

function iconStub() {
  return {
    props: ['name'],
    template: '<span :data-icon="name" />',
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  getTotpStatus.mockReset()
  getTotpStatus.mockResolvedValue({
    feature_enabled: true,
    enabled: false,
  })
  updateProfile.mockReset()
  updateProfile.mockResolvedValue({ ...user })
  setLocalePreference.mockReset()
  setThemePreference.mockReset()
})

afterEach(() => {
  for (const wrapper of wrappers) wrapper.unmount()
  wrappers.clear()
  vi.restoreAllMocks()
})

describe('PersonalSettingsGeneralPanel', () => {
  it('writes the selected three-state theme and locale preferences', async () => {
    const wrapper = track(shallowMount(PersonalSettingsGeneralPanel))
    const menus = wrapper.findAllComponents(SettingsChoiceMenu)

    expect(menus).toHaveLength(2)
    expect(menus[0]?.props('modelValue')).toBe('system')
    expect(menus[0]?.props('options').map((option: { value: string }) => option.value))
      .toEqual(['system', 'light', 'dark'])
    expect(menus[1]?.props('modelValue')).toBe('auto')
    expect(menus[1]?.props('options').map((option: { value: string }) => option.value))
      .toEqual(['auto', 'zh', 'en'])

    menus[0]?.vm.$emit('update:modelValue', 'dark')
    menus[1]?.vm.$emit('update:modelValue', 'zh')
    await wrapper.vm.$nextTick()

    expect(setThemePreference).toHaveBeenCalledWith('dark')
    expect(setLocalePreference).toHaveBeenCalledWith('zh')
  })
})

describe('Personal settings category introductions', () => {
  it('does not repeat the active category title and description inside the panel body', () => {
    const panels = [
      track(shallowMount(PersonalSettingsGeneralPanel)),
      track(shallowMount(PersonalSettingsAccountPanel, {
        props: {
          user,
          publicSettings,
          detail: null,
          loading: false,
          error: false,
        },
      })),
      track(shallowMount(PersonalSettingsSecurityPanel, {
        props: { detail: null },
      })),
      track(shallowMount(PersonalSettingsNotificationsPanel, {
        props: {
          user,
          publicSettings,
          detail: null,
          loading: false,
          error: false,
        },
      })),
    ]

    for (const panel of panels) {
      expect(panel.element.tagName).toBe('DIV')
      expect(panel.attributes('aria-labelledby')).toBeUndefined()
      expect(panel.find('.personal-settings-panel-heading').exists()).toBe(false)
    }

    expect(panels.map((panel) => (
      panel.findAll('.personal-settings-row-description').map((row) => row.text())
    ))).toEqual([
      [
        'personalSettings.general.appearanceHint',
        'personalSettings.general.languageHint',
      ],
      [
        'personalSettings.account.profileHint',
        'personalSettings.account.connectionsHint',
        'personalSettings.account.desktopDevicesHint',
      ],
      [
        'personalSettings.security.passwordHint',
        'personalSettings.security.totpHint',
      ],
      [
        'personalSettings.notifications.balanceAlertHint',
        'personalSettings.notifications.recipientsHint',
      ],
    ])
  })
})

describe('Personal settings divider hierarchy', () => {
  it('removes only the first content divider while preserving row and section separators', () => {
    expect(settingsStyles).toContain(
      '.personal-settings-panel > .personal-settings-group:first-child {\n  border-top: 0;\n}',
    )
    expect(settingsStyles).toContain(
      '.personal-settings-group {\n  border-top: 1px solid var(--lx-clay-border);\n}',
    )
    expect(settingsStyles).toContain(
      'border-bottom: 1px solid var(--lx-clay-border);',
    )
    expect(walletPanelSource).toContain(
      '.wallet-settings__section:first-child {\n  border-top: 0;\n}',
    )
    expect(walletPanelSource).toContain(
      'border-top: 1px solid rgb(226 232 240 / 0.86);',
    )
  })
})

describe('PersonalSettingsAccountPanel', () => {
  it('renders cached identity during refresh failure and exposes both secondary editors', async () => {
    const wrapper = track(shallowMount(PersonalSettingsAccountPanel, {
      props: {
        user,
        publicSettings,
        detail: null,
        loading: false,
        error: true,
      },
      global: {
        stubs: { Icon: iconStub() },
      },
    }))

    expect(wrapper.get('[data-testid="personal-settings-account-error"]').attributes('role'))
      .toBe('alert')
    expect(wrapper.text()).toContain('Riley Quinn')

    await wrapper.get('[data-testid="personal-settings-open-profile"]').trigger('click')
    await wrapper.get('[data-testid="personal-settings-open-connections"]').trigger('click')
    await wrapper.get('[data-testid="personal-settings-account-error"] button').trigger('click')

    expect(wrapper.emitted('open-detail')).toEqual([['profile'], ['connections']])
    expect(wrapper.emitted('retry')).toEqual([[]])
  })

  it('keeps the bindings editor inside the account detail surface', () => {
    const wrapper = track(shallowMount(PersonalSettingsAccountPanel, {
      props: {
        user,
        publicSettings,
        detail: 'connections',
        loading: false,
        error: false,
      },
    }))

    const bindings = wrapper.getComponent({ name: 'ProfileIdentityBindingsSection' })
    expect(bindings.props('embedded')).toBe(true)
    expect(bindings.props('compact')).toBe(true)
    expect(bindings.props('linuxdoEnabled')).toBe(true)
  })

  it('does not misreport unavailable connection capabilities after a load failure', () => {
    const wrapper = track(shallowMount(PersonalSettingsAccountPanel, {
      props: {
        user,
        publicSettings: null,
        detail: 'connections',
        loading: false,
        error: true,
      },
    }))

    expect(wrapper.get('[data-testid="personal-settings-connections-error"]').attributes('role'))
      .toBe('alert')
    expect(wrapper.findComponent({ name: 'ProfileIdentityBindingsSection' }).exists()).toBe(false)
  })
})

describe('PersonalSettingsSecurityPanel', () => {
  it('loads TOTP status only for the first-level summary and supports retry', async () => {
    getTotpStatus.mockRejectedValueOnce(new Error('network unavailable'))
    const wrapper = track(shallowMount(PersonalSettingsSecurityPanel, {
      props: { detail: null },
      global: { stubs: { Icon: iconStub() } },
    }))
    await flushPromises()

    expect(getTotpStatus).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-testid="personal-settings-security-error"]').attributes('role'))
      .toBe('alert')

    getTotpStatus.mockResolvedValueOnce({
      feature_enabled: true,
      enabled: true,
    })
    await wrapper.get('[data-testid="personal-settings-security-error"] button').trigger('click')
    await flushPromises()

    expect(getTotpStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('personalSettings.security.totpEnabled')
  })

  it('does not duplicate the secondary editor own security request', async () => {
    const wrapper = track(shallowMount(PersonalSettingsSecurityPanel, {
      props: { detail: 'totp' },
    }))
    await flushPromises()

    expect(getTotpStatus).not.toHaveBeenCalled()
    expect(wrapper.getComponent({ name: 'ProfileTotpCard' }).props('embedded')).toBe(true)

    await wrapper.setProps({ detail: null })
    await flushPromises()
    expect(getTotpStatus).toHaveBeenCalledOnce()
  })
})

describe('PersonalSettingsNotificationsPanel', () => {
  it('marks the balance threshold summary as Points', () => {
    const wrapper = track(shallowMount(PersonalSettingsNotificationsPanel, {
      props: {
        user,
        publicSettings,
        detail: null,
        loading: false,
        error: false,
      },
    }))

    const threshold = wrapper.getComponent(CreditAmount)
    expect(threshold.props('value')).toBe(
      'personalSettings.notifications.systemDefault:{"value":"10.00"}',
    )
    expect(threshold.props('iconSize')).toBe('xs')
  })

  it('distinguishes a disabled system capability from a loading failure', () => {
    const disabled = track(shallowMount(PersonalSettingsNotificationsPanel, {
      props: {
        user,
        publicSettings: {
          ...publicSettings,
          balance_low_notify_enabled: false,
        },
        detail: null,
        loading: false,
        error: false,
      },
    }))
    expect(disabled.find('[data-testid="personal-settings-notifications-disabled"]').exists())
      .toBe(true)

    const failed = track(shallowMount(PersonalSettingsNotificationsPanel, {
      props: {
        user,
        publicSettings: null,
        detail: null,
        loading: false,
        error: true,
      },
    }))
    expect(failed.get('[data-testid="personal-settings-notifications-error"]').attributes('role'))
      .toBe('alert')
    expect(failed.find('[data-testid="personal-settings-notifications-disabled"]').exists())
      .toBe(false)
  })

  it('rolls back a failed quick toggle and reports it through the app toast', async () => {
    updateProfile.mockRejectedValueOnce(new Error('save failed'))
    const pinia = createPinia()
    setActivePinia(pinia)
    const appStore = useAppStore()
    const showError = vi.spyOn(appStore, 'showError')
    const wrapper = track(shallowMount(PersonalSettingsNotificationsPanel, {
      props: {
        user,
        publicSettings,
        detail: null,
        loading: false,
        error: false,
      },
      global: {
        plugins: [pinia],
        stubs: { Icon: iconStub() },
      },
    }))
    const toggle = wrapper.get<HTMLInputElement>(
      '[data-testid="personal-settings-notification-toggle"]',
    )

    await toggle.setValue(false)
    await flushPromises()

    expect(updateProfile).toHaveBeenCalledWith({
      balance_notify_enabled: false,
    })
    expect(toggle.element.checked).toBe(true)
    expect(showError).toHaveBeenCalledOnce()
  })
})
