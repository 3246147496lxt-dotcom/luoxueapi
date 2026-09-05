<template>
      <!-- API Key input (only for apikey type, excluding Antigravity which has its own fields) -->
      <div v-if="accountType === 'apikey' && platform !== 'antigravity'" class="space-y-4">
        <!-- 智谱 GLM account mode/protocol and optional team metadata. -->
        <div v-if="platform === 'zhipu'" class="space-y-4 rounded-lg border border-indigo-200 bg-indigo-50/50 p-3 dark:border-indigo-900 dark:bg-indigo-950/20">
          <div>
            <label class="input-label">{{ t('admin.accounts.cnProviders.accountMode.title') }}</label>
            <div class="mt-2 grid grid-cols-2 gap-2">
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-left text-sm transition-colors"
                :class="accountMode === 'payg' ? 'border-indigo-500 bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200' : 'border-gray-200 bg-white text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'"
                @click="accountMode = 'payg'"
              >
                <span class="block font-medium">{{ t('admin.accounts.cnProviders.accountMode.payg') }}</span>
                <span class="mt-0.5 block text-xs opacity-75">{{ t('admin.accounts.cnProviders.accountMode.paygDesc') }}</span>
              </button>
              <button
                type="button"
                class="rounded-md border px-3 py-2 text-left text-sm transition-colors"
                :class="accountMode === 'coding' ? 'border-indigo-500 bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200' : 'border-gray-200 bg-white text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'"
                @click="accountMode = 'coding'"
              >
                <span class="block font-medium">{{ t('admin.accounts.cnProviders.accountMode.coding') }}</span>
                <span class="mt-0.5 block text-xs opacity-75">{{ t('admin.accounts.cnProviders.accountMode.codingDesc') }}</span>
              </button>
            </div>
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.cnProviders.apiProtocol.title') }}</label>
            <div class="mt-2 grid grid-cols-2 gap-2">
              <button
                v-for="option in zhipuProtocolOptions"
                :key="option.value"
                type="button"
                class="rounded-md border px-3 py-2 text-left text-sm transition-colors"
                :class="apiProtocol === option.value ? 'border-indigo-500 bg-indigo-100 text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200' : 'border-gray-200 bg-white text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300'"
                @click="apiProtocol = option.value"
              >
                <span class="block font-medium">{{ t(`admin.accounts.cnProviders.apiProtocol.${option.key}`) }}</span>
                <span class="mt-0.5 block text-xs opacity-75">{{ t(`admin.accounts.cnProviders.apiProtocol.${option.key}Desc`) }}</span>
              </button>
            </div>
          </div>
          <div v-if="accountMode === 'coding'" class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.accounts.cnProviders.zhipuTeam.organization') }}</label>
              <input v-model="zhipuOrganization" type="text" class="input" :placeholder="t('admin.accounts.cnProviders.zhipuTeam.organizationPlaceholder')" />
            </div>
            <div>
              <label class="input-label">{{ t('admin.accounts.cnProviders.zhipuTeam.project') }}</label>
              <input v-model="zhipuProject" type="text" class="input" :placeholder="t('admin.accounts.cnProviders.zhipuTeam.projectPlaceholder')" />
            </div>
          </div>
          <p class="text-xs text-indigo-700 dark:text-indigo-300">{{ t('admin.accounts.cnProviders.protocolHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
          <input
            v-model="apiKeyBaseUrl"
            type="text"
            class="input"
            :placeholder="
              platform === 'openai'
                ? 'https://api.openai.com'
                : platform === 'gemini'
                  ? 'https://generativelanguage.googleapis.com'
                  : platform === 'grok'
                    ? 'https://api.x.ai/v1'
                    : platform === 'zhipu'
                      ? 'https://open.bigmodel.cn/api/paas/v4'
                      : platform === 'deepseek'
                      ? 'https://api.deepseek.com'
                      : 'https://api.anthropic.com'
            "
          />
          <p v-if="baseUrlHint" class="input-hint">{{ baseUrlHint }}</p>
          <div v-if="platform === 'deepseek' || platform === 'zhipu'" class="mt-2 flex flex-wrap gap-2">
            <button
              v-for="preset in platform === 'zhipu' ? zhipuBaseUrlPresets : deepseekBaseUrlPresets"
              :key="preset.url"
              type="button"
              class="rounded-md border border-sky-200 px-2.5 py-1 text-xs text-sky-700 transition-colors hover:bg-sky-50 dark:border-sky-800 dark:text-sky-300 dark:hover:bg-sky-900/30"
              @click="platform === 'zhipu' ? selectZhipuPreset(preset.url) : (apiKeyBaseUrl = preset.url)"
            >
              {{ preset.label }}
            </button>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.apiKeyRequired') }}</label>
          <input
            v-model="apiKeyValue"
            type="password"
            required
            class="input font-mono"
            :placeholder="
              platform === 'openai'
                ? 'sk-proj-...'
                : platform === 'gemini'
                  ? 'AIza...'
                  : platform === 'grok'
                    ? 'xai-...'
                    : platform === 'zhipu'
                      ? 'sk-...'
                      : platform === 'deepseek'
                      ? 'sk-...'
                      : 'sk-ant-...'
            "
          />
          <p v-if="apiKeyHint" class="input-hint">{{ apiKeyHint }}</p>
        </div>

        <!-- Gemini API Key tier selection -->
        <div v-if="platform === 'gemini'">
          <label class="input-label">{{ t('admin.accounts.gemini.tier.label') }}</label>
          <select v-model="geminiTierAIStudio" class="input">
            <option value="aistudio_free">{{ t('admin.accounts.gemini.tier.aiStudio.free') }}</option>
            <option value="aistudio_paid">{{ t('admin.accounts.gemini.tier.aiStudio.paid') }}</option>
          </select>
          <p class="input-hint">{{ t('admin.accounts.gemini.tier.aiStudioHint') }}</p>
        </div>

        <!-- Model Restriction Section (Antigravity 已在上层条件排除) -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

          <div
            v-if="openaiModelRestrictionDisabled"
            class="mb-3 rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20"
          >
            <p class="text-xs text-amber-700 dark:text-amber-400">
              {{ t('admin.accounts.openai.modelRestrictionDisabledByPassthrough') }}
            </p>
          </div>

          <template v-else>
            <!-- Mode Toggle -->
            <div class="mb-4 flex gap-2">
              <button
                type="button"
                @click="modelRestrictionMode = 'whitelist'"
                :class="[
                  'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
                  modelRestrictionMode === 'whitelist'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <svg
                  class="mr-1.5 inline h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
                {{ t('admin.accounts.modelWhitelist') }}
              </button>
              <button
                type="button"
                @click="modelRestrictionMode = 'mapping'"
                :class="[
                  'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
                  modelRestrictionMode === 'mapping'
                    ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                <svg
                  class="mr-1.5 inline h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
                  />
                </svg>
                {{ t('admin.accounts.modelMapping') }}
              </button>
            </div>

            <!-- Whitelist Mode -->
            <div v-if="modelRestrictionMode === 'whitelist'">
              <ModelWhitelistSelector v-model="allowedModels" :platform="platform" :sync-credentials="syncPreviewCredentials" />
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
                <span v-if="allowedModels.length === 0">{{
                  t('admin.accounts.supportsAllModels')
                }}</span>
              </p>
            </div>

            <!-- Mapping Mode -->
            <div v-else>
              <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
                <p class="text-xs text-purple-700 dark:text-purple-400">
                  <svg
                    class="mr-1 inline h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  {{ t('admin.accounts.mapRequestModels') }}
                </p>
              </div>

            <!-- Model Mapping List -->
            <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
              <div
                v-for="(mapping, index) in modelMappings"
                :key="getModelMappingKey(mapping)"
                class="flex items-center gap-2"
              >
                <input
                  v-model="mapping.from"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.accounts.requestModel')"
                />
                <svg
                  class="h-4 w-4 flex-shrink-0 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M14 5l7 7m0 0l-7 7m7-7H3"
                  />
                </svg>
                <input
                  v-model="mapping.to"
                  type="text"
                  class="input flex-1"
                  :placeholder="t('admin.accounts.actualModel')"
                />
                <button
                  type="button"
                  @click="removeModelMapping(index)"
                  class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                >
                  <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              </div>
            </div>

            <button
              type="button"
              @click="addModelMapping"
              class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
            >
              <svg
                class="mr-1 inline h-4 w-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M12 4v16m8-8H4"
                />
              </svg>
              {{ t('admin.accounts.addMapping') }}
            </button>

              <!-- Quick Add Buttons -->
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="preset in presetMappings"
                  :key="preset.label"
                  type="button"
                  @click="addPresetMapping(preset.from, preset.to)"
                  :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
                >
                  + {{ preset.label }}
                </button>
              </div>
            </div>
          </template>
        </div>

        <!-- Pool Mode Section -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.poolMode') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.poolModeHint') }}
              </p>
            </div>
            <button
              type="button"
              @click="poolModeEnabled = !poolModeEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                poolModeEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  poolModeEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="poolModeEnabled" class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
            <p class="text-xs text-blue-700 dark:text-blue-400">
              <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
              {{ t('admin.accounts.poolModeInfo') }}
            </p>
          </div>
          <div v-if="poolModeEnabled" class="mt-3">
            <label class="input-label">{{ t('admin.accounts.poolModeRetryCount') }}</label>
            <input
              v-model.number="poolModeRetryCount"
              type="number"
              min="0"
              :max="MAX_POOL_MODE_RETRY_COUNT"
              step="1"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{
                t('admin.accounts.poolModeRetryCountHint', {
                  default: DEFAULT_POOL_MODE_RETRY_COUNT,
                  max: MAX_POOL_MODE_RETRY_COUNT
                })
              }}
            </p>
          </div>
          <div v-if="poolModeEnabled" class="mt-3">
            <label class="input-label">{{ t('admin.accounts.poolModeRetryStatusCodes') }}</label>
            <input
              v-model="poolModeRetryStatusCodesInput"
              type="text"
              class="input"
              :placeholder="DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ')"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.poolModeRetryStatusCodesHint', { default: DEFAULT_POOL_MODE_RETRY_STATUS_CODES.join(', ') }) }}
            </p>
          </div>
        </div>

        <!-- Custom Error Codes Section -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.customErrorCodes') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.customErrorCodesHint') }}
              </p>
            </div>
            <button
              type="button"
              @click="customErrorCodesEnabled = !customErrorCodesEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                customErrorCodesEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  customErrorCodesEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="customErrorCodesEnabled" class="space-y-3">
            <div class="rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
              <p class="text-xs text-amber-700 dark:text-amber-400">
                <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.accounts.customErrorCodesWarning') }}
              </p>
            </div>

            <!-- Error Code Buttons -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="code in commonErrorCodes"
                :key="code.value"
                type="button"
                @click="toggleErrorCode(code.value)"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
                  selectedErrorCodes.includes(code.value)
                    ? 'bg-red-100 text-red-700 ring-1 ring-red-500 dark:bg-red-900/30 dark:text-red-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                {{ code.value }} {{ code.label }}
              </button>
            </div>

            <!-- Manual input -->
            <div class="flex items-center gap-2">
              <input
                v-model.number="customErrorCodeInput"
                type="number"
                min="100"
                max="599"
                class="input flex-1"
                :placeholder="t('admin.accounts.enterErrorCode')"
                @keyup.enter="addCustomErrorCode"
              />
              <button type="button" @click="addCustomErrorCode" class="btn btn-secondary px-3">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 4v16m8-8H4"
                  />
                </svg>
              </button>
            </div>

            <!-- Selected codes summary -->
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="code in selectedErrorCodes.sort((a, b) => a - b)"
                :key="code"
                class="inline-flex items-center gap-1 rounded-full bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
              >
                {{ code }}
                <button
                  type="button"
                  @click="removeErrorCode(code)"
                  class="hover:text-red-900 dark:hover:text-red-300"
                >
                  <Icon name="x" size="sm" :stroke-width="2" />
                </button>
              </span>
              <span v-if="selectedErrorCodes.length === 0" class="text-xs text-gray-400">
                {{ t('admin.accounts.noneSelectedUsesDefault') }}
              </span>
            </div>
          </div>
        </div>

        <!-- Header Override Section (anthropic/openai apikey only) -->
        <div
          v-if="isHeaderOverrideCapable(platform, 'apikey')"
          class="border-t border-gray-200 pt-4 dark:border-dark-600"
        >
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.headerOverride.title') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.headerOverride.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="headerOverrideEnabled = !headerOverrideEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                headerOverrideEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  headerOverrideEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="headerOverrideEnabled" class="space-y-3">
            <div class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
              <p class="text-xs text-blue-700 dark:text-blue-400">
                <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.accounts.headerOverride.info') }}
              </p>
            </div>

            <HeaderOverrideEditor
              :rows="headerOverrideRows"
              @update:rows="headerOverrideRows = $event"
            />
          </div>
        </div>

      </div>

      <AnthropicBedrockPanel
        v-if="platform === 'anthropic' && accountCategory === 'bedrock'"
        v-model:auth-mode="bedrockAuthMode"
        v-model:access-key-id="bedrockAccessKeyId"
        v-model:secret-access-key="bedrockSecretAccessKey"
        v-model:session-token="bedrockSessionToken"
        v-model:api-key="bedrockApiKeyValue"
        v-model:region="bedrockRegion"
        v-model:force-global="bedrockForceGlobal"
        v-model:restriction-mode="modelRestrictionMode"
        v-model:allowed-models="allowedModels"
        :model-mappings="modelMappings"
        :presets="bedrockPresets"
        :sync-credentials="syncPreviewCredentials"
        v-model:pool-enabled="poolModeEnabled"
        v-model:pool-retry-count="poolModeRetryCount"
        v-model:pool-retry-status-codes="poolModeRetryStatusCodesInput"
        :default-pool-retry-count="DEFAULT_POOL_MODE_RETRY_COUNT"
        :max-pool-retry-count="MAX_POOL_MODE_RETRY_COUNT"
        :default-pool-retry-status-codes="DEFAULT_POOL_MODE_RETRY_STATUS_CODES"
        @add-mapping="addModelMapping"
        @remove-mapping="removeModelMapping"
        @update-mapping="updateModelMapping"
        @add-preset="addPresetMapping"
      />


