import type { LibraryFile } from '@/types/library'

export function formatLibraryBytes(value: number, locale?: string): string {
  if (!Number.isFinite(value) || value <= 0) return '0 KB'
  const units = ['B', 'KB', 'MB', 'GB', 'TB'] as const
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const amount = value / (1024 ** exponent)
  const maximumFractionDigits = exponent === 0 || amount >= 10 ? 0 : 1
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits }).format(amount)} ${units[exponent]}`
}

export function formatLibraryDate(value: string, locale?: string): string {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return '—'
  const now = Date.now()
  const sameYear = date.getFullYear() === new Date(now).getFullYear()
  return new Intl.DateTimeFormat(locale, {
    ...(sameYear ? {} : { year: 'numeric' }),
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export function libraryFileExtension(file: Pick<LibraryFile, 'name' | 'type'>): string {
  const extension = file.name.split('.').pop()?.trim().toUpperCase() ?? ''
  if (extension && extension.length <= 6 && extension !== file.name.toUpperCase()) return extension
  const fallback: Record<LibraryFile['type'], string> = {
    image: 'IMG',
    pdf: 'PDF',
    document: 'DOC',
    spreadsheet: 'XLS',
    presentation: 'PPT',
    other: 'FILE',
  }
  return fallback[file.type]
}

export function libraryFileCanPreview(file: LibraryFile): boolean {
  return file.previewable ?? (file.type === 'image' || file.type === 'pdf')
}

export function saveLibraryBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = name
  anchor.rel = 'noopener'
  anchor.style.display = 'none'
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}

const libraryDownloadPathPattern = /^(.*\/api\/v1\/library\/download\/)([A-Za-z0-9_-]{22})$/
const libraryDownloadPreparationLifetimeMs = 120_000
const libraryDownloadNavigationLifetimeMs = 16 * 60_000

interface ResolvedLibraryDownloadURL {
  url: URL
  pathPrefix: string
}

export interface PreparedLibraryBrowserDownload {
  start(downloadPath: string): void
  cancel(): void
}

function resolveLibraryDownloadURL(value: string): ResolvedLibraryDownloadURL {
  const resolved = new URL(value, window.location.href)
  const pathMatch = resolved.pathname.match(libraryDownloadPathPattern)
  if (
    (resolved.protocol !== 'http:' && resolved.protocol !== 'https:')
    || resolved.username
    || resolved.password
    || resolved.search
    || resolved.hash
    || !pathMatch?.[1]
  ) {
    throw new TypeError('Invalid library download URL.')
  }
  return { url: resolved, pathPrefix: pathMatch[1] }
}

/**
 * Reserves the browser navigation synchronously, before the ticket request can
 * consume transient user activation. The returned starter then accepts only a
 * canonical ticket URL on the exact origin and deployment prefix that were
 * approved by the probe URL.
 */
export function prepareLibraryBrowserDownload(probePath: string): PreparedLibraryBrowserDownload {
  const probe = resolveLibraryDownloadURL(probePath)
  const sameOrigin = probe.url.origin === window.location.origin
  let frame: HTMLIFrameElement | null = null
  let popup: Window | null = null
  let cleanupTimer: number | undefined
  let started = false
  let cancelled = false

  const clearCleanupTimer = (): void => {
    if (cleanupTimer === undefined) return
    window.clearTimeout(cleanupTimer)
    cleanupTimer = undefined
  }
  const cleanup = (): void => {
    clearCleanupTimer()
    frame?.remove()
    frame = null
    if (popup && !popup.closed) popup.close()
    popup = null
  }
  const scheduleCleanup = (delay: number): void => {
    clearCleanupTimer()
    cleanupTimer = window.setTimeout(() => {
      cancelled = true
      cleanup()
    }, delay)
  }

  if (!sameOrigin) {
    const popupName = `sub2api-library-download-${Date.now()}-${Math.random().toString(36).slice(2)}`
    popup = window.open('about:blank', popupName)
    if (!popup) throw new Error('The browser blocked the download window.')
    try {
      popup.opener = null
      if (popup.opener !== null) throw new Error('Unable to isolate the download window.')
    } catch {
      cleanup()
      throw new Error('Unable to isolate the download window.')
    }
    scheduleCleanup(libraryDownloadPreparationLifetimeMs)
  }

  return {
    start(downloadPath: string): void {
      if (started || cancelled) throw new Error('The library download has already been used.')

      let resolved: ResolvedLibraryDownloadURL
      try {
        resolved = resolveLibraryDownloadURL(downloadPath)
        if (
          resolved.url.origin !== probe.url.origin
          || resolved.pathPrefix !== probe.pathPrefix
        ) {
          throw new TypeError('Invalid library download URL.')
        }
      } catch (error) {
        cancelled = true
        cleanup()
        throw error
      }

      started = true
      clearCleanupTimer()
      if (sameOrigin) {
        try {
          frame = document.createElement('iframe')
          frame.src = resolved.url.href
          frame.hidden = true
          frame.tabIndex = -1
          frame.setAttribute('aria-hidden', 'true')
          frame.setAttribute('referrerpolicy', 'no-referrer')
          frame.style.display = 'none'
          document.body.append(frame)
          scheduleCleanup(libraryDownloadNavigationLifetimeMs)
        } catch (error) {
          cancelled = true
          cleanup()
          throw error
        }
        return
      }

      const reservedPopup = popup
      if (!reservedPopup || reservedPopup.closed) {
        cancelled = true
        cleanup()
        throw new Error('The download window is no longer available.')
      }
      try {
        reservedPopup.location.replace(resolved.url.href)
        scheduleCleanup(libraryDownloadNavigationLifetimeMs)
      } catch (error) {
        cancelled = true
        cleanup()
        throw error
      }
    },
    cancel(): void {
      if (cancelled) return
      cancelled = true
      cleanup()
    },
  }
}
