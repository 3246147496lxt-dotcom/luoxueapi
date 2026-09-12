<template>
  <section class="account-assistant" data-test="account-assistant" aria-label="account assistant">
    <div class="account-assistant__workspace">
      <div class="account-assistant__chat">
        <header class="account-assistant__chat-head">
          <h2>{{ t('admin.accounts.copilot.chatTitle') }}</h2>
        </header>

        <div
          ref="transcriptEl"
          class="account-assistant__transcript"
          data-test="account-assistant-transcript"
          role="log"
          aria-live="polite"
        >
          <article
            v-for="(message, index) in transcript"
            :key="index"
            class="account-assistant__message"
            :class="`account-assistant__message--${message.role}`"
          >
            <span class="account-assistant__role">
              {{ message.role === 'user' ? t('admin.accounts.copilot.you') : t('admin.accounts.copilot.assistant') }}
            </span>
            <p>{{ message.content }}</p>
            <div v-if="message.traces?.length" class="account-assistant__traces">
              <span
                v-for="(trace, traceIndex) in message.traces"
                :key="`${trace.name}-${traceIndex}`"
                class="account-assistant__trace"
                :class="`account-assistant__trace--${trace.status}`"
                :data-test="`account-assistant-tool-${trace.name}`"
              >
                {{ toolLabel(trace.name) }}
              </span>
            </div>
          </article>
          <p v-if="busy" class="account-assistant__thinking" data-test="account-assistant-thinking">
            {{ t('admin.accounts.copilot.thinking') }}
          </p>
        </div>

        <div class="account-assistant__composer">
          <div class="account-assistant__chips" role="group">
            <button
              v-for="chip in suggestionChips"
              :key="chip.id"
              type="button"
              class="account-assistant__chip"
              :disabled="busy"
              :data-test="`account-assistant-chip-${chip.id}`"
              @click="sendSuggestion(chip.id)"
            >
              {{ chip.label }}
            </button>
          </div>
          <form
            class="account-assistant__form"
            data-test="account-assistant-form"
            @submit.prevent="sendDraft"
          >
            <label class="sr-only" :for="inputId">{{ t('admin.accounts.copilot.inputPlaceholder') }}</label>
            <textarea
              :id="inputId"
              ref="inputEl"
              v-model="draft"
              class="account-assistant__input"
              rows="1"
              :disabled="busy"
              :placeholder="t('admin.accounts.copilot.inputPlaceholder')"
              data-test="account-assistant-input"
              @input="resizeComposer"
              @keydown="onComposerKeydown"
            />
            <button
              type="submit"
              class="btn btn-primary account-assistant__send"
              :disabled="busy || !draft.trim()"
              data-test="account-assistant-send"
            >
              <Icon name="chatSend" size="sm" aria-hidden="true" />
              <span>{{ t('admin.accounts.copilot.send') }}</span>
            </button>
          </form>
        </div>
      </div>

      <aside class="account-assistant__queue" aria-labelledby="account-assistant-queue-title">
        <div class="account-assistant__queue-head">
          <h2 id="account-assistant-queue-title">{{ t('admin.accounts.copilot.reviewTitle') }}</h2>
          <span>{{ t('admin.accounts.copilot.selectedCount', { count: selectedCount }) }}</span>
        </div>

        <div class="account-assistant__tabs" role="tablist">
          <button
            type="button"
            role="tab"
            class="account-assistant__tab"
            :class="{ 'is-active': queueTab === 'cleanup' }"
            :aria-selected="queueTab === 'cleanup'"
            data-test="account-assistant-tab-cleanup"
            @click="queueTab = 'cleanup'"
          >
            {{ t('admin.accounts.copilot.tabCleanup') }}
          </button>
          <button
            type="button"
            role="tab"
            class="account-assistant__tab"
            :class="{ 'is-active': queueTab === 'add' }"
            :aria-selected="queueTab === 'add'"
            data-test="account-assistant-tab-add"
            @click="queueTab = 'add'"
          >
            {{ t('admin.accounts.copilot.tabAdd') }}
          </button>
        </div>

        <div v-if="activeQueueLength > 0" class="account-assistant__queue-actions">
          <button
            type="button"
            :disabled="busy || selectedCount === activeQueueLength"
            data-test="account-assistant-select-all"
            @click="selectAll"
          >
            {{ t('admin.accounts.copilot.selectAll') }}
          </button>
          <button
            type="button"
            :disabled="busy || selectedCount === 0"
            data-test="account-assistant-select-none"
            @click="selectNone"
          >
            {{ t('admin.accounts.copilot.selectNone') }}
          </button>
        </div>

        <template v-if="queueTab === 'cleanup'">
          <p v-if="reviewItems.length === 0" class="account-assistant__empty">
            {{ t('admin.accounts.copilot.emptyQueue') }}
          </p>
          <ul v-else class="account-assistant__list">
            <li
              v-for="item in reviewItems"
              :key="item.account_id"
              class="account-assistant__row"
              data-test="account-assistant-item"
            >
              <label class="account-assistant__item">
                <input
                  type="checkbox"
                  :checked="selectedIds.includes(item.account_id)"
                  :data-test="`account-assistant-item-${item.account_id}`"
                  @change="toggleItem(item.account_id)"
                />
                <span class="account-assistant__item-copy">
                  <span class="account-assistant__item-name">{{ item.name || `#${item.account_id}` }}</span>
                  <span class="account-assistant__item-meta">
                    {{ item.platform }} · {{ reasonLabel(item.reason) }}
                  </span>
                </span>
              </label>
            </li>
          </ul>
          <button
            type="button"
            class="btn btn-danger account-assistant__delete"
            :disabled="!canDelete"
            data-test="account-assistant-delete"
            @click="showDeleteDialog = true"
          >
            {{ t('admin.accounts.copilot.deleteSelected') }}
          </button>
        </template>

        <template v-else>
          <p v-if="addItems.length === 0" class="account-assistant__empty">
            {{ t('admin.accounts.copilot.emptyAddQueue') }}
          </p>
          <ul v-else class="account-assistant__list">
            <li
              v-for="item in addItems"
              :key="item.index"
              class="account-assistant__row"
              data-test="account-assistant-add-item"
            >
              <label class="account-assistant__item">
                <input
                  type="checkbox"
                  :checked="selectedAddIndexes.includes(item.index)"
                  :data-test="`account-assistant-add-item-${item.index}`"
                  @change="toggleAddItem(item.index)"
                />
                <span class="account-assistant__item-copy">
                  <span class="account-assistant__item-name">{{ item.name || item.key_hint }}</span>
                  <span class="account-assistant__item-meta">
                    {{ item.platform }} · {{ item.summary || item.key_hint }}
                  </span>
                </span>
              </label>
            </li>
          </ul>
          <div v-if="ssoMode" class="account-assistant__sso-options" data-test="account-assistant-sso-options">
            <p class="account-assistant__sso-title">{{ t('admin.accounts.copilot.ssoOptionsTitle') }}</p>
            <label class="account-assistant__sso-field">
              <span>{{ t('admin.accounts.copilot.proxyLabel') }}</span>
              <select v-model="selectedProxyId" data-test="account-assistant-sso-proxy">
                <option :value="null">{{ t('admin.accounts.copilot.proxyDirect') }}</option>
                <option v-for="proxy in proxyOptions" :key="proxy.id" :value="proxy.id">
                  {{ proxy.name }} ({{ proxy.protocol }})
                </option>
              </select>
            </label>
            <fieldset class="account-assistant__sso-field">
              <legend>{{ t('admin.accounts.copilot.groupsLabel') }}</legend>
              <label v-for="group in groupOptions" :key="group.id" class="account-assistant__group-option">
                <input v-model="selectedGroupIds" type="checkbox" :value="group.id" />
                <span>{{ group.name }}</span>
              </label>
              <span v-if="groupOptions.length === 0" class="account-assistant__sso-empty">
                {{ t('admin.accounts.copilot.groupsEmpty') }}
              </span>
            </fieldset>
          </div>
          <button
            type="button"
            class="btn btn-primary account-assistant__delete"
            :disabled="!canAdd"
            data-test="account-assistant-add"
            @click="showAddDialog = true"
          >
            {{ t('admin.accounts.copilot.addSelected') }}
          </button>
        </template>
      </aside>
    </div>

    <section class="account-assistant__overview" aria-label="account assistant overview">
      <div class="account-assistant__overview-card account-assistant__overview-card--pending">
        <span>{{ t('admin.accounts.copilot.queuePending') }}</span>
        <strong>{{ pendingCount }}</strong>
      </div>
      <div class="account-assistant__overview-card account-assistant__overview-card--importing">
        <span>{{ t('admin.accounts.copilot.queueImporting') }}</span>
        <strong>{{ importingCount }}</strong>
      </div>
      <div class="account-assistant__overview-card account-assistant__overview-card--failed">
        <span>{{ t('admin.accounts.copilot.queueFailed') }}</span>
        <strong>{{ failedItems.length }}</strong>
      </div>
      <div class="account-assistant__overview-card account-assistant__overview-card--quarantined">
        <span>{{ t('admin.accounts.copilot.queueQuarantined') }}</span>
        <strong>{{ quarantinedCount }}</strong>
      </div>
      <details class="account-assistant__settings" data-test="account-assistant-settings" @toggle="loadAdvancedSettings">
        <summary>{{ t('admin.accounts.copilot.advancedSettings') }}</summary>
        <div class="account-assistant__settings-grid">
          <label>
            <span>{{ t('admin.accounts.copilot.scanFrequency') }}</span>
            <select v-model="settings.scanFrequency" @change="persistSettings">
              <option value="daily">{{ t('admin.accounts.copilot.frequencyDaily') }}</option>
              <option value="weekly">{{ t('admin.accounts.copilot.frequencyWeekly') }}</option>
              <option value="manual">{{ t('admin.accounts.copilot.frequencyManual') }}</option>
            </select>
          </label>
          <label>
            <span>{{ t('admin.accounts.copilot.failurePolicy') }}</span>
            <select v-model="settings.failurePolicy" @change="persistSettings">
              <option value="quarantine">{{ t('admin.accounts.copilot.policyQuarantine') }}</option>
              <option value="review">{{ t('admin.accounts.copilot.policyReview') }}</option>
            </select>
          </label>
          <label class="account-assistant__settings-check">
            <input v-model="settings.notifications" type="checkbox" @change="persistSettings" />
            <span>{{ t('admin.accounts.copilot.enableNotifications') }}</span>
          </label>
          <label class="account-assistant__settings-check">
            <input v-model="settings.emailNotifications" type="checkbox" @change="persistSettings" />
            <span>{{ t('admin.accounts.copilot.enableEmailNotifications') }}</span>
          </label>
          <label class="account-assistant__settings-check">
            <input v-model="settings.webhookNotifications" type="checkbox" @change="persistSettings" />
            <span>{{ t('admin.accounts.copilot.enableWebhookNotifications') }}</span>
          </label>
          <label v-if="settings.webhookNotifications">
            <span>{{ t('admin.accounts.copilot.webhookUrl') }}</span>
            <input v-model="settings.webhookUrl" type="url" :placeholder="t('admin.accounts.copilot.webhookUrlPlaceholder')" @change="persistSettings" />
          </label>
          <div class="account-assistant__settings-divider" />
          <label class="account-assistant__settings-check">
            <input v-model="watcherSettings.enabled" type="checkbox" data-test="account-assistant-watcher-enabled" @change="persistWatcherSettings" />
            <span>{{ t('admin.accounts.copilot.enableRegisterWatcher') }}</span>
          </label>
          <label>
            <span>{{ t('admin.accounts.copilot.registerOutputDir') }}</span>
            <input v-model="watcherSettings.output_dir" type="text" data-test="account-assistant-watcher-dir" :placeholder="t('admin.accounts.copilot.registerOutputDirPlaceholder')" @change="persistWatcherSettings" />
          </label>
          <label class="account-assistant__settings-check">
            <input v-model="watcherSettings.auto_discover" type="checkbox" data-test="account-assistant-watcher-auto" @change="persistWatcherSettings" />
            <span>{{ t('admin.accounts.copilot.registerAutoDiscover') }}</span>
          </label>
          <button
            type="button"
            class="btn btn-secondary account-assistant__watcher-scan"
            data-test="account-assistant-watcher-scan"
            :disabled="busy || watcherScanning || !watcherSettings.output_dir.trim()"
            @click="scanRegisterOutput()"
          >
            {{ watcherScanning ? t('admin.accounts.copilot.registerScanning') : t('admin.accounts.copilot.registerScan') }}
          </button>
        </div>
        <p class="account-assistant__settings-hint">{{ t('admin.accounts.copilot.settingsHint') }}</p>
        <p v-if="watcherLastScanAt" class="account-assistant__settings-hint" data-test="account-assistant-watcher-status">
          {{ t('admin.accounts.copilot.registerLastScan', { time: watcherLastScanAt }) }}
        </p>
      </details>
      <button
        v-if="failedItems.length > 0 && lastHandoff"
        type="button"
        class="btn btn-secondary account-assistant__retry-failed"
        data-test="account-assistant-retry-failed"
        :disabled="busy"
        @click="retryFailed"
      >
        {{ t('admin.accounts.copilot.retryFailed', { count: failedItems.length }) }}
      </button>
    </section>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.accounts.copilot.confirmTitle')"
      :message="t('admin.accounts.copilot.confirmMessage', { count: selectedIds.length })"
      :confirm-text="t('admin.accounts.copilot.deleteSelected')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      data-test="account-assistant-confirm"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
    <ConfirmDialog
      :show="showAddDialog"
      :title="t('admin.accounts.copilot.confirmAddTitle')"
      :message="addConfirmMessage"
      :confirm-text="t('admin.accounts.copilot.addSelected')"
      :cancel-text="t('common.cancel')"
      :danger="false"
      data-test="account-assistant-add-confirm"
      @confirm="confirmAdd"
      @cancel="showAddDialog = false"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  addHealth,
  chatHealth,
  cleanupHealth,
  getHealthSettings,
  updateHealthSettings,
  getImportWatcherSettings,
  updateImportWatcherSettings,
  scanImportWatcher,
  importWatcherFile,
  type AccountHealthSettings,
  type AccountHealthAddItem,
  type AccountHealthAssistantResult,
  type AccountHealthAssistantRequest,
  type AccountHealthItem,
  type AccountHealthProposal,
  type AccountHealthToolTrace
} from '@/api/admin/accounts'
import grokAPI from '@/api/admin/grok'
import proxiesAPI from '@/api/admin/proxies'
import groupsAPI from '@/api/admin/groups'
import type { Proxy, AdminGroup, GroupPlatform } from '@/types'