</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { commonErrorCodes, getPresetMappingsByPlatform } from '@/composables/useModelWhitelist'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'
import HeaderOverrideEditor from '@/components/account/HeaderOverrideEditor.vue'
import Icon from '@/components/icons/Icon.vue'
import { isHeaderOverrideCapable, type HeaderOverrideRow } from '@/components/account/credentialsBuilder'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import type { AccountPlatform, AccountType } from '@/types'
import AnthropicBedrockPanel from './AnthropicBedrockPanel.vue'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  DEFAULT_POOL_MODE_RETRY_STATUS_CODES,
  MAX_POOL_MODE_RETRY_COUNT,
  DEEPSEEK_BASE_URL_PRESETS,
  ZHIPU_BASE_URL_PRESETS,
  inferZhipuRoutingFromBaseURL,
} from '../credentialDraftBuilders'
import { apiKeyBaseURLHintKey, apiKeyValueHintKey } from '../platformFormPolicy'
import type {
  AccountModelMappingDraft,
  BedrockAuthMode,
  CreateAccountCategory,
  GeminiAIStudioTier,
  ModelRestrictionMode,
} from '../formDraft'
import type { CnApiProtocol, CnAccountMode } from '../credentialDraftBuilders'

interface Props {
  platform: AccountPlatform
  accountType: AccountType
  accountCategory: CreateAccountCategory
  openaiModelRestrictionDisabled: boolean
}

