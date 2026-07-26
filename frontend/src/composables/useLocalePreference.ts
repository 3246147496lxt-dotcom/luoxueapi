import { computed } from 'vue'
import {
  getLocale,
  installLocalePreferenceSync,
  localePreference,
  setLocalePreference,
  type LocalePreference,
} from '@/i18n'

export type { LocalePreference } from '@/i18n'

export function useLocalePreference() {
  installLocalePreferenceSync()

  return {
    preference: localePreference,
    resolvedLocale: computed(() => getLocale()),
    setPreference: (preference: LocalePreference) => setLocalePreference(preference),
  }
}
