import type {
  CodexSessionImportRequest,
  CreateAccountRequest,
  OpenAICodexPATCreateRequest,
} from '@/types'
import {
  assembleAccountPayload,
  assertNever,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type ExistingCredentialDraft,
  type GeneratedCredentialDraft,
} from './shared'

const OPENAI_DEFAULT_BASE_URL = 'https://api.openai.com'

export type OpenAIAccountDraft =
  | (ExistingCredentialDraft & {
      platform: 'openai'
      kind: 'oauth'
    })
  | (GeneratedCredentialDraft & {
      platform: 'openai'
      kind: 'apikey'
      apiKey: string
      baseUrl?: string
    })

export function buildOpenAIAccountPayload(
  base: AccountBaseDraft,
  draft: OpenAIAccountDraft,
): CreateAccountRequest {
  switch (draft.kind) {
    case 'oauth':
      return assembleAccountPayload(
        base,
        draft.platform,
        'oauth',
        { ...draft.credentials },
        draft.extra,
      )
    case 'apikey':
      return assembleAccountPayload(
        base,
        draft.platform,
        'apikey',
        mergeCredentialOptions(
          {
            base_url: draft.baseUrl?.trim() || OPENAI_DEFAULT_BASE_URL,
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

function nonEmptyRecord(
  value?: Record<string, unknown>,
): Record<string, unknown> | undefined {
  return value && Object.keys(value).length > 0 ? { ...value } : undefined
}

export function buildOpenAICodexSessionImportRequest(
  base: AccountBaseDraft,
  content: string,
  credentialExtras?: Record<string, unknown>,
  extra?: Record<string, unknown>,
): CodexSessionImportRequest {
  return {
    content: content.trim(),
    name: base.name,
    notes: base.notes || null,
    proxy_id: base.proxy_id,
    concurrency: base.concurrency,
    load_factor: base.load_factor ?? undefined,
    priority: base.priority,
    rate_multiplier: base.rate_multiplier,
    group_ids: [...base.group_ids],
    expires_at: base.expires_at,
    auto_pause_on_expired: base.auto_pause_on_expired,
    credential_extras: nonEmptyRecord(credentialExtras),
    extra: extra ? { ...extra } : undefined,
    update_existing: true,
  }
}

export function buildOpenAICodexPATCreateRequest(
  base: AccountBaseDraft,
  accessToken: string,
  credentialExtras?: Record<string, unknown>,
  extra?: Record<string, unknown>,
): OpenAICodexPATCreateRequest {
  return {
    access_token: accessToken.trim(),
    name: base.name,
    notes: base.notes || null,
    proxy_id: base.proxy_id,
    concurrency: base.concurrency,
    load_factor: base.load_factor ?? undefined,
    priority: base.priority,
    rate_multiplier: base.rate_multiplier,
    group_ids: [...base.group_ids],
    expires_at: base.expires_at,
    auto_pause_on_expired: base.auto_pause_on_expired,
    credential_extras: nonEmptyRecord(credentialExtras),
    extra: extra ? { ...extra } : undefined,
  }
}
