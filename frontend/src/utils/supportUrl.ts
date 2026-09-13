import { sanitizeUrl } from './url'

export type SupportContactDestination = {
  kind: 'contact' | 'documentation'
  url: string
}

/**
 * Resolve the customer-support destination used by navigation and empty states.
 * Free-form contact text is intentionally not treated as a URL; when no safe
 * destination is configured, the first-party contact page is used.
 */
export function resolveSupportContactDestination(
  contactInfo: string | null | undefined,
  _documentationUrl: string | null | undefined,
): SupportContactDestination {
  const configuredUrl = sanitizeUrl(contactInfo || '', { allowRelative: true })
  if (configuredUrl) {
    return { kind: 'contact', url: configuredUrl }
  }

  return {
    kind: 'contact',
    url: '/contact',
  }
}

export function resolveSupportContactUrl(
  contactInfo: string | null | undefined,
  documentationUrl: string | null | undefined,
): string {
  return resolveSupportContactDestination(contactInfo, documentationUrl).url
}
