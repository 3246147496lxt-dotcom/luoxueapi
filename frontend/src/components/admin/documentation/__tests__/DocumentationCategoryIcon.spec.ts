import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DocumentationCategoryIcon from '../DocumentationCategoryIcon.vue'
import Icon from '@/components/icons/Icon.vue'

describe('DocumentationCategoryIcon', () => {
  it('prefers a sanitized custom SVG over the built-in fallback', () => {
    const wrapper = mount(DocumentationCategoryIcon, {
      props: {
        icon: 'key',
        iconSvg: '<svg viewBox="0 0 24 24" onload="alert(1)"><script>alert(1)</script><path d="M1 1h4v4z"/></svg>',
      },
    })

    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.find('path').exists()).toBe(true)
    expect(wrapper.html()).not.toContain('onload')
    expect(wrapper.html()).not.toContain('<script')
    expect(wrapper.findComponent(Icon).exists()).toBe(false)
  })

  it('falls back to the mapped built-in icon when custom markup is empty or invalid', async () => {
    const wrapper = mount(DocumentationCategoryIcon, {
      props: { icon: 'client', iconSvg: '<p>not an svg</p>' },
    })

    expect(wrapper.getComponent(Icon).props('name')).toBe('terminal')

    await wrapper.setProps({ icon: 'unknown', iconSvg: '' })
    expect(wrapper.getComponent(Icon).props('name')).toBe('questionCircle')
  })
})
