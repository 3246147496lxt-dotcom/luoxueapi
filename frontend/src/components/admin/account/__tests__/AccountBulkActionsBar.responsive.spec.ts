import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('AccountBulkActionsBar responsive layout', () => {
  it('wraps contextual operations without introducing another primary CTA', async () => {
    const wrapper = mount(AccountBulkActionsBar, { props: { selectedIds: [7, 11] } })

    expect(wrapper.classes()).toEqual(expect.arrayContaining(['flex-col', 'sm:flex-row']))
    expect(wrapper.find('.flex.flex-wrap.gap-2').exists()).toBe(true)
    expect(wrapper.findAll('.btn-primary')).toHaveLength(0)
    expect(wrapper.findAll('button').every(button => button.classes().includes('min-h-11'))).toBe(true)

    const edit = wrapper.findAll('button').find(button => button.text() === 'admin.accounts.bulkActions.edit')
    expect(edit).toBeDefined()
    await edit!.trigger('click')
    expect(wrapper.emitted('edit-selected')).toHaveLength(1)
  })
})
