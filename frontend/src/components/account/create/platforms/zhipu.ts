import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type GeneratedCredentialDraft,
} from './shared'
import type { CnAccountMode, CnApiProtocol } from '../credentialDraftBuilders'

const ZHIPU_DEFAULT_BASE_URL = 'https://open.bigmodel.cn/api/paas/v4'

/** Credentials for 智谱 GLM accounts.  The backend uses account_mode and
 * api_protocol to select the pay-as-you-go/coding-plan quota path and the
 * Chat Completions/Anthropic/Responses bridge respectively. */
export type ZhipuAccountDraft = GeneratedCredentialDraft & {
  platform: 'zhipu'
  kind: 'apikey'
  apiKey: string
  baseUrl?: string
  /** Optional routing metadata kept alongside the API-key credentials. */
  accountMode?: CnAccountMode
  apiProtocol?: CnApiProtocol
  zhipuOrganization?: string
  zhipuProject?: string
}

export function buildZhipuAccountPayload(
  base: AccountBaseDraft,
  draft: ZhipuAccountDraft,
): CreateAccountRequest {
  const credentials: Record<string, unknown> = {
    base_url: draft.baseUrl?.trim() || ZHIPU_DEFAULT_BASE_URL,
    api_key: draft.apiKey.trim(),
  }
  // Keep the typed draft contract in sync with buildAPIKeyCredentials.  Put
  // explicit fields in the fixed portion so they win over stale values in
  // credentialOptions, while still preserving unknown/legacy options.
  if (draft.accountMode) credentials.account_mode = draft.accountMode
  if (draft.apiProtocol) credentials.api_protocol = draft.apiProtocol
  const organization = draft.zhipuOrganization?.trim()
  if (organization) credentials.zhipu_organization = organization
  const project = draft.zhipuProject?.trim()
  if (project) credentials.zhipu_project = project

  return assembleAccountPayload(
    base,
    draft.platform,
    'apikey',
    mergeCredentialOptions(credentials, draft.credentialOptions),
    draft.extra,
  )
}
