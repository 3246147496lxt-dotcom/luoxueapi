<template>
  <div
    class="personal-settings-panel"
    data-testid="personal-settings-general"
  >
    <div class="personal-settings-group">
      <div class="personal-settings-row">
        <span class="personal-settings-row-copy">
          <span class="personal-settings-row-title">
            {{ t('personalSettings.general.appearance') }}
          </span>
          <span class="personal-settings-row-description">
            {{ t('personalSettings.general.appearanceHint') }}
          </span>
        </span>
        <SettingsChoiceMenu
          :model-value="themePreference"
          :options="themeOptions"
          :ariaLabel="t('personalSettings.general.appearance')"
          test-id="personal-settings-theme"
          @update:model-value="handleThemeChange"
        />
      </div>

      <div class="personal-settings-row">
        <span class="personal-settings-row-copy">
          <span class="personal-settings-row-title">
            {{ t('personalSettings.general.language') }}
          </span>
          <span class="personal-settings-row-description">
            {{ t('personalSettings.general.languageHint') }}
          </span>
        </span>
        <SettingsChoiceMenu
          :model-value="localePreference"
          :options="localeOptions"
          :ariaLabel="t('personalSettings.general.language')"
          test-id="personal-settings-locale"
          @update:model-value="handleLocaleChange"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  useThemePreference,
  type ThemePreference,
} from '@/composables/useThemePreference'
import {
  useLocalePreference,
  type LocalePreference,
} from '@/composables/useLocalePreference'
import SettingsChoiceMenu, {
  type SettingsChoiceOption,
} from './SettingsChoiceMenu.vue'

const { t } = useI18n()
const {
  preference: themePreference,
  setPreference: setThemePreference,
} = useThemePreference()
const {
  preference: localePreference,
  setPreference: setLocalePreference,
} = useLocalePreference()

const themeOptions = computed<SettingsChoiceOption[]>(() => [
  { value: 'system', label: t('personalSettings.general.system') },
  { value: 'light', label: t('personalSettings.general.light') },
  { value: 'dark', label: t('personalSettings.general.dark') },
])

const localeOptions = computed<SettingsChoiceOption[]>(() => [
  { value: 'auto', label: t('personalSettings.general.auto') },
  { value: 'zh', label: t('personalSettings.general.zh') },
  { value: 'en', label: t('personalSettings.general.en') },
])

function handleThemeChange(value: string) {
  if (value === 'system' || value === 'light' || value === 'dark') {
    setThemePreference(value satisfies ThemePreference)
  }
}

function handleLocaleChange(value: string) {
  if (value === 'auto' || value === 'zh' || value === 'en') {
    void setLocalePreference(value satisfies LocalePreference)
  }
}
</script>