type SuggestionId = 'scan' | 'test' | 'add' | 'help'
type QueueTab = 'cleanup' | 'add'
type ChatLine = {
  role: 'user' | 'assistant'
  content: string
  traces?: AccountHealthToolTrace[]
}

const { t } = useI18n()
const inputId = 'account-assistant-input'
const draft = ref('')
const busy = ref(false)
const showDeleteDialog = ref(false)
const showAddDialog = ref(false)
const queueTab = ref<QueueTab>('cleanup')
const transcript = ref<ChatLine[]>([])
const reviewItems = ref<AccountHealthItem[]>([])
const addItems = ref<AccountHealthAddItem[]>([])
const selectedIds = ref<number[]>([])
const selectedAddIndexes = ref<number[]>([])
const pendingProposal = ref<AccountHealthProposal | null>(null)
const lastHandoff = ref('')
const ssoMode = ref(false)
const ssoTokens = ref<string[]>([])
const proxyOptions = ref<Proxy[]>([])
const groupOptions = ref<AdminGroup[]>([])
const selectedProxyId = ref<number | null>(null)
const selectedGroupIds = ref<number[]>([])
const watcherMode = ref(false)
const defaultRegisterOutputDir = '/Volumes/T7Dev/grok-register/tokens'
const watcherSettings = ref({ enabled: false, output_dir: defaultRegisterOutputDir, auto_discover: true })
const watcherSettingsLoaded = ref(false)
const watcherScanning = ref(false)
const watcherLastScanAt = ref('')
let watcherTimer: ReturnType<typeof setInterval> | undefined
const watcherFiles = ref<Record<number, string>>({})
const failedItems = ref<Array<{ index: number; message?: string; error?: string }>>([])
const settings = ref({
  scanFrequency: 'daily',
  failurePolicy: 'quarantine',
  notifications: true,
  emailNotifications: false,
  webhookNotifications: false,
  webhookUrl: ''
})
const healthSettings = ref<AccountHealthSettings>({
  enabled: true,
  scan_interval_minutes: 1440,
  auto_quarantine: true,
  require_delete_confirmation: true,
  routing_rules: {},
  notifications: { in_app: true, email: false, webhook: false }
})
const settingsLoaded = ref(false)
const transcriptEl = ref<HTMLElement | null>(null)
const inputEl = ref<HTMLTextAreaElement | null>(null)

