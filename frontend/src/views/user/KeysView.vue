<template>
  <AppLayout variant="chat">
    <section class="keys-workspace" data-test="keys-workspace">
      <header class="keys-page-header">
        <h1>{{ t('keys.title') }}</h1>
      </header>

      <section class="keys-endpoint-card" aria-labelledby="keys-endpoint-title">
        <div class="keys-endpoint-label">
          <h2 id="keys-endpoint-title">{{ t('keys.endpoints.title') }}</h2>
        </div>
        <div class="keys-endpoint-field">
          <code :title="apiEndpoint">{{ apiEndpoint }}</code>
          <button
            type="button"
            class="keys-icon-button"
            :title="endpointCopied ? t('keys.endpoints.copied') : t('keys.endpoints.clickToCopy')"
            :aria-label="endpointCopied ? t('keys.endpoints.copied') : t('keys.endpoints.clickToCopy')"
            data-test="endpoint-copy"
            @click="copyEndpoint"
          >
            <KeysLucideIcon :name="endpointCopied ? 'check' : 'copy'" :size="17" aria-hidden="true" />
          </button>
          <button
            type="button"
            class="keys-icon-button"
            :class="`keys-endpoint-test--${endpointTestState}`"
            :disabled="endpointTestState === 'testing'"
            :title="endpointTestLabel"
            :aria-label="endpointTestLabel"
            data-test="endpoint-test"
            @click="testEndpoint"
          >
            <KeysLucideIcon
              :name="endpointTestState === 'testing' ? 'loaderCircle' : endpointTestState === 'ready' ? 'check' : endpointTestState === 'failed' ? 'refreshCw' : 'activity'"
              :size="17"
              :class="endpointTestState === 'testing' && 'animate-spin'"
              aria-hidden="true"
            />
          </button>
          <span
            v-if="endpointTestState !== 'idle'"
            class="keys-endpoint-status"
            :class="`is-${endpointTestState}`"
            aria-live="polite"
          >
            {{ endpointTestMessage }}
          </span>
        </div>
        <div class="keys-endpoint-actions">
          <button
            type="button"
            role="switch"
            class="keys-fast-mode"
            :aria-checked="gptFastMode"
            :title="t('keys.gptFastModeHint')"
            :aria-label="t('keys.gptFastMode')"
            data-test="gpt-fast-mode"
            @click="toggleGptFastMode"
          >
            <span>{{ t('keys.gptFastMode') }}</span>
            <span class="keys-fast-mode-track" aria-hidden="true"><span /></span>
          </button>
          <button
            type="button"
            class="keys-create-button"
            data-tour="keys-create-btn"
            data-test="key-create-header"
            @click="showCreateModal = true"
          >
            <KeysLucideIcon name="plus" :size="17" aria-hidden="true" />
            <span>{{ t('keys.createKey') }}</span>
          </button>
        </div>
      </section>

      <div v-if="loading" class="keys-loading-state" aria-live="polite" :aria-label="t('common.loading')">
        <div class="keys-loading-table">
          <div v-for="index in 5" :key="index" class="keys-loading-row" />
        </div>
        <div class="keys-loading-cards">
          <div v-for="index in 2" :key="index" class="keys-loading-card" />
        </div>
      </div>

      <div
        v-else-if="loadError && apiKeys.length === 0"
        class="keys-error-state"
        role="alert"
      >
        <p>{{ t('keys.failedToLoad') }}</p>
        <button type="button" class="keys-error-retry" @click="loadApiKeys">
          <KeysLucideIcon name="refreshCw" :size="15" aria-hidden="true" />
          <span>{{ t('keys.retryLoad') }}</span>
        </button>
      </div>

      <div v-else-if="apiKeys.length > 0" class="keys-list-region">
        <ApiKeyWorkspaceList
          class="keys-desktop-table"
          :api-keys="apiKeys"
          :copied-key-id="copiedKeyId"
          :now="now"
          :show-ccs-import="!publicSettings?.hide_ccs_import_button"
          @copy-key="copyKey"
          @change-group="openGroupSelector"
          @use-key="openUseKeyModal"
          @edit="editKey"
          @import-ccs="importToCcswitch"
          @delete="confirmDelete"
        />

        <div class="keys-mobile-list" data-test="api-key-mobile-list">
          <ApiKeySummaryCard
            v-for="row in apiKeys"
            :key="row.id"
            :api-key="row"
            :copied="copiedKeyId === row.id"
            :now="now"
            :show-ccs-import="!publicSettings?.hide_ccs_import_button"
            @copy-key="copyKey"
            @change-group="openGroupSelector"
            @use-key="openUseKeyModal"
            @edit="editKey"
            @import-ccs="importToCcswitch"
            @delete="confirmDelete"
          />
        </div>
      </div>

      <div v-else class="keys-empty-state">
        <EmptyState :title="t('keys.noKeysYet')" :description="t('keys.createFirstKey')">
          <template #action>
            <button type="button" class="key-empty-create btn btn-primary min-h-11" data-test="key-create-empty" @click="showCreateModal = true">
              <KeysLucideIcon name="plus" :size="20" class="mr-2" aria-hidden="true" />
              {{ t('keys.createKey') }}
            </button>
          </template>
        </EmptyState>
      </div>
    </section>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
      width="normal"
      @close="closeModals"
    >
      <form id="key-form" @submit.prevent="handleSubmit" class="space-y-5">
        <div>
          <label class="input-label">{{ t('keys.nameLabel') }}</label>
          <input
            v-model="formData.name"
            type="text"
            required
            class="input"
            :placeholder="t('keys.namePlaceholder')"
            data-tour="key-form-name"
          />
        </div>

        <div>
          <label class="input-label">{{ t('keys.groupLabel') }}</label>
          <Select
            v-model="formData.group_id"
            :options="groupOptions"
            :placeholder="t('keys.selectGroup')"
            :searchable="true"
            :search-placeholder="t('keys.searchGroup')"
            data-tour="key-form-group"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
              />
              <span v-else class="keys-text-muted">{{ t('keys.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :user-rate-multiplier="(option as unknown as GroupOption).userRate"
                :peak-rate-enabled="(option as unknown as GroupOption).peakRateEnabled"
                :peak-start="(option as unknown as GroupOption).peakStart"
                :peak-end="(option as unknown as GroupOption).peakEnd"
                :peak-rate-multiplier="(option as unknown as GroupOption).peakRateMultiplier"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
        </div>

        <!-- Custom Key Section (only for create) -->
        <div v-if="!showEditModal" class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.customKeyLabel') }}</label>
            <button
              type="button"
              @click="formData.use_custom_key = !formData.use_custom_key"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.use_custom_key ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.use_custom_key ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="formData.use_custom_key">
            <input
              v-model="formData.custom_key"
              type="text"
              class="input font-mono"
              :placeholder="t('keys.customKeyPlaceholder')"
              :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            />
            <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
            <p v-else class="input-hint">{{ t('keys.customKeyHint') }}</p>
          </div>
        </div>

        <div v-if="showEditModal">
          <label class="input-label">{{ t('keys.statusLabel') }}</label>
          <Select
            v-model="formData.status"
            :options="statusOptions"
            :placeholder="t('keys.selectStatus')"
          />
        </div>

        <!-- IP Restriction Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.ipRestriction') }}</label>
            <button
              type="button"
              @click="formData.enable_ip_restriction = !formData.enable_ip_restriction"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_ip_restriction ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_ip_restriction ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_ip_restriction" class="space-y-4 pt-2">
            <div>
              <label class="input-label">{{ t('keys.ipWhitelist') }}</label>
              <textarea
                v-model="formData.ip_whitelist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipWhitelistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipWhitelistHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('keys.ipBlacklist') }}</label>
              <textarea
                v-model="formData.ip_blacklist"
                rows="3"
                class="input font-mono text-sm"
                :placeholder="t('keys.ipBlacklistPlaceholder')"
              />
              <p class="input-hint">{{ t('keys.ipBlacklistHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Quota Limit Section -->
        <div class="space-y-3">
          <label class="input-label">{{ t('keys.quotaLimit') }}</label>
          <!-- Switch commented out - always show input, 0 = unlimited
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.quotaLimit') }}</label>
            <button
              type="button"
              @click="formData.enable_quota = !formData.enable_quota"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_quota ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_quota ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          -->

          <div class="space-y-4">
            <div>
              <div class="relative">
                <PointsIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.quota"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="t('keys.quotaAmountPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('keys.quotaAmountHint') }}</p>
            </div>

            <!-- Quota used display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
              <label class="input-label">{{ t('keys.quotaUsed') }}</label>
              <div class="flex items-center gap-2">
                <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2">
                  <CreditAmount
                    class="keys-text-primary font-medium"
                    :value="selectedKey.quota_used?.toFixed(4) || '0.0000'"
                    icon-size="xs"
                  />
                  <span class="keys-text-muted mx-2">/</span>
                  <CreditAmount
                    class="keys-text-secondary"
                    :value="selectedKey.quota?.toFixed(2) || '0.00'"
                    icon-size="xs"
                  />
                </div>
                <button
                  type="button"
                  @click="confirmResetQuota"
                  class="btn btn-secondary text-sm"
                  :title="t('keys.resetQuotaUsed')"
                >
                  {{ t('keys.reset') }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Rate Limit Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.rateLimitSection') }}</label>
            <button
              type="button"
              @click="formData.enable_rate_limit = !formData.enable_rate_limit"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_rate_limit ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_rate_limit ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_rate_limit" class="space-y-4 pt-2">
            <p class="input-hint -mt-2">{{ t('keys.rateLimitHint') }}</p>
            <!-- 5-Hour Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit5h') }}</label>
              <div class="relative">
                <PointsIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_5h"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_5h > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_5h?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'text-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_5h?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h ? 'bg-red-500' :
                      selectedKey.usage_5h >= selectedKey.rate_limit_5h * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_5h / selectedKey.rate_limit_5h) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Daily Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit1d') }}</label>
              <div class="relative">
                <PointsIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_1d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_1d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_1d?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'text-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_1d?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d ? 'bg-red-500' :
                      selectedKey.usage_1d >= selectedKey.rate_limit_1d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_1d / selectedKey.rate_limit_1d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- 7-Day Limit -->
            <div>
              <label class="input-label">{{ t('keys.rateLimit7d') }}</label>
              <div class="relative">
                <PointsIcon class="absolute left-3 top-1/2 -translate-y-1/2" size="sm" />
                <input
                  v-model.number="formData.rate_limit_7d"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input pl-10"
                  :placeholder="'0'"
                />
              </div>
              <!-- Usage info (edit mode only) -->
              <div v-if="showEditModal && selectedKey && selectedKey.rate_limit_7d > 0" class="mt-2">
                <div class="flex items-center gap-2">
                  <div class="keys-form-usage-surface flex-1 rounded-lg px-3 py-2 text-sm">
                    <CreditAmount
                      :value="selectedKey.usage_7d?.toFixed(4) || '0.0000'"
                      icon-size="xs"
                      :class="[
                      'font-medium',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'text-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'text-yellow-500' :
                      'keys-text-primary'
                    ]"
                    />
                    <span class="keys-text-muted mx-2">/</span>
                    <CreditAmount
                      class="keys-text-secondary"
                      :value="selectedKey.rate_limit_7d?.toFixed(2) || '0.00'"
                      icon-size="xs"
                    />
                  </div>
                </div>
                <div class="keys-form-progress-track mt-1 h-1.5 w-full overflow-hidden rounded-full">
                  <div
                    :class="[
                      'h-full rounded-full transition-all',
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d ? 'bg-red-500' :
                      selectedKey.usage_7d >= selectedKey.rate_limit_7d * 0.8 ? 'bg-yellow-500' :
                      'bg-green-500'
                    ]"
                    :style="{ width: Math.min((selectedKey.usage_7d / selectedKey.rate_limit_7d) * 100, 100) + '%' }"
                  />
                </div>
              </div>
            </div>

            <!-- Reset Rate Limit button (edit mode only) -->
            <div v-if="showEditModal && selectedKey && (selectedKey.rate_limit_5h > 0 || selectedKey.rate_limit_1d > 0 || selectedKey.rate_limit_7d > 0)">
              <button
                type="button"
                @click="confirmResetRateLimit"
                class="btn btn-secondary text-sm"
              >
                {{ t('keys.resetRateLimitUsage') }}
              </button>
            </div>
          </div>
        </div>

        <!-- Expiration Section -->
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <label class="input-label mb-0">{{ t('keys.expiration') }}</label>
            <button
              type="button"
              @click="formData.enable_expiration = !formData.enable_expiration"
              :class="[
                'relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none',
                formData.enable_expiration ? 'bg-primary-600' : 'keys-switch-track'
              ]"
            >
              <span
                :class="[
                  'keys-switch-thumb pointer-events-none inline-block h-4 w-4 transform rounded-full shadow ring-0 transition duration-200 ease-in-out',
                  formData.enable_expiration ? 'translate-x-4' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
            <!-- Quick select buttons (for both create and edit mode) -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="days in ['7', '30', '90']"
                :key="days"
                type="button"
                @click="setExpirationDays(parseInt(days))"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === days
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'keys-neutral-choice'
                ]"
              >
                {{ showEditModal ? t('keys.extendDays', { days }) : t('keys.expiresInDays', { days }) }}
              </button>
              <button
                type="button"
                @click="formData.expiration_preset = 'custom'"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm transition-colors',
                  formData.expiration_preset === 'custom'
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                    : 'keys-neutral-choice'
                ]"
              >
                {{ t('keys.customDate') }}
              </button>
            </div>

            <!-- Date picker (always show for precise adjustment) -->
            <div>
              <label class="input-label">{{ t('keys.expirationDate') }}</label>
              <input
                v-model="formData.expiration_date"
                type="datetime-local"
                class="input"
              />
              <p class="input-hint">{{ t('keys.expirationDateHint') }}</p>
            </div>

            <!-- Current expiration display (only in edit mode) -->
            <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
              <span class="keys-text-secondary">{{ t('keys.currentExpiration') }}: </span>
              <span class="keys-text-primary font-medium">
                {{ formatDateTime(selectedKey.expires_at) }}
              </span>
            </div>
          </div>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModals" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            form="key-form"
            type="submit"
            :disabled="submitting"
            class="btn btn-primary"
            data-tour="key-form-submit"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              submitting
                ? t('keys.saving')
                : showEditModal
                  ? t('common.update')
                  : t('common.create')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation Dialog -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('keys.deleteKey')"
      :message="t('keys.deleteConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="handleDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaDialog"
      :title="t('keys.resetQuotaTitle')"
      :message="t('keys.resetQuotaConfirmMessage', { name: selectedKey?.name, used: selectedKey?.quota_used?.toFixed(4) })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetQuotaUsed"
      @cancel="showResetQuotaDialog = false"
    />

    <!-- Reset Rate Limit Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetRateLimitDialog"
      :title="t('keys.resetRateLimitTitle')"
      :message="t('keys.resetRateLimitConfirmMessage', { name: selectedKey?.name })"
      :confirm-text="t('keys.reset')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="resetRateLimitUsage"
      @cancel="showResetRateLimitDialog = false"
    />

    <!-- Use Key Modal -->
    <UseKeyModal
      :show="showUseKeyModal"
      :api-key="selectedKey?.key || ''"
      :base-url="apiEndpoint"
      :platform="selectedKey?.group?.platform || null"
      :allow-messages-dispatch="selectedKey?.group?.allow_messages_dispatch || false"
      @close="closeUseKeyModal"
    />

    <!-- CCS Client Selection Dialog for Antigravity -->
    <BaseDialog
      :show="showCcsClientSelect"
      :title="t('keys.ccsClientSelect.title')"
      width="narrow"
      @close="closeCcsClientSelect"
    >
      <div class="space-y-4">
        <p class="keys-text-secondary text-sm">
          {{ t('keys.ccsClientSelect.description') }}
	        </p>
	        <div class="grid grid-cols-2 gap-3">
	          <button
	            @click="handleCcsClientSelect('claude')"
	            class="keys-client-option flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
	          >
	            <Icon name="terminal" size="xl" class="keys-text-secondary" />
	            <span class="keys-text-primary font-medium">{{
	              t('keys.ccsClientSelect.claudeCode')
	            }}</span>
	            <span class="keys-text-secondary text-xs">{{
	              t('keys.ccsClientSelect.claudeCodeDesc')
	            }}</span>
	          </button>
	          <button
	            @click="handleCcsClientSelect('gemini')"
	            class="keys-client-option flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:border-primary-500 hover:bg-primary-50 dark:hover:border-primary-500 dark:hover:bg-primary-900/20"
	          >
	            <Icon name="sparkles" size="xl" class="keys-text-secondary" />
	            <span class="keys-text-primary font-medium">{{
	              t('keys.ccsClientSelect.geminiCli')
	            }}</span>
	            <span class="keys-text-secondary text-xs">{{
	              t('keys.ccsClientSelect.geminiCliDesc')
	            }}</span>
	          </button>
	        </div>
	      </div>
      <template #footer>
        <div class="flex justify-end">
          <button @click="closeCcsClientSelect" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Group Selector Dropdown (Teleported to body to avoid overflow clipping) -->
    <Teleport to="body">
      <div
        v-if="groupSelectorKeyId !== null && dropdownPosition"
        ref="dropdownRef"
        role="dialog"
        :aria-label="t('keys.selectGroup')"
        class="keys-group-selector animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-[min(380px,calc(100vw-2rem))] overflow-hidden rounded-xl shadow-lg duration-200"
        style="pointer-events: auto !important;"
        :style="{
          top: dropdownPosition.top !== undefined ? dropdownPosition.top + 'px' : undefined,
          bottom: dropdownPosition.bottom !== undefined ? dropdownPosition.bottom + 'px' : undefined,
          left: dropdownPosition.left + 'px'
        }"
      >
        <!-- Search box -->
        <div class="keys-group-selector-header border-b p-2">
          <div class="relative">
            <svg class="keys-text-muted absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <input
              v-model="groupSearchQuery"
              type="text"
              class="keys-group-search w-full rounded-lg border py-1.5 pl-8 pr-3 text-sm outline-none focus:border-primary-300 focus:ring-1 focus:ring-primary-300 dark:focus:border-primary-600 dark:focus:ring-primary-600"
              :placeholder="t('keys.searchGroup')"
              :aria-label="t('keys.searchGroup')"
              @click.stop
              @keydown.esc.stop="closeGroupSelectorMenu"
            />
          </div>
        </div>
        <!-- Group list -->
        <div class="max-h-80 overflow-y-auto p-1.5" role="listbox" :aria-label="t('keys.selectGroup')">
          <button
            v-for="option in filteredGroupOptions"
            :key="option.value ?? 'null'"
            role="option"
            :aria-selected="selectedKeyForGroup?.group_id === option.value"
            @click="changeSelectedGroup(option.value)"
            :class="[
              'flex w-full items-center justify-between rounded-lg px-3 py-2.5 text-sm transition-colors',
              'keys-group-option border-b last:border-0',
              selectedKeyForGroup?.group_id === option.value ||
              (!selectedKeyForGroup?.group_id && option.value === null)
                ? 'keys-group-option--selected'
                : 'keys-group-option--idle'
            ]"
            :title="option.description || undefined"
          >
            <GroupOptionItem
              :name="option.label"
              :platform="option.platform"
              :subscription-type="option.subscriptionType"
              :rate-multiplier="option.rate"
              :user-rate-multiplier="option.userRate"
              :peak-rate-enabled="option.peakRateEnabled"
              :peak-start="option.peakStart"
              :peak-end="option.peakEnd"
              :peak-rate-multiplier="option.peakRateMultiplier"
              :description="option.description"
              :selected="
                selectedKeyForGroup?.group_id === option.value ||
                (!selectedKeyForGroup?.group_id && option.value === null)
              "
            />
          </button>
          <!-- Empty state when search has no results -->
          <div v-if="filteredGroupOptions.length === 0" class="keys-text-muted py-4 text-center text-sm">
            {{ t('keys.noGroupFound') }}
          </div>
        </div>
      </div>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

