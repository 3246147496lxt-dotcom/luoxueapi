import type { AccountType, CreateAccountRequest } from '@/types'

export interface AccountBaseDraft {
  name: string
  notes: string
  proxy_id: number | null
  concurrency: number
  load_factor: number | null
  priority: number
  rate_multiplier: number
  group_ids: number[]
  expires_at: number | null
  auto_pause_on_expired: boolean
}

export interface ExistingCredentialDraft {
  credentials: Record<string, unknown>
  extra?: Record<string, unknown>
}

export interface GeneratedCredentialDraft {
  credentialOptions?: Record<string, unknown>
  extra?: Record<string, unknown>
}

export function mergeCredentialOptions(
  credentials: Record<string, unknown>,
  options?: Record<string, unknown>,
): Record<string, unknown> {
  return {
    ...(options || {}),
    ...credentials,
  }
}

export function assembleAccountPayload(
  base: AccountBaseDraft,
  platform: CreateAccountRequest['platform'],
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>,
): CreateAccountRequest {
  return {
    name: base.name,
    notes: base.notes,
    platform,
    type,
    credentials,
    extra: extra ? { ...extra } : undefined,
    proxy_id: base.proxy_id,
    concurrency: base.concurrency,
    load_factor: base.load_factor ?? undefined,
    priority: base.priority,
    rate_multiplier: base.rate_multiplier,
    group_ids: [...base.group_ids],
    expires_at: base.expires_at,
    auto_pause_on_expired: base.auto_pause_on_expired,
  }
}

export function assertNever(_value: never): never {
  throw new Error('Unsupported account draft')
}
