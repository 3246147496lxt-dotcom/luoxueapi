export type LocaleCode = 'en' | 'zh'
export type LocalePreference = 'auto' | LocaleCode

export const LOCALE_STORAGE_KEY = 'sub2api_locale'
export const DEFAULT_LOCALE: LocaleCode = 'en'

function getStorage(): Storage | null {
  if (typeof localStorage === 'undefined') return null

  try {
    return localStorage
  } catch {
    return null
  }
}

function getBrowserLanguage(): string {
  if (typeof navigator === 'undefined') return DEFAULT_LOCALE
  return navigator.language
}

export function isLocaleCode(value: string): value is LocaleCode {
  return value === 'en' || value === 'zh'
}

export function parseLocalePreference(value: string | null): LocalePreference {
  return value && isLocaleCode(value) ? value : 'auto'
}

export function readLocalePreference(storage: Storage | null = getStorage()): LocalePreference {
  if (!storage) return 'auto'

  try {
    return parseLocalePreference(storage.getItem(LOCALE_STORAGE_KEY))
  } catch {
    return 'auto'
  }
}

export function detectBrowserLocale(language = getBrowserLanguage()): LocaleCode {
  return language.toLowerCase().startsWith('zh') ? 'zh' : DEFAULT_LOCALE
}

export function resolveLocalePreference(
  preference: LocalePreference,
  browserLanguage = getBrowserLanguage(),
): LocaleCode {
  return preference === 'auto' ? detectBrowserLocale(browserLanguage) : preference
}

export function persistLocalePreference(
  preference: LocalePreference,
  storage: Storage | null = getStorage(),
) {
  if (!storage) return

  try {
    if (preference === 'auto') {
      storage.removeItem(LOCALE_STORAGE_KEY)
    } else {
      storage.setItem(LOCALE_STORAGE_KEY, preference)
    }
  } catch {
    // Storage can be unavailable in privacy-restricted browsing contexts.
  }
}
