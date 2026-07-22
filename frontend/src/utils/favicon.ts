import { sanitizeUrl } from './url'

export const DEFAULT_FAVICON = '/brand/luoxue-snowpuff-extracted.svg'
export const DEFAULT_APPLE_TOUCH_ICON = '/brand/luoxue-snowpuff-extracted-touch-180.png'

function resolveFaviconMimeType(url: string): string {
  if (/\.svg(?:[?#].*)?$/i.test(url) || url.startsWith('data:image/svg+xml')) {
    return 'image/svg+xml'
  }
  if (/\.png(?:[?#].*)?$/i.test(url) || url.startsWith('data:image/png')) {
    return 'image/png'
  }
  return 'image/x-icon'
}

export function syncFavicon(
  customLogoUrl: string | null | undefined,
  targetDocument: Document = document
): HTMLLinkElement {
  const customIconUrl = sanitizeUrl(customLogoUrl || '', {
    allowRelative: true,
    allowDataUrl: true
  })
  const faviconUrl = customIconUrl || DEFAULT_FAVICON
  const touchIconUrl = customIconUrl || DEFAULT_APPLE_TOUCH_ICON
  let link = targetDocument.querySelector<HTMLLinkElement>('link[rel="icon"]')
  let touchLink = targetDocument.querySelector<HTMLLinkElement>('link[rel="apple-touch-icon"]')

  if (!link) {
    link = targetDocument.createElement('link')
    link.rel = 'icon'
    targetDocument.head.appendChild(link)
  }

  if (!touchLink) {
    touchLink = targetDocument.createElement('link')
    touchLink.rel = 'apple-touch-icon'
    touchLink.setAttribute('sizes', '180x180')
    targetDocument.head.appendChild(touchLink)
  }

  link.type = resolveFaviconMimeType(faviconUrl)
  link.href = faviconUrl
  touchLink.href = touchIconUrl
  return link
}
