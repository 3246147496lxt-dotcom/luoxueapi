import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type GeneratedCredentialDraft,
} from './shared'

/** Kimi's public API is API-key based and uses the OpenAI-compatible
 * credential shape (base_url + api_key).  Protocol-specific endpoint routing
 * is handled by the backend; the create form only needs to preserve the
 * caller-supplied base URL.
 */
const KIMI_DEFAULT_BASE_URL = 'https://api.moonshot.cn/v1'

export type KimiAccountDraft = GeneratedCredentialDraft & {
  platform: 'kimi'
  kind: 'apikey'
  apiKey: string
  baseUrl?: string
}
export function buildKimiAccountPayload(
  base: AccountBaseDraft,
  draft: KimiAccountDraft,
): CreateAccountRequest {
  return assembleAccountPayload(
    base,
    draft.platform,
    'apikey',
    mergeCredentialOptions(
      {
        base_url: draft.baseUrl?.trim() || KIMI_DEFAULT_BASE_URL,
        api_key: draft.apiKey.trim(),
      },
      draft.credentialOptions,
    ),
    draft.extra,
  )
}
