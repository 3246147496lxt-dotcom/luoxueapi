import { readonly, ref } from 'vue'
import { createI18n } from 'vue-i18n'
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

const localeLoaders: Record<LocaleCode, () => Promise<{ default: LocaleMessages }>> = {
  en: () => import('./locales/en'),
  zh: () => import('./locales/zh')
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

const loadedLocales = new Set<LocaleCode>()
let localeApplySequence = 0
let preferenceSyncInstalled = false

export const localePreference = readonly(preferenceState)

export async function loadLocaleMessages(locale: LocaleCode): Promise<void> {
  if (loadedLocales.has(locale)) {
    return
  }

  const loader = localeLoaders[locale]
  const module = await loader()
  i18n.global.setLocaleMessage(locale, module.default)
  loadedLocales.add(locale)
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
  const { default: router } = await import('@/router')
  const { useAppStore } = await import('@/stores/app')
  const { useAuthStore } = await import('@/stores/auth')
  const { useAdminSettingsStore } = await import('@/stores/adminSettings')
  const route = router.currentRoute.value
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  const customMenuItems = [
    ...(appStore.cachedPublicSettings?.custom_menu_items ?? []),
    ...(authStore.isAdmin ? adminSettingsStore.customMenuItems : []),
  ]
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

export async function initI18n(): Promise<void> {
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
