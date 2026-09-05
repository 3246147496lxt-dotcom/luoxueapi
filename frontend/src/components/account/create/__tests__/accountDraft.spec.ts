import { describe, expect, it } from 'vitest'
import type { AccountPlatform } from '@/types'
import {
  accountPayloadAdapters,
  buildAccountCreatePayload,
  toAccountCredentialDraft,
  type AccountBaseDraft,
} from '../accountDraft'

const base: AccountBaseDraft = {
  name: 'account',
  notes: 'note',
  proxy_id: 3,
  concurrency: 4,
  load_factor: null,
  priority: 1,
  rate_multiplier: 1,
  group_ids: [9],
  expires_at: null,
  auto_pause_on_expired: true,
}

describe('account payload adapters', () => {
  it('registers every supported platform exactly once', () => {
    expect(Object.keys(accountPayloadAdapters)).toEqual([
      'anthropic',
      'openai',
      'gemini',
      'antigravity',
      'grok',
      'kimi',
      'deepseek',
      'zhipu',
    ])
  })

  const cases: Array<{
    platform: AccountPlatform
    type: 'oauth' | 'apikey'
    expectedType: string
  }> = [
    { platform: 'anthropic', type: 'oauth', expectedType: 'oauth' },
    { platform: 'openai', type: 'apikey', expectedType: 'apikey' },
    { platform: 'gemini', type: 'apikey', expectedType: 'apikey' },
    { platform: 'antigravity', type: 'apikey', expectedType: 'apikey' },
    { platform: 'grok', type: 'oauth', expectedType: 'oauth' },
    { platform: 'kimi', type: 'apikey', expectedType: 'apikey' },
    { platform: 'deepseek', type: 'apikey', expectedType: 'apikey' },
    { platform: 'zhipu', type: 'apikey', expectedType: 'apikey' },
  ]

  it.each(cases)('builds the $platform payload', ({ platform, type, expectedType }) => {
    const payload = buildAccountCreatePayload(
      base,
      toAccountCredentialDraft(platform, type, { marker: platform }),
    )
    expect(payload).toMatchObject({
      name: 'account',
      platform,
      type: expectedType,
      credentials: { marker: platform },
      group_ids: [9],
      auto_pause_on_expired: true,
    })
    expect(payload.load_factor).toBeUndefined()
  })

  it('rejects account kinds unsupported by a platform', () => {
    expect(() =>
      toAccountCredentialDraft('openai', 'bedrock', {}),
    ).toThrow('Unsupported openai account kind')
  })
})