const canDelete = computed(() => !busy.value && selectedIds.value.length > 0)
const canAdd = computed(() => !busy.value && selectedAddIndexes.value.length > 0 && (
  lastHandoff.value.trim() !== '' || watcherMode.value
))
const selectedCount = computed(() => (
  queueTab.value === 'add' ? selectedAddIndexes.value.length : selectedIds.value.length
))
const addConfirmMessage = computed(() => t(
  ssoMode.value ? 'admin.accounts.copilot.ssoConfirmAddMessage' : 'admin.accounts.copilot.confirmAddMessage',
  { count: selectedAddIndexes.value.length }
))
const activeQueueLength = computed(() => (
  queueTab.value === 'add' ? addItems.value.length : reviewItems.value.length
))
const pendingCount = computed(() => reviewItems.value.length + addItems.value.length)
const importingCount = computed(() => busy.value && addItems.value.length > 0 ? 1 : 0)
const quarantinedCount = computed(() => reviewItems.value.filter((item) => (
  item.status === 'quarantined' || item.reason === 'expired' || item.reason === 'auth_failed'
)).length)
const suggestionChips = computed(() => ([
  { id: 'scan' as const, label: t('admin.accounts.copilot.suggestions.scan') },
  { id: 'test' as const, label: t('admin.accounts.copilot.suggestions.test') },
  { id: 'add' as const, label: t('admin.accounts.copilot.suggestions.add') },
  { id: 'help' as const, label: t('admin.accounts.copilot.suggestions.help') }
]))

