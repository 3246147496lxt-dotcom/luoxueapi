<template>
  <div>
    <div class="flex items-center justify-between">
      <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
      <button
        type="button"
        @click="emit('open-help')"
        class="flex items-center gap-1 rounded px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 dark:text-blue-400 dark:hover:bg-blue-900/20"
      >
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
        </svg>
        {{ t('admin.accounts.gemini.helpButton') }}
      </button>
    </div>
    <div class="mt-2 grid grid-cols-3 gap-3" data-tour="account-form-type">
      <button
        type="button"
        @click="category = 'oauth-based'"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          category === 'oauth-based'
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
            : 'border-gray-200 hover:border-blue-300 dark:border-dark-600 dark:hover:border-blue-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            category === 'oauth-based'
              ? 'bg-blue-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="key" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accounts.gemini.accountType.oauthTitle') }}
          </span>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.gemini.accountType.oauthDesc') }}
          </span>
        </div>
      </button>

      <button
        type="button"
        @click="category = 'apikey'"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          category === 'apikey'
            ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
            : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            category === 'apikey'
              ? 'bg-purple-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <svg
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="1.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1721.75 8.25z"
            />
          </svg>
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accounts.gemini.accountType.apiKeyTitle') }}
          </span>
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.gemini.accountType.apiKeyDesc') }}
          </span>
        </div>
      </button>

      <button
        type="button"
        @click="category = 'service_account'"
        :class="[
          'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
          category === 'service_account'
            ? 'border-sky-500 bg-sky-50 dark:bg-sky-900/20'
            : 'border-gray-200 hover:border-sky-300 dark:border-dark-600 dark:hover:border-sky-700'
        ]"
      >
        <div
          :class="[
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
            category === 'service_account'
              ? 'bg-sky-500 text-white'
              : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
          ]"
        >
          <Icon name="cloud" size="sm" />
        </div>
        <div>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">Vertex</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">Service Account</span>
        </div>
      </button>
    </div>

    <div
      v-if="category === 'apikey'"
      class="mt-3 rounded-lg border border-purple-200 bg-purple-50 px-3 py-2 text-xs text-purple-800 dark:border-purple-800/40 dark:bg-purple-900/20 dark:text-purple-200"
    >
      <p>{{ t('admin.accounts.gemini.accountType.apiKeyNote') }}</p>
      <div class="mt-2 flex flex-wrap gap-2">
        <a
          :href="helpLinks.apiKey"
          class="font-medium text-blue-600 hover:underline dark:text-blue-400"
          target="_blank"
          rel="noreferrer"
        >
          {{ t('admin.accounts.gemini.accountType.apiKeyLink') }}
        </a>
      </div>
    </div>

    <div
      v-if="category === 'service_account'"
      class="mt-3 rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-xs text-sky-800 dark:border-sky-800/40 dark:bg-sky-900/20 dark:text-sky-200"
    >
      <p>{{ t('admin.accounts.vertexGeminiHint') }}</p>
    </div>

    <div v-if="category === 'oauth-based'" class="mt-4">
      <label class="input-label">{{ t('admin.accounts.oauth.gemini.oauthTypeLabel') }}</label>
      <div class="mt-2 grid grid-cols-2 gap-3">
        <button
          type="button"
          @click="emit('select-oauth-type', 'google_one')"
          :class="[
            'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
            oauthType === 'google_one'
              ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
              : 'border-gray-200 hover:border-purple-300 dark:border-dark-600 dark:hover:border-purple-700'
          ]"
        >
          <div
            :class="[
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
              oauthType === 'google_one'
                ? 'bg-purple-500 text-white'
                : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
            ]"
          >
            <Icon name="user" size="sm" />
          </div>
          <div class="min-w-0">
            <span class="block text-sm font-medium text-gray-900 dark:text-white">Google One</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.gemini.oauthType.googleOneDesc') }}
            </span>
            <div class="mt-2 flex flex-wrap gap-1">
              <span class="rounded bg-purple-100 px-2 py-0.5 text-[10px] font-semibold text-purple-700 dark:bg-purple-900/40 dark:text-purple-300">
                {{ t('admin.accounts.gemini.oauthType.badges.individuals') }}
              </span>
              <span class="rounded bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                {{ t('admin.accounts.gemini.oauthType.badges.noGcp') }}
              </span>
            </div>
          </div>
        </button>

        <button
          type="button"
          @click="emit('select-oauth-type', 'code_assist')"
          :class="[
            'flex items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
            oauthType === 'code_assist'
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
              : 'border-gray-200 hover:border-blue-300 dark:border-dark-600 dark:hover:border-blue-700'
          ]"
        >
          <div
            :class="[
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
              oauthType === 'code_assist'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
            ]"
          >
            <Icon name="cloud" size="sm" />
          </div>
          <div class="min-w-0">
            <span class="block text-sm font-medium text-gray-900 dark:text-white">GCP Code Assist</span>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.gemini.oauthType.codeAssistDesc') }}
            </span>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.gemini.oauthType.codeAssistRequirement') }}
              <a
                :href="helpLinks.gcpProject"
                class="ml-1 text-blue-600 hover:underline dark:text-blue-400"
                target="_blank"
                rel="noreferrer"
              >
                {{ t('admin.accounts.gemini.oauthType.gcpProjectLink') }}
              </a>
            </div>
            <div class="mt-2 flex flex-wrap gap-1">
              <span class="rounded bg-blue-100 px-2 py-0.5 text-[10px] font-semibold text-blue-700 dark:bg-blue-900/40 dark:text-blue-300">
                {{ t('admin.accounts.gemini.oauthType.badges.enterprise') }}
              </span>
              <span class="rounded bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
                {{ t('admin.accounts.gemini.oauthType.badges.highConcurrency') }}
              </span>
            </div>
          </div>
        </button>
      </div>

      <div class="mt-3">
        <button
          type="button"
          @click="showAdvanced = !showAdvanced"
          class="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
        >
          <svg
            :class="['h-4 w-4 transition-transform', showAdvanced ? 'rotate-90' : '']"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
          </svg>
          <span>
            {{
              showAdvanced
                ? t('admin.accounts.gemini.oauthType.hideAdvanced')
                : t('admin.accounts.gemini.oauthType.showAdvanced')
            }}
          </span>
        </button>
      </div>

      <div v-if="showAdvanced" class="mt-3 group relative">
        <button
          type="button"
          :disabled="!aiStudioOAuthEnabled"
          @click="emit('select-oauth-type', 'ai_studio')"
          :class="[
            'flex w-full items-center gap-3 rounded-lg border-2 p-3 text-left transition-all',
            !aiStudioOAuthEnabled ? 'cursor-not-allowed opacity-60' : '',
            oauthType === 'ai_studio'
              ? 'border-amber-500 bg-amber-50 dark:bg-amber-900/20'
              : 'border-gray-200 hover:border-amber-300 dark:border-dark-600 dark:hover:border-amber-700'
          ]"
        >
          <div
            :class="[
              'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg',
              oauthType === 'ai_studio'
                ? 'bg-amber-500 text-white'
                : 'bg-gray-100 text-gray-500 dark:bg-dark-600 dark:text-gray-400'
            ]"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 18.75l-.813-2.846a4.5 4.5 0 00-3.09-3.09L2.25 12l2.846-.813a4.5 4.5 0 003.09-3.09L9 5.25l.813 2.846a4.5 4.5 0 003.09 3.09L15.75 12l-2.846.813a4.5 4.5 0 00-3.09 3.09z" />
            </svg>
          </div>
          <div class="min-w-0">
            <span class="block text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.accounts.gemini.oauthType.customTitle') }}
            </span>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.gemini.oauthType.customDesc') }}
            </span>
            <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.gemini.oauthType.customRequirement') }}
            </div>
            <div class="mt-2 flex flex-wrap gap-1">
              <span class="rounded bg-amber-100 px-2 py-0.5 text-[10px] font-semibold text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
                {{ t('admin.accounts.gemini.oauthType.badges.orgManaged') }}
              </span>
              <span class="rounded bg-amber-100 px-2 py-0.5 text-[10px] font-semibold text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">
                {{ t('admin.accounts.gemini.oauthType.badges.adminRequired') }}
              </span>
            </div>
          </div>
          <span
            v-if="!aiStudioOAuthEnabled"
            class="ml-auto shrink-0 rounded bg-amber-100 px-2 py-0.5 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
          >
            {{ t('admin.accounts.oauth.gemini.aiStudioNotConfiguredShort') }}
          </span>
        </button>

        <div
          v-if="!aiStudioOAuthEnabled"
          class="pointer-events-none absolute right-0 top-full z-50 mt-2 w-80 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 opacity-0 shadow-lg transition-opacity group-hover:opacity-100 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-200"
        >
          {{ t('admin.accounts.oauth.gemini.aiStudioNotConfiguredTip') }}
        </div>
      </div>
    </div>

    <div v-if="category !== 'service_account'" class="mt-4">
      <label class="input-label">{{ t('admin.accounts.gemini.tier.label') }}</label>
      <div class="mt-2">
        <select v-if="oauthType === 'google_one'" v-model="googleOneTier" class="input">
          <option value="google_one_free">{{ t('admin.accounts.gemini.tier.googleOne.free') }}</option>
          <option value="google_ai_pro">{{ t('admin.accounts.gemini.tier.googleOne.pro') }}</option>
          <option value="google_ai_ultra">{{ t('admin.accounts.gemini.tier.googleOne.ultra') }}</option>
        </select>

        <select v-else-if="oauthType === 'code_assist'" v-model="gcpTier" class="input">
          <option value="gcp_standard">{{ t('admin.accounts.gemini.tier.gcp.standard') }}</option>
          <option value="gcp_enterprise">{{ t('admin.accounts.gemini.tier.gcp.enterprise') }}</option>
        </select>

        <select v-else v-model="aiStudioTier" class="input">
          <option value="aistudio_free">{{ t('admin.accounts.gemini.tier.aiStudio.free') }}</option>
          <option value="aistudio_paid">{{ t('admin.accounts.gemini.tier.aiStudio.paid') }}</option>
        </select>
      </div>
      <p class="input-hint">{{ t('admin.accounts.gemini.tier.hint') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type {
  CreateAccountCategory,
  GeminiAIStudioTier,
  GeminiGCPTier,
  GeminiGoogleOneTier,
  GeminiOAuthType,
} from '../formDraft'

const props = defineProps<{
  modelValue: CreateAccountCategory
  oauthType: GeminiOAuthType
  aiStudioOAuthEnabled: boolean
  showAdvanced: boolean
  googleOneTier: GeminiGoogleOneTier
  gcpTier: GeminiGCPTier
  aiStudioTier: GeminiAIStudioTier
  helpLinks: { apiKey: string; gcpProject: string }
}>()
const emit = defineEmits<{
  'update:modelValue': [value: CreateAccountCategory]
  'update:showAdvanced': [value: boolean]
  'update:googleOneTier': [value: GeminiGoogleOneTier]
  'update:gcpTier': [value: GeminiGCPTier]
  'update:aiStudioTier': [value: GeminiAIStudioTier]
  'select-oauth-type': [value: GeminiOAuthType]
  'open-help': []
}>()

const { t } = useI18n()
const category = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const showAdvanced = computed({
  get: () => props.showAdvanced,
  set: (value) => emit('update:showAdvanced', value),
})
const googleOneTier = computed({
  get: () => props.googleOneTier,
  set: (value) => emit('update:googleOneTier', value),
})
const gcpTier = computed({
  get: () => props.gcpTier,
  set: (value) => emit('update:gcpTier', value),
})
const aiStudioTier = computed({
  get: () => props.aiStudioTier,
  set: (value) => emit('update:aiStudioTier', value),
})
</script>
