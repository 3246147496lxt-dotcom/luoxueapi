import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import AdminPageHeader from '../AdminPageHeader.vue'

describe('AdminPageHeader', () => {
  it('renders one semantic page heading with supporting copy', () => {
    const wrapper = mount(AdminPageHeader, {
      props: {
        title: 'User Management',
        description: 'Manage users and their permissions'
      }
    })

    expect(wrapper.element.tagName).toBe('HEADER')
    expect(wrapper.findAll('h1')).toHaveLength(1)
    expect(wrapper.get('h1').text()).toBe('User Management')
    expect(wrapper.text()).toContain('Manage users and their permissions')
  })

  it('keeps metadata and primary/secondary actions in explicit regions', () => {
    const wrapper = mount(AdminPageHeader, {
      props: { title: 'Accounts' },
      slots: {
        meta: '<span data-testid="meta">24 accounts</span>',
        'secondary-actions': '<button data-testid="secondary">Refresh</button>',
        'primary-actions': '<button data-testid="primary">Create</button>'
      }
    })

    expect(wrapper.get('[data-testid="meta"]').text()).toBe('24 accounts')
    expect(wrapper.get('[data-testid="secondary"]').text()).toBe('Refresh')
    expect(wrapper.get('[data-testid="primary"]').text()).toBe('Create')
  })
})
