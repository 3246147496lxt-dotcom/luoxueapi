import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import UserAccountCard from '../UserAccountCard.vue'

describe('UserAccountCard', () => {
  it.each([
    { context: 'work' as const, collapsed: false },
    { context: 'work' as const, collapsed: true },
    { context: 'chat' as const, collapsed: false },
    { context: 'chat' as const, collapsed: true },
  ])('delegates the shared $context account surface to the canonical dock', ({ context, collapsed }) => {
    const wrapper = mount(UserAccountCard, {
      props: { context, collapsed },
      global: {
        stubs: {
          SidebarAccountDock: {
            props: ['context', 'collapsed'],
            template: '<div data-testid="dock" :data-context="context" :data-collapsed="String(collapsed)" />',
          },
        },
      },
    })

    const dock = wrapper.get('[data-testid="dock"]')
    expect(dock.attributes('data-context')).toBe(context)
    expect(dock.attributes('data-collapsed')).toBe(String(collapsed))
  })
})