import { authAPI, keysAPI, userGroupsAPI } from '@/api'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import PointsIcon from '@/components/icons/PointsIcon.vue'
import ApiKeySummaryCard from '@/components/keys/ApiKeySummaryCard.vue'
import ApiKeyWorkspaceList from '@/components/keys/ApiKeyWorkspaceList.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { useOnboardingStore } from '@/stores/onboarding'
import type { ApiKey, Group, GroupPlatform, PublicSettings, SubscriptionType, UpdateApiKeyRequest } from '@/types'
import { formatDateTime } from '@/utils/format'
import {
  buildCcSwitchImportDeeplink,
  type CcSwitchClientType
} from '@/utils/ccswitchImport'

const { t } = useI18n()

// Helper to format date for datetime-local input
const formatDateTimeLocal = (isoDate: string): string => {
  const date = new Date(isoDate)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

interface GroupOption {
  value: number
  label: string
  description: string | null
  rate: number
  userRate: number | null
  peakRateEnabled: boolean
  peakStart: string
  peakEnd: string
  peakRateMultiplier: number
  subscriptionType: SubscriptionType
  platform: GroupPlatform
}

const appStore = useAppStore()
const onboardingStore = useOnboardingStore()
const { copyToClipboard: clipboardCopy } = useClipboard()

const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const loadError = ref(false)
const submitting = ref(false)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null
const userGroupRates = ref<Record<number, number>>({})
const publicSettings = ref<PublicSettings | null>(null)

// The endpoint is intentionally shown in the page-level control so it is easy
// to copy before configuring a client. Public settings remain the source of
// truth when configured; the product default keeps the page useful in a fresh
// installation.
const DEFAULT_API_ENDPOINT = 'https://luoxueapi.cc'
const apiEndpoint = computed(() => {
  const configured = publicSettings.value?.api_base_url?.trim().replace(/\/+$/, '')
  return configured || DEFAULT_API_ENDPOINT
})
// Public settings may already contain the API version suffix. Keep the
// displayed endpoint intact, but use a root form when probing `/v1/*` routes
// or generating a CCS usage URL so we never produce `/v1/v1/...`.
const apiEndpointRoot = computed(() => apiEndpoint.value.replace(/\/v1\/?$/i, ''))
const endpointCopied = ref(false)
type EndpointTestState = 'idle' | 'testing' | 'ready' | 'failed'
const endpointTestState = ref<EndpointTestState>('idle')
const endpointTestLatency = ref<number | null>(null)
const GPT_FAST_MODE_STORAGE_KEY = 'api-keys-gpt-fast-mode'
const readGptFastMode = () => {
  if (typeof window === 'undefined') return true
  try {
    const stored = window.localStorage.getItem(GPT_FAST_MODE_STORAGE_KEY)
    return stored === null ? true : stored === 'true'
  } catch {
    return true
  }
}
const gptFastMode = ref(readGptFastMode())

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showResetQuotaDialog = ref(false)
const showResetRateLimitDialog = ref(false)
const showUseKeyModal = ref(false)
const showCcsClientSelect = ref(false)
const pendingCcsRow = ref<ApiKey | null>(null)
const selectedKey = ref<ApiKey | null>(null)
const copiedKeyId = ref<number | null>(null)
const groupSelectorKeyId = ref<number | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<{ top?: number; bottom?: number; left: number } | null>(null)
let abortController: AbortController | null = null

// Get the currently selected key for group change
const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const formData = ref({
  name: '',
  group_id: null as number | null,
  status: 'active' as 'active' | 'inactive',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  // Quota settings (empty = unlimited)
  enable_quota: false,
  quota: null as number | null,
  // Rate limit settings
  enable_rate_limit: false,
  rate_limit_5h: null as number | null,
  rate_limit_1d: null as number | null,
  rate_limit_7d: null as number | null,
  enable_expiration: false,
  expiration_preset: '30' as '7' | '30' | '90' | 'custom',
  expiration_date: ''
})

// 自定义Key验证
const customKeyError = computed(() => {
  if (!formData.value.use_custom_key || !formData.value.custom_key) {
    return ''
  }
  const key = formData.value.custom_key
  if (key.length < 16) {
    return t('keys.customKeyTooShort')
  }
  // 检查字符：只允许字母、数字、下划线、连字符
  if (!/^[a-zA-Z0-9_-]+$/.test(key)) {
    return t('keys.customKeyInvalidChars')
  }
  return ''
})

const statusOptions = computed(() => [
  { value: 'active', label: t('common.active') },
  { value: 'inactive', label: t('common.inactive') }
])

const shouldSubmitEditStatus = (key: ApiKey, status: 'active' | 'inactive') => {
  if (key.status === 'quota_exhausted' || key.status === 'expired') {
    return status === 'active'
  }
  return true
}

// Convert groups to Select options format with rate multiplier and subscription type
const groupOptions = computed(() =>
  groups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    rate: group.rate_multiplier,
    userRate: userGroupRates.value[group.id] ?? null,
    peakRateEnabled: group.peak_rate_enabled,
    peakStart: group.peak_start,
    peakEnd: group.peak_end,
    peakRateMultiplier: group.peak_rate_multiplier,
    subscriptionType: group.subscription_type,
    platform: group.platform
  }))
)

