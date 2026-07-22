import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTableActions from '../AccountTableActions.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const mountActions = () =>
  mount(AccountTableActions, {
    props: { loading: false },
    slots: {
      after: '<button data-testid="secondary-action">secondary</button>'
    },
    global: {
      stubs: { Icon: true }
    }
  })

describe('AccountTableActions responsive disclosure', () => {
  it('keeps account creation as the only primary page action', () => {
    const wrapper = mountActions()
    const primaryActions = wrapper.findAll('.btn-primary')

    expect(primaryActions).toHaveLength(1)
    expect(primaryActions[0].text()).toBe('admin.accounts.createAccount')
    expect(primaryActions[0].classes()).toEqual(expect.arrayContaining(['order-1', 'min-h-11', 'lg:order-none']))
  })

  it('collapses low-frequency actions on mobile and exposes accessible state', async () => {
    const wrapper = mountActions()
    const toggle = wrapper.get('[data-testid="account-actions-toggle"]')
    const secondary = wrapper.get('[data-testid="account-secondary-actions"]')

    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('account-secondary-actions')
    expect(toggle.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11', 'lg:hidden']))
    expect(secondary.classes()).toEqual(expect.arrayContaining(['hidden', 'lg:contents']))

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(secondary.classes()).toContain('flex')
    expect(secondary.get('[data-testid="secondary-action"]').exists()).toBe(true)
  })

  it('keeps refresh directly available with an accessible 44px target', async () => {
    const wrapper = mountActions()
    const refresh = wrapper.get('button[aria-label="common.refresh"]')

    expect(refresh.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11']))
    await refresh.trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
})