const props = defineProps<Props>()
const { t } = useI18n()
const appStore = useAppStore()

const apiKeyBaseUrl = defineModel<string>('apiKeyBaseUrl', { required: true })
const apiKeyValue = defineModel<string>('apiKeyValue', { required: true })
const geminiTierAIStudio = defineModel<GeminiAIStudioTier>('geminiTierAiStudio', { required: true })
const modelRestrictionMode = defineModel<ModelRestrictionMode>('modelRestrictionMode', { required: true })
const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const modelMappings = defineModel<AccountModelMappingDraft[]>('modelMappings', { required: true })
const poolModeEnabled = defineModel<boolean>('poolModeEnabled', { required: true })
const poolModeRetryCount = defineModel<number>('poolModeRetryCount', { required: true })
const poolModeRetryStatusCodesInput = defineModel<string>('poolModeRetryStatusCodesInput', { required: true })
const customErrorCodesEnabled = defineModel<boolean>('customErrorCodesEnabled', { required: true })
const selectedErrorCodes = defineModel<number[]>('selectedErrorCodes', { required: true })
const customErrorCodeInput = defineModel<number | null>('customErrorCodeInput', { required: true })
const headerOverrideEnabled = defineModel<boolean>('headerOverrideEnabled', { required: true })
const headerOverrideRows = defineModel<HeaderOverrideRow[]>('headerOverrideRows', { required: true })
const accountMode = defineModel<CnAccountMode>('accountMode', { default: 'payg' })
const apiProtocol = defineModel<CnApiProtocol>('apiProtocol', { default: 'chat_completions' })
const zhipuOrganization = defineModel<string>('zhipuOrganization', { default: '' })
const zhipuProject = defineModel<string>('zhipuProject', { default: '' })
const bedrockAuthMode = defineModel<BedrockAuthMode>('bedrockAuthMode', { required: true })
const bedrockAccessKeyId = defineModel<string>('bedrockAccessKeyId', { required: true })
const bedrockSecretAccessKey = defineModel<string>('bedrockSecretAccessKey', { required: true })
const bedrockSessionToken = defineModel<string>('bedrockSessionToken', { required: true })
const bedrockApiKeyValue = defineModel<string>('bedrockApiKeyValue', { required: true })
const bedrockRegion = defineModel<string>('bedrockRegion', { required: true })
const bedrockForceGlobal = defineModel<boolean>('bedrockForceGlobal', { required: true })

