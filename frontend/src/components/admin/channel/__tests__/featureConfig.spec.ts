import { describe, expect, it } from 'vitest'
import {
  applyBedrockCCCompatFeatureConfig,
  readBedrockCCCompatFeatureConfig,
} from '../featureConfig'

describe('Bedrock CC compatibility feature config', () => {
  it('serializes the enabled state as the backend boolean contract', () => {
    const featuresConfig: Record<string, unknown> = {
      existing_feature: true,
      bedrock_cc_compat: { anthropic: false },
    }

    applyBedrockCCCompatFeatureConfig(featuresConfig, [
      {
        platform: 'anthropic',
        enabled: true,
        bedrock_cc_compat: true,
      },
    ])

    expect(featuresConfig).toEqual({
      existing_feature: true,
      bedrock_cc_compat: true,
    })
  })

  it('persists an explicit false value for an enabled Anthropic section', () => {
    const featuresConfig: Record<string, unknown> = {
      bedrock_cc_compat: true,
    }

    applyBedrockCCCompatFeatureConfig(featuresConfig, [
      {
        platform: 'anthropic',
        enabled: true,
        bedrock_cc_compat: false,
      },
    ])

    expect(featuresConfig.bedrock_cc_compat).toBe(false)
  })

  it('removes the feature when no Anthropic section is enabled', () => {
    const featuresConfig: Record<string, unknown> = {
      bedrock_cc_compat: true,
    }

    applyBedrockCCCompatFeatureConfig(featuresConfig, [
      {
        platform: 'openai',
        enabled: true,
        bedrock_cc_compat: false,
      },
    ])

    expect(featuresConfig).not.toHaveProperty('bedrock_cc_compat')
  })

  it('reads both the boolean contract and the legacy platform map', () => {
    expect(readBedrockCCCompatFeatureConfig({ bedrock_cc_compat: true })).toBe(true)
    expect(readBedrockCCCompatFeatureConfig({ bedrock_cc_compat: false })).toBe(false)
    expect(readBedrockCCCompatFeatureConfig({
      bedrock_cc_compat: { anthropic: true },
    })).toBe(true)
  })
})
