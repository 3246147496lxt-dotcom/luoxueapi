import type { LocationQuery } from 'vue-router'

/**
 * Read only an unambiguous scalar query value. Repeated parameters are treated
 * as invalid so route guards and purchase views cannot interpret them
 * differently.
 */
export function readSingleQueryString(
  query: LocationQuery,
  key: string,
): string {
  const value = query[key]
  return typeof value === 'string' ? value : ''
}

/**
 * A syntactically complete WeChat payment return always carries the explicit
 * resume marker plus at least one unambiguous scalar credential. Auxiliary
 * callback fields alone must never turn a fresh subscription checkout into a
 * recovery flow.
 */
export function hasCompleteWechatResumeQuery(
  query: LocationQuery,
): boolean {
  if (readSingleQueryString(query, 'wechat_resume') !== '1') {
    return false
  }

  return readSingleQueryString(query, 'wechat_resume_token').trim() !== ''
    || readSingleQueryString(query, 'openid').trim() !== ''
}
