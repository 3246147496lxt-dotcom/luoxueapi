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
      after: `
        <button class="account-toolbar-icon-button" data-testid="auto-refresh-action" aria-label="auto refresh"></button>
        <button class="account-toolbar-icon-button" data-testid="columns-action" aria-label="columns"></button>
        <button class="account-toolbar-icon-button" data-testid="more-action" aria-label="more"></button>
      `
    },
    global: {
      stubs: { Icon: true }
    }
  })

describe('AccountTableActions responsive disclosure', () => {
  it('renders refresh and after tools as one always-visible icon group', () => {
    const wrapper = mountActions()
    const group = wrapper.get('.account-table-actions')
    const buttons = group.findAll('button')

    expect(buttons).toHaveLength(4)
    expect(buttons[0]?.attributes('aria-label')).toBe('common.refresh')
    expect(buttons[0]?.classes()).toContain('account-table-action-button')
    expect(buttons.slice(1).every(button => button.classes().includes('account-toolbar-icon-button'))).toBe(true)
    expect(wrapper.find('[data-testid="account-actions-toggle"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-secondary-actions"]').exists()).toBe(false)
    expect(wrapper.find('.btn-primary').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.accounts.createAccount')
  })

  it('keeps the 44px refresh action accessible and emits refresh directly', async () => {
    const wrapper = mountActions()
    const refresh = wrapper.get('button[aria-label="common.refresh"]')

    expect(refresh.attributes('title')).toBe('common.refresh')
    expect(refresh.classes()).toContain('account-table-action-button')
    await refresh.trigger('click')
    expect(wrapper.emitted('refresh')).toHaveLength(1)
  })
})
