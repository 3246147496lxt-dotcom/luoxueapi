import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import StatusBadge from '../StatusBadge.vue'

describe('StatusBadge', () => {
  it.each(['disabled', 'inactive'])('uses a neutral marker for %s', (status) => {
    const wrapper = mount(StatusBadge, {
      props: { status, label: status }
    })

    expect(wrapper.find('.status-badge__dot').classes()).toContain('status-badge__dot--neutral')
    expect(wrapper.find('.status-badge__dot').classes()).not.toContain('status-badge__dot--warning')
  })

  it('reserves amber for warning states', () => {
    const wrapper = mount(StatusBadge, {
      props: { status: 'warning', label: 'Warning' }
    })

    expect(wrapper.find('.status-badge__dot').classes()).toContain('status-badge__dot--warning')
  })

  it('uses the success semantic marker for active states', () => {
    const wrapper = mount(StatusBadge, {
      props: { status: 'active', label: 'Active' }
    })

    expect(wrapper.find('.status-badge__dot').classes()).toContain('status-badge__dot--success')
    expect(wrapper.attributes('data-status')).toBe('active')
  })
})
