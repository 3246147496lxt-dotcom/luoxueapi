import { describe, expect, it } from 'vitest'

import { resolveSupportContactDestination, resolveSupportContactUrl } from '../supportUrl'

describe('support URL resolution', () => {
  it('keeps safe absolute and relative contact destinations', () => {
    expect(resolveSupportContactUrl('https://support.example.com/help', ''))
      .toBe('https://support.example.com/help')
    expect(resolveSupportContactUrl('/support', ''))
      .toBe('/support')
    expect(resolveSupportContactDestination('/support', '')).toEqual({
      kind: 'contact',
      url: '/support',
    })
  })

  it('falls back to the first-party contact page for free-form contact text', () => {
    expect(resolveSupportContactUrl('QQ 123456', '/tutorial-docs/'))
      .toBe('/contact')
    expect(resolveSupportContactUrl('help@example.com', '/tutorial-docs/'))
      .toBe('/contact')
  })

  it('rejects unsafe protocols and replaces stale documentation hashes', () => {
    expect(resolveSupportContactUrl('javascript:alert(1)', '/tutorial-docs/#old'))
      .toBe('/contact')
  })

  it('uses the first-party contact page when both values are empty', () => {
    expect(resolveSupportContactUrl('', ''))
      .toBe('/contact')
    expect(resolveSupportContactDestination('', '')).toEqual({
      kind: 'contact',
      url: '/contact',
    })
  })
})
