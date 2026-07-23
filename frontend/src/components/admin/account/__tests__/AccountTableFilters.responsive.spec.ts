import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue', 'change'],
  template: '<div class="select-stub"></div>'
}

const SearchInputStub = {
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue', 'search'],
  template: '<div class="search-input-stub"></div>'
}

const mountFilters = (filters: Record<string, unknown> = {}) =>
  mount(AccountTableFilters, {
    props: {
      searchQuery: '',
      filters: {
        platform: '',
        type: '',
        status: '',
        privacy_mode: '',
        group: '',
        ...filters
      },
      groups: []
    },
    global: {
      stubs: {
        Select: SelectStub,
        SearchInput: SearchInputStub,
        Icon: true
      }
    }
  })

describe('AccountTableFilters responsive disclosure', () => {
  it('keeps search and status outside the collapsed secondary filters', () => {
    const wrapper = mountFilters()
    const secondary = wrapper.get('[data-testid="account-secondary-filters"]')

    expect(wrapper.find('[data-testid="account-search-filter"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="account-status-filter"]').exists()).toBe(true)
    expect(secondary.find('[data-testid="account-search-filter"]').exists()).toBe(false)
    expect(secondary.find('[data-testid="account-status-filter"]').exists()).toBe(false)
    expect(secondary.classes()).toContain('hidden')
    expect(secondary.classes()).toContain('lg:contents')
  })

  it('expands secondary filters with accessible state and a 44px touch target', async () => {
    const wrapper = mountFilters({ platform: 'openai', group: 'ungrouped' })
    const toggle = wrapper.get('[data-testid="account-filters-toggle"]')

    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('account-secondary-filters')
    expect(toggle.classes()).toEqual(expect.arrayContaining(['min-h-11', 'min-w-11', 'lg:hidden']))
    expect(toggle.text()).toContain('2')

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('[data-testid="account-secondary-filters"]').classes()).toContain('grid')
  })

  it('preserves the original desktop filter order through responsive order classes', () => {
    const wrapper = mountFilters()

    expect(wrapper.get('[data-testid="account-search-filter"]').classes()).toContain('lg:order-1')
    expect(wrapper.get('[data-testid="account-platform-filter"]').classes()).toContain('lg:order-2')
    expect(wrapper.get('[data-testid="account-type-filter"]').classes()).toContain('lg:order-3')
    expect(wrapper.get('[data-testid="account-status-filter"]').classes()).toContain('lg:order-4')
    expect(wrapper.get('[data-testid="account-privacy-filter"]').classes()).toContain('lg:order-5')
    expect(wrapper.get('[data-testid="account-group-filter"]').classes()).toContain('lg:order-6')
  })
})