const baseUrlHint = computed(() => {
  const key = apiKeyBaseURLHintKey(props.platform)
  return key ? t(key) : ''
})
const apiKeyHint = computed(() => {
  const key = apiKeyValueHintKey(props.platform)
  return key ? t(key) : ''
})
const deepseekBaseUrlPresets = DEEPSEEK_BASE_URL_PRESETS
const zhipuBaseUrlPresets = ZHIPU_BASE_URL_PRESETS
const zhipuProtocolOptions = [
  { value: 'chat_completions' as const, key: 'chatCompletions' },
  { value: 'anthropic' as const, key: 'anthropic' },
]

// A URL preset represents a complete routing choice, not just a text snippet.
// Keep the mode/protocol models in sync so selecting "Coding" or "Anthropic"
// cannot submit a mismatched account_mode/api_protocol pair.
const selectZhipuPreset = (url: string) => {
  apiKeyBaseUrl.value = url
  const routing = inferZhipuRoutingFromBaseURL(url)
  if (routing?.protocol) apiProtocol.value = routing.protocol
  if (routing?.mode) accountMode.value = routing.mode
}
const syncPreviewCredentials = computed(() =>
  apiKeyValue.value
    ? {
        platform: props.platform,
        type: props.accountType,
        base_url: apiKeyBaseUrl.value || undefined,
        api_key: apiKeyValue.value,
        ...(props.platform === 'zhipu'
          ? {
              account_mode: accountMode.value,
              api_protocol: apiProtocol.value,
              zhipu_organization: zhipuOrganization.value || undefined,
              zhipu_project: zhipuProject.value || undefined,
            }
          : {}),
      }
    : undefined,
)
const presetMappings = computed(() => getPresetMappingsByPlatform(props.platform))
const bedrockPresets = computed(() => getPresetMappingsByPlatform('bedrock'))
const getModelMappingKey = createStableObjectKeyResolver<AccountModelMappingDraft>('create-model-mapping')

