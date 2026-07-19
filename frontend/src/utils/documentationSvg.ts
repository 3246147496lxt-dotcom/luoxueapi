import { sanitizeSvg } from './sanitize'

const BLOCKED_ELEMENT_PATTERN = /<\s*\/?\s*(?:a|animate|animatemotion|animatetransform|audio|canvas|discard|embed|feimage|foreignobject|iframe|image|link|meta|mpath|object|set|style|video)\b/iu
const ACTIVE_ATTRIBUTE_PATTERN = /\s(?:style|src|xml:base)\s*=/iu

export function sanitizeDocumentationIconSvg(value: string | null | undefined): string {
  const sanitized = sanitizeSvg(value ?? '').trim()
  if (!/^<svg(?:\s|>)/iu.test(sanitized)) return ''
  if (BLOCKED_ELEMENT_PATTERN.test(sanitized) || ACTIVE_ATTRIBUTE_PATTERN.test(sanitized)) return ''

  for (const match of sanitized.matchAll(/\b(?:href|xlink:href)\s*=\s*(["'])(.*?)\1/giu)) {
    if (!match[2].trim().startsWith('#')) return ''
  }
  for (const match of sanitized.matchAll(/url\(\s*(["']?)(.*?)\1\s*\)/giu)) {
    if (!match[2].trim().startsWith('#')) return ''
  }
  return sanitized
}
