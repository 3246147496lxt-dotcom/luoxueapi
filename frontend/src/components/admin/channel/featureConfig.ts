interface BedrockCCCompatSection {
  platform: string
  enabled: boolean
  bedrock_cc_compat: boolean
}

export function applyBedrockCCCompatFeatureConfig(
  featuresConfig: Record<string, unknown>,
  sections: readonly BedrockCCCompatSection[],
): void {
  const anthropicSection = sections.find(
    section => section.enabled && section.platform === 'anthropic',
  )

  if (!anthropicSection) {
    delete featuresConfig.bedrock_cc_compat
    return
  }

  // The backend contract is a single channel-level boolean.
  featuresConfig.bedrock_cc_compat = !!anthropicSection.bedrock_cc_compat
}

export function readBedrockCCCompatFeatureConfig(
  featuresConfig?: Record<string, unknown>,
): boolean {
  const value = featuresConfig?.bedrock_cc_compat
  if (typeof value === 'boolean') return value

  // Migrate values written by the previous frontend implementation.
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return (value as Record<string, unknown>).anthropic === true
  }

  return false
}
