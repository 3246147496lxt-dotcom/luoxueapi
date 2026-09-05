import type { AccountPlatform, AccountType } from '@/types'
import {
  buildAccountCreatePayload,
  accountPayloadAdapters,
  type AccountCredentialDraft,
} from './platformRegistry'
import type { AccountBaseDraft } from './platforms/shared'

export {
  accountPayloadAdapters,
  buildAccountCreatePayload,
  type AccountBaseDraft,
  type AccountCredentialDraft,
}
export type { AnthropicAccountDraft } from './platforms/anthropic'
export type { AntigravityAccountDraft } from './platforms/antigravity'
export type { GeminiAccountDraft } from './platforms/gemini'
export type { GrokAccountDraft } from './platforms/grok'
export type { OpenAIAccountDraft } from './platforms/openai'
export type { KimiAccountDraft } from './platforms/kimi'
export type { DeepSeekAccountDraft } from './platforms/deepseek'
export type { ZhipuAccountDraft } from './platforms/zhipu'
export { buildGrokSSOImportRequest } from './platforms/grok'
export {
  buildOpenAICodexPATCreateRequest,
  buildOpenAICodexSessionImportRequest,
} from './platforms/openai'

function stringCredential(credentials: Record<string, unknown>, key: string): string {
  const value = credentials[key]
  return typeof value === 'string' ? value : value == null ? '' : String(value)
}

function credentialOptions(
  credentials: Record<string, unknown>,
  excludedKeys: readonly string[],
): Record<string, unknown> | undefined {
  const options = { ...credentials }
  excludedKeys.forEach((key) => delete options[key])
  return Object.keys(options).length > 0 ? options : undefined
}

function unsupported(platform: AccountPlatform, type: AccountType): never {
  throw new Error(`Unsupported ${platform} account kind: ${type}`)
}

/**
 * Compatibility boundary for the modal's OAuth composables. It converts their
 * existing credential records into the discriminated draft consumed by the
 * platform registry; no standard create-account request bypasses the registry
 * after this point. Dedicated bulk-import endpoints keep their own API schema.
 */
export function toAccountCredentialDraft(
  platform: AccountPlatform,
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>,
): AccountCredentialDraft {
  switch (platform) {
    case 'anthropic':
      switch (type) {
        case 'oauth':
        case 'setup-token':
          return { platform, kind: type, credentials, extra }
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        case 'bedrock': {
          const authMode = credentials.auth_mode === 'apikey' ? 'apikey' : 'sigv4'
          const common = {
            platform,
            kind: 'bedrock' as const,
            region: stringCredential(credentials, 'aws_region'),
            forceGlobal:
              credentials.aws_force_global === true || credentials.aws_force_global === 'true',
            credentialOptions: credentialOptions(credentials, [
              'auth_mode',
              'aws_region',
              'aws_access_key_id',
              'aws_secret_access_key',
              'aws_session_token',
              'aws_force_global',
              'api_key',
            ]),
            extra,
          }
          if (authMode === 'apikey') {
            return {
              ...common,
              authMode,
              apiKey: stringCredential(credentials, 'api_key'),
            }
          }
          return {
            ...common,
            authMode,
            accessKeyId: stringCredential(credentials, 'aws_access_key_id'),
            secretAccessKey: stringCredential(credentials, 'aws_secret_access_key'),
            sessionToken: stringCredential(credentials, 'aws_session_token'),
          }
        }
        case 'service_account':
          return {
            platform,
            kind: 'service_account',
            serviceAccountJson: stringCredential(credentials, 'service_account_json'),
            projectId: stringCredential(credentials, 'project_id'),
            clientEmail: stringCredential(credentials, 'client_email'),
            location: stringCredential(credentials, 'location'),
            credentialOptions: credentialOptions(credentials, [
              'service_account_json',
              'project_id',
              'client_email',
              'location',
              'tier_id',
            ]),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'openai':
      switch (type) {
        case 'oauth':
          return { platform, kind: 'oauth', credentials, extra }
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'kimi':
      switch (type) {
        case 'apikey':
          return { platform, kind: 'apikey', baseUrl: stringCredential(credentials, 'base_url'), apiKey: stringCredential(credentials, 'api_key'), credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']), extra }
        default: return unsupported(platform, type)
      }
    case 'deepseek':
      switch (type) {
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'zhipu':
      switch (type) {
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'gemini':
      switch (type) {
        case 'oauth':
          return { platform, kind: 'oauth', credentials, extra }
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            tierId: stringCredential(credentials, 'tier_id'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key', 'tier_id']),
            extra,
          }
        case 'service_account':
          return {
            platform,
            kind: 'service_account',
            serviceAccountJson: stringCredential(credentials, 'service_account_json'),
            projectId: stringCredential(credentials, 'project_id'),
            clientEmail: stringCredential(credentials, 'client_email'),
            location: stringCredential(credentials, 'location'),
            credentialOptions: credentialOptions(credentials, [
              'service_account_json',
              'project_id',
              'client_email',
              'location',
              'tier_id',
            ]),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'antigravity':
      switch (type) {
        case 'oauth':
          return { platform, kind: 'oauth', credentials, extra }
        case 'apikey':
        case 'upstream':
          return {
            platform,
            kind: 'upstream',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
    case 'grok':
      switch (type) {
        case 'oauth':
          return { platform, kind: 'oauth', credentials, extra }
        case 'apikey':
          return {
            platform,
            kind: 'apikey',
            baseUrl: stringCredential(credentials, 'base_url'),
            apiKey: stringCredential(credentials, 'api_key'),
            credentialOptions: credentialOptions(credentials, ['base_url', 'api_key']),
            extra,
          }
        default:
          return unsupported(platform, type)
      }
  }
}
