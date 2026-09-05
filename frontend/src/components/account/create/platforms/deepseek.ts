import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type GeneratedCredentialDraft,
} from './shared'

/** DeepSeek's public API is API-key based and uses the OpenAI-compatible
 * credential shape (base_url + api_key).  Protocol-specific endpoint routing
 * is handled by the backend; the create form only needs to preserve the
 * caller-supplied base URL.
 */
const DEEPSEEK_DEFAULT_BASE_URL = 'https://api.deepseek.com'

export type DeepSeekAccountDraft = GeneratedCredentialDraft & {
  platform: 'deepseek'
  kind: 'apikey'
  apiKey: string
  baseUrl?: string
}
export function buildDeepSeekAccountPayload(
  base: AccountBaseDraft,
  draft: DeepSeekAccountDraft,
): CreateAccountRequest {
  return assembleAccountPayload(
    base,
    draft.platform,
    'apikey',
    mergeCredentialOptions(
      {
        base_url: draft.baseUrl?.trim() || DEEPSEEK_DEFAULT_BASE_URL,
        api_key: draft.apiKey.trim(),
      },
      draft.credentialOptions,
    ),
    draft.extra,
  )
}
