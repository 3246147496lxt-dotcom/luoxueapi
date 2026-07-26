import { describe, expect, it } from 'vitest'

import {
  LEGACY_OAUTH_BINDING_CONNECTIONS_PATH,
  resolveOAuthBindingCompletionRedirect,
  resolveOAuthBindingConnectionsRedirect,
} from '@/navigation/oauthBindingRedirect'

describe('oauth binding settings redirects', () => {
  it('resolves canonical connections settings for known user roles', () => {
    expect(resolveOAuthBindingConnectionsRedirect('user')).toBe(
      '/dashboard?account_settings=account&account_settings_detail=connections'
    )
    expect(resolveOAuthBindingConnectionsRedirect('admin')).toBe(
      '/admin/dashboard?account_settings=account&account_settings_detail=connections'
    )
  })

  it('uses the role-aware legacy route when auth hydration has not completed', () => {
    expect(resolveOAuthBindingConnectionsRedirect()).toBe(
      LEGACY_OAUTH_BINDING_CONNECTIONS_PATH
    )
  })

  it('continues to honor explicit and legacy backend redirects', () => {
    expect(resolveOAuthBindingCompletionRedirect('/profile', 'admin')).toBe('/profile')
    expect(resolveOAuthBindingCompletionRedirect('/profile/connections', 'user')).toBe(
      '/profile/connections'
    )
    expect(resolveOAuthBindingCompletionRedirect('  /custom-return  ', 'user')).toBe(
      '/custom-return'
    )
  })
})
