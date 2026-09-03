import { describe, expect, it } from 'vitest'
import {
  CHAT_PRODUCT_MODELS,
  DEFAULT_CHAT_PRODUCT_MODEL_ID,
} from '../chatProductModels'

describe('Web Chat product model configuration', () => {
  it('keeps the fixed picker order and Sol default independent from runtime APIs', () => {
    expect(CHAT_PRODUCT_MODELS.map(({ id }) => id)).toEqual([
      'gpt-5.6-sol',
      'gpt-5.5',
      'gpt-5.6-luna',
      'gpt-5.6-terra',
    ])
    expect(DEFAULT_CHAT_PRODUCT_MODEL_ID).toBe('gpt-5.6-sol')
    expect(CHAT_PRODUCT_MODELS.filter(({ recommended }) => recommended)).toHaveLength(1)
  })

  it('declares stable product capabilities while leaving reasoning to runtime capability data', () => {
    expect(CHAT_PRODUCT_MODELS.every(({ supports_vision }) => supports_vision)).toBe(true)
    expect(CHAT_PRODUCT_MODELS.every(({ supports_reasoning_slider }) => (
      supports_reasoning_slider === undefined
    ))).toBe(true)
  })
})