function looksLikeAccountHealthHandoff(raw: string) {
  const n = raw.toLowerCase()
  const hasJwtLine = extractSSOTokens(raw).length > 0
  return hasJwtLine
    || n.includes('sk-')
    || n.includes('rk-')
    || n.includes('bearer ')
    || n.includes('"api_key"')
    || n.includes('api_key:')
    || n.includes('access_token')
    || n.includes('refresh_token')
    || n.includes('id_token')
    || n.includes('client_secret')
    || n.includes('session_token')
    || n.includes('authorization:')
}

function extractSSOTokens(raw: string) {
  const found: string[] = []
  const visit = (value: unknown) => {
    if (typeof value === 'string') {
      const token = value.trim()
      const parts = token.split('.')
      if (parts.length === 3 && parts[0].toLowerCase().startsWith('eyj')) found.push(token)
      return
    }
    if (Array.isArray(value)) {
      value.forEach(visit)
      return
    }
    if (value && typeof value === 'object') {
      Object.values(value as Record<string, unknown>).forEach(visit)
    }
  }
  try {
    visit(JSON.parse(raw))
  } catch {
    raw.split(/\r?\n/).forEach((line) => {
      const trimmed = line.trim()
      const pipeParts = trimmed.split('|').map((part) => part.trim())
      visit(trimmed)
      pipeParts.forEach(visit)
      const fields = trimmed.split(/\s+/)
      fields.forEach(visit)
    })
  }
  return [...new Set(found)]
}

async function loadSSOOptions() {
  try {
    await loadHealthSettings()
    const [proxies, groups] = await Promise.all([proxiesAPI.getAll(), groupsAPI.getAll('grok' as GroupPlatform)])
    proxyOptions.value = proxies.filter((proxy) => proxy.status === 'active')
    groupOptions.value = groups.filter((group) => group.status === 'active')
    const recommendation = healthSettings.value.routing_rules?.grok
    if (recommendation) {
      selectedProxyId.value = recommendation.proxy_id ?? null
      selectedGroupIds.value = (recommendation.group_ids ?? []).filter((id) => groupOptions.value.some((group) => group.id === id))
    }
  } catch {
    proxyOptions.value = []
    groupOptions.value = []
  }
}

onMounted(() => {
  try {
    const stored = globalThis.localStorage?.getItem('sub2api:account-assistant:settings')
    if (stored) settings.value = { ...settings.value, ...JSON.parse(stored) }
  } catch {
    // Use safe defaults when browser storage is unavailable or malformed.
  }
  if (transcript.value.length === 0) {
    transcript.value = [{ role: 'assistant', content: t('admin.accounts.copilot.welcome') }]
  }
  inputEl.value?.focus()
  resizeComposer()
})

async function loadHealthSettings() {
  if (settingsLoaded.value) return
  try {
    const remote = await getHealthSettings()
    healthSettings.value = remote
    settings.value.scanFrequency = remote.enabled === false
      ? 'manual'
      : remote.scan_interval_minutes <= 1440 ? 'daily' : 'weekly'
    settings.value.failurePolicy = remote.auto_quarantine ? 'quarantine' : 'review'
    settings.value.notifications = remote.notifications?.in_app ?? true
    settings.value.emailNotifications = remote.notifications?.email ?? false
    settings.value.webhookNotifications = remote.notifications?.webhook ?? false
    settings.value.webhookUrl = remote.notifications?.webhook_url ?? ''
    settingsLoaded.value = true
  } catch {
    // Keep local defaults when the optional settings endpoint is unavailable.
  }
}

async function loadImportWatcherSettings() {
  if (watcherSettingsLoaded.value) return
  try {
    const remote = await getImportWatcherSettings()
    watcherSettings.value = {
      enabled: remote.enabled === true,
      output_dir: remote.output_dir || defaultRegisterOutputDir,
      auto_discover: remote.auto_discover !== false
    }
    watcherSettingsLoaded.value = true
  } catch {
    // Keep local defaults when the watcher endpoint is unavailable.
  }
}

