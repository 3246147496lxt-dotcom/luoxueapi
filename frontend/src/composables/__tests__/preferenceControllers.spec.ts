import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const titleMocks = vi.hoisted(() => ({
  resolveRouteDocumentTitle: vi.fn(() => 'Resolved route title'),
}))

vi.mock('@/router/title', () => ({
  resolveRouteDocumentTitle: titleMocks.resolveRouteDocumentTitle,
}))

vi.mock('@/router', () => ({
  default: {
    currentRoute: {
      value: { name: 'dashboard', meta: {} },
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    siteName: '落雪API',
    cachedPublicSettings: null,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isAdmin: false,
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    customMenuItems: [],
  }),
}))

import {
  initializeThemePreference,
  resolveThemePreference,
  setThemePreference,
  stopThemePreferenceSync,
  useThemePreference,
} from '@/composables/useThemePreference'
import { useLocalePreference } from '@/composables/useLocalePreference'
import {
  initI18n,
  setLocale,
  setLocalePreference,
  stopLocalePreferenceSync,
} from '@/i18n'
import {
  detectBrowserLocale,
  parseLocalePreference,
  resolveLocalePreference,
} from '@/i18n/preference'

class FakeMediaQueryList {
  matches: boolean
  readonly media = '(prefers-color-scheme: dark)'
  onchange: ((this: MediaQueryList, ev: MediaQueryListEvent) => any) | null = null
  private listeners = new Set<() => void>()

  constructor(matches: boolean) {
    this.matches = matches
  }

  addEventListener(_type: string, listener: () => void) {
    this.listeners.add(listener)
  }

  removeEventListener(_type: string, listener: () => void) {
    this.listeners.delete(listener)
  }

  addListener(listener: () => void) {
    this.listeners.add(listener)
  }

  removeListener(listener: () => void) {
    this.listeners.delete(listener)
  }

  dispatchEvent(): boolean {
    this.listeners.forEach(listener => listener())
    return true
  }

  setMatches(matches: boolean) {
    this.matches = matches
    this.dispatchEvent()
  }
}

let colorSchemeQuery: FakeMediaQueryList

function setBrowserLanguage(language: string) {
  Object.defineProperty(window.navigator, 'language', {
    configurable: true,
    value: language,
  })
}

