import type { OpenAIEndpointCapability } from '@/types'

export type { OpenAIEndpointCapability } from '@/types'

export const OPENAI_ENDPOINT_CAPABILITIES = [
  'chat_completions',
  'embeddings',
  'audio_transcriptions',
] as const

export const DEFAULT_OPENAI_ENDPOINT_CAPABILITIES = [
  'chat_completions',
  'embeddings',
] as const satisfies readonly OpenAIEndpointCapability[]

const OPENAI_ENDPOINT_CAPABILITY_SET = new Set<string>(
  OPENAI_ENDPOINT_CAPABILITIES,
)

export interface ParsedOpenAIEndpointCapabilities {
  selected: OpenAIEndpointCapability[]
  unknown: string[]
}

export interface SerializedOpenAIEndpointCapabilities {
  omit: boolean
  values: string[]
}

export function normalizeOpenAIEndpointCapabilities(
  values: readonly unknown[],
): OpenAIEndpointCapability[] {
  const selected = OPENAI_ENDPOINT_CAPABILITIES.filter((capability) =>
    values.includes(capability),
  )
  return selected.length > 0
    ? [...selected]
    : [...DEFAULT_OPENAI_ENDPOINT_CAPABILITIES]
}

function pushUnknownCapability(
  unknown: string[],
  seen: Set<string>,
  value: unknown,
): void {
  if (
    typeof value !== 'string' ||
    value.length === 0 ||
    OPENAI_ENDPOINT_CAPABILITY_SET.has(value) ||
    seen.has(value)
  ) {
    return
  }
  seen.add(value)
  unknown.push(value)
}

export function parseOpenAIEndpointCapabilities(
  raw: unknown,
): ParsedOpenAIEndpointCapabilities {
  const configured = Array.isArray(raw) || (raw !== null && typeof raw === 'object')
  const known: unknown[] = []
  const unknown: string[] = []
  const unknownSeen = new Set<string>()

  if (Array.isArray(raw)) {
    for (const value of raw) {
      if (
        typeof value === 'string' &&
        OPENAI_ENDPOINT_CAPABILITY_SET.has(value)
      ) {
        known.push(value)
      } else {
        pushUnknownCapability(unknown, unknownSeen, value)
      }
    }
  } else if (raw !== null && typeof raw === 'object') {
    for (const [capability, enabled] of Object.entries(raw)) {
      if (enabled !== true) continue
      if (OPENAI_ENDPOINT_CAPABILITY_SET.has(capability)) {
        known.push(capability)
      } else {
        pushUnknownCapability(unknown, unknownSeen, capability)
      }
    }
  }

  return {
    selected: configured
      ? OPENAI_ENDPOINT_CAPABILITIES.filter((capability) => known.includes(capability))
      : [...DEFAULT_OPENAI_ENDPOINT_CAPABILITIES],
    unknown,
  }
}

function isLegacyDefaultCapabilitySet(
  capabilities: readonly OpenAIEndpointCapability[],
): boolean {
  return (
    capabilities.length === DEFAULT_OPENAI_ENDPOINT_CAPABILITIES.length &&
    DEFAULT_OPENAI_ENDPOINT_CAPABILITIES.every((capability) =>
      capabilities.includes(capability),
    )
  )
}

export function serializeOpenAIEndpointCapabilities(
  selected: readonly OpenAIEndpointCapability[],
  unknownValues: readonly string[] = [],
): SerializedOpenAIEndpointCapabilities {
  const capabilities = OPENAI_ENDPOINT_CAPABILITIES.filter((capability) =>
    selected.includes(capability),
  )
  const unknown: string[] = []
  const unknownSeen = new Set<string>()
  for (const value of unknownValues) {
    pushUnknownCapability(unknown, unknownSeen, value)
  }

  return {
    omit: unknown.length === 0 && isLegacyDefaultCapabilitySet(capabilities),
    values: [...capabilities, ...unknown],
  }
}
