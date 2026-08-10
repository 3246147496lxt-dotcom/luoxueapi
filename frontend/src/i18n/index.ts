import { readonly, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import type { Router } from 'vue-router'
import {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  isLocaleCode,
  parseLocalePreference,
  persistLocalePreference,
  readLocalePreference,
  resolveLocalePreference,
  type LocaleCode,
  type LocalePreference,
} from './preference'

export type { LocaleCode, LocalePreference } from './preference'
export {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  detectBrowserLocale,
  parseLocalePreference,
  readLocalePreference,
  resolveLocalePreference,
} from './preference'

type LocaleMessages = Record<string, any>
export type I18nSurface = 'user' | 'admin'

const localeLoaders: Record<I18nSurface, Record<LocaleCode, () => Promise<{ default: LocaleMessages }>>> = {
  user: {
    en: () => import('./locales/en/portal'),
    zh: () => import('./locales/zh/portal'),
  },
  admin: {
    en: () => import('./locales/en'),
    zh: () => import('./locales/zh'),
  },
}

const preferenceState = ref<LocalePreference>(readLocalePreference())

export const i18n = createI18n({
  legacy: false,
  locale: resolveLocalePreference(preferenceState.value),
  fallbackLocale: DEFAULT_LOCALE,
  messages: {},
  // 禁用 HTML 消息警告 - 引导步骤使用富文本内容（driver.js 支持 HTML）
  // 这些内容是内部定义的，不存在 XSS 风险
  warnHtmlMessage: false
})

const loadedLocales = new Set<string>()
let localeApplySequence = 0
let preferenceSyncInstalled = false
let activeSurface: I18nSurface = 'user'
let activeRouter: Router | null = null

export const localePreference = readonly(preferenceState)

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  const loadKey = `${activeSurface}:${locale}`
  if (loadedLocales.has(loadKey)) {
    return
  }

  const loader = localeLoaders[activeSurface][locale]
  const module = await loader()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(loadKey)
}

async function applyLocale(locale: LocaleCode, updateTitle: boolean): Promise<void> {
  const requestSequence = ++localeApplySequence
  await loadLocaleMessages(locale)
  if (requestSequence !== localeApplySequence) return

  i18n.global.locale.value = locale
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('lang', locale)
  }

  if (!updateTitle || typeof document === 'undefined') return

  // 同步更新浏览器页签标题，使其跟随语言切换
  const { resolveRouteDocumentTitle } = await import('@/router/title')
  const { useAppStore } = await import('@/stores/app')
  const router = activeRouter ?? (await import('@/router')).default
  const route = router.currentRoute.value
  const appStore = useAppStore()
  const customMenuItems = [...(appStore.cachedPublicSettings?.custom_menu_items ?? [])]
  if (activeSurface === 'admin') {
    const { useAdminSettingsStore } = await import('@/stores/adminSettings')
    customMenuItems.push(...useAdminSettingsStore().customMenuItems)
  }
  document.title = resolveRouteDocumentTitle(route, appStore.siteName, customMenuItems)
}

function handleBrowserLanguageChange() {
  if (preferenceState.value !== 'auto') return
  void applyLocale(resolveLocalePreference('auto'), true)
}

function handleLocaleStorage(event: StorageEvent) {
  if (event.key !== LOCALE_STORAGE_KEY) return

  preferenceState.value = parseLocalePreference(event.newValue)
  void applyLocale(resolveLocalePreference(preferenceState.value), true)
}

export function installLocalePreferenceSync() {
  if (preferenceSyncInstalled || typeof window === 'undefined') return

  preferenceSyncInstalled = true
  window.addEventListener('languagechange', handleBrowserLanguageChange)
  window.addEventListener('storage', handleLocaleStorage)
}

export function stopLocalePreferenceSync() {
  if (!preferenceSyncInstalled || typeof window === 'undefined') return

  preferenceSyncInstalled = false
  window.removeEventListener('languagechange', handleBrowserLanguageChange)
  window.removeEventListener('storage', handleLocaleStorage)
}

export async function initI18n(options?: {
  surface?: I18nSurface
  router?: Router
}): Promise<void> {
  activeSurface = options?.surface ?? 'user'
  activeRouter = options?.router ?? null
  preferenceState.value = readLocalePreference()
  await applyLocale(resolveLocalePreference(preferenceState.value), false)
  installLocalePreferenceSync()
}

export async function setLocalePreference(preference: LocalePreference): Promise<void> {
  preferenceState.value = preference
  persistLocalePreference(preference)
  await applyLocale(resolveLocalePreference(preference), true)
}

export async function setLocale(locale: string): Promise<void> {
  if (!isLocaleCode(locale)) return
  await setLocalePreference(locale)
}

export function getLocale(): LocaleCode {
  const current = i18n.global.locale.value
  return isLocaleCode(current) ? current : DEFAULT_LOCALE
}

export const availableLocales = [
  { code: 'en', name: 'English', flag: '🇺🇸' },
  { code: 'zh', name: '中文', flag: '🇨🇳' }
] as const

if (import.meta.hot) {
  import.meta.hot.dispose(stopLocalePreferenceSync)
}

export default i18n
