import type { CreateAccountRequest } from '@/types'
import type { GrokSSOToOAuthRequest } from '@/api/admin/grok'
import {
  assembleAccountPayload,
  assertNever,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type ExistingCredentialDraft,
  type GeneratedCredentialDraft,
} from './shared'

const GROK_DEFAULT_BASE_URL = 'https://api.x.ai/v1'

export type GrokAccountDraft =
  | (ExistingCredentialDraft & {
      platform: 'grok'
      kind: 'oauth'
      baseUrl?: string
    })
  | (GeneratedCredentialDraft & {
      platform: 'grok'
      kind: 'apikey'
      apiKey: string
      baseUrl?: string
    })

export function buildGrokAccountPayload(
  base: AccountBaseDraft,
  draft: GrokAccountDraft,
): CreateAccountRequest {
  switch (draft.kind) {
    case 'oauth': {
      const credentials = { ...draft.credentials }
      if (!credentials.base_url && draft.baseUrl?.trim()) {
        credentials.base_url = draft.baseUrl.trim()
      }
      return assembleAccountPayload(base, draft.platform, 'oauth', credentials, draft.extra)
    }
    case 'apikey':
      return assembleAccountPayload(
        base,
        draft.platform,
        'apikey',
        mergeCredentialOptions(
          {
            base_url: draft.baseUrl?.trim() || GROK_DEFAULT_BASE_URL,
            api_key: draft.apiKey.trim(),
          },
          draft.credentialOptions,
        ),
        draft.extra,
      )
    default:
      return assertNever(draft)
  }
}

export function buildGrokSSOImportRequest(
  base: AccountBaseDraft,
  ssoTokens: string[],
  credentials: Record<string, unknown>,
): GrokSSOToOAuthRequest {
  return {
    sso_tokens: [...ssoTokens],
    name: base.name || undefined,
    notes: base.notes || undefined,
    proxy_id: base.proxy_id,
    group_ids: [...base.group_ids],
    credentials: { ...credentials },
    concurrency: base.concurrency,
    load_factor: base.load_factor ?? undefined,
    priority: base.priority,
    rate_multiplier: base.rate_multiplier,
    expires_at: base.expires_at,
    auto_pause_on_expired: base.auto_pause_on_expired,
  }
}
