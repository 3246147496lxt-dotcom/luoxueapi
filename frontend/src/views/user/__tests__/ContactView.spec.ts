import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ContactView from '@/views/user/ContactView.vue'

describe('ContactView', () => {
  it('renders the four configured support channels and live actions', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })

    const wrapper = mount(ContactView, {
      global: {
        stubs: { Icon: { template: '<span />' } },
      },
    })

    expect(wrapper.find('[data-testid="contact-page"]').exists()).toBe(true)
    expect(wrapper.findAll('.contact-card')).toHaveLength(4)
    expect(wrapper.find('[data-testid="contact-card-qq"]').text()).toContain('2456772148')
    expect(wrapper.find('[data-testid="contact-card-qqGroup"]').text()).toContain('1072675573')
    expect(wrapper.find('[data-testid="contact-card-wechat"]').text()).toContain('Xxxttt_111')
    expect(wrapper.find('[data-testid="contact-card-wechat"] img').attributes('src')).toBe('/contact-wechat.jpg')

    const telegram = wrapper.find('[data-testid="contact-card-telegram"] a')
    expect(telegram.attributes('href')).toBe('https://t.me/+g4eFhwpWMukyNGVl')
    expect(telegram.attributes('target')).toBe('_blank')
    expect(wrapper.findAll('button:disabled')).toHaveLength(0)

    const copyButton = wrapper.find('[aria-label="复制QQ号"]')
    await copyButton.trigger('click')
    expect(writeText).toHaveBeenCalledWith('2456772148')
    expect(copyButton.attributes('aria-pressed')).toBe('true')
  })
})
