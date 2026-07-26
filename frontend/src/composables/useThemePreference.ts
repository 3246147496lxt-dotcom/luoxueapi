import { computed, readonly, ref } from 'vue'

export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = Exclude<ThemePreference, 'system'>

export const THEME_STORAGE_KEY = 'theme'

const preferenceState = ref<ThemePreference>('system')
const resolvedThemeState = ref<ResolvedTheme>('light')

let initialized = false
let colorSchemeQuery: MediaQueryList | null = null

function getStorage(): Storage | null {
  if (typeof localStorage === 'undefined') return null

  try {
    return localStorage
  } catch {
    return null
  }
}

function getColorSchemeQuery(): MediaQueryList | null {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return null
  }

  return window.matchMedia('(prefers-color-scheme: dark)')
}

export function parseThemePreference(value: string | null): ThemePreference {
  return value === 'light' || value === 'dark' ? value : 'system'
}

export function readThemePreference(storage: Storage | null = getStorage()): ThemePreference {
  if (!storage) return 'system'

  try {
    return parseThemePreference(storage.getItem(THEME_STORAGE_KEY))
  } catch {
    return 'system'
  }
}

export function resolveThemePreference(
  preference: ThemePreference,
  prefersDark = getColorSchemeQuery()?.matches ?? false,
): ResolvedTheme {
  if (preference === 'system') {
    return prefersDark ? 'dark' : 'light'
  }

  return preference
}

function applyThemePreference() {
  const resolved = resolveThemePreference(
    preferenceState.value,
    colorSchemeQuery?.matches ?? false,
  )
  resolvedThemeState.value = resolved

  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', resolved === 'dark')
  }
}

function persistThemePreference(preference: ThemePreference) {
  const storage = getStorage()
  if (!storage) return

  try {
    if (preference === 'system') {
      storage.removeItem(THEME_STORAGE_KEY)
    } else {
      storage.setItem(THEME_STORAGE_KEY, preference)
    }
  } catch {
    // Storage can be unavailable in privacy-restricted browsing contexts.
  }
}

function handleColorSchemeChange() {
  if (preferenceState.value === 'system') {
    applyThemePreference()
  }
}

function handleThemeStorage(event: StorageEvent) {
  if (event.key !== THEME_STORAGE_KEY) return

  preferenceState.value = parseThemePreference(event.newValue)
  applyThemePreference()
}

function addColorSchemeListener(query: MediaQueryList) {
  if (typeof query.addEventListener === 'function') {
    query.addEventListener('change', handleColorSchemeChange)
  } else {
    query.addListener(handleColorSchemeChange)
  }
}

function removeColorSchemeListener(query: MediaQueryList) {
  if (typeof query.removeEventListener === 'function') {
    query.removeEventListener('change', handleColorSchemeChange)
  } else {
    query.removeListener(handleColorSchemeChange)
  }
}

export function initializeThemePreference(): ThemePreference {
  preferenceState.value = readThemePreference()
  colorSchemeQuery ??= getColorSchemeQuery()
  applyThemePreference()

  if (!initialized && typeof window !== 'undefined') {
    initialized = true
    if (colorSchemeQuery) {
      addColorSchemeListener(colorSchemeQuery)
    }
    window.addEventListener('storage', handleThemeStorage)
  }

  return preferenceState.value
}

export function setThemePreference(preference: ThemePreference) {
  preferenceState.value = preference
  persistThemePreference(preference)
  colorSchemeQuery ??= getColorSchemeQuery()
  applyThemePreference()
}

export function stopThemePreferenceSync() {
  if (!initialized || typeof window === 'undefined') return

  initialized = false
  if (colorSchemeQuery) {
    removeColorSchemeListener(colorSchemeQuery)
  }
  window.removeEventListener('storage', handleThemeStorage)
  colorSchemeQuery = null
}

export function useThemePreference() {
  initializeThemePreference()

  const isDark = computed(() => resolvedThemeState.value === 'dark')

  function setDarkMode(enabled: boolean) {
    setThemePreference(enabled ? 'dark' : 'light')
  }

  function toggleTheme() {
    setDarkMode(!isDark.value)
  }

  return {
    preference: readonly(preferenceState),
    resolvedTheme: readonly(resolvedThemeState),
    isDark,
    setPreference: setThemePreference,
    // Compatibility helpers for compact public/auth controls.
    setDarkMode,
    toggleTheme,
  }
}

if (import.meta.hot) {
  import.meta.hot.dispose(stopThemePreferenceSync)
}