// Group dropdown search
const groupSearchQuery = ref('')
const filteredGroupOptions = computed(() => {
  const query = groupSearchQuery.value.trim().toLowerCase()
  if (!query) return groupOptions.value
  return groupOptions.value.filter((opt) => {
    return opt.label.toLowerCase().includes(query) ||
      (opt.description && opt.description.toLowerCase().includes(query))
  })
})

const copyToClipboard = async (text: string, keyId: number) => {
  const success = await clipboardCopy(text, t('keys.copied'))
  if (success) {
    copiedKeyId.value = keyId
    setTimeout(() => {
      copiedKeyId.value = null
    }, 800)
  }
}

const copyKey = (key: ApiKey) => copyToClipboard(key.key, key.id)

const isAbortError = (error: unknown) => {
  if (!error || typeof error !== 'object') return false
  const { name, code } = error as { name?: string; code?: string }
  return name === 'AbortError' || code === 'ERR_CANCELED'
}

const endpointTestLabel = computed(() => {
  if (endpointTestState.value === 'testing') return t('keys.endpoints.testing')
  if (endpointTestState.value === 'ready') {
    return endpointTestLatency.value === null
      ? t('keys.endpoints.connected')
      : t('keys.endpoints.connectedWithLatency', { latency: endpointTestLatency.value })
  }
  if (endpointTestState.value === 'failed') return t('keys.endpoints.retry')
  return t('keys.endpoints.speedTest')
})

