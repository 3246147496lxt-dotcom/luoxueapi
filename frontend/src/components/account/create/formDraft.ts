export type CreateAccountCategory =
  | 'oauth-based'
  | 'apikey'
  | 'bedrock'
  | 'service_account'

export type AddMethod = 'oauth' | 'setup-token'

export type AuthInputMethod =
  | 'manual'
  | 'cookie'
  | 'refresh_token'
  | 'mobile_refresh_token'
  | 'session_token'
  | 'access_token'
  | 'codex_session'
  | 'agent_identity'
  | 'codex_pat'
  | 'sso_cookie'

export type AntigravityAccountKind = 'oauth' | 'upstream'

export type GeminiOAuthType = 'code_assist' | 'google_one' | 'ai_studio'
export type GeminiGoogleOneTier = 'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'
export type GeminiGCPTier = 'gcp_standard' | 'gcp_enterprise'
export type GeminiAIStudioTier = 'aistudio_free' | 'aistudio_paid'

export interface AccountModelMappingDraft {
  from: string
  to: string
}

export type BedrockAuthMode = 'sigv4' | 'apikey'
export type ModelRestrictionMode = 'whitelist' | 'mapping'
