export function sanitizeInternalRedirect(
  value: unknown,
  fallback = '/dashboard',
): string {
  const candidate = Array.isArray(value) ? value[0] : value
  if (typeof candidate !== 'string') return fallback

  const raw = candidate.trim()
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.includes('\\')) return fallback

  try {
    const parsed = new URL(raw, 'https://luoxue.internal')
    if (parsed.origin !== 'https://luoxue.internal') return fallback
    return `${parsed.pathname}${parsed.search}${parsed.hash}`
  } catch {
    return fallback
  }
}

export function authRedirectRoute(
  path: '/login' | '/register',
  value: unknown,
): { path: '/login' | '/register'; query?: { redirect: string } } {
  const redirect = sanitizeInternalRedirect(value, '')
  return redirect ? { path, query: { redirect } } : { path }
}