async function loadAdvancedSettings() {
  await Promise.all([loadHealthSettings(), loadImportWatcherSettings()])
  configureWatcherPolling()
}

function configureWatcherPolling() {
  if (watcherTimer) {
    clearInterval(watcherTimer)
    watcherTimer = undefined
  }
  if (watcherSettings.value.enabled && watcherSettings.value.auto_discover && watcherSettings.value.output_dir.trim()) {
    watcherTimer = setInterval(() => { void scanRegisterOutput(true) }, 30_000)
  }
}

onUnmounted(() => {
  if (watcherTimer) clearInterval(watcherTimer)
})

function persistSettings() {
  healthSettings.value.enabled = settings.value.scanFrequency !== 'manual'
  healthSettings.value.scan_interval_minutes = settings.value.scanFrequency === 'daily'
    ? 1440 : settings.value.scanFrequency === 'weekly' ? 10080 : 1440
  healthSettings.value.auto_quarantine = settings.value.failurePolicy === 'quarantine'
  healthSettings.value.notifications.in_app = settings.value.notifications
  healthSettings.value.notifications.email = settings.value.emailNotifications
  healthSettings.value.notifications.webhook = settings.value.webhookNotifications
  healthSettings.value.notifications.webhook_url = settings.value.webhookUrl.trim() || undefined
  try {
    globalThis.localStorage?.setItem('sub2api:account-assistant:settings', JSON.stringify(settings.value))
  } catch {
    // Settings remain active for the current session.
  }
  void updateHealthSettings(healthSettings.value).then((saved) => {
    healthSettings.value = saved
    settingsLoaded.value = true
  }).catch(() => undefined)
}

function persistWatcherSettings() {
  const payload = {
    enabled: watcherSettings.value.enabled,
    output_dir: watcherSettings.value.output_dir.trim(),
    auto_discover: watcherSettings.value.auto_discover
  }
  watcherSettings.value.output_dir = payload.output_dir
  void updateImportWatcherSettings(payload).then((saved) => {
    watcherSettings.value = {
      enabled: saved.enabled === true,
      output_dir: saved.output_dir ?? '',
      auto_discover: saved.auto_discover !== false
    }
    watcherSettingsLoaded.value = true
    configureWatcherPolling()
  }).catch(() => undefined)
}

async function scanRegisterOutput(silent = false) {
  if (watcherScanning.value || !watcherSettings.value.output_dir.trim()) return
  watcherScanning.value = true
  try {
    void loadSSOOptions()
    const result = await scanImportWatcher()
    const files = result.files ?? result.discovered_files ?? []
    const nextFiles: Record<number, string> = {}
    const items: AccountHealthAddItem[] = files
      .filter((file) => file.already_imported !== true && file.new !== false)
      .map((file, index) => {
        const name = file.name || file.file || file.path || `accounts_${index + 1}.txt`
        const count = file.token_count ?? file.sso_count ?? 0
        nextFiles[index] = file.file || file.name || file.path || name
        return {
          index,
          name,
          platform: file.platform || 'grok',
          type: 'sso',
          auth_method: 'grok_sso',
          key_hint: `SSO ••••${String(index + 1).padStart(2, '0')}`,
          summary: count > 0 ? `Grok SSO · ${count} tokens` : 'Grok SSO',
          source_file: nextFiles[index],
          token_count: count
        } as AccountHealthAddItem
      })
    watcherFiles.value = nextFiles
    addItems.value = items
    selectedAddIndexes.value = items.map((item) => item.index)
    watcherMode.value = items.length > 0
    ssoMode.value = false
    queueTab.value = 'add'
    watcherLastScanAt.value = new Date().toLocaleTimeString()
    if (!silent || items.length > 0) {
      transcript.value = [...transcript.value, {
        role: 'assistant',
        content: items.length > 0
          ? t('admin.accounts.copilot.registerFound', { count: items.length })
          : t('admin.accounts.copilot.registerNone')
      }]
    }
  } catch {
    if (!silent) transcript.value = [...transcript.value, { role: 'assistant', content: t('admin.accounts.copilot.registerScanFailed') }]
  } finally {
    watcherScanning.value = false
  }
}

function retryFailed() {
  if (failedItems.value.length === 0) return
  queueTab.value = 'add'
  selectedAddIndexes.value = failedItems.value
    .map((item) => item.index)
    .filter((index) => addItems.value.some((item) => item.index === index))
  if (selectedAddIndexes.value.length > 0) showAddDialog.value = true
}

watch(transcript, async () => {
  await nextTick()
  const el = transcriptEl.value
  if (el) el.scrollTop = el.scrollHeight
}, { deep: true })

function reasonLabel(reason: string) {
  const key = `admin.accounts.copilot.reasons.${reason}`
  const translated = t(key)
  return translated === key ? reason : translated
}

function toolLabel(name: string) {
  const key = `admin.accounts.copilot.tools.${name}`
  const translated = t(key)
  return translated === key ? name : translated
}

function toggleItem(id: number) {
  if (selectedIds.value.includes(id)) {
    selectedIds.value = selectedIds.value.filter((item) => item !== id)
    return
  }
  selectedIds.value = [...selectedIds.value, id]
}

function toggleAddItem(index: number) {
  if (selectedAddIndexes.value.includes(index)) {
    selectedAddIndexes.value = selectedAddIndexes.value.filter((item) => item !== index)
    return
  }
  selectedAddIndexes.value = [...selectedAddIndexes.value, index]
}

function selectAll() {
  if (queueTab.value === 'add') {
    selectedAddIndexes.value = addItems.value.map((item) => item.index)
    return
  }
  selectedIds.value = reviewItems.value.map((item) => item.account_id)
}

