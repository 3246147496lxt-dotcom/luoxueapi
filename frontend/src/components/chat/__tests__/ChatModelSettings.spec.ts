import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import ChatModelSettings from '../ChatModelSettings.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

let wrapper: VueWrapper | undefined

function panelElement(selector: string): HTMLElement {
  const element = document.body.querySelector<HTMLElement>(selector)
  if (!element) throw new Error(`Missing teleported element: ${selector}`)
  return element
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('ChatModelSettings', () => {
  it('uses one compact trigger for the selected model and reasoning effort', () => {
    wrapper = mount(ChatModelSettings, {
      attachTo: document.body,
      props: {
        modelValue: 'gpt-5.6-sol',
        reasoningEffort: 'xhigh',
        modelOptions: [
          { value: 'gpt-5.6-sol', label: 'GPT-5.6 Sol' },
        ],
      },
      global: {
        stubs: {
          Icon: IconStub,
          Transition: true,
        },
      },
    })

    const trigger = wrapper.get('[data-test="chat-model-settings-trigger"]')
    expect(trigger.text()).toContain('GPT-5.6 Sol')
    expect(trigger.text()).toContain('chat.settings.reasoningLevels.xhigh')
  })

  it('selects a model and reasoning level from the layered menu', async () => {
    wrapper = mount(ChatModelSettings, {
      attachTo: document.body,
      props: {
        modelValue: 'gpt-5',
        reasoningEffort: '',
        modelOptions: [
          { value: 'gpt-5', label: 'GPT-5' },
          { value: 'gpt-5.6-sol', label: 'GPT-5.6 Sol', recommended: true },
        ],
      },
      global: {
        stubs: {
          Icon: IconStub,
          Transition: true,
        },
      },
    })

    await wrapper.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    panelElement('[data-test="chat-settings-model-menu"]').click()
    await nextTick()
    panelElement('[role="menuitemradio"][data-value="gpt-5.6-sol"]').click()
    await nextTick()

    expect(wrapper.emitted('update:modelValue')).toEqual([['gpt-5.6-sol']])
    expect(document.body.querySelector('.chat-model-settings-popover')).toBeNull()

    await wrapper.get('[data-test="chat-model-settings-trigger"]').trigger('click')
    panelElement('[data-test="chat-settings-reasoning-menu"]').click()
    await nextTick()
    panelElement('[role="menuitemradio"][data-value="high"]').click()
    await nextTick()

    expect(wrapper.emitted('update:reasoningEffort')).toEqual([['high']])
  })
})
