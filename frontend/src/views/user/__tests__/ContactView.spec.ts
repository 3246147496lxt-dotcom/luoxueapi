import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ContactView from '@/views/user/ContactView.vue'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('ContactView', () => {
  it('renders the four configured support channels with placeholder actions', () => {
    const wrapper = mount(ContactView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: { template: '<span />' },
        },
      },
    })

    expect(wrapper.find('[data-testid="contact-page"]').exists()).toBe(true)
    expect(wrapper.findAll('.contact-card')).toHaveLength(4)
    expect(wrapper.find('[data-testid="contact-card-qq"]').text()).toContain('contact.channels.qq.title')
    expect(wrapper.find('[data-testid="contact-card-qqGroup"]').text()).toContain('contact.channels.qqGroup.title')
    expect(wrapper.find('[data-testid="contact-card-wechat"]').text()).toContain('contact.channels.wechat.title')
    expect(wrapper.find('[data-testid="contact-card-telegram"]').text()).toContain('contact.channels.telegram.title')
    expect(wrapper.findAll('button:disabled')).toHaveLength(4)
  })
})
