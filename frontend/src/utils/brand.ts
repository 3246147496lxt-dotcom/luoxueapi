export interface BrandNameParts {
  base: string
  apiSuffix: string
}

/**
 * Keeps administrator-defined brand text intact while exposing a trailing
 * "API" token for the shared two-colour Luoxue wordmark treatment.
 */
export function splitBrandApiSuffix(value: string): BrandNameParts {
  const normalized = value.trim()
  const match = /^(.*?)(?:\s*)(api)$/iu.exec(normalized)

  if (!match?.[1]?.trim()) {
    return { base: normalized, apiSuffix: '' }
  }

  return {
    base: match[1].trimEnd(),
    apiSuffix: match[2]
  }
}