function selectNone() {
  if (queueTab.value === 'add') {
    selectedAddIndexes.value = []
    return
  }
  selectedIds.value = []
}

function applyAssistantResult(result: AccountHealthAssistantResult) {
  transcript.value = [
    ...transcript.value,
    {
      role: 'assistant',
      content: result.reply,
      traces: result.tool_traces
    }
  ]
  if (result.scan?.items) {
    const cleanupItems = result.scan.items.filter((item) => item.cleanup)
    reviewItems.value = cleanupItems
    selectedIds.value = []
    queueTab.value = 'cleanup'
  }
  if (result.proposal) {
    pendingProposal.value = result.proposal
    if (!result.scan && result.proposal.items?.length) {
      reviewItems.value = result.proposal.items
      selectedIds.value = []
      queueTab.value = 'cleanup'
    }
  }
  if (result.add_proposal) {
    addItems.value = result.add_proposal.items ?? []
    selectedAddIndexes.value = []
    watcherMode.value = false
    watcherFiles.value = {}
    ssoMode.value = addItems.value.some((item) => (
      item.auth_method === 'grok_sso' || item.type === 'sso' || item.platform === 'grok-sso'
      || (item.type === 'oauth' && item.platform === 'grok')
    ))
    queueTab.value = 'add'
  }
}

async function sendContent(content: string, explicitIntent?: AccountHealthAssistantRequest['intent']) {
  const text = content.trim()
  if (!text || busy.value) return
  const handoff = looksLikeAccountHealthHandoff(text) ? text : undefined
  const display = handoff ? t('admin.accounts.copilot.handoffSubmitted') : text
  draft.value = ''
  resizeComposer()
  busy.value = true
  transcript.value = [...transcript.value, { role: 'user', content: display }]
  if (handoff) {
    lastHandoff.value = handoff
    const tokens = extractSSOTokens(handoff)
    if (tokens.length > 0) {
      // SSO credentials are handled locally and never sent to the LLM.
      watcherMode.value = false
      watcherFiles.value = {}
      ssoMode.value = true
      ssoTokens.value = tokens
      addItems.value = tokens.map((_, index) => ({
        index,
        name: `grok-sso-${index + 1}`,
        platform: 'grok',
        type: 'sso',
        key_hint: `SSO ••••${String(index + 1).padStart(2, '0')}`,
        summary: 'Grok SSO'
      }))
      selectedAddIndexes.value = []
      queueTab.value = 'add'
      void loadSSOOptions()
      transcript.value = [...transcript.value, { role: 'assistant', content: t('admin.accounts.copilot.ssoDetected') }]
      busy.value = false
      await nextTick()
      inputEl.value?.focus()
      return
    }
    // A subsequent API-key handoff switches back to the regular importer.
    ssoMode.value = false
    ssoTokens.value = []
    selectedProxyId.value = null
    selectedGroupIds.value = []
  }
  try {
    const result = await chatHealth({
      messages: transcript.value.map((message) => ({ role: message.role, content: message.content })),
      pending_proposal: pendingProposal.value ?? undefined,
      ...(explicitIntent ? { intent: explicitIntent } : {}),
      ...(handoff ? { handoff } : {})
    })
    applyAssistantResult(result)
  } catch {
    transcript.value = [...transcript.value, { role: 'assistant', content: t('admin.accounts.copilot.error') }]
  } finally {
    busy.value = false
    await nextTick()
    inputEl.value?.focus()
  }
}

function sendDraft() {
  void sendContent(draft.value)
}

function sendSuggestion(id: SuggestionId) {
  const content = t(`admin.accounts.copilot.suggestions.${id}`)
  // The primary scan action is a live, pool-wide health check. Send the
  // explicit intent so it cannot fall back to the legacy metadata-only scan.
  const intent = id === 'scan' || id === 'test' ? 'test' : undefined
  void sendContent(content, intent)
}

function onComposerKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey) return
  event.preventDefault()
  sendDraft()
}

