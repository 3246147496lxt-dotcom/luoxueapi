import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { prepareLibraryBrowserDownload } from '../libraryFileUi'

const downloadHandle = 'ABCdef0123456789_-abcd'
const probeHandle = 'AAAAAAAAAAAAAAAAAAAAAA'
const downloadPath = `/api/v1/library/download/${downloadHandle}`
const probePath = `/api/v1/library/download/${probeHandle}`
const navigationLifetimeMs = 16 * 60_000

beforeEach(() => {
  vi.useFakeTimers()
  document.body.replaceChildren()
})

afterEach(() => {
  vi.restoreAllMocks()
  vi.useRealTimers()
  document.body.replaceChildren()
})

describe('prepareLibraryBrowserDownload', () => {
  it('uses a hidden same-origin frame for the complete server operation window', () => {
    const unrelatedFrame = document.createElement('iframe')
    unrelatedFrame.id = 'unrelated-frame'
    document.body.append(unrelatedFrame)

    const prepared = prepareLibraryBrowserDownload(probePath)
    prepared.start(downloadPath)

    const frames = Array.from(document.querySelectorAll('iframe'))
    const downloadFrame = frames.find((frame) => frame !== unrelatedFrame)
    expect(downloadFrame).toBeDefined()
    expect(downloadFrame?.src).toBe(`${window.location.origin}${downloadPath}`)
    expect(downloadFrame?.hidden).toBe(true)
    expect(downloadFrame?.tabIndex).toBe(-1)
    expect(downloadFrame?.getAttribute('aria-hidden')).toBe('true')
    expect(downloadFrame?.hasAttribute('sandbox')).toBe(false)
    expect(downloadFrame?.getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(document.querySelector('a')).toBeNull()

    vi.advanceTimersByTime(navigationLifetimeMs - 1)
    expect(downloadFrame?.isConnected).toBe(true)
    vi.advanceTimersByTime(1)
    expect(downloadFrame?.isConnected).toBe(false)
    expect(unrelatedFrame.isConnected).toBe(true)
  })

  it('locks a same-origin deployment prefix between preparation and navigation', () => {
    const prefixedProbe = `/sub2api${probePath}`
    const prefixedDownload = `/sub2api${downloadPath}`
    const prepared = prepareLibraryBrowserDownload(prefixedProbe)

    prepared.start(prefixedDownload)

    const frame = document.querySelector('iframe')
    expect(frame?.src).toBe(`${window.location.origin}${prefixedDownload}`)
    expect(frame?.getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(document.querySelector('a')).toBeNull()
  })

  it('reserves an isolated cross-origin window before navigating it', () => {
    const replace = vi.fn()
    const close = vi.fn()
    const popup = {
      closed: false,
      opener: window,
      location: { replace },
      close,
    } as unknown as Window
    const open = vi.spyOn(window, 'open').mockReturnValue(popup)
    const crossOriginProbe = `https://files.example.test${probePath}`
    const crossOriginDownload = `https://files.example.test${downloadPath}`

    const prepared = prepareLibraryBrowserDownload(crossOriginProbe)

    expect(open).toHaveBeenCalledOnce()
    expect(open).toHaveBeenCalledWith('about:blank', expect.stringMatching(/^sub2api-library-download-/))
    expect(popup.opener).toBeNull()
    expect(replace).not.toHaveBeenCalled()

    prepared.start(crossOriginDownload)

    expect(replace).toHaveBeenCalledWith(crossOriginDownload)
    expect(document.querySelector('a')).toBeNull()
    expect(document.querySelector('iframe')).toBeNull()
    vi.advanceTimersByTime(navigationLifetimeMs - 1)
    expect(close).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(close).toHaveBeenCalledOnce()
  })

  it('fails before ticket issuance when the browser blocks the cross-origin window', () => {
    vi.spyOn(window, 'open').mockReturnValue(null)

    expect(() => prepareLibraryBrowserDownload(`https://files.example.test${probePath}`)).toThrow(
      'The browser blocked the download window.',
    )
    expect(document.body.childElementCount).toBe(0)
  })

  it('cancels an unused cross-origin reservation precisely', () => {
    const close = vi.fn()
    const popup = {
      closed: false,
      opener: window,
      location: { replace: vi.fn() },
      close,
    } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(popup)
    const prepared = prepareLibraryBrowserDownload(`https://files.example.test${probePath}`)

    prepared.cancel()
    prepared.cancel()

    expect(close).toHaveBeenCalledOnce()
    expect(() => prepared.start(`https://files.example.test${downloadPath}`)).toThrow(
      'The library download has already been used.',
    )
  })

  it.each([
    `https://other.example.test${downloadPath}`,
    `https://files.example.test/other${downloadPath}`,
    `https://files.example.test${downloadPath}?code=secret`,
    `https://files.example.test${downloadPath}#secret`,
  ])('rejects a final URL outside the prepared origin and prefix: %s', (invalidURL) => {
    const close = vi.fn()
    const popup = {
      closed: false,
      opener: window,
      location: { replace: vi.fn() },
      close,
    } as unknown as Window
    vi.spyOn(window, 'open').mockReturnValue(popup)
    const prepared = prepareLibraryBrowserDownload(`https://files.example.test${probePath}`)

    expect(() => prepared.start(invalidURL)).toThrow(TypeError)
    expect(close).toHaveBeenCalledOnce()
  })

  it.each([
    `${probePath}?code=secret`,
    `${probePath}#secret`,
    `${probePath}/extra`,
    '/api/v1/library/download/too-short',
    'javascript:alert(1)',
  ])('rejects invalid probe URL %s before creating navigation state', (invalidURL) => {
    const open = vi.spyOn(window, 'open')

    expect(() => prepareLibraryBrowserDownload(invalidURL)).toThrow(TypeError)
    expect(open).not.toHaveBeenCalled()
    expect(document.body.childElementCount).toBe(0)
  })
})
