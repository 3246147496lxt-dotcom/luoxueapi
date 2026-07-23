import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  buildEmbeddedPageContext,
  buildEmbeddedUrl,
  detectTheme,
} from '../embedded-url'

describe('embedded-url', () => {
  const originalLocation = window.location

  beforeEach(() => {
    Object.defineProperty(window, 'location', {
      value: {
        origin: 'https://app.example.com',
        href: 'https://app.example.com/user/custom/example',
      },
      writable: true,
      configurable: true,
    })
  })

  afterEach(() => {
    Object.defineProperty(window, 'location', {
      value: originalLocation,
      writable: true,
      configurable: true,
    })
    document.documentElement.classList.remove('dark')
    vi.restoreAllMocks()
  })

  it('adds only non-secret presentation and source-host context', () => {
    const result = buildEmbeddedUrl(
      'https://external.example/checkout?plan=pro&token=legacy&USER_ID=7&src_url=https%3A%2F%2Fsecret.example%2Fpath&s2a_launch_code=stale#token=legacy-fragment',
      {
      theme: 'dark',
      lang: 'zh-CN',
      uiMode: 'embedded',
      },
    )

    const url = new URL(result)
    expect(Object.fromEntries(url.searchParams)).toEqual({
      theme: 'dark',
      lang: 'zh-CN',
      ui_mode: 'embedded',
      src_host: 'https://app.example.com',
    })
    expect(url.searchParams.has('token')).toBe(false)
    expect(url.searchParams.has('user_id')).toBe(false)
    expect(url.searchParams.has('src_url')).toBe(false)
    expect(url.searchParams.has('USER_ID')).toBe(false)
    expect(url.searchParams.has('s2a_launch_code')).toBe(false)
    expect(url.hash).toBe('')
  })

  it('builds a separate new-tab context without the current page URL', () => {
    expect(buildEmbeddedPageContext({
      theme: 'light',
      lang: 'en-US',
      uiMode: 'new_tab',
    })).toEqual({
      theme: 'light',
      lang: 'en-US',
      ui_mode: 'new_tab',
      src_host: 'https://app.example.com',
    })
  })

  it('fails closed for invalid URL input', () => {
    expect(buildEmbeddedUrl('not a url?token=legacy', { theme: 'light' })).toBe('')
  })

  it('detects dark mode from the document root class', () => {
    document.documentElement.classList.add('dark')
    expect(detectTheme()).toBe('dark')
  })
})
