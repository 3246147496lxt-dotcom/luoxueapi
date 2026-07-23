<template>
  <div class="space-y-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.bedrockAuthMode') }}</label>
      <div class="mt-2 flex gap-4">
        <label class="flex cursor-pointer items-center">
          <input
            v-model="authMode"
            type="radio"
            value="sigv4"
            class="mr-2 text-primary-600 focus:ring-primary-500"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.bedrockAuthModeSigv4') }}</span>
        </label>
        <label class="flex cursor-pointer items-center">
          <input
            v-model="authMode"
            type="radio"
            value="apikey"
            class="mr-2 text-primary-600 focus:ring-primary-500"
          />
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.bedrockAuthModeApikey') }}</span>
        </label>
      </div>
    </div>

    <template v-if="authMode === 'sigv4'">
      <div>
        <label class="input-label">{{ t('admin.accounts.bedrockAccessKeyId') }}</label>
        <input v-model="accessKeyId" type="text" required class="input font-mono" placeholder="AKIA..." />
      </div>
      <div>
        <label class="input-label">{{ t('admin.accounts.bedrockSecretAccessKey') }}</label>
        <input v-model="secretAccessKey" type="password" required class="input font-mono" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.accounts.bedrockSessionToken') }}</label>
        <input v-model="sessionToken" type="password" class="input font-mono" />
        <p class="input-hint">{{ t('admin.accounts.bedrockSessionTokenHint') }}</p>
      </div>
    </template>

    <div v-if="authMode === 'apikey'">
      <label class="input-label">{{ t('admin.accounts.bedrockApiKeyInput') }}</label>
      <input v-model="apiKey" type="password" required class="input font-mono" />
    </div>

    <div>
      <label class="input-label">{{ t('admin.accounts.bedrockRegion') }}</label>
      <select v-model="region" class="input">
        <optgroup label="US">
          <option value="us-east-1">us-east-1 (N. Virginia)</option>
          <option value="us-east-2">us-east-2 (Ohio)</option>
          <option value="us-west-1">us-west-1 (N. California)</option>
          <option value="us-west-2">us-west-2 (Oregon)</option>
          <option value="us-gov-east-1">us-gov-east-1 (GovCloud US-East)</option>
          <option value="us-gov-west-1">us-gov-west-1 (GovCloud US-West)</option>
        </optgroup>
        <optgroup label="Europe">
          <option value="eu-west-1">eu-west-1 (Ireland)</option>
          <option value="eu-west-2">eu-west-2 (London)</option>
          <option value="eu-west-3">eu-west-3 (Paris)</option>
          <option value="eu-central-1">eu-central-1 (Frankfurt)</option>
          <option value="eu-central-2">eu-central-2 (Zurich)</option>
          <option value="eu-south-1">eu-south-1 (Milan)</option>
          <option value="eu-south-2">eu-south-2 (Spain)</option>
          <option value="eu-north-1">eu-north-1 (Stockholm)</option>
        </optgroup>
        <optgroup label="Asia Pacific">
          <option value="ap-northeast-1">ap-northeast-1 (Tokyo)</option>
          <option value="ap-northeast-2">ap-northeast-2 (Seoul)</option>
          <option value="ap-northeast-3">ap-northeast-3 (Osaka)</option>
          <option value="ap-south-1">ap-south-1 (Mumbai)</option>
          <option value="ap-south-2">ap-south-2 (Hyderabad)</option>
          <option value="ap-southeast-1">ap-southeast-1 (Singapore)</option>
          <option value="ap-southeast-2">ap-southeast-2 (Sydney)</option>
        </optgroup>
        <optgroup label="Canada">
          <option value="ca-central-1">ca-central-1 (Canada)</option>
        </optgroup>
        <optgroup label="South America">
          <option value="sa-east-1">sa-east-1 (São Paulo)</option>
        </optgroup>
      </select>
      <p class="input-hint">{{ t('admin.accounts.bedrockRegionHint') }}</p>
    </div>

    <div>
      <label class="flex items-center gap-2 cursor-pointer">
        <input
          v-model="forceGlobal"
          type="checkbox"
          class="rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
        />
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.bedrockForceGlobal') }}</span>
      </label>
      <p class="input-hint mt-1">{{ t('admin.accounts.bedrockForceGlobalHint') }}</p>
    </div>

    <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
      <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>
      <div class="mb-4 flex gap-2">
        <button
          type="button"
          @click="restrictionMode = 'whitelist'"
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            restrictionMode === 'whitelist'
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
        >
          {{ t('admin.accounts.modelWhitelist') }}
        </button>
        <button
          type="button"
          @click="restrictionMode = 'mapping'"
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            restrictionMode === 'mapping'
              ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
        >
          {{ t('admin.accounts.modelMapping') }}
        </button>
      </div>

      <div v-if="restrictionMode === 'whitelist'">
        <ModelWhitelistSelector v-model="allowedModels" platform="anthropic" :sync-credentials="syncCredentials" />
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
          <span v-if="allowedModels.length === 0">{{ t('admin.accounts.supportsAllModels') }}</span>
        </p>
      </div>

      <div v-else class="space-y-3">
        <div v-for="(mapping, index) in modelMappings" :key="index" class="flex items-center gap-2">
          <input
            :value="mapping.from"
            type="text"
            class="input flex-1"
            :placeholder="t('admin.accounts.fromModel')"
            @input="updateMapping(index, 'from', $event)"
          />
          <span class="text-gray-400">→</span>
          <input
            :value="mapping.to"
            type="text"
            class="input flex-1"
            :placeholder="t('admin.accounts.toModel')"
            @input="updateMapping(index, 'to', $event)"
          />
          <button type="button" @click="emit('remove-mapping', index)" class="text-red-500 hover:text-red-700">
            <Icon name="trash" size="sm" />
          </button>
        </div>
        <button type="button" @click="emit('add-mapping')" class="btn btn-secondary text-sm">
          + {{ t('admin.accounts.addMapping') }}
        </button>
        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in presets"
            :key="preset.from"
            type="button"
            @click="emit('add-preset', preset.from, preset.to)"
            :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
          >
            + {{ preset.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
      <div class="mb-3 flex items-center justify-between">
        <div>
          <label class="input-label mb-0">{{ t('admin.accounts.poolMode') }}</label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accounts.poolModeHint') }}</p>
        </div>
        <button
          type="button"
          @click="poolEnabled = !poolEnabled"
          :class="[
            'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
            poolEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
          ]"
        >
          <span
            :class="[
              'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
              poolEnabled ? 'translate-x-5' : 'translate-x-0'
            ]"
          />
        </button>
      </div>
      <div v-if="poolEnabled" class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
        <p class="text-xs text-blue-700 dark:text-blue-400">
          <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.accounts.poolModeInfo') }}
        </p>
      </div>
      <div v-if="poolEnabled" class="mt-3">
        <label class="input-label">{{ t('admin.accounts.poolModeRetryCount') }}</label>
        <input v-model.number="poolRetryCount" type="number" min="0" :max="maxPoolRetryCount" step="1" class="input" />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.poolModeRetryCountHint', { default: defaultPoolRetryCount, max: maxPoolRetryCount }) }}
        </p>
      </div>
      <div v-if="poolEnabled" class="mt-3">
        <label class="input-label">{{ t('admin.accounts.poolModeRetryStatusCodes') }}</label>
        <input
          v-model="poolRetryStatusCodes"
          type="text"
          class="input"
          :placeholder="defaultPoolRetryStatusCodes.join(', ')"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.poolModeRetryStatusCodesHint', { default: defaultPoolRetryStatusCodes.join(', ') }) }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'
