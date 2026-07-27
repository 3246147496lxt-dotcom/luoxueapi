import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import PaymentHelpPanel from '../PaymentHelpPanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('PaymentHelpPanel', () => {
  it('opens an accessible image dialog and restores focus after Escape', async () => {
    wrapper = mount(PaymentHelpPanel, {
      attachTo: document.body,
      props: {
        text: 'Use this payment guide.',
        imageUrl: 'https://example.com/help.png',
      },
      global: {
        stubs: {
          Teleport: true,
          Transition: { props: ['name'], template: '<slot />' },
        },
      },
    })

    const trigger = wrapper.get('button[aria-label="payment.previewHelpImage"]')
    await trigger.trigger('click')
    await flushPromises()

    const dialog = wrapper.get('[role="dialog"]')
    const closeButton = dialog.get('button[aria-label="common.close"]')
    expect(dialog.attributes('aria-modal')).toBe('true')
    expect(dialog.attributes('aria-label')).toBe('payment.helpImagePreview')
    expect(dialog.get('img').attributes('alt')).toBe('payment.helpImagePreview')
    expect(document.activeElement).toBe(closeButton.element)

    dialog.element.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    }))
    await flushPromises()

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('click')
    await flushPromises()
    await wrapper.get('[role="dialog"] button[aria-label="common.close"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
  })

  it('renders text without an empty image control', () => {
    wrapper = mount(PaymentHelpPanel, {
      props: { text: 'Text-only payment help.' },
      global: {
        stubs: {
          Teleport: true,
          Transition: { props: ['name'], template: '<slot />' },
        },
      },
    })

    expect(wrapper.text()).toContain('Text-only payment help.')
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })
})
