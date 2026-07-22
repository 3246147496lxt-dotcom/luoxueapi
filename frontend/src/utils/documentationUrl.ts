import { sanitizeUrl } from './url'

export const PRODUCTION_DOCUMENTATION_URL = '/tutorial-docs/'
const DEVELOPMENT_DOCUMENTATION_URL = 'http://127.0.0.1:4179/tutorial-docs/'

const configuredDevelopmentUrl = import.meta.env.VITE_DEV_DOCUMENTATION_URL || ''
const DEFAULT_DOCUMENTATION_URL = import.meta.env.DEV
  ? sanitizeUrl(configuredDevelopmentUrl, { allowRelative: true })
    || DEVELOPMENT_DOCUMENTATION_URL
  : PRODUCTION_DOCUMENTATION_URL

type FirstPartyDocumentationLocation = {
  pathname: string
  search: string
  hash: string
}

const TUTORIAL_PATH = '/tutorial-docs'
const RELATIVE_URL_BASE = 'https://local-docs.invalid'

function isTutorialPath(pathname: string): boolean {
  return pathname === TUTORIAL_PATH || pathname.startsWith(`${TUTORIAL_PATH}/`)
}

function getFirstPartyDocumentationLocation(url: string): FirstPartyDocumentationLocation | null {
  try {
    if (url.startsWith('/')) {
      const parsed = new URL(url, RELATIVE_URL_BASE)
      if (parsed.origin !== RELATIVE_URL_BASE || !isTutorialPath(parsed.pathname)) return null

      return {
        pathname: parsed.pathname.slice(TUTORIAL_PATH.length) || '/',
        search: parsed.search,
        hash: parsed.hash,
      }
    }

    const parsed = new URL(url)
    if (
      parsed.protocol !== 'https:'
      || parsed.port !== ''
      || parsed.username !== ''
      || parsed.password !== ''
    ) {
      return null
    }

    if (parsed.hostname === 'luoxueapi.cc' && isTutorialPath(parsed.pathname)) {
      return {
        pathname: parsed.pathname.slice(TUTORIAL_PATH.length) || '/',
        search: parsed.search,
        hash: parsed.hash,
      }
    }

    if (parsed.hostname === 'docs.luoxueapi.cc') {
      return {
        pathname: parsed.pathname || '/',
        search: parsed.search,
        hash: parsed.hash,
      }
    }
  } catch {
    return null
  }

  return null
}

function resolveDevelopmentDocumentationUrl(location: FirstPartyDocumentationLocation): string {
  const base = DEFAULT_DOCUMENTATION_URL.replace(/[?#].*$/, '').replace(/\/+$/, '')
  const pathname = location.pathname === '/'
    ? '/'
    : `/${location.pathname.replace(/^\/+/, '')}`

  return `${base}${pathname}${location.search}${location.hash}`
}

export function getDefaultDocumentationUrl(): string {
  return DEFAULT_DOCUMENTATION_URL
}

export function resolveDocumentationUrl(
  configuredUrl: string | null | undefined,
  fallback = DEFAULT_DOCUMENTATION_URL,
): string {
  const safeFallback = sanitizeUrl(fallback, { allowRelative: true })
    || DEFAULT_DOCUMENTATION_URL
  const safeConfiguredUrl = sanitizeUrl(configuredUrl || '', { allowRelative: true })
  const resolvedUrl = safeConfiguredUrl || safeFallback

  // First-party documentation links should stay on the local docs app during
  // development, whether they came from administrator settings or a fallback.
  if (import.meta.env.DEV) {
    const firstPartyLocation = getFirstPartyDocumentationLocation(resolvedUrl)
    if (firstPartyLocation) {
      return resolveDevelopmentDocumentationUrl(firstPartyLocation)
    }
  }

  return resolvedUrl
}

export function resolveTutorialUrl(
  configuredUrl: string | null | undefined,
  hash = 'quick-start',
  fallback = DEFAULT_DOCUMENTATION_URL,
): string {
  const base = resolveDocumentationUrl(configuredUrl, fallback).replace(/#.*$/, '')
  return `${base}#${hash.replace(/^#/, '')}`
}
