import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import type { AccountPlatform } from '@/types'
import {
  OAuthDriverRegistry,
  type OAuthDriver,
} from '../oauthDriverRegistry'

function driver(): OAuthDriver {
  return {
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    generateAuthorization: vi.fn(async () => undefined),
    exchangeAuthorizationCode: vi.fn(async () => undefined),
    validateRefreshToken: vi.fn(async () => undefined),
    reset: vi.fn(),
  }
}

describe('OAuthDriverRegistry', () => {
  it('selects by platform and resets shared drivers only once', () => {
    const anthropic = driver()
    const drivers = {
      anthropic,
      openai: driver(),
      gemini: driver(),
      antigravity: driver(),
      grok: driver(),
    } satisfies Record<AccountPlatform, OAuthDriver>
    const registry = new OAuthDriverRegistry(drivers)

    expect(registry.get('openai')).toBe(drivers.openai)
    registry.resetAll()
    Object.values(drivers).forEach((value) => {
      expect(value.reset).toHaveBeenCalledTimes(1)
    })
  })

  it('routes exchange and refresh-token commands through the selected driver', async () => {
    const drivers = {
      anthropic: driver(),
      openai: driver(),
      gemini: driver(),
      antigravity: driver(),
      grok: driver(),
    } satisfies Record<AccountPlatform, OAuthDriver>
    const registry = new OAuthDriverRegistry(drivers)

    await registry.exchangeAuthorizationCode('gemini', 'oauth-code')
    await registry.validateRefreshToken('grok', 'refresh-token')

    expect(drivers.gemini.exchangeAuthorizationCode).toHaveBeenCalledWith(
      'oauth-code',
    )
    expect(drivers.grok.validateRefreshToken).toHaveBeenCalledWith(
      'refresh-token',
    )
    expect(drivers.openai.exchangeAuthorizationCode).not.toHaveBeenCalled()
  })
})
