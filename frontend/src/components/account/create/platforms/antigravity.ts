import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  assertNever,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type ExistingCredentialDraft,
  type GeneratedCredentialDraft,
} from './shared'

export type AntigravityAccountDraft =
  | (ExistingCredentialDraft & {
      platform: 'antigravity'
      kind: 'oauth'
    })
  | (GeneratedCredentialDraft & {
      platform: 'antigravity'
      kind: 'upstream'
      apiKey: string
      baseUrl: string
    })

export function buildAntigravityAccountPayload(
  base: AccountBaseDraft,
  draft: AntigravityAccountDraft,
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
    case 'upstream':
      return assembleAccountPayload(
        base,
        draft.platform,
        // The backend's compatibility contract stores Antigravity upstream
        // accounts as API-key accounts even though the UI draft is explicit.
        'apikey',
        mergeCredentialOptions(
          {
            base_url: draft.baseUrl.trim(),
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
