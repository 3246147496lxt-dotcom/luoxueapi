/** Safe, non-secret context shared with external custom pages. */

export type EmbeddedUIMode = 'embedded' | 'new_tab'

export interface EmbeddedPageContext {
  theme: 'light' | 'dark'
  lang?: string
  ui_mode: EmbeddedUIMode
  src_host?: string
}

export interface EmbeddedUrlOptions {
  theme?: 'light' | 'dark'
  lang?: string
  uiMode?: EmbeddedUIMode
}

export function buildEmbeddedPageContext(
  options: EmbeddedUrlOptions = {},
): EmbeddedPageContext {
  const context: EmbeddedPageContext = {
    theme: options.theme ?? 'light',
    ui_mode: options.uiMode ?? 'embedded',
  }
  if (options.lang?.trim()) context.lang = options.lang.trim()
  if (typeof window !== 'undefined' && window.location.origin) {
    context.src_host = window.location.origin
  }
  return context
}

/**
 * Add only presentation context to an external URL.
 *
 * Authentication is intentionally excluded. Pages that opt into
 * `exchange_code` receive a one-time launch URL from the backend instead.
 */
export function buildEmbeddedUrl(
  baseUrl: string,
  options: EmbeddedUrlOptions = {},
): string {
  if (!baseUrl) return baseUrl
  try {
    const url = new URL(baseUrl)
    const context = buildEmbeddedPageContext(options)
    url.search = ''
    url.hash = ''
    url.searchParams.set('theme', context.theme)
    if (context.lang) url.searchParams.set('lang', context.lang)
    url.searchParams.set('ui_mode', context.ui_mode)
    if (context.src_host) url.searchParams.set('src_host', context.src_host)
    return url.toString()
  } catch {
    // Fail closed: returning an unparseable original string could preserve a
    // legacy token/query fragment if a future caller navigates without first
    // validating the configured URL.
    return ''
  }
}

export function detectTheme(): 'light' | 'dark' {
  if (typeof document === 'undefined') return 'light'
  return document.documentElement.classList.contains('dark') ? 'dark' : 'light'
}
