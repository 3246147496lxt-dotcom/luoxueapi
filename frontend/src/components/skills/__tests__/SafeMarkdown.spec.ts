import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import SafeMarkdown from '../SafeMarkdown.vue'

describe('SafeMarkdown', () => {
  it('allows only http(s) links and adds defensive rel attributes', () => {
    const wrapper = mount(SafeMarkdown, {
      props: {
        content: [
          '[Safe](https://example.com/docs)',
          '[Script](javascript:alert(1))',
          '[Data](data:text/html,bad)',
          '<iframe src="https://evil.example"></iframe>',
          '<img src=x onerror="alert(1)">',
        ].join('\n\n'),
      },
    })

    const safeLink = wrapper.get('a[href="https://example.com/docs"]')
    expect(safeLink.attributes('rel')).toBe('noopener noreferrer nofollow')
    expect(wrapper.html()).not.toContain('javascript:')
    expect(wrapper.html()).not.toContain('data:text')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.html()).not.toContain('onerror')
  })
})
