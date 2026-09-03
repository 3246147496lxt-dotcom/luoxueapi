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

  it('falls back to the recharge documentation section for free-form contact text', () => {
    expect(resolveSupportContactUrl('QQ 123456', '/tutorial-docs/'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
    expect(resolveSupportContactUrl('help@example.com', '/tutorial-docs/'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
  })

  it('rejects unsafe protocols and replaces stale documentation hashes', () => {
    expect(resolveSupportContactUrl('javascript:alert(1)', '/tutorial-docs/#old'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
  })

  it('uses the default documentation destination when both values are empty', () => {
    expect(resolveSupportContactUrl('', ''))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
    expect(resolveSupportContactDestination('', '')).toEqual({
      kind: 'documentation',
      url: 'http://127.0.0.1:4179/tutorial-docs/#recharge',
    })
  })
})
