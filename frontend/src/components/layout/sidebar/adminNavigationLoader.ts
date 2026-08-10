import type { AdminNavigationDefinition } from './types'

let request: Promise<AdminNavigationDefinition> | null = null

/** Keep administrator navigation out of the personal-workspace static graph. */
export function loadAdminNavigation(): Promise<AdminNavigationDefinition> {
  if (!request) {
    request = import('./adminNavigation').then(({ adminNavigationDefinition }) => (
      adminNavigationDefinition
    ))
  }
  return request
}
