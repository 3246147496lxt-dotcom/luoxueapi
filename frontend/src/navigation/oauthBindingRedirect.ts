import type { User } from '@/types'

const CONNECTIONS_QUERY = 'account_settings=account&account_settings_detail=connections'

export const LEGACY_OAUTH_BINDING_CONNECTIONS_PATH = '/settings/profile'

/**
 * Returns the canonical settings destination when the signed-in role is known.
 *
 * OAuth binding can also resume before the auth store has hydrated. In that
 * case the legacy bridge is intentional: the router resolves the correct
 * user/admin host after authentication without guessing the audience here.
 */
export function resolveOAuthBindingConnectionsRedirect(
  role?: User['role'] | null,
): string {
  if (role === 'admin') {
    return `/admin/dashboard?${CONNECTIONS_QUERY}`
  }
  if (role === 'user') {
    return `/dashboard?${CONNECTIONS_QUERY}`
  }
  return LEGACY_OAUTH_BINDING_CONNECTIONS_PATH
}

/**
 * Backend-provided and caller-provided redirects remain authoritative so old
 * `/profile*` callbacks keep working through the compatibility routes.
 */
export function resolveOAuthBindingCompletionRedirect(
  explicitRedirect: string | null | undefined,
  role?: User['role'] | null,
): string {
  return explicitRedirect?.trim() || resolveOAuthBindingConnectionsRedirect(role)
}
