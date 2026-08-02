import { describe, expect, it } from 'vitest'
import { sessionExpiredLoginURL } from '../quotaViewerAuthorizationRedirect'

describe('quota viewer authorization session expiry redirect', () => {
  it('preserves only a valid quota authorization deep link', () => {
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer/authorize',
      search: '?user_code=ABCD-EFGH',
      hash: '#confirm',
    })).toBe(
      '/login?redirect=%2Fquota-viewer%2Fauthorize%3Fuser_code%3DABCD-EFGH%23confirm',
    )
  })

  it('does not preserve other routes or malformed authorization links', () => {
    expect(sessionExpiredLoginURL({
      pathname: '/dashboard',
      search: '?redirect=https://example.com',
    })).toBe('/login')
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer/authorize',
      search: '?user_code=INVALID',
    })).toBe('/login')
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer/authorize',
      search: '?user_code=ABCD-EFGH&user_code=JKLM-NPQR',
    })).toBe('/login')
  })

  it('preserves only a supported installer intent on the public product page', () => {
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer',
      search: '?download=macos',
    })).toBe('/login?redirect=%2Fquota-viewer%3Fdownload%3Dmacos')
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer',
      search: '?download=linux',
    })).toBe('/login')
    expect(sessionExpiredLoginURL({
      pathname: '/quota-viewer',
      search: '?download=windows&next=https://example.com',
    })).toBe('/login')
  })
})
