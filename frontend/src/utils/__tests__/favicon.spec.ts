import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { beforeEach, describe, expect, it } from 'vitest'

import {
  DEFAULT_APPLE_TOUCH_ICON,
  DEFAULT_FAVICON,
  syncFavicon
} from '@/utils/favicon'

const frontendDirectory = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')

describe('syncFavicon', () => {
  beforeEach(() => {
    document.head
      .querySelectorAll('link[rel="icon"], link[rel="apple-touch-icon"]')
      .forEach((link) => link.remove())
  })

  it('uses the canonical browser and touch icons when no custom logo is configured', () => {
    const link = syncFavicon('')
    const touchLink = document.head.querySelector<HTMLLinkElement>('link[rel="apple-touch-icon"]')

    expect(link.getAttribute('href')).toBe(DEFAULT_FAVICON)
    expect(link.type).toBe('image/png')
    expect(touchLink?.getAttribute('href')).toBe(DEFAULT_APPLE_TOUCH_ICON)
    expect(touchLink?.getAttribute('sizes')).toBe('180x180')
  })

  it('uses a custom site logo for both browser and touch icons', () => {
    const link = syncFavicon('/uploads/site-logo.svg?v=2')
    const touchLink = document.head.querySelector<HTMLLinkElement>('link[rel="apple-touch-icon"]')

    expect(link.getAttribute('href')).toBe('/uploads/site-logo.svg?v=2')
    expect(link.type).toBe('image/svg+xml')
    expect(touchLink?.getAttribute('href')).toBe('/uploads/site-logo.svg?v=2')
  })

  it('restores both default icons after a custom logo is cleared', () => {
    const customLink = syncFavicon('data:image/png;base64,cHVycGxl')
    const defaultLink = syncFavicon(null)
    const touchLink = document.head.querySelector<HTMLLinkElement>('link[rel="apple-touch-icon"]')

    expect(defaultLink).toBe(customLink)
    expect(defaultLink.getAttribute('href')).toBe(DEFAULT_FAVICON)
    expect(defaultLink.type).toBe('image/png')
    expect(touchLink?.getAttribute('href')).toBe(DEFAULT_APPLE_TOUCH_ICON)
    expect(document.head.querySelectorAll('link[rel="icon"]')).toHaveLength(1)
    expect(document.head.querySelectorAll('link[rel="apple-touch-icon"]')).toHaveLength(1)
  })

  it('falls back to the built-in icons for an unsafe custom URL', () => {
    syncFavicon('javascript:alert(1)')

    expect(document.head.querySelector<HTMLLinkElement>('link[rel="icon"]')?.getAttribute('href'))
      .toBe(DEFAULT_FAVICON)
    expect(document.head.querySelector<HTMLLinkElement>('link[rel="apple-touch-icon"]')?.getAttribute('href'))
      .toBe(DEFAULT_APPLE_TOUCH_ICON)
  })

  it('keeps the initial HTML icon fallbacks aligned with the runtime defaults', () => {
    const indexHtml = readFileSync(resolve(frontendDirectory, 'index.html'), 'utf8')

    expect(indexHtml).toContain(`rel="icon" type="image/png" href="${DEFAULT_FAVICON}"`)
    expect(indexHtml).toContain(
      `rel="apple-touch-icon" sizes="180x180" href="${DEFAULT_APPLE_TOUCH_ICON}"`
    )
  })
})
