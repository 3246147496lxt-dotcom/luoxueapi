const QUOTA_VIEWER_AUTHORIZATION_PATH = '/quota-viewer/authorize'
const QUOTA_VIEWER_LANDING_PATH = '/quota-viewer'
const QUOTA_VIEWER_CODE_PATTERN = /^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{8}$/

export interface LocationSnapshot {
  pathname: string
  search?: string
  hash?: string
}

function hasValidQuotaViewerCode(search: string): boolean {
  const params = new URLSearchParams(search)
  const userCodes = params.getAll('user_code')
  if (userCodes.length !== 1) return false

  const compact = userCodes[0].trim().toUpperCase().replace(/[\s-]+/g, '')
  return QUOTA_VIEWER_CODE_PATTERN.test(compact)
}

function hasValidQuotaViewerDownload(search: string): boolean {
  const params = new URLSearchParams(search)
  const platforms = params.getAll('download')
  if (platforms.length !== 1 || Array.from(params.keys()).some((key) => key !== 'download')) {
    return false
  }
  return platforms[0] === 'macos' || platforms[0] === 'windows'
}

export function sessionExpiredLoginURL(location: LocationSnapshot): string {
  const pathname = location.pathname || ''
  const search = location.search || ''
  const hash = location.hash || ''

  const validAuthorization = pathname === QUOTA_VIEWER_AUTHORIZATION_PATH
    && hasValidQuotaViewerCode(search)
  const validDownload = pathname === QUOTA_VIEWER_LANDING_PATH
    && hasValidQuotaViewerDownload(search)

  if (!validAuthorization && !validDownload) {
    return '/login'
  }

  const redirect = `${pathname}${search}${hash}`
  return `/login?redirect=${encodeURIComponent(redirect)}`
}
