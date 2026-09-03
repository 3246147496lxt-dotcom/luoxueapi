import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import ModelIcon from '../ModelIcon.vue'

describe('ModelIcon', () => {
  it.each(['openai', ' OpenAI ', 'oai', 'o', 'openai/gpt-5.5'])('renders the OpenAI mark for the %s alias', (model) => {
    const wrapper = mount(ModelIcon, { props: { model } })

    expect(wrapper.find('img.model-icon-image').exists()).toBe(true)
    expect(wrapper.find('img.model-icon-image').attributes('src')).toContain('openai-mark')
    expect(wrapper.find('img.model-icon-image').attributes('style')).toContain('width: 18px')
    expect(wrapper.find('img.model-icon-image').attributes('style')).toContain('height: 18px')
    expect(wrapper.find('.model-icon-fallback').exists()).toBe(false)
  })

  it.each([
    'anthropic',
    ' Anthropic ',
    'anthropic/claude-sonnet-5',
    'claude-opus-4-6',
  ])('renders the supplied Claude mark for the %s alias', (model) => {
    const wrapper = mount(ModelIcon, { props: { model } })

    const icon = wrapper.find('img.model-icon-image')
    expect(icon.exists()).toBe(true)
    expect(icon.attributes('src')).toContain('claude-mark')
    expect(icon.attributes('style')).toContain('width: 18px')
    expect(icon.attributes('style')).toContain('height: 18px')
    expect(wrapper.find('svg.model-icon').exists()).toBe(false)
    expect(wrapper.find('.model-icon-fallback').exists()).toBe(false)
  })

  it('keeps the letter fallback for an unknown model', () => {
    const wrapper = mount(ModelIcon, { props: { model: 'unknown-model' } })

    expect(wrapper.find('svg.model-icon').exists()).toBe(false)
    expect(wrapper.find('.model-icon-fallback').text()).toBe('U')
  })
})