const endpointTestMessage = computed(() => {
  if (endpointTestState.value === 'testing') return t('keys.endpoints.testing')
  if (endpointTestState.value === 'ready') {
    return endpointTestLatency.value === null
      ? t('keys.endpoints.connected')
      : t('keys.endpoints.connectedWithLatency', { latency: endpointTestLatency.value })
  }
  return t('keys.endpoints.failed')
})

const copyEndpoint = async () => {
  const success = await clipboardCopy(apiEndpoint.value, t('keys.endpoints.copied'))
  if (!success) return
  endpointCopied.value = true
  window.setTimeout(() => {
    endpointCopied.value = false
  }, 1200)
}

/**
 * Probe the configured endpoint without requiring an API key. A 401/403 is
 * still a healthy network response, while transport errors are reported as a
 * failed probe. The control is intentionally compact so it never competes
 * with the primary copy action.
 */
const testEndpoint = async () => {
  if (endpointTestState.value === 'testing') return
  endpointTestState.value = 'testing'
  endpointTestLatency.value = null
  const startedAt = typeof performance !== 'undefined' ? performance.now() : Date.now()
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 5000)
  try {
    const response = await fetch(`${apiEndpointRoot.value}/v1/models`, {
      method: 'GET',
      mode: 'cors',
      signal: controller.signal,
      headers: { Accept: 'application/json' }
    })
    if (!response.ok && ![401, 403, 404, 405].includes(response.status)) {
      throw new Error(`Endpoint probe returned ${response.status}`)
    }
    endpointTestLatency.value = Math.max(1, Math.round(
      (typeof performance !== 'undefined' ? performance.now() : Date.now()) - startedAt
    ))
    endpointTestState.value = 'ready'
  } catch {
    endpointTestState.value = 'failed'
  } finally {
    window.clearTimeout(timeout)
  }
}

const toggleGptFastMode = () => {
  gptFastMode.value = !gptFastMode.value
  try {
    window.localStorage.setItem(GPT_FAST_MODE_STORAGE_KEY, String(gptFastMode.value))
  } catch {
    // A locked-down browser can reject localStorage; the in-memory switch
    // remains usable for the current page.
  }
}

const loadApiKeys = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const { signal } = controller
  loading.value = true
  loadError.value = false
  try {
    const filters = { sort_by: 'created_at', sort_order: 'desc' as const }
    const pageSize = 1000
    const firstPage = await keysAPI.list(1, pageSize, filters, { signal })
    if (signal.aborted) return
    const items = [...firstPage.items]
    const totalPages = Math.max(
      firstPage.pages || 1,
      Math.ceil(firstPage.total / Math.max(firstPage.page_size || pageSize, 1))
    )
    // The endpoint caps a page at 1000 items. Fetch additional pages only for
    // unusually large accounts so the UI can remain intentionally pagination-
    // free without silently hiding keys.
    for (let page = 2; page <= totalPages && items.length < firstPage.total; page += 1) {
      const nextPage = await keysAPI.list(page, pageSize, filters, { signal })
      if (signal.aborted) return
      items.push(...nextPage.items)
    }
    apiKeys.value = items
  } catch (error) {
    if (isAbortError(error)) {
      return
    }
    loadError.value = true
    appStore.showError(t('keys.failedToLoad'))
  } finally {
    if (abortController === controller) {
      loading.value = false
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await userGroupsAPI.getAvailable()
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const loadUserGroupRates = async () => {
  try {
    userGroupRates.value = await userGroupsAPI.getUserGroupRates()
  } catch (error) {
    console.error('Failed to load user group rates:', error)
  }
}

const loadPublicSettings = async () => {
  try {
    publicSettings.value = await authAPI.getPublicSettings()
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
}

const openUseKeyModal = (key: ApiKey) => {
  selectedKey.value = key
  showUseKeyModal.value = true
}

const closeUseKeyModal = () => {
  showUseKeyModal.value = false
  selectedKey.value = null
}

const editKey = (key: ApiKey) => {
  selectedKey.value = key
  const hasIPRestriction = (key.ip_whitelist?.length > 0) || (key.ip_blacklist?.length > 0)
  const hasExpiration = !!key.expires_at
  formData.value = {
    name: key.name,
    group_id: key.group_id,
    status: key.status === 'quota_exhausted' || key.status === 'expired' ? 'inactive' : key.status,
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: hasIPRestriction,
    ip_whitelist: (key.ip_whitelist || []).join('\n'),
    ip_blacklist: (key.ip_blacklist || []).join('\n'),
    enable_quota: key.quota > 0,
    quota: key.quota > 0 ? key.quota : null,
    enable_rate_limit: (key.rate_limit_5h > 0) || (key.rate_limit_1d > 0) || (key.rate_limit_7d > 0),
    rate_limit_5h: key.rate_limit_5h || null,
    rate_limit_1d: key.rate_limit_1d || null,
    rate_limit_7d: key.rate_limit_7d || null,
    enable_expiration: hasExpiration,
    expiration_preset: 'custom',
    expiration_date: key.expires_at ? formatDateTimeLocal(key.expires_at) : ''
  }
  showEditModal.value = true
}

const openGroupSelector = (key: ApiKey, event: MouseEvent) => {
  if (groupSelectorKeyId.value === key.id) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  } else {
    const buttonEl = event.currentTarget as HTMLElement | null
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const dropdownEstHeight = 400 // estimated max dropdown height
      const dropdownWidth = Math.min(380, window.innerWidth - 32)
      const spaceBelow = window.innerHeight - rect.bottom
      const spaceAbove = rect.top
      const left = Math.min(
        Math.max(16, rect.left),
        Math.max(16, window.innerWidth - dropdownWidth - 16)
      )

      if (spaceBelow < dropdownEstHeight && spaceAbove > spaceBelow) {
        // Not enough space below, pop upward
        dropdownPosition.value = {
          bottom: window.innerHeight - rect.top + 4,
          left
        }
      } else {
        // Default: pop downward
        dropdownPosition.value = {
          top: rect.bottom + 4,
          left
        }
      }
    }
    groupSelectorKeyId.value = key.id
    groupSearchQuery.value = ''
  }
}

const changeGroup = async (key: ApiKey, newGroupId: number | null) => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
  if (key.group_id === newGroupId) return

  try {
    await keysAPI.update(key.id, { group_id: newGroupId })
    appStore.showSuccess(t('keys.groupChangedSuccess'))
    loadApiKeys()
  } catch (error) {
    appStore.showError(t('keys.failedToChangeGroup'))
  }
}

const changeSelectedGroup = (newGroupId: number | null) => {
  const key = selectedKeyForGroup.value
  if (!key) {
    closeGroupSelectorMenu()
    return
  }
  changeGroup(key, newGroupId)
}

const closeGroupSelectorMenu = () => {
  groupSelectorKeyId.value = null
  dropdownPosition.value = null
}

const closeGroupSelector = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  // Check if click is inside the dropdown or the trigger button
  // The v31 list uses semantic class names instead of the legacy Tailwind
  // `group/dropdown` token, so keep all trigger variants inside the popup
  // boundary. Otherwise the document listener would close the menu
  // immediately after a group chip is clicked.
  const isGroupTrigger = target.closest(
    '.group\\/dropdown, .workspace-group-button, .api-key-summary-card__group'
  )
  if (!isGroupTrigger && !dropdownRef.value?.contains(target)) {
    groupSelectorKeyId.value = null
    dropdownPosition.value = null
  }
}

