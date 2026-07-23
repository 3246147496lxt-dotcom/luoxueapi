import type { CreateAccountRequest } from '@/types'
import {
  assembleAccountPayload,
  assertNever,
  mergeCredentialOptions,
  type AccountBaseDraft,
  type ExistingCredentialDraft,
  type GeneratedCredentialDraft,
} from './shared'

const ANTHROPIC_DEFAULT_BASE_URL = 'https://api.anthropic.com'

export type AnthropicAccountDraft =
  | (ExistingCredentialDraft & {
      platform: 'anthropic'
      kind: 'oauth' | 'setup-token'
    })
  | (GeneratedCredentialDraft & {
      platform: 'anthropic'
      kind: 'apikey'
      apiKey: string
      baseUrl?: string
    })
  | (GeneratedCredentialDraft & {
      platform: 'anthropic'
      kind: 'bedrock'
      authMode: 'sigv4'
      region?: string
      accessKeyId: string
      secretAccessKey: string
      sessionToken?: string
      forceGlobal?: boolean
    })
  | (GeneratedCredentialDraft & {
      platform: 'anthropic'
      kind: 'bedrock'
      authMode: 'apikey'
      region?: string
      apiKey: string
      forceGlobal?: boolean
    })
  | (GeneratedCredentialDraft & {
      platform: 'anthropic'
      kind: 'service_account'
      serviceAccountJson: string
      projectId: string
      clientEmail: string
      location: string
    })

export function buildAnthropicAccountPayload(
  base: AccountBaseDraft,
  draft: AnthropicAccountDraft,
): CreateAccountRequest {
  switch (draft.kind) {
    case 'oauth':
    case 'setup-token':
      return assembleAccountPayload(
        base,
        draft.platform,
        draft.kind,
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
            base_url: draft.baseUrl?.trim() || ANTHROPIC_DEFAULT_BASE_URL,
            api_key: draft.apiKey.trim(),
          },
          draft.credentialOptions,
        ),
        draft.extra,
      )
    case 'bedrock': {
      const credentials: Record<string, unknown> = {
        auth_mode: draft.authMode,
        aws_region: draft.region?.trim() || 'us-east-1',
      }
      if (draft.authMode === 'sigv4') {
        credentials.aws_access_key_id = draft.accessKeyId.trim()
        credentials.aws_secret_access_key = draft.secretAccessKey.trim()
        if (draft.sessionToken?.trim()) {
          credentials.aws_session_token = draft.sessionToken.trim()
        }
      } else {
        credentials.api_key = draft.apiKey.trim()
      }
      if (draft.forceGlobal) {
        credentials.aws_force_global = 'true'
      }
      return assembleAccountPayload(
        base,
        draft.platform,
        'bedrock',
        mergeCredentialOptions(credentials, draft.credentialOptions),
        draft.extra,
      )
    }
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
