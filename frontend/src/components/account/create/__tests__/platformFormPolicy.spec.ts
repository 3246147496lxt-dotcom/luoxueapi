import { describe, expect, it } from 'vitest'
import {
  apiKeyBaseURLHintKey,
  apiKeyValueHintKey,
  oauthStepTitleKey,
  resolveCreateAccountType,
  resolveGeminiSelectedTier,
  usesCreateAccountOAuthFlow,
} from '../platformFormPolicy'

describe('create-account platform form policy', () => {
  it.each([
    {
      platform: 'anthropic' as const,
      category: 'oauth-based' as const,
      addMethod: 'setup-token' as const,
      antigravityKind: 'oauth' as const,
      expected: 'setup-token',
    },
    {
      platform: 'anthropic' as const,
      category: 'bedrock' as const,
      addMethod: 'oauth' as const,
      antigravityKind: 'oauth' as const,
      expected: 'bedrock',
    },
    {
      platform: 'gemini' as const,
      category: 'service_account' as const,
      addMethod: 'oauth' as const,
      antigravityKind: 'oauth' as const,
      expected: 'service_account',
    },
    {
      platform: 'antigravity' as const,
      category: 'oauth-based' as const,
      addMethod: 'oauth' as const,
      antigravityKind: 'upstream' as const,
      expected: 'apikey',
    },
    {
      platform: 'grok' as const,
      category: 'oauth-based' as const,
      addMethod: 'oauth' as const,
      antigravityKind: 'oauth' as const,
      expected: 'oauth',
    },
  ])('resolves $platform/$category to $expected', (input) => {
    expect(resolveCreateAccountType(input)).toBe(input.expected)
  })

  it('keeps direct Bedrock and upstream flows out of the OAuth step', () => {
    expect(
      usesCreateAccountOAuthFlow({
        platform: 'anthropic',
        category: 'bedrock',
        antigravityKind: 'oauth',
      }),
    ).toBe(false)
    expect(
      usesCreateAccountOAuthFlow({
        platform: 'antigravity',
        category: 'oauth-based',
        antigravityKind: 'upstream',
      }),
    ).toBe(false)
    expect(
      usesCreateAccountOAuthFlow({
        platform: 'openai',
        category: 'oauth-based',
        antigravityKind: 'oauth',
      }),
    ).toBe(true)
  })

  it('owns platform-specific copy keys and Grok empty hints', () => {
    expect(oauthStepTitleKey('anthropic')).toBe('admin.accounts.oauth.title')
    expect(oauthStepTitleKey('openai')).toBe('admin.accounts.oauth.openai.title')
    expect(apiKeyBaseURLHintKey('gemini')).toBe(
      'admin.accounts.gemini.baseUrlHint',
    )
    expect(apiKeyValueHintKey('openai')).toBe(
      'admin.accounts.openai.apiKeyHint',
    )
    expect(apiKeyBaseURLHintKey('grok')).toBeNull()
    expect(apiKeyValueHintKey('grok')).toBeNull()
  })

  it('resolves Gemini tiers without leaking branches into the modal', () => {
    const base = {
      platform: 'gemini' as const,
      category: 'oauth-based' as const,
      googleOneTier: 'google-tier',
      gcpTier: 'gcp-tier',
      aiStudioTier: 'studio-tier',
    }
    expect(
      resolveGeminiSelectedTier({ ...base, oauthType: 'google_one' }),
    ).toBe('google-tier')
    expect(
      resolveGeminiSelectedTier({ ...base, oauthType: 'code_assist' }),
    ).toBe('gcp-tier')
    expect(
      resolveGeminiSelectedTier({
        ...base,
        category: 'apikey',
        oauthType: 'ai_studio',
      }),
    ).toBe('studio-tier')
  })
})