const addModelMapping = () => modelMappings.value.push({ from: '', to: '' })
const removeModelMapping = (index: number) => modelMappings.value.splice(index, 1)
const updateModelMapping = (
  index: number,
  field: keyof AccountModelMappingDraft,
  value: string,
) => {
  const mapping = modelMappings.value[index]
  if (mapping) mapping[field] = value
}
const addPresetMapping = (from: string, to: string) => {
  if (modelMappings.value.some((mapping) => mapping.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const confirmSensitiveErrorCode = (code: number) => {
  if (code === 429) return confirm(t('admin.accounts.customErrorCodes429Warning'))
  if (code === 529) return confirm(t('admin.accounts.customErrorCodes529Warning'))
  return true
}
const toggleErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index >= 0) {
    selectedErrorCodes.value.splice(index, 1)
  } else if (confirmSensitiveErrorCode(code)) {
    selectedErrorCodes.value.push(code)
  }
}
const addCustomErrorCode = () => {
  const code = customErrorCodeInput.value
  if (code === null || code < 100 || code > 599) {
    appStore.showError(t('admin.accounts.invalidErrorCode'))
    return
  }
  if (selectedErrorCodes.value.includes(code)) {
    appStore.showInfo(t('admin.accounts.errorCodeExists'))
    return
  }
  if (!confirmSensitiveErrorCode(code)) return
  selectedErrorCodes.value.push(code)
  customErrorCodeInput.value = null
}
const removeErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index >= 0) selectedErrorCodes.value.splice(index, 1)
}
</script>
