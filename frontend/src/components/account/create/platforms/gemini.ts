import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  assertNever,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type ExistingCredentialDraft,
  type GeneratedCredentialDraft,
} from './shared'

const GEMINI_DEFAULT_BASE_URL = 'https://generativelanguage.googleapis.com'

export type GeminiAccountDraft =
  | (ExistingCredentialDraft & {
      platform: 'gemini'
      kind: 'oauth'
    })
  | (GeneratedCredentialDraft & {
      platform: 'gemini'
      kind: 'apikey'
      apiKey: string
      baseUrl?: string
      tierId: string
    })
  | (GeneratedCredentialDraft & {
      platform: 'gemini'
      kind: 'service_account'
      serviceAccountJson: string
      projectId: string
      clientEmail: string
      location: string
    })

export function buildGeminiAccountPayload(
  base: AccountBaseDraft,
  draft: GeminiAccountDraft,
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
            base_url: draft.baseUrl?.trim() || GEMINI_DEFAULT_BASE_URL,
            api_key: draft.apiKey.trim(),
            tier_id: draft.tierId,
          },
          draft.credentialOptions,
        ),
        draft.extra,
      )
    case 'service_account':
      return assembleAccountPayload(
        base,
        draft.platform,
        'service_account',
        mergeCredentialOptions(
          {
            service_account_json: draft.serviceAccountJson.trim(),
            project_id: draft.projectId.trim(),
            client_email: draft.clientEmail.trim(),
            location: draft.location.trim(),
            tier_id: 'vertex',
          },
          draft.credentialOptions,
        ),
        draft.extra,
      )
    default:
      return assertNever(draft)
  }
}
