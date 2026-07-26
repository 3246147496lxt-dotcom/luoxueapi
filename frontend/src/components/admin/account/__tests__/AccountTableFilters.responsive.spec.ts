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
  it('keeps search visible and opens advanced filters as an accessible popover', async () => {
    const wrapper = mountFilters()
    const toggle = wrapper.get('[data-testid="account-filters-toggle"]')

    expect(wrapper.find('[data-testid="account-search-filter"]').exists()).toBe(true)
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(toggle.attributes('aria-controls')).toBe('account-secondary-filters')
    expect(toggle.text()).toContain('admin.accounts.advancedFilters')
    expect(wrapper.find('[data-testid="account-secondary-filters"]').exists()).toBe(false)

    await toggle.trigger('click')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    const secondary = wrapper.get('[data-testid="account-secondary-filters"]')
    expect(secondary.attributes('id')).toBe('account-secondary-filters')
    expect(secondary.classes()).toContain('account-filter-popover')

    await secondary.trigger('keydown', { key: 'Escape' })
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="account-secondary-filters"]').exists()).toBe(false)
  })

  it('keeps all five secondary filters inside the advanced popover', async () => {
    const wrapper = mountFilters()
    await wrapper.get('[data-testid="account-filters-toggle"]').trigger('click')

    const secondary = wrapper.get('[data-testid="account-secondary-filters"]')
    const filterTestIds = [
      'account-platform-filter',
      'account-type-filter',
      'account-status-filter',
      'account-privacy-filter',
      'account-group-filter'
    ]

    expect(secondary.findAll('.select-stub')).toHaveLength(5)
    for (const testId of filterTestIds) {
      expect(secondary.find(`[data-testid="${testId}"]`).exists()).toBe(true)
    }
    expect(secondary.find('[data-testid="account-search-filter"]').exists()).toBe(false)
  })

  it('shows applied filter chips and supports clearing one or all filters', async () => {
    const wrapper = mountFilters({
      platform: 'openai',
      type: 'oauth',
      status: 'active',
      privacy_mode: 'training_off',
      group: 'ungrouped'
    })
    const applied = wrapper.get('[data-testid="account-applied-filters"]')
    const chips = applied.findAll('.account-filter-chip')

    expect(chips).toHaveLength(5)
    expect(chips[0]?.text()).toContain('OpenAI')
    expect(chips[0]?.classes()).toContain('account-filter-chip--openai')
    expect(wrapper.get('[data-testid="account-filters-toggle"]').text()).toContain('5')

    await chips[0]?.trigger('click')

    const updatesAfterChip = wrapper.emitted('update:filters')
    expect(updatesAfterChip?.[0]?.[0]).toEqual(expect.objectContaining({ platform: '' }))
    expect(wrapper.emitted('change')).toHaveLength(1)

    await applied.get('.account-applied-filters__clear').trigger('click')

    const updatesAfterClearAll = wrapper.emitted('update:filters')
    expect(updatesAfterClearAll?.at(-1)?.[0]).toEqual(expect.objectContaining({
      platform: '',
      type: '',
      status: '',
      privacy_mode: '',
      group: ''
    }))
    expect(wrapper.emitted('change')).toHaveLength(2)
  })
})
