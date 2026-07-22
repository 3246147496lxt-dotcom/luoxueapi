import { describe, expect, it } from 'vitest'

import {
  PRODUCTION_DOCUMENTATION_URL,
  getDefaultDocumentationUrl,
  resolveDocumentationUrl,
  resolveTutorialUrl
} from '../documentationUrl'

describe('documentation URL resolution', () => {
  it('uses the local docs server during development', () => {
    expect(getDefaultDocumentationUrl())
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
  })

  it('keeps a same-origin production fallback on the local docs server', () => {
    expect(resolveDocumentationUrl('', PRODUCTION_DOCUMENTATION_URL))
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
  })

  it('keeps a safe administrator-configured URL ahead of environment defaults', () => {
    expect(resolveDocumentationUrl('https://docs.example.com/guide'))
      .toBe('https://docs.example.com/guide')
    expect(resolveDocumentationUrl('/custom-docs/', PRODUCTION_DOCUMENTATION_URL))
      .toBe('/custom-docs/')
  })

  it('keeps production documentation links local during development', () => {
    expect(resolveDocumentationUrl('https://luoxueapi.cc/tutorial-docs'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
    expect(resolveDocumentationUrl('https://luoxueapi.cc/tutorial-docs/?source=admin#old'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/?source=admin#old')
    expect(resolveDocumentationUrl('/tutorial-docs?source=sidebar#quick-start'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/?source=sidebar#quick-start')
    expect(resolveDocumentationUrl('/tutorial-docs/orders#quick-start'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/orders#quick-start')
  })

  it('rewrites the historical first-party documentation host during development', () => {
    expect(resolveDocumentationUrl('https://docs.luoxueapi.cc'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
    expect(resolveDocumentationUrl('https://docs.luoxueapi.cc/orders?source=legacy#quick-start'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/orders?source=legacy#quick-start')
  })

  it('does not rewrite safe custom or lookalike documentation hosts', () => {
    expect(resolveDocumentationUrl('https://docs.example.com/tutorial-docs/'))
      .toBe('https://docs.example.com/tutorial-docs/')
    expect(resolveDocumentationUrl('https://luoxueapi.cc.example.com/tutorial-docs/'))
      .toBe('https://luoxueapi.cc.example.com/tutorial-docs/')
    expect(resolveDocumentationUrl('https://luoxueapi.cc/another-guide/'))
      .toBe('https://luoxueapi.cc/another-guide/')
    expect(resolveDocumentationUrl('/tutorial-docs-old/'))
      .toBe('/tutorial-docs-old/')
    expect(resolveDocumentationUrl('https://docs.luoxueapi.cc.example.com/'))
      .toBe('https://docs.luoxueapi.cc.example.com/')
    expect(resolveDocumentationUrl('https://docs.luoxueapi.cc:444/'))
      .toBe('https://docs.luoxueapi.cc:444/')
  })

  it('rejects unsafe configured and fallback URLs', () => {
    expect(resolveDocumentationUrl('javascript:alert(1)', PRODUCTION_DOCUMENTATION_URL))
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
    expect(resolveDocumentationUrl('', 'javascript:alert(1)'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/')
  })

  it('normalizes tutorial anchors without retaining a stale hash', () => {
    expect(resolveTutorialUrl('/tutorial-docs/#old', 'quick-start', PRODUCTION_DOCUMENTATION_URL))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#quick-start')
    expect(resolveTutorialUrl('https://luoxueapi.cc/tutorial-docs/#old', 'recharge'))
      .toBe('http://127.0.0.1:4179/tutorial-docs/#recharge')
  })
})
