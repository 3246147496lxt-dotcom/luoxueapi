import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import Pagination from '../Pagination.vue'
import Select from '../Select.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, values?: Record<string, unknown>) =>
      values ? `${key}:${JSON.stringify(values)}` : key
  })
}))

describe('Pagination', () => {
  it('treats an empty result as a single disabled navigation page', () => {
    const wrapper = mount(Pagination, {
      props: {
        total: 0,
        page: 1,
        pageSize: 20
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    expect(wrapper.find('.pagination-button--previous').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.pagination-button--next').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.pagination-status').text()).toContain('"total":1')
  })

  it('uses the page-size options supplied by the caller', () => {
    const wrapper = mount(Pagination, {
      props: {
        total: 40,
        page: 1,
        pageSize: 15,
        pageSizeOptions: [15, 30]
      },
      global: {
        stubs: { Teleport: true }
      }
    })

    expect(wrapper.getComponent(Select).props('options')).toEqual([
      { value: 15, label: '15' },
      { value: 30, label: '30' }
    ])
  })
})