function resizeComposer() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(Math.max(el.scrollHeight, 44), 128)}px`
}

async function confirmDelete() {
  if (!canDelete.value) return
  const ids = [...selectedIds.value]
  showDeleteDialog.value = false
  busy.value = true
  try {
    const result = await cleanupHealth({
      account_ids: ids,
      confirm: 'DELETE'
    })
    const deleted = new Set(result.deleted)
    reviewItems.value = reviewItems.value.filter((item) => !deleted.has(item.account_id))
    selectedIds.value = selectedIds.value.filter((id) => !deleted.has(id))
    pendingProposal.value = {
      account_ids: selectedIds.value,
      items: reviewItems.value
    }
    const parts = [t('admin.accounts.copilot.deleted', { count: result.deleted.length })]
    if (result.failed.length > 0) {
      parts.push(t('admin.accounts.copilot.failed', { count: result.failed.length }))
    }
    transcript.value = [...transcript.value, { role: 'assistant', content: parts.join(' ') }]
  } catch {
    transcript.value = [...transcript.value, { role: 'assistant', content: t('admin.accounts.copilot.error') }]
  } finally {
    busy.value = false
  }
}

async function confirmAdd() {
  if (!canAdd.value) return
  const indexes = [...selectedAddIndexes.value]
  const handoff = lastHandoff.value
  showAddDialog.value = false
  busy.value = true
  try {
    if (watcherMode.value) {
      let createdCount = 0
      let failedCount = 0
      const completed = new Set<number>()
      const failed: Array<{ index: number; message?: string; error?: string }> = []
      for (const index of indexes) {
        const file = watcherFiles.value[index] || addItems.value.find((item) => item.index === index)?.source_file
        if (!file) continue
        const result = await importWatcherFile({
          file,
          indexes: [],
          proxy_id: selectedProxyId.value,
          group_ids: selectedGroupIds.value,
          confirm: 'ADD'
        })
        const imported = result.imported ?? result.created?.length ?? 0
        const fileFailed = result.failed?.length ?? 0
        createdCount += imported
        failedCount += fileFailed
        if (fileFailed === 0) completed.add(index)
        else failed.push({ index, message: t('admin.accounts.copilot.addFailed', { count: fileFailed }) })
      }
      addItems.value = addItems.value.filter((item) => !completed.has(item.index))
      selectedAddIndexes.value = selectedAddIndexes.value.filter((index) => !completed.has(index))
      failedItems.value = failed
      watcherMode.value = addItems.value.length > 0
      if (!watcherMode.value) watcherFiles.value = {}
      transcript.value = [...transcript.value, {
        role: 'assistant',
        content: failedCount > 0
          ? `${t('admin.accounts.copilot.added', { count: createdCount })} ${t('admin.accounts.copilot.addFailed', { count: failedCount })}`
          : t('admin.accounts.copilot.added', { count: createdCount })
      }]
      return
    }
    if (ssoMode.value) {
      const tokens = indexes.map((index) => ssoTokens.value[index]).filter(Boolean)
      const result = await grokAPI.createFromSSO({
        sso_tokens: tokens,
        proxy_id: selectedProxyId.value,
        group_ids: selectedGroupIds.value
      })
      // The Grok importer numbers results relative to the submitted subset;
      // map those indexes back to the queue indexes before removing rows.
      const createdIndexes = new Set(result.created
        // The Grok importer reports one-based token indexes.
        .map((item) => indexes[item.index - 1])
        .filter((index): index is number => typeof index === 'number'))
      addItems.value = addItems.value.filter((item) => !createdIndexes.has(item.index))
      selectedAddIndexes.value = selectedAddIndexes.value.filter((index) => !createdIndexes.has(index))
      failedItems.value = result.failed.map((item) => ({ index: item.index, message: item.error, error: item.error }))
      if (addItems.value.length === 0) {
        lastHandoff.value = ''
        ssoTokens.value = []
        ssoMode.value = false
        selectedProxyId.value = null
        selectedGroupIds.value = []
      }
      const parts = [t('admin.accounts.copilot.added', { count: result.created.length })]
      if (result.failed.length > 0) parts.push(t('admin.accounts.copilot.addFailed', { count: result.failed.length }))
      transcript.value = [...transcript.value, { role: 'assistant', content: parts.join(' ') }]
      return
    }
    const result = await addHealth({
      handoff,
      indexes,
      confirm: 'ADD'
    })
    const created = new Set(result.created.map((item) => item.index))
    addItems.value = addItems.value.filter((item) => !created.has(item.index))
    selectedAddIndexes.value = selectedAddIndexes.value.filter((index) => !created.has(index))
    failedItems.value = result.failed
    if (addItems.value.length === 0) lastHandoff.value = ''
    const parts = [t('admin.accounts.copilot.added', { count: result.created.length })]
    if (result.failed.length > 0) {
      parts.push(t('admin.accounts.copilot.addFailed', { count: result.failed.length }))
    }
    transcript.value = [...transcript.value, { role: 'assistant', content: parts.join(' ') }]
  } catch {
    transcript.value = [...transcript.value, { role: 'assistant', content: t('admin.accounts.copilot.error') }]
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.account-assistant {
  display: flex;
  flex-direction: column;
  min-height: 0;
  flex: 1;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
}

.account-assistant__overview {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.account-assistant__overview-card {
  display: flex;
  min-height: 62px;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
  padding: 10px 14px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-surface);
}

.account-assistant__overview-card span { color: var(--lx-clay-text-secondary); font-size: 0.75rem; }
.account-assistant__overview-card strong { font-size: 1.25rem; line-height: 1.2; }
.account-assistant__overview-card--failed strong { color: var(--lx-clay-danger); }
.account-assistant__overview-card--quarantined strong { color: var(--lx-clay-warning, #a66a00); }

.account-assistant__settings {
  grid-column: 1 / -1;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: var(--lx-clay-surface-soft);
}
.account-assistant__settings summary { padding: 10px 14px; cursor: pointer; color: var(--lx-clay-text-secondary); font-size: 0.8125rem; font-weight: 650; }
.account-assistant__settings-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; padding: 0 14px 10px; }
.account-assistant__settings-grid label { display: flex; flex-direction: column; gap: 5px; color: var(--lx-clay-text-secondary); font-size: 0.75rem; }
.account-assistant__settings-grid select,
.account-assistant__settings-grid input[type='url'],
.account-assistant__settings-grid input[type='text'] { min-height: 34px; padding: 0 8px; border: 1px solid var(--lx-clay-border); border-radius: 6px; color: var(--lx-clay-text); background: var(--lx-clay-surface); }
.account-assistant__settings-check { flex-direction: row !important; align-items: center; align-self: end; min-height: 34px; }
.account-assistant__settings-divider { grid-column: 1 / -1; height: 1px; background: var(--lx-clay-border); }
.account-assistant__watcher-scan { align-self: end; min-height: 34px; }
.account-assistant__settings-hint { margin: 0; padding: 0 14px 12px; color: var(--lx-clay-text-secondary); font-size: 0.75rem; }

.account-assistant__workspace {
  display: grid;
  grid-template-columns: minmax(0, 1fr) min(22rem, 34%);
  min-height: 32rem;
  flex: 1;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-ops);
  background: var(--lx-clay-surface);
}

.account-assistant__chat,
.account-assistant__queue {
  display: flex;
  min-height: 0;
  flex-direction: column;
}

.account-assistant__chat {
  background: var(--lx-clay-surface);
}

.account-assistant__queue {
  border-left: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface-soft);
}

.account-assistant__chat-head,
.account-assistant__queue-head {
  display: flex;
  gap: 8px;
  padding: 16px 22px 12px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.account-assistant__chat-head {
  flex-direction: column;
}

.account-assistant__chat-head h2,
.account-assistant__queue-head h2 {
  margin: 0;
  font-size: 0.9375rem;
  font-weight: 700;
}

.account-assistant__chat-head p,
.account-assistant__queue-head span {
  margin: 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  line-height: 1.45;
}

.account-assistant__transcript {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 14px;
  overflow-y: auto;
  padding: 16px 22px 8px;
}

.account-assistant__message {
  display: flex;
  max-width: 85%;
  flex-direction: column;
  margin: 0;
}

.account-assistant__role {
  margin: 0 4px 4px;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-weight: 650;
}

.account-assistant__message p {
  margin: 0;
  padding: 10px 14px;
  border-radius: 16px;
  line-height: 1.55;
  font-size: 0.9375rem;
  white-space: pre-wrap;
}

.account-assistant__message--assistant {
  align-self: flex-start;
}

.account-assistant__message--assistant p {
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.account-assistant__message--user {
  align-self: flex-end;
}

.account-assistant__message--user .account-assistant__role {
  text-align: right;
}

.account-assistant__message--user p {
  color: var(--lx-clay-on-accent);
  background: linear-gradient(145deg, var(--lx-clay-accent-gradient-start), var(--lx-clay-accent-deep));
}

.account-assistant__traces {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}

.account-assistant__trace {
  display: inline-flex;
  min-height: 28px;
  align-items: center;
  padding: 0 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--lx-clay-accent) 12%, var(--lx-clay-surface));
  color: var(--lx-clay-accent-deep);
  font-size: 0.75rem;
  font-weight: 650;
}

.account-assistant__trace--error {
  background: var(--lx-clay-danger-soft);
  color: var(--lx-clay-danger);
}

.account-assistant__thinking {
  margin: 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.875rem;
}

.account-assistant__composer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 18px 18px;
  border-top: 1px solid var(--lx-clay-border);
}

.account-assistant__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.account-assistant__chip {
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 999px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 0.8125rem;
}

.account-assistant__chip:hover:not(:disabled) {
  border-color: var(--lx-clay-accent);
  color: var(--lx-clay-accent-deep);
  background: color-mix(in srgb, var(--lx-clay-accent) 10%, transparent);
}

.account-assistant__chip:disabled {
  opacity: 0.55;
}

.account-assistant__form {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.account-assistant__input {
  min-height: 44px;
  max-height: 8rem;
  flex: 1;
  resize: none;
  padding: 10px 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
  font-family: inherit;
  line-height: 1.45;
}

.account-assistant__input:focus {
  outline: none;
  border-color: var(--lx-clay-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 20%, transparent);
}

.account-assistant__send {
  display: inline-flex;
  gap: 6px;
  min-width: 5.5rem;
}

.account-assistant__tabs {
  display: flex;
  gap: 4px;
  margin: 0 18px 8px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.account-assistant__tab {
  min-height: 36px;
  padding: 0 10px;
  border: 0;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 0.8125rem;
  font-weight: 650;
}

.account-assistant__tab.is-active {
  color: var(--lx-clay-accent-deep);
  border-bottom-color: var(--lx-clay-accent);
}

.account-assistant__queue-head {
  align-items: baseline;
  justify-content: space-between;
  margin: 0;
  padding: 16px 18px 10px;
  border-bottom: 0;
}

.account-assistant__queue-head span {
  font-weight: 600;
}

.account-assistant__queue-actions {
  display: flex;
  gap: 8px;
  margin: 0 18px 8px;
}

.account-assistant__queue-actions button {
  min-height: 32px;
  padding: 0 4px;
  border: 0;
  color: var(--lx-clay-text-secondary);
  background: transparent;
  font-size: 0.75rem;
  font-weight: 650;
}

.account-assistant__queue-actions button:hover:not(:disabled) {
  color: var(--lx-clay-accent-deep);
}

.account-assistant__queue-actions button:disabled {
  opacity: 0.45;
}

.account-assistant__empty,
.account-assistant__list,
.account-assistant__delete {
  margin: 0 18px;
}

.account-assistant__empty {
  color: var(--lx-clay-text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.account-assistant__list {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;
  overflow-y: auto;
  padding: 0;
  list-style: none;
}

.account-assistant__item {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  min-height: 44px;
  padding: 8px 4px;
  cursor: pointer;
}

.account-assistant__item input {
  margin-top: 4px;
}

.account-assistant__item-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.account-assistant__item-name {
  font-weight: 650;
}

.account-assistant__item-meta {
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
}

.account-assistant__sso-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 12px 18px;
  padding: 12px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
}

.account-assistant__sso-title {
  margin: 0;
  font-size: 0.875rem;
  font-weight: 650;
}

.account-assistant__sso-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  border: 0;
  padding: 0;
  font-size: 0.8125rem;
}

.account-assistant__sso-field select {
  min-height: 34px;
  padding: 4px 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 6px;
  background: var(--lx-clay-surface);
  color: inherit;
}

.account-assistant__group-option {
  display: flex;
  gap: 7px;
  align-items: center;
}

.account-assistant__sso-empty {
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
}

.account-assistant__delete {
  margin-top: auto;
  margin-bottom: 18px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 1023px) {
  .account-assistant__overview { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .account-assistant__settings-grid { grid-template-columns: 1fr; }
  .account-assistant__workspace {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(22rem, 1fr) auto;
  }

  .account-assistant__queue {
    border-left: 0;
    border-top: 1px solid var(--lx-clay-border);
  }
}
</style>
