<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { CircleUserRound, Download, LogOut, MonitorCog, RotateCcw, Send, ShieldCheck, Trash2, X } from 'lucide-vue-next'
import { desktopApi } from '@/api'
import type { DiagnosticReceipt, DiagnosticSummary, DesktopSettings, DesktopSnapshot, ThemeMode } from '@/types'

const props = defineProps<{ snapshot: DesktopSnapshot; busy: string }>()
const emit = defineEmits<{
  save: [settings: DesktopSettings]
  checkUpdate: []
  installUpdate: []
  prepareUninstall: []
  logout: []
}>()
const { t, locale } = useI18n()
const form = reactive<DesktopSettings>({ ...props.snapshot.settings })
const diagnosticSummary = ref<DiagnosticSummary | null>(null)
const diagnosticReceipt = ref<DiagnosticReceipt | null>(null)
const diagnosticBusy = ref(false)
const diagnosticError = ref('')

const diagnosticJson = computed(() => diagnosticSummary.value
  ? JSON.stringify(diagnosticSummary.value, null, 2)
  : '')

watch(() => props.snapshot.settings, (value) => Object.assign(form, value), { deep: true })

function save() {
  locale.value = form.locale
  localStorage.setItem('luoxue-desktop-locale', form.locale)
  emit('save', { ...form })
}

function setTheme(theme: ThemeMode) {
  form.theme = theme
  save()
}

async function previewDiagnostic() {
  diagnosticBusy.value = true
  diagnosticError.value = ''
  diagnosticReceipt.value = null
  try {
    diagnosticSummary.value = await desktopApi.diagnosticSummary()
  } catch (error) {
    diagnosticError.value = error instanceof Error ? error.message : String(error)
  } finally {
    diagnosticBusy.value = false
  }
}

async function uploadDiagnostic() {
  diagnosticBusy.value = true
  diagnosticError.value = ''
  try {
    diagnosticReceipt.value = await desktopApi.uploadDiagnostic()
  } catch (error) {
    diagnosticError.value = error instanceof Error ? error.message : String(error)
  } finally {
    diagnosticBusy.value = false
  }
}

function closeDiagnostic() {
  diagnosticSummary.value = null
  diagnosticReceipt.value = null
  diagnosticError.value = ''
}
</script>