const handleEscapeKey = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  closeGroupSelectorMenu()
}

const confirmDelete = (key: ApiKey) => {
  selectedKey.value = key
  showDeleteDialog.value = true
}

const handleSubmit = async () => {
  // Validate group_id is required
  if (formData.value.group_id === null) {
    appStore.showError(t('keys.groupRequired'))
    return
  }

  // Validate custom key if enabled
  if (!showEditModal.value && formData.value.use_custom_key) {
    if (!formData.value.custom_key) {
      appStore.showError(t('keys.customKeyRequired'))
      return
    }
    if (customKeyError.value) {
      appStore.showError(customKeyError.value)
      return
    }
  }

  // Parse IP lists only if IP restriction is enabled
  const parseIPList = (text: string): string[] =>
    text.split('\n').map(ip => ip.trim()).filter(ip => ip.length > 0)
  const ipWhitelist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_whitelist) : []
  const ipBlacklist = formData.value.enable_ip_restriction ? parseIPList(formData.value.ip_blacklist) : []

  // Calculate quota value (null/empty/0 = unlimited, stored as 0)
  const quota = formData.value.quota && formData.value.quota > 0 ? formData.value.quota : 0

  // Calculate expiration
  let expiresInDays: number | undefined
  let expiresAt: string | null | undefined
  if (formData.value.enable_expiration && formData.value.expiration_date) {
    if (!showEditModal.value) {
      // Create mode: calculate days from date
      const expDate = new Date(formData.value.expiration_date)
      const now = new Date()
      const diffDays = Math.ceil((expDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
      expiresInDays = diffDays > 0 ? diffDays : 1
    } else {
      // Edit mode: use custom date directly
      expiresAt = new Date(formData.value.expiration_date).toISOString()
    }
  } else if (showEditModal.value) {
    // Edit mode: if expiration disabled or date cleared, send empty string to clear
    expiresAt = ''
  }

  // Calculate rate limit values (send 0 when toggle is off)
  const rateLimitData = formData.value.enable_rate_limit ? {
    rate_limit_5h: formData.value.rate_limit_5h && formData.value.rate_limit_5h > 0 ? formData.value.rate_limit_5h : 0,
    rate_limit_1d: formData.value.rate_limit_1d && formData.value.rate_limit_1d > 0 ? formData.value.rate_limit_1d : 0,
    rate_limit_7d: formData.value.rate_limit_7d && formData.value.rate_limit_7d > 0 ? formData.value.rate_limit_7d : 0,
  } : { rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0 }

  submitting.value = true
  try {
    if (showEditModal.value && selectedKey.value) {
      const updates: UpdateApiKeyRequest = {
        name: formData.value.name,
        group_id: formData.value.group_id,
        ip_whitelist: ipWhitelist,
        ip_blacklist: ipBlacklist,
        quota: quota,
        expires_at: expiresAt,
        rate_limit_5h: rateLimitData.rate_limit_5h,
        rate_limit_1d: rateLimitData.rate_limit_1d,
        rate_limit_7d: rateLimitData.rate_limit_7d,
      }
      if (shouldSubmitEditStatus(selectedKey.value, formData.value.status)) {
        updates.status = formData.value.status
      }
      await keysAPI.update(selectedKey.value.id, updates)
      appStore.showSuccess(t('keys.keyUpdatedSuccess'))
    } else {
      const customKey = formData.value.use_custom_key ? formData.value.custom_key : undefined
      await keysAPI.create(
        formData.value.name,
        formData.value.group_id,
        customKey,
        ipWhitelist,
        ipBlacklist,
        quota,
        expiresInDays,
        rateLimitData
      )
      appStore.showSuccess(t('keys.keyCreatedSuccess'))
      // Only advance tour if active, on submit step, and creation succeeded
      if (onboardingStore.isCurrentStep('[data-tour="key-form-submit"]')) {
        onboardingStore.nextStep(500)
      }
    }
    closeModals()
    loadApiKeys()
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToSave')
    appStore.showError(errorMsg)
    // Don't advance tour on error
  } finally {
    submitting.value = false
  }
}

/**
 * 处理删除 API Key 的操作
 * 优化：错误处理改进，优先显示后端返回的具体错误消息（如权限不足等），
 * 若后端未返回消息则显示默认的国际化文本
 */
const handleDelete = async () => {
  if (!selectedKey.value) return

  try {
    await keysAPI.delete(selectedKey.value.id)
    appStore.showSuccess(t('keys.keyDeletedSuccess'))
    showDeleteDialog.value = false
    loadApiKeys()
  } catch (error: any) {
    // 优先使用后端返回的错误消息，提供更具体的错误信息给用户
    const errorMsg = error?.message || t('keys.failedToDelete')
    appStore.showError(errorMsg)
  }
}

const closeModals = () => {
  showCreateModal.value = false
  showEditModal.value = false
  selectedKey.value = null
  formData.value = {
    name: '',
    group_id: null,
    status: 'active',
    use_custom_key: false,
    custom_key: '',
    enable_ip_restriction: false,
    ip_whitelist: '',
    ip_blacklist: '',
    enable_quota: false,
    quota: null,
    enable_rate_limit: false,
    rate_limit_5h: null,
    rate_limit_1d: null,
    rate_limit_7d: null,
    enable_expiration: false,
    expiration_preset: '30',
    expiration_date: ''
  }
}

// Show reset quota confirmation dialog
const confirmResetQuota = () => {
  showResetQuotaDialog.value = true
}

// Set expiration date based on quick select days
const setExpirationDays = (days: number) => {
  formData.value.expiration_preset = days.toString() as '7' | '30' | '90'
  const expDate = new Date()
  expDate.setDate(expDate.getDate() + days)
  formData.value.expiration_date = formatDateTimeLocal(expDate.toISOString())
}

// Reset quota used for an API key
const resetQuotaUsed = async () => {
  if (!selectedKey.value) return
  showResetQuotaDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_quota: true })
    appStore.showSuccess(t('keys.quotaResetSuccess'))
    // Update local state
    if (selectedKey.value) {
      selectedKey.value.quota_used = 0
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetQuota')
    appStore.showError(errorMsg)
  }
}

// Show reset rate limit confirmation dialog (from edit modal)
const confirmResetRateLimit = () => {
  showResetRateLimitDialog.value = true
}

