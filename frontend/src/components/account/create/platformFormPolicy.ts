import type { AccountPlatform, AccountType } from '@/types'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type {
  AntigravityAccountKind,
  CreateAccountCategory,
  GeminiOAuthType,
} from './formDraft'

export function resolveCreateAccountType(input: {
  platform: AccountPlatform
  category: CreateAccountCategory
  addMethod: AddMethod
  antigravityKind: AntigravityAccountKind
}): AccountType {
  if (
    input.platform === 'antigravity' &&
    input.antigravityKind === 'upstream'
  ) {
    return 'apikey'
  }
  if (input.platform === 'anthropic' && input.category === 'bedrock') {
    return 'bedrock' as AccountType
  }
  if (
    (input.platform === 'gemini' || input.platform === 'anthropic') &&
    input.category === 'service_account'
  ) {
    return 'service_account' as AccountType
  }
  if (input.category === 'oauth-based') {
    return input.platform === 'anthropic' ? input.addMethod : 'oauth'
  }
  return 'apikey'
}

export function usesCreateAccountOAuthFlow(input: {
  platform: AccountPlatform
  category: CreateAccountCategory
  antigravityKind: AntigravityAccountKind
}): boolean {
  if (
    input.platform === 'antigravity' &&
    input.antigravityKind === 'upstream'
  ) {
    return false
  }
  if (input.platform === 'anthropic' && input.category === 'bedrock') {
    return false
  }
  return input.category === 'oauth-based'
}

export function oauthStepTitleKey(platform: AccountPlatform): string {
  return platform === 'anthropic'
    ? 'admin.accounts.oauth.title'
    : `admin.accounts.oauth.${platform}.title`
}

export function apiKeyBaseURLHintKey(
  platform: AccountPlatform,
): string | null {
  if (platform === 'grok') return null
  if (platform === 'openai' || platform === 'gemini') {
    return `admin.accounts.${platform}.baseUrlHint`
  }
  return 'admin.accounts.baseUrlHint'
}

export function apiKeyValueHintKey(
  platform: AccountPlatform,
): string | null {
  if (platform === 'grok') return null
  if (platform === 'openai' || platform === 'gemini') {
    return `admin.accounts.${platform}.apiKeyHint`
  }
  return 'admin.accounts.apiKeyHint'
}

export function resolveGeminiSelectedTier(input: {
  platform: AccountPlatform
  category: CreateAccountCategory
  oauthType: GeminiOAuthType
  googleOneTier: string
  gcpTier: string
  aiStudioTier: string
}): string {
  if (input.platform !== 'gemini') return ''
  if (input.category === 'apikey') return input.aiStudioTier
  if (input.oauthType === 'google_one') return input.googleOneTier
  if (input.oauthType === 'code_assist') return input.gcpTier
  return input.aiStudioTier
}