describe('preference controllers', () => {
  beforeEach(() => {
    stopThemePreferenceSync()
    stopLocalePreferenceSync()
    localStorage.clear()
    document.documentElement.classList.remove('dark')
    document.documentElement.lang = ''
    document.title = ''
    titleMocks.resolveRouteDocumentTitle.mockClear()
    colorSchemeQuery = new FakeMediaQueryList(false)
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn(() => colorSchemeQuery),
    })
    setBrowserLanguage('en-US')
  })

  afterEach(() => {
    stopThemePreferenceSync()
    stopLocalePreferenceSync()
  })

  describe('theme preference', () => {
    it('treats missing storage as system and follows color-scheme changes', () => {
      colorSchemeQuery.matches = true

      expect(initializeThemePreference()).toBe('system')
      const controller = useThemePreference()

      expect(controller.preference.value).toBe('system')
      expect(controller.isDark.value).toBe(true)
      expect(document.documentElement.classList.contains('dark')).toBe(true)

      colorSchemeQuery.setMatches(false)

      expect(controller.isDark.value).toBe(false)
      expect(document.documentElement.classList.contains('dark')).toBe(false)
    })

    it('keeps legacy light/dark values and removes storage for system', () => {
      localStorage.setItem('theme', 'light')
      colorSchemeQuery.matches = true

      initializeThemePreference()
      const controller = useThemePreference()

      expect(controller.preference.value).toBe('light')
      expect(controller.isDark.value).toBe(false)

      controller.setPreference('dark')
      expect(localStorage.getItem('theme')).toBe('dark')
      expect(controller.isDark.value).toBe(true)

      controller.setPreference('system')
      expect(localStorage.getItem('theme')).toBeNull()
      expect(controller.preference.value).toBe('system')
      expect(controller.isDark.value).toBe(true)
    })

    it('resolves explicit themes independently from the system theme', () => {
      expect(resolveThemePreference('system', true)).toBe('dark')
      expect(resolveThemePreference('system', false)).toBe('light')
      expect(resolveThemePreference('light', true)).toBe('light')
      expect(resolveThemePreference('dark', false)).toBe('dark')

      setThemePreference('light')
      colorSchemeQuery.setMatches(true)
      expect(document.documentElement.classList.contains('dark')).toBe(false)
    })
  })

  describe('locale preference', () => {
    it('normalizes legacy values and detects Chinese browser locales', () => {
      expect(parseLocalePreference('en')).toBe('en')
      expect(parseLocalePreference('zh')).toBe('zh')
      expect(parseLocalePreference(null)).toBe('auto')
      expect(parseLocalePreference('unsupported')).toBe('auto')
      expect(detectBrowserLocale('zh-CN')).toBe('zh')
      expect(detectBrowserLocale('en-GB')).toBe('en')
      expect(resolveLocalePreference('auto', 'zh-Hant')).toBe('zh')
      expect(resolveLocalePreference('en', 'zh-CN')).toBe('en')
    })

    it('uses auto when storage is empty and keeps lang in sync', async () => {
      setBrowserLanguage('zh-CN')
      await initI18n()
      const controller = useLocalePreference()

      expect(controller.preference.value).toBe('auto')
      expect(controller.resolvedLocale.value).toBe('zh')
      expect(document.documentElement.lang).toBe('zh')
      expect(localStorage.getItem('sub2api_locale')).toBeNull()
    })

    it('hydrates the explicit locale written by the legacy switcher', async () => {
      localStorage.setItem('sub2api_locale', 'zh')
      setBrowserLanguage('en-US')

      await initI18n()
      const controller = useLocalePreference()

      expect(controller.preference.value).toBe('zh')
      expect(controller.resolvedLocale.value).toBe('zh')
      expect(document.documentElement.lang).toBe('zh')
    })

    it('persists explicit locales and removes storage when returning to auto', async () => {
      setBrowserLanguage('zh-CN')
      await initI18n()
      const controller = useLocalePreference()

      await controller.setPreference('en')

      expect(controller.preference.value).toBe('en')
      expect(controller.resolvedLocale.value).toBe('en')
      expect(localStorage.getItem('sub2api_locale')).toBe('en')
      expect(document.documentElement.lang).toBe('en')
      expect(document.title).toBe('Resolved route title')

      await controller.setPreference('auto')

      expect(controller.preference.value).toBe('auto')
      expect(controller.resolvedLocale.value).toBe('zh')
      expect(localStorage.getItem('sub2api_locale')).toBeNull()
      expect(document.documentElement.lang).toBe('zh')
    })

    it('follows languagechange only while preference is auto', async () => {
      await initI18n()
      const controller = useLocalePreference()

      setBrowserLanguage('zh-CN')
      window.dispatchEvent(new Event('languagechange'))
      await vi.waitFor(() => expect(controller.resolvedLocale.value).toBe('zh'))

      await controller.setPreference('en')
      setBrowserLanguage('zh-TW')
      window.dispatchEvent(new Event('languagechange'))

      await new Promise(resolve => setTimeout(resolve, 0))
      expect(controller.resolvedLocale.value).toBe('en')
    })

    it('keeps setLocale as the explicit-locale compatibility API', async () => {
      await initI18n()

      await setLocale('zh')

      expect(localStorage.getItem('sub2api_locale')).toBe('zh')
      expect(document.documentElement.lang).toBe('zh')

      await setLocalePreference('auto')
      expect(localStorage.getItem('sub2api_locale')).toBeNull()
    })
  })
})
