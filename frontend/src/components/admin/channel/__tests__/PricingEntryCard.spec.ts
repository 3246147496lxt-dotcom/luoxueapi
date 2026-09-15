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

const MiniMaxModelTagInputStub = defineComponent({
  emits: ['update:models'],
  template: '<button data-testid="model-tag-input" type="button" @click="$emit(\'update:models\', [\'minimax-m3\'])">add model</button>'
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

  it('uses the 1x channel multiplier when auto-filling a new model', async () => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 5e-6,
      output_price: 30e-6,
      cache_write_price: 6.25e-6,
      cache_read_price: 0.5e-6,
      image_input_price: 2e-6,
      image_output_price: 4e-6,
      channel_pricing_multiplier: 1
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
      input_price: 5,
      output_price: 30,
      cache_write_price: 6.25,
      cache_read_price: 0.5,
      image_input_price: 2,
      image_output_price: 4
    })
  })

  it.each([true, false])('auto-fills MiniMax-M3 tiers and preserves unknown cache writes (channel multiplier: %s)', async (applyChannelPricingMultiplier) => {
    testState.getModelDefaultPricing.mockResolvedValue({
      found: true,
      input_price: 0.3e-6,
      output_price: 1.2e-6,
      cache_write_price: null,
      cache_read_price: 0.06e-6,
      channel_pricing_multiplier: 2,
      intervals: [
        { min_tokens: 0, max_tokens: 524288, tier_label: '', input_price: 0.3e-6, output_price: 1.2e-6, cache_write_price: null, cache_read_price: 0.06e-6, per_request_price: null, sort_order: 0 },
        { min_tokens: 524288, max_tokens: null, tier_label: 'long_context', input_price: 0.6e-6, output_price: 2.4e-6, cache_write_price: null, cache_read_price: 0.12e-6, per_request_price: null, sort_order: 1 },
      ],
    })
    const wrapper = mount(PricingEntryCard, {
      props: { entry: makeEntry(), platform: 'openai', applyChannelPricingMultiplier },
      global: { stubs: { Icon: true, IntervalRow: true, Select: true, ModelTagInput: MiniMaxModelTagInputStub } },
    })

    await wrapper.get('[data-testid="model-tag-input"]').trigger('click')
    await flushPromises()

    expect(testState.getModelDefaultPricing).toHaveBeenCalledWith('minimax-m3')
    const updated = wrapper.emitted('update')?.[1][0] as PricingFormEntry
    const multiplier = applyChannelPricingMultiplier ? 2 : 1
    expect(updated.cache_write_price).toBeNull()
    expect(updated.intervals).toHaveLength(2)
    expect(updated.intervals[0]).toMatchObject({ min_tokens: 0, max_tokens: 524288, input_price: 0.3 * multiplier, cache_write_price: null })
    expect(updated.intervals[1]).toMatchObject({ min_tokens: 524288, max_tokens: null, input_price: 0.6 * multiplier, output_price: 2.4 * multiplier, cache_read_price: 0.12 * multiplier, cache_write_price: null })
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
