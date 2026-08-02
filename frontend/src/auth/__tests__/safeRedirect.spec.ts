import { describe, expect, it } from 'vitest'
import { authRedirectRoute, sanitizeInternalRedirect } from '../safeRedirect'

describe('safe internal auth redirects', () => {
  it('keeps an installer intent including its query', () => {
    expect(sanitizeInternalRedirect('/quota-viewer?download=macos')).toBe(
      '/quota-viewer?download=macos',
    )
    expect(authRedirectRoute('/register', '/quota-viewer?download=windows')).toEqual({
      path: '/register',
      query: { redirect: '/quota-viewer?download=windows' },
    })
  })

  it('rejects external, protocol-relative, and backslash redirects', () => {
    expect(sanitizeInternalRedirect('https://example.com', '/dashboard')).toBe('/dashboard')
    expect(sanitizeInternalRedirect('//example.com', '')).toBe('')
    expect(sanitizeInternalRedirect('/\\example.com', '')).toBe('')
  })
})