import type {
  AccountModelMappingDraft,
  BedrockAuthMode,
  ModelRestrictionMode,
} from '../formDraft'

const props = defineProps<{
  authMode: BedrockAuthMode
  accessKeyId: string
  secretAccessKey: string
  sessionToken: string
  apiKey: string
  region: string
  forceGlobal: boolean
  restrictionMode: ModelRestrictionMode
  allowedModels: string[]
  modelMappings: AccountModelMappingDraft[]
  presets: Array<{ label: string; from: string; to: string; color: string }>
  syncCredentials?: {
    platform: string
    type: string
    base_url?: string
    api_key: string
  }
  poolEnabled: boolean
  poolRetryCount: number
  poolRetryStatusCodes: string
  defaultPoolRetryCount: number
  maxPoolRetryCount: number
  defaultPoolRetryStatusCodes: number[]
}>()
const emit = defineEmits<{
  'update:authMode': [value: BedrockAuthMode]
  'update:accessKeyId': [value: string]
  'update:secretAccessKey': [value: string]
  'update:sessionToken': [value: string]
  'update:apiKey': [value: string]
  'update:region': [value: string]
  'update:forceGlobal': [value: boolean]
  'update:restrictionMode': [value: ModelRestrictionMode]
  'update:allowedModels': [value: string[]]
  'update:poolEnabled': [value: boolean]
  'update:poolRetryCount': [value: number]
  'update:poolRetryStatusCodes': [value: string]
  'add-mapping': []
  'remove-mapping': [index: number]
  'update-mapping': [index: number, field: keyof AccountModelMappingDraft, value: string]
  'add-preset': [from: string, to: string]
}>()

const { t } = useI18n()
const authMode = computed({
  get: () => props.authMode,
  set: (value) => emit('update:authMode', value),
})
const accessKeyId = computed({
  get: () => props.accessKeyId,
  set: (value) => emit('update:accessKeyId', value),
})
const secretAccessKey = computed({
  get: () => props.secretAccessKey,
  set: (value) => emit('update:secretAccessKey', value),
})
const sessionToken = computed({
  get: () => props.sessionToken,
  set: (value) => emit('update:sessionToken', value),
})
const apiKey = computed({
  get: () => props.apiKey,
  set: (value) => emit('update:apiKey', value),
})
const region = computed({
  get: () => props.region,
  set: (value) => emit('update:region', value),
})
const forceGlobal = computed({
  get: () => props.forceGlobal,
  set: (value) => emit('update:forceGlobal', value),
})
const restrictionMode = computed({
  get: () => props.restrictionMode,
  set: (value) => emit('update:restrictionMode', value),
})
const allowedModels = computed({
  get: () => props.allowedModels,
  set: (value) => emit('update:allowedModels', value),
})
const modelMappings = computed(() => props.modelMappings)
const poolEnabled = computed({
  get: () => props.poolEnabled,
  set: (value) => emit('update:poolEnabled', value),
})
const poolRetryCount = computed({
  get: () => props.poolRetryCount,
  set: (value) => emit('update:poolRetryCount', value),
})
const poolRetryStatusCodes = computed({
  get: () => props.poolRetryStatusCodes,
  set: (value) => emit('update:poolRetryStatusCodes', value),
})

const updateMapping = (
  index: number,
  field: keyof AccountModelMappingDraft,
  event: Event,
) => {
  emit('update-mapping', index, field, (event.target as HTMLInputElement).value)
}

</script>
