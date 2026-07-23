import type { AccountPlatform, CreateAccountRequest } from '@/types'
import {
  buildAnthropicAccountPayload,
  type AnthropicAccountDraft,
} from './platforms/anthropic'
import {
  buildAntigravityAccountPayload,
  type AntigravityAccountDraft,
} from './platforms/antigravity'
import { buildGeminiAccountPayload, type GeminiAccountDraft } from './platforms/gemini'
import { buildGrokAccountPayload, type GrokAccountDraft } from './platforms/grok'
import { buildOpenAIAccountPayload, type OpenAIAccountDraft } from './platforms/openai'
import type { AccountBaseDraft } from './platforms/shared'

export type AccountCredentialDraft =
  | AnthropicAccountDraft
  | OpenAIAccountDraft
  | GeminiAccountDraft
  | AntigravityAccountDraft
  | GrokAccountDraft

interface PlatformPayloadAdapter<Draft extends AccountCredentialDraft> {
  build(base: AccountBaseDraft, draft: Draft): CreateAccountRequest
}

export const accountPayloadAdapters = {
  anthropic: { build: buildAnthropicAccountPayload },
  openai: { build: buildOpenAIAccountPayload },
  gemini: { build: buildGeminiAccountPayload },
  antigravity: { build: buildAntigravityAccountPayload },
  grok: { build: buildGrokAccountPayload },
} satisfies {
  [Platform in AccountPlatform]: PlatformPayloadAdapter<
    Extract<AccountCredentialDraft, { platform: Platform }>
  >
}

export function buildAccountCreatePayload(
  base: AccountBaseDraft,
  draft: AccountCredentialDraft,
): CreateAccountRequest {
  switch (draft.platform) {
    case 'anthropic':
      return accountPayloadAdapters.anthropic.build(base, draft)
    case 'openai':
      return accountPayloadAdapters.openai.build(base, draft)
    case 'gemini':
      return accountPayloadAdapters.gemini.build(base, draft)
    case 'antigravity':
      return accountPayloadAdapters.antigravity.build(base, draft)
    case 'grok':
      return accountPayloadAdapters.grok.build(base, draft)
  }
}