<template>
  <div class="view settings-view">
    <section class="settings-section">
      <div class="settings-section__heading">
        <CircleUserRound :size="20" />
        <h2>{{ t('settings.account') }}</h2>
      </div>
      <div class="settings-row">
        <div>
          <strong>{{ snapshot.accountEmail }}</strong>
          <span>{{ t('settings.device') }} · {{ snapshot.deviceName }}</span>
        </div>
        <button class="secondary-button" type="button" @click="$emit('logout')">
          <LogOut :size="16" />{{ t('settings.logout') }}
        </button>
      </div>
    </section>

    <section class="settings-section">
      <div class="settings-section__heading"><ShieldCheck :size="20" /><h2>{{ t('settings.diagnostics') }}</h2></div>
      <div class="settings-row">
        <div>
          <strong>{{ t('settings.diagnosticPackage') }}</strong>
          <span>{{ t('settings.diagnosticHint') }}</span>
        </div>
        <button class="secondary-button" type="button" :disabled="diagnosticBusy" @click="previewDiagnostic">
          <ShieldCheck :size="16" />{{ t('settings.previewDiagnostic') }}
        </button>
      </div>
    </section>

    <section class="settings-section">
      <div class="settings-section__heading"><MonitorCog :size="20" /><h2>{{ t('settings.systemSection') }}</h2></div>
      <label class="settings-row settings-row--interactive">
        <div>
          <strong>{{ t('settings.startup') }}</strong>
          <span>{{ t('settings.startupWarning') }}</span>
        </div>
        <button class="toggle" type="button" role="switch" :aria-checked="form.launchAtLogin" @click="form.launchAtLogin = !form.launchAtLogin; save()"><span /></button>
      </label>
      <label class="settings-row settings-row--interactive">
        <div><strong>{{ t('settings.notifications') }}</strong><span>{{ t('settings.notificationsHint') }}</span></div>
        <button class="toggle" type="button" role="switch" :aria-checked="form.notifications" @click="form.notifications = !form.notifications; save()"><span /></button>
      </label>
      <div class="settings-row">
        <div><strong>{{ t('settings.theme') }}</strong><span>{{ t('settings.appearanceHint') }}</span></div>
        <div class="segmented-control">
          <button v-for="theme in (['system', 'light', 'dark'] as ThemeMode[])" :key="theme" type="button" :class="{ active: form.theme === theme }" @click="setTheme(theme)">{{ t(`settings.${theme}`) }}</button>
        </div>
      </div>
      <label class="settings-row settings-row--interactive">
        <div><strong>{{ t('settings.language') }}</strong><span>简体中文 / English</span></div>
        <select v-model="form.locale" @change="save">
          <option value="zh-CN">简体中文</option>
          <option value="en">English</option>
        </select>
      </label>
      <label class="settings-row settings-row--interactive">
        <div><strong>{{ t('settings.retention') }}</strong><span>{{ t('settings.retentionHint') }}</span></div>
        <select v-model.number="form.retentionDays" @change="save">
          <option :value="1">{{ t('settings.days', { count: 1 }) }}</option>
          <option :value="7">{{ t('settings.days', { count: 7 }) }}</option>
          <option :value="14">{{ t('settings.days', { count: 14 }) }}</option>
          <option :value="30">{{ t('settings.days', { count: 30 }) }}</option>
        </select>
      </label>
    </section>

    <section class="settings-section">
      <div class="settings-section__heading"><Download :size="20" /><h2>{{ t('settings.update') }}</h2></div>
      <div class="settings-row">
        <div>
          <strong>{{ snapshot.update?.currentVersion }}</strong>
          <span v-if="snapshot.update?.state === 'available'">
            {{ t('settings.updateAvailable', { version: snapshot.update.version }) }}
          </span>
          <span v-else-if="snapshot.update?.state === 'error'">{{ t('settings.updateError') }}</span>
          <span v-else>{{ t('settings.upToDate') }}</span>
        </div>
        <button
          v-if="snapshot.update?.state === 'available'"
          class="primary-button"
          type="button"
          :disabled="busy === 'update'"
          @click="$emit('installUpdate')"
        >
          <Download :size="16" />{{ busy === 'update' ? t('settings.installingUpdate') : t('settings.installUpdate') }}
        </button>
        <button v-else class="secondary-button" type="button" :disabled="busy === 'update'" @click="$emit('checkUpdate')">
          <RotateCcw :size="16" :class="{ spin: busy === 'update' }" />{{ t('settings.checkUpdate') }}
        </button>
      </div>
    </section>

    <section class="settings-section settings-section--danger">
      <div class="settings-section__heading"><Trash2 :size="20" /><h2>{{ t('settings.uninstallSection') }}</h2></div>
      <div class="settings-row">
        <div><strong>{{ t('settings.uninstall') }}</strong><span>{{ t('settings.uninstallHint') }}</span></div>
        <button class="danger-button" type="button" @click="$emit('prepareUninstall')">{{ t('settings.uninstall') }}</button>
      </div>
    </section>

    <div v-if="diagnosticSummary" class="dialog-backdrop" @click.self="closeDiagnostic">
      <section class="diagnostic-dialog" role="dialog" aria-modal="true" :aria-label="t('settings.diagnosticPreview')">
        <header class="diagnostic-dialog__header">
          <div>
            <h2>{{ t('settings.diagnosticPreview') }}</h2>
            <p>{{ t('settings.diagnosticPrivacy') }}</p>
          </div>
          <button class="icon-button icon-button--small" type="button" :aria-label="t('common.cancel')" @click="closeDiagnostic">
            <X :size="17" />
          </button>
        </header>

        <div v-if="diagnosticReceipt" class="diagnostic-success" role="status">
          <ShieldCheck :size="20" />
          <div>
            <strong>{{ t('settings.diagnosticUploaded') }}</strong>
            <span>{{ t('settings.diagnosticId') }}: {{ diagnosticReceipt.id }}</span>
          </div>
        </div>
        <template v-else>
          <pre class="diagnostic-preview">{{ diagnosticJson }}</pre>
          <p v-if="diagnosticError" class="diagnostic-error" role="alert">{{ diagnosticError }}</p>
          <footer class="diagnostic-dialog__actions">
            <button class="secondary-button" type="button" @click="closeDiagnostic">{{ t('common.cancel') }}</button>
            <button class="primary-button" type="button" :disabled="diagnosticBusy" @click="uploadDiagnostic">
              <Send :size="16" />{{ diagnosticBusy ? t('settings.uploadingDiagnostic') : t('settings.confirmUpload') }}
            </button>
          </footer>
        </template>
      </section>
    </div>
  </div>
</template>
