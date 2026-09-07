import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import PricingEntryCard from '../PricingEntryCard.vue'
import type { PricingFormEntry } from '../types'

const testState = vi.hoisted(() => ({
  getModelDefaultPricing: vi.fn()
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/api/admin/channels', () => ({
  default: {
    getModelDefaultPricing: testState.getModelDefaultPricing
  }
}))

const ModelTagInputStub = defineComponent({
  name: 'ModelTagInput',
  props: {
    models: { type: Array, default: () => [] },
    platform: { type: String, default: '' }
  },
  emits: ['update:models'],
  template: '<button data-testid="model-tag-input" type="button" @click="$emit(\'update:models\', [\'gpt-5.5\'])">add model</button>'
})

const MultiModelTagInputStub = defineComponent({
  props: {
    models: { type: Array, default: () => [] },
    platform: { type: String, default: '' }
  },
  emits: ['update:models'],
  template: '<button data-testid="model-tag-input" type="button" @click="$emit(\'update:models\', [\'gpt-5.5\', \'gpt-5.6-sol\'])">add models</button>'
})

const AppendModelTagInputStub = defineComponent({
  props: {
    models: { type: Array, default: () => ['existing-model'] },
    platform: { type: String, default: '' }
  },
  emits: ['update:models'],
  template: '<button data-testid="model-tag-input" type="button" @click="$emit(\'update:models\', [\'existing-model\', \'new-model\'])">append model</button>'
})

function makeEntry(overrides: Partial<PricingFormEntry> = {}): PricingFormEntry {
  return {
    models: [],
    billing_mode: 'token',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    image_input_price: null,
    image_output_price: null,
    per_request_price: null,
    intervals: [],
    ...overrides
  }
}

describe('PricingEntryCard', () => {
  beforeEach(() => {
    testState.getModelDefaultPricing.mockReset()
  })

  it('multiplies official default token quotes by 70 when auto-filling a new model', async () => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 5e-6,
      output_price: 30e-6,
      cache_write_price: 6.25e-6,
      cache_read_price: 0.5e-6,
      image_input_price: 2e-6,
      image_output_price: 4e-6,
      channel_pricing_multiplier: 70
    })
    const wrapper = mount(PricingEntryCard, {
      props: { entry: makeEntry(), platform: 'openai' },
      global: {
        stubs: {
          Icon: true,
          IntervalRow: true,
          Select: true,
          ModelTagInput: ModelTagInputStub
        }
      }
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    expect(testState.getModelDefaultPricing).toHaveBeenCalledWith('gpt-5.5')
    const updates = wrapper.emitted('update') ?? []
    expect(updates).toHaveLength(2)
    expect(updates[1][0]).toMatchObject({
      models: ['gpt-5.5'],
      input_price: 350,
      output_price: 2100,
      cache_write_price: 437.5,
      cache_read_price: 35,
      image_input_price: 140,
      image_output_price: 280
    })
  })

  it('does not overwrite manually entered channel prices', async () => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 5e-6,
      output_price: 30e-6
    })
    const wrapper = mount(PricingEntryCard, {
      props: {
        entry: makeEntry({ input_price: 123 }),
        platform: 'openai'
      },
      global: {
        stubs: {
          Icon: true,
          IntervalRow: true,
          Select: true,
          ModelTagInput: ModelTagInputStub
        }
      }
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    expect(testState.getModelDefaultPricing).not.toHaveBeenCalled()
    expect(wrapper.emitted('update')).toHaveLength(1)
  })

  it('keeps official prices for account-stats entries when the channel multiplier is disabled', async () => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 5e-6,
      output_price: 30e-6,
      cache_write_price: 6.25e-6,
      cache_read_price: 0.5e-6
    })
    const wrapper = mount(PricingEntryCard, {
      props: {
        entry: makeEntry(),
        platform: 'openai',
        applyChannelPricingMultiplier: false
      },
      global: {
        stubs: {
          Icon: true,
          IntervalRow: true,
          Select: true,
          ModelTagInput: ModelTagInputStub
        }
      }
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    const updates = wrapper.emitted('update') ?? []
    expect(updates[1][0]).toMatchObject({
      input_price: 5,
      output_price: 30,
      cache_write_price: 6.25,
      cache_read_price: 0.5
    })
  })

  it('does not copy the first quote when several models are added at once', async () => {
    const wrapper = mount(PricingEntryCard, {
      props: { entry: makeEntry(), platform: 'openai' },
      global: {
        stubs: {
          Icon: true,
          IntervalRow: true,
          Select: true,
          ModelTagInput: MultiModelTagInputStub
        }
      }
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    expect(testState.getModelDefaultPricing).not.toHaveBeenCalled()
    expect(wrapper.emitted('update')).toHaveLength(1)
    expect(wrapper.emitted('update')?.[0][0]).toMatchObject({
      models: ['gpt-5.5', 'gpt-5.6-sol'],
      input_price: null,
      output_price: null
    })
  })

  it('does not fill an appended model into an existing blank row', async () => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 5e-6,
      output_price: 30e-6
    })
    const wrapper = mount(PricingEntryCard, {
      props: {
        entry: makeEntry({ models: ['existing-model'] }),
        platform: 'openai'
      },
      global: {
        stubs: {
          Icon: true,
          IntervalRow: true,
          Select: true,
          ModelTagInput: AppendModelTagInputStub
        }
      }
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    expect(testState.getModelDefaultPricing).not.toHaveBeenCalled()
    expect(wrapper.emitted('update')).toHaveLength(1)
    expect(wrapper.emitted('update')?.[0][0]).toMatchObject({
      models: ['existing-model', 'new-model'],
      input_price: null,
      output_price: null
    })
  })
})
