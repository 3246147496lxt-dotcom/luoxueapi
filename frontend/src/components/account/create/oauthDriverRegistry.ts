import type { AccountPlatform } from '@/types'
import type { Ref } from 'vue'

export interface OAuthAuthorizationContext {
  proxyId: number | null
  projectId?: string
  oauthType?: string
  tier?: string
}

export interface OAuthDriver {
  readonly authUrl: Ref<string>
  readonly sessionId: Ref<string>
  readonly loading: Ref<boolean>
  readonly error: Ref<string>
  generateAuthorization(context: OAuthAuthorizationContext): Promise<void>
  exchangeAuthorizationCode(code: string): Promise<void>
  validateRefreshToken?(refreshToken: string): Promise<void>
  reset(): void
}

type OAuthDriverMap = Record<AccountPlatform, OAuthDriver>

export class OAuthDriverRegistry {
  constructor(private readonly drivers: OAuthDriverMap) {}

  get(platform: AccountPlatform): OAuthDriver {
    return this.drivers[platform]
  }

  exchangeAuthorizationCode(
    platform: AccountPlatform,
    code: string,
  ): Promise<void> {
    return this.get(platform).exchangeAuthorizationCode(code)
  }

  async validateRefreshToken(
    platform: AccountPlatform,
    refreshToken: string,
  ): Promise<void> {
    await this.get(platform).validateRefreshToken?.(refreshToken)
  }

  resetAll(): void {
    const visited = new Set<OAuthDriver>()
    Object.values(this.drivers).forEach((driver) => {
      if (visited.has(driver)) return
      visited.add(driver)
      driver.reset()
    })
  }
}