// Reset rate limit usage for an API key
const resetRateLimitUsage = async () => {
  if (!selectedKey.value) return
  showResetRateLimitDialog.value = false
  try {
    await keysAPI.update(selectedKey.value.id, { reset_rate_limit_usage: true })
    appStore.showSuccess(t('keys.rateLimitResetSuccess'))
    // Refresh key data
    await loadApiKeys()
    // Update the editing key with fresh data
    const refreshedKey = apiKeys.value.find(k => k.id === selectedKey.value!.id)
    if (refreshedKey) {
      selectedKey.value = refreshedKey
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.detail || t('keys.failedToResetRateLimit')
    appStore.showError(errorMsg)
  }
}

const importToCcswitch = (row: ApiKey) => {
  const platform = row.group?.platform || 'anthropic'

  // For antigravity platform, show client selection dialog
  if (platform === 'antigravity') {
    pendingCcsRow.value = row
    showCcsClientSelect.value = true
    return
  }

  // For other platforms, execute directly
  executeCcsImport(row, platform === 'gemini' ? 'gemini' : 'claude')
}

const executeCcsImport = (row: ApiKey, clientType: CcSwitchClientType) => {
  const baseUrl = apiEndpointRoot.value
  const platform = row.group?.platform || 'anthropic'

  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  const providerName = (publicSettings.value?.site_name || '落雪API').trim() || '落雪API'
  const deeplink = buildCcSwitchImportDeeplink({
    baseUrl,
    platform,
    clientType,
    providerName,
    apiKey: row.key,
    usageScript
  })

  try {
    window.open(deeplink, '_self')

    // Check if the protocol handler worked by detecting if we're still focused
    setTimeout(() => {
      if (document.hasFocus()) {
        // Still focused means the protocol handler likely failed
        appStore.showError(t('keys.ccSwitchNotInstalled'))
      }
    }, 100)
  } catch (error) {
    appStore.showError(t('keys.ccSwitchNotInstalled'))
  }
}

const handleCcsClientSelect = (clientType: CcSwitchClientType) => {
  if (pendingCcsRow.value) {
    executeCcsImport(pendingCcsRow.value, clientType)
  }
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

const closeCcsClientSelect = () => {
  showCcsClientSelect.value = false
  pendingCcsRow.value = null
}

onMounted(() => {
  loadApiKeys()
  loadGroups()
  loadUserGroupRates()
  loadPublicSettings()
  document.addEventListener('click', closeGroupSelector)
  document.addEventListener('keydown', handleEscapeKey)
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  abortController?.abort()
  document.removeEventListener('click', closeGroupSelector)
  document.removeEventListener('keydown', handleEscapeKey)
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
.keys-workspace {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  color: var(--workspace-text);
  background: var(--workspace-canvas);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
  -webkit-font-smoothing: antialiased;
}

.keys-mobile-header,
.keys-desktop-header,
.keys-filter-toolbar {
  flex: 0 0 auto;
  border-bottom: 1px solid var(--workspace-border);
  background: var(--workspace-card-surface);
}

.keys-mobile-header {
  display: flex;
  height: 56px;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
}

.keys-mobile-title,
.keys-mobile-actions,
.keys-desktop-title-wrap,
.keys-desktop-actions {
  display: flex;
  align-items: center;
}

.keys-mobile-title {
  gap: 8px;
  min-width: 0;
  color: var(--workspace-text);
  font-size: var(--workspace-type-brand-size);
  font-weight: var(--workspace-type-brand-weight);
  line-height: 24px;
}

.keys-mobile-title > :first-child {
  color: #7c3aed;
}

.keys-mobile-actions {
  gap: 8px;
}

.keys-mobile-refresh,
.keys-mobile-create,
.keys-desktop-refresh,
.keys-desktop-create,
.keys-detail-settings-button,
.keys-density-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  outline: none;
  transition: all 300ms cubic-bezier(0.4, 0, 0.2, 1);
}

.keys-mobile-refresh {
  width: 40px;
  height: 40px;
  color: var(--workspace-text-muted);
  border-radius: 12px;
}

.keys-mobile-refresh :deep(svg) {
  width: 18px;
  height: 18px;
}

.keys-mobile-create {
  height: 36px;
  gap: 6px;
  padding: 0 16px;
  border-radius: 12px;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 10px 15px -3px rgb(124 58 237 / 0.3), 0 4px 6px -4px rgb(124 58 237 / 0.3);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-mobile-create :deep(svg) {
  width: 12px;
  height: 12px;
}

.keys-mobile-refresh:active,
.keys-mobile-create:active {
  transform: scale(0.95);
}

.keys-desktop-header {
  min-height: 89px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 32px;
}

.keys-desktop-title-wrap {
  min-width: 0;
  gap: 16px;
}

.keys-desktop-title-icon {
  display: grid;
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  place-items: center;
  border-radius: 16px;
  color: #7c3aed;
  background: #ede9fe;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.05);
}

.keys-desktop-title-copy {
  min-width: 0;
}

.keys-desktop-title-copy h1 {
  margin: 0;
  color: var(--workspace-text);
  font-size: var(--workspace-type-page-title-size);
  font-weight: var(--workspace-type-page-title-weight);
  line-height: 28px;
}

.keys-desktop-title-copy p {
  overflow: hidden;
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 16px;
  letter-spacing: 0;
  text-overflow: ellipsis;
  text-transform: uppercase;
  white-space: nowrap;
}

.keys-desktop-actions {
  gap: 12px;
}

.keys-desktop-refresh {
  width: 48px;
  height: 48px;
  flex: 0 0 48px;
  border-radius: 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
}

.keys-desktop-refresh:hover {
  background: var(--workspace-hover);
}

.keys-desktop-create {
  height: 48px;
  gap: 8px;
  padding: 0 24px;
  border-radius: 12px;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 10px 15px -3px rgb(124 58 237 / 0.3), 0 4px 6px -4px rgb(124 58 237 / 0.3);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-desktop-create:hover {
  transform: translateY(-2px);
}

.keys-desktop-create:active {
  transform: scale(0.95);
}

.keys-filter-toolbar {
  padding: 16px;
}

.keys-filter-grid {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
}

.key-filter-search,
.key-secondary-filter {
  min-width: 0;
}

.key-filter-search {
  margin-bottom: 4px;
}

.key-filter-search :deep(.input),
.key-secondary-filter :deep(.select-trigger) {
  width: 100%;
  min-height: 44px;
  border: 0;
  border-radius: 12px;
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
  box-shadow: none;
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.key-filter-search :deep(.input) {
  height: 44px;
  padding-left: 48px;
}

.key-filter-search :deep(> div > div) {
  padding-left: 16px;
}

.key-filter-search :deep(> div > div svg) {
  width: 16px;
  height: 16px;
}

.key-secondary-filter :deep(.select-trigger) {
  height: 44px;
  padding: 0 16px;
}

.key-secondary-filter :deep(.select-icon) {
  display: none;
}

.key-filter-search :deep(.input:focus),
.key-secondary-filter :deep(.select-trigger:focus-visible),
.key-secondary-filter :deep(.select-trigger-open) {
  border-color: transparent;
  outline: none;
  box-shadow: 0 0 0 2px rgb(139 92 246 / 0.2);
}

.keys-filter-actions {
  display: flex;
  min-width: 0;
  height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 4px;
}

.keys-detail-settings-button,
.keys-density-button {
  height: 40px;
  border-radius: 8px;
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-detail-settings-button {
  gap: 8px;
  padding: 0 12px;
  color: #6d28d9;
  background: #f5f3ff;
  font-size: var(--workspace-type-navigation-size);
}

.keys-detail-settings-button :deep(svg) {
  width: 18px;
  height: 18px;
}

.keys-settings-icon--desktop {
  display: none;
}

.keys-density-button {
  padding: 0 12px;
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text-muted);
  background: var(--workspace-surface-subtle);
  font-size: var(--workspace-type-navigation-size);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-detail-settings-button:focus-visible,
.keys-density-button:focus-visible,
.keys-mobile-refresh:focus-visible,
.keys-mobile-create:focus-visible,
.keys-desktop-refresh:focus-visible,
.keys-desktop-create:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.keys-detail-menu {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 50;
  width: 224px;
  max-height: 320px;
  overflow-y: auto;
  margin-top: 8px;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 8px 0;
  background: var(--workspace-popup-surface);
  box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1);
}

.keys-detail-menu-item {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  color: var(--workspace-text-secondary);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-align: left;
}

.keys-detail-menu-item:hover {
  background: var(--workspace-hover);
}

.keys-content {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  overflow: hidden;
  background: var(--workspace-canvas);
}

.keys-master-pane {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow: hidden;
  background: var(--workspace-canvas);
}

.keys-detail-pane {
  width: 40%;
  min-width: 0;
  flex: 0 0 40%;
  overflow: hidden;
  border-left: 1px solid var(--workspace-border);
  background: var(--workspace-canvas);
}

.keys-mobile-list {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  background: var(--workspace-canvas);
  scrollbar-color: var(--workspace-border-strong) transparent;
  scrollbar-width: thin;
}

.keys-mobile-list::-webkit-scrollbar {
  width: 5px;
}

.keys-mobile-list::-webkit-scrollbar-thumb {
  border-radius: 10px;
  background: var(--workspace-border-strong);
}

.keys-mobile-list > * + * {
  margin-top: 16px;
}

.keys-mobile-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0 32px;
}

.keys-mobile-pagination p {
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-mobile-pagination > div {
  display: flex;
  gap: 8px;
}

.keys-mobile-pagination button {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

.keys-mobile-pagination button:disabled {
  opacity: 0.4;
}

.keys-desktop-pagination {
  flex: 0 0 auto;
  overflow: hidden;
}

.keys-desktop-pagination :deep(.pagination) {
  min-height: 73px;
  border-top: 1px solid var(--workspace-border);
  padding: 16px 32px;
  background: var(--workspace-card-surface);
}

.keys-desktop-pagination :deep(.pagination-summary) {
  color: var(--workspace-text-muted);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  letter-spacing: 0;
  text-transform: uppercase;
}

.keys-desktop-pagination :deep(.pagination-nav) {
  display: flex;
  gap: 8px;
  border-radius: 0;
  box-shadow: none;
}

.keys-desktop-pagination :deep(.pagination-nav .pagination-button) {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  margin: 0;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 0;
  color: var(--workspace-text-muted);
  background: var(--workspace-card-surface);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.keys-desktop-pagination :deep(.pagination-nav .pagination-button--active) {
  border-color: #7c3aed;
  color: #fff;
  background: #7c3aed;
  box-shadow: 0 4px 6px -1px rgb(124 58 237 / 0.2);
}

.keys-empty-state {
  display: flex;
  min-height: 0;
  flex: 1 1 0;
  align-items: center;
  justify-content: center;
  overflow-y: auto;
  padding: 24px;
  background: var(--workspace-canvas);
}

.keys-loading-master {
  background: var(--workspace-surface-subtle);
}

.keys-loading-mobile {
  background: var(--workspace-canvas);
}

.keys-loading-detail {
  border-color: var(--workspace-border);
  background: var(--workspace-surface-subtle);
}

.keys-loading-card,
.keys-client-option {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
}

.keys-switch-track,
.keys-form-progress-track {
  background: var(--workspace-border-strong);
}

.keys-switch-thumb {
  background: var(--workspace-light-surface);
}

.keys-form-usage-surface,
.keys-neutral-choice {
  background: var(--workspace-surface-subtle);
}

.keys-neutral-choice {
  color: var(--workspace-text-secondary);
}

.keys-neutral-choice:hover {
  background: var(--workspace-hover);
}

.keys-text-primary {
  color: var(--workspace-text);
}

.keys-text-secondary {
  color: var(--workspace-text-secondary);
}

.keys-text-muted {
  color: var(--workspace-text-muted);
}

.keys-group-selector {
  border: 1px solid var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-popup-surface);
}

.keys-group-selector-header,
.keys-group-option {
  border-color: var(--workspace-border);
}

.keys-group-search {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
}

.keys-group-search::placeholder {
  color: var(--workspace-text-muted);
}

.keys-group-option--selected {
  background: var(--workspace-selected);
}

.keys-group-option--idle:hover {
  background: var(--workspace-hover);
}

:global(.app-layout--snow-shell:has(.keys-workspace)) {
  --app-shell-top-offset: 0px;
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
  position: relative;
  z-index: 0;
  overflow-x: hidden;
  overflow-y: auto;
  background: var(--workspace-canvas) !important;
  box-shadow: none;
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat .app-main-content) {
  padding: 0;
}

:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat .app-main-content > div) {
  max-width: none;
}

:global(.dark) .keys-workspace,
:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-content,
:global(.dark) .keys-master-pane {
  color: var(--workspace-text);
  background: var(--workspace-canvas);
}

:global(.dark .app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
  background: var(--workspace-canvas) !important;
}

:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-detail-pane {
  border-color: var(--workspace-border);
}

:global(.dark) .keys-mobile-title,
:global(.dark) .keys-desktop-title-copy h1 {
  color: var(--workspace-text);
}

:global(.dark) .keys-desktop-title-icon {
  color: #a78bfa;
  background: rgb(76 29 149 / 0.3);
}

:global(.dark) .keys-desktop-refresh,
:global(.dark) .keys-filter-toolbar,
:global(.dark) .keys-mobile-header,
:global(.dark) .keys-desktop-header {
  background: var(--workspace-card-surface);
}

:global(.dark) .key-filter-search :deep(.input),
:global(.dark) .key-secondary-filter :deep(.select-trigger),
:global(.dark) .keys-density-button {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
}

:global(.dark) .keys-detail-settings-button {
  color: #a78bfa;
  background: rgb(76 29 149 / 0.1);
}

:global(.dark) .keys-detail-menu {
  border-color: var(--workspace-border);
  background: var(--workspace-popup-surface);
}

:global(.dark) .keys-detail-menu-item {
  color: var(--workspace-text-secondary);
}

:global(.dark) .keys-detail-menu-item:hover {
  background: var(--workspace-hover);
}

:global(.dark) .keys-detail-pane {
  background: var(--workspace-canvas);
}

:global(.dark) .keys-mobile-list,
:global(.dark) .keys-empty-state {
  background: var(--workspace-canvas);
}

:global(.dark) .keys-mobile-pagination button,
:global(.dark) .keys-desktop-pagination :deep(.pagination),
:global(.dark) .keys-desktop-pagination :deep(.pagination-nav .pagination-button) {
  border-color: var(--workspace-border);
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

@media (min-width: 768px) {
  .keys-settings-icon--mobile {
    display: none;
  }

  .keys-settings-icon--desktop {
    display: inline-flex;
  }

  .keys-filter-toolbar {
    padding: 16px 32px;
  }

  .keys-filter-grid {
    flex-direction: row;
    align-items: center;
    gap: 8px;
  }

  .key-filter-search {
    max-width: 512px;
    flex: 1 1 260px;
    margin: 0 8px 0 0;
  }

  .key-secondary-filter {
    width: 110px;
    flex: 0 0 110px;
  }

  .key-sort-filter {
    width: 152px;
    flex-basis: 152px;
  }

  .key-filter-search :deep(.input),
  .key-secondary-filter :deep(.select-trigger) {
    border: 1px solid var(--workspace-border);
  }

  .key-secondary-filter :deep(.select-trigger) {
    gap: 8px;
    padding-right: 12px;
    padding-left: 12px;
    background: var(--workspace-card-surface);
  }

  .key-secondary-filter :deep(.select-icon) {
    display: inline-flex;
  }

  .key-secondary-filter :deep(.select-icon svg) {
    width: 16px;
    height: 16px;
  }

  .keys-mobile-header {
    display: none;
  }

  .keys-filter-actions {
    width: 44px;
    height: 44px;
    flex: 0 0 44px;
    margin: 0;
  }

  .keys-detail-settings-button {
    width: 44px;
    height: 44px;
    padding: 0;
    border: 1px solid var(--workspace-border);
    border-radius: 12px;
    color: var(--workspace-text-secondary);
    background: var(--workspace-card-surface);
  }

  .keys-detail-settings-button:hover {
    background: var(--workspace-hover);
  }

  .keys-detail-settings-button span,
  .keys-density-button {
    display: none;
  }

  .keys-detail-menu {
    right: 0;
    left: auto;
  }

  .key-empty-create {
    display: none;
  }

  :global(.dark) .key-secondary-filter :deep(.select-trigger),
  :global(.dark) .keys-detail-settings-button {
    border-color: var(--workspace-border);
    color: var(--workspace-text-secondary);
    background: var(--workspace-surface-subtle);
  }
}

@media (max-width: 1023px) {
  :global(.app-layout--snow-shell:has(.keys-workspace) [data-testid='app-mobile-header']) {
    display: none !important;
  }

  :global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
    height: 100dvh;
  }
}

@media (prefers-reduced-motion: reduce) {
  .keys-mobile-refresh,
  .keys-mobile-create,
  .keys-desktop-refresh,
  .keys-desktop-create,
  .keys-detail-settings-button,
  .keys-density-button {
    transition: none;
  }
}

/* --------------------------------------------------------------------------
 * API keys v31 workspace
 *
 * The page intentionally uses one calm content column. The shared AppLayout
 * and sidebar remain untouched; these rules only shape the slotted right-hand
 * workspace shown above.
 * -------------------------------------------------------------------------- */
.keys-workspace {
  display: block;
  width: 100%;
  height: auto;
  min-height: calc(100dvh - 24px);
  overflow: visible;
  padding: 0 0 32px;
  color: var(--workspace-text);
  background: var(--workspace-canvas);
}

.keys-page-header {
  width: min(calc(100% - 64px), 1280px);
  margin: 0 auto;
  padding: 52px 0 30px;
}

.keys-page-header h1 {
  margin: 0;
  color: var(--workspace-text);
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.03em;
  line-height: 1.15;
}

.keys-endpoint-card {
  display: grid;
  width: min(calc(100% - 64px), 1280px);
  min-height: 82px;
  grid-template-columns: minmax(150px, 180px) minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  margin: 0 auto;
  border: 1px solid var(--workspace-border);
  border-radius: 18px;
  padding: 18px 20px;
  background: var(--workspace-card-surface);
  box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
}

.keys-endpoint-label h2 {
  margin: 0;
  color: var(--workspace-text);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.keys-endpoint-field {
  display: flex;
  min-width: 0;
  min-height: 46px;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-pill);
  padding: 0 6px 0 16px;
  background: var(--workspace-surface-subtle);
}

.keys-endpoint-field code {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--workspace-text-secondary);
  font-family: var(--workspace-font-mono);
  font-size: 13px;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.keys-icon-button {
  display: inline-grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  border: 0;
  border-radius: 50%;
  color: var(--workspace-text-muted);
  background: transparent;
  outline: none;
  transition: color 150ms ease, background-color 150ms ease;
}

.keys-icon-button:hover,
.keys-icon-button:focus-visible {
  color: var(--workspace-work-accent);
  background: var(--workspace-hover);
}

.keys-icon-button:focus-visible,
.keys-fast-mode:focus-visible,
.keys-create-button:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.keys-icon-button:disabled {
  cursor: wait;
  opacity: 0.72;
}

.keys-endpoint-test--ready {
  color: var(--workspace-work-success);
}

.keys-endpoint-test--failed {
  color: var(--workspace-text-secondary);
}

.keys-endpoint-status {
  display: inline-flex;
  min-width: max-content;
  align-items: center;
  margin-left: 2px;
  border-left: 1px solid var(--workspace-border);
  padding-left: 10px;
  color: var(--workspace-text-muted);
  font-size: 11px;
  line-height: 18px;
  white-space: nowrap;
}

.keys-endpoint-status.is-testing {
  color: var(--workspace-work-accent);
}

.keys-endpoint-status.is-ready {
  color: var(--workspace-work-success);
}

.keys-endpoint-status.is-failed {
  color: var(--workspace-text-secondary);
}

.keys-endpoint-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.keys-fast-mode,
.keys-create-button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  border-radius: var(--workspace-radius-pill);
  outline: none;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
}

.keys-fast-mode {
  gap: 9px;
  border: 1px solid var(--workspace-border);
  padding: 0 12px 0 15px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
}

.keys-fast-mode:hover {
  border-color: var(--workspace-border-strong);
  background: var(--workspace-hover);
}

.keys-fast-mode[aria-checked="true"] {
  border-color: var(--workspace-work-accent-border);
  color: var(--workspace-work-accent);
  background: color-mix(in srgb, var(--workspace-work-accent-soft) 35%, var(--workspace-card-surface));
}

.keys-fast-mode-track {
  position: relative;
  display: inline-flex;
  width: 34px;
  height: 20px;
  flex: 0 0 34px;
  border-radius: var(--workspace-radius-pill);
  background: var(--workspace-border-strong);
  transition: background-color 150ms ease;
}

.keys-fast-mode-track > span {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--workspace-card-surface);
  box-shadow: 0 1px 2px rgb(15 23 42 / 18%);
  transition: transform 150ms ease;
}

.keys-fast-mode[aria-checked="true"] .keys-fast-mode-track {
  background: var(--workspace-work-accent);
}

.keys-fast-mode[aria-checked="true"] .keys-fast-mode-track > span {
  transform: translateX(14px);
}

.keys-create-button {
  gap: 7px;
  border: 1px solid var(--workspace-work-accent);
  padding: 0 18px;
  color: var(--workspace-work-on-accent);
  background: var(--workspace-work-accent);
}

.keys-create-button:hover {
  border-color: var(--workspace-work-accent-hover);
  background: var(--workspace-work-accent-hover);
}

.keys-list-region {
  width: min(calc(100% - 64px), 1280px);
  min-width: 0;
  margin: 28px auto 32px;
}

.keys-desktop-table {
  display: block !important;
  width: 100%;
  min-width: 0;
  height: auto !important;
}

.keys-desktop-table :deep(.key-workspace-list) {
  height: auto;
  max-height: none;
  overflow: auto;
}

.keys-mobile-list {
  display: none !important;
  height: auto;
  min-width: 0;
  overflow: visible;
  padding: 0;
}

.keys-loading-state {
  width: min(calc(100% - 64px), 1280px);
  margin: 28px auto 32px;
}

.keys-loading-table {
  overflow: hidden;
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  background: var(--workspace-card-surface);
}

.keys-loading-row {
  height: 64px;
  border-bottom: 1px solid var(--workspace-border);
  background: linear-gradient(90deg, var(--workspace-surface-subtle), var(--workspace-card-surface), var(--workspace-surface-subtle));
  background-size: 240% 100%;
  animation: keys-loading-shimmer 1.4s ease-in-out infinite;
}

.keys-loading-row:last-child {
  border-bottom: 0;
}

.keys-loading-cards {
  display: none;
}

.keys-empty-state {
  min-height: 300px;
  padding: 48px 24px;
}

.keys-error-state {
  display: flex;
  width: min(calc(100% - 64px), 1280px);
  min-height: 260px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 12px;
  margin: 28px auto 32px;
  padding: 48px 24px;
  color: var(--workspace-text-muted);
  text-align: center;
}

.keys-error-state p {
  margin: 0;
  font-size: 13px;
}

.keys-error-retry {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  gap: 7px;
  border: 1px solid var(--workspace-border);
  border-radius: var(--workspace-radius-pill);
  padding: 0 13px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-card-surface);
  font-size: 12px;
  font-weight: 500;
  transition: border-color 150ms ease, color 150ms ease, background-color 150ms ease;
}

.keys-error-retry:hover,
.keys-error-retry:focus-visible {
  border-color: var(--workspace-border-strong);
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.keys-error-retry:focus-visible {
  outline: 2px solid var(--workspace-work-accent);
  outline-offset: 2px;
}

.keys-group-selector {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-popup-surface);
}

.keys-group-selector-header,
.keys-group-option {
  border-color: var(--workspace-border);
}

.keys-group-search {
  border-color: var(--workspace-border);
  color: var(--workspace-text);
  background: var(--workspace-surface-subtle);
}

@keyframes keys-loading-shimmer {
  0% { background-position: 100% 0; }
  100% { background-position: -100% 0; }
}

@media (max-width: 1100px) and (min-width: 961px) {
  .keys-page-header,
  .keys-endpoint-card,
  .keys-list-region,
  .keys-loading-state,
  .keys-error-state {
    width: calc(100% - 48px);
  }

  .keys-endpoint-card {
    grid-template-columns: minmax(130px, 160px) minmax(0, 1fr) auto;
    gap: 14px;
  }
}

@media (max-width: 960px) {
  .keys-workspace {
    min-height: 100dvh;
    padding-bottom: 24px;
  }

  .keys-page-header {
    width: auto;
    margin: 0;
    padding: 30px 16px 24px;
  }

  .keys-page-header h1 {
    font-size: 28px;
  }

  .keys-endpoint-card {
    display: block;
    width: calc(100% - 32px);
    min-height: 0;
    margin: 0 16px;
    border-radius: 16px;
    padding: 16px;
  }

  .keys-endpoint-label {
    margin-bottom: 12px;
  }

  .keys-endpoint-label h2 {
    font-size: 19px;
  }

  .keys-endpoint-field {
    width: 100%;
  }

  .keys-endpoint-actions {
    width: 100%;
    justify-content: flex-start;
    margin-top: 12px;
  }

  .keys-fast-mode,
  .keys-create-button {
    flex: 1 1 auto;
  }

  .keys-endpoint-status {
    order: 3;
    width: 100%;
    min-height: 18px;
    margin: 2px 0 0;
    border-left: 0;
    padding-left: 0;
  }

  .keys-list-region,
  .keys-loading-state,
  .keys-error-state {
    width: calc(100% - 32px);
    margin: 20px 16px 24px;
  }

  .keys-desktop-table {
    display: none !important;
  }

  .keys-mobile-list {
    display: flex !important;
    flex-direction: column;
    gap: 10px;
  }

  .keys-mobile-list > * + * {
    margin-top: 0;
  }

  .keys-loading-table {
    display: none;
  }

  .keys-loading-cards {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .keys-loading-card {
    height: 226px;
    border: 1px solid var(--workspace-border);
    border-radius: 12px;
    background: linear-gradient(90deg, var(--workspace-surface-subtle), var(--workspace-card-surface), var(--workspace-surface-subtle));
    background-size: 240% 100%;
    animation: keys-loading-shimmer 1.4s ease-in-out infinite;
  }
}

@media (prefers-reduced-motion: reduce) {
  .keys-icon-button,
  .keys-fast-mode,
  .keys-fast-mode-track,
  .keys-fast-mode-track > span,
  .keys-create-button,
  .keys-error-retry,
  .keys-loading-row,
  .keys-loading-card {
    animation: none;
    transition: none;
  }
}

:global(.dark) .keys-endpoint-card,
:global(.dark) .keys-loading-table,
:global(.dark) .keys-loading-card {
  border-color: var(--workspace-border);
  background-color: var(--workspace-card-surface);
}

:global(.dark) .keys-endpoint-field {
  background: var(--workspace-surface-subtle);
}

:global(.dark) .keys-fast-mode[aria-checked="true"] {
  background: color-mix(in srgb, var(--workspace-work-accent-soft) 28%, var(--workspace-card-surface));
}

/* The chat shell normally clips its content to keep conversation panes fixed.
   This page is a document-style, pagination-free list, so let the right pane
   scroll vertically as the number of keys grows while leaving the sidebar
   untouched. */
:global(.app-layout--snow-shell:has(.keys-workspace) .app-main-shell.app-layout--chat) {
  overflow-x: hidden;
  overflow-y: auto;
  box-shadow: none;
}
</style>
