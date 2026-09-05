<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.createAccount')"
    width="wide"
    @close="handleClose"
  >
    <!-- Step Indicator for OAuth accounts -->
    <div v-if="isOAuthFlow" class="mb-6 flex items-center justify-center">
      <div class="flex items-center space-x-4">
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 1 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            1
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            t('admin.accounts.oauth.authMethod')
          }}</span>
        </div>
        <div class="h-0.5 w-8 bg-gray-300 dark:bg-dark-600" />
        <div class="flex items-center">
          <div
            :class="[
              'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
              step >= 2 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
            ]"
          >
            2
          </div>
          <span class="ml-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{
            oauthStepTitle
          }}</span>
        </div>
      </div>
    </div>

    <!-- Step 1: Basic Info -->
    <form
      v-if="step === 1"
      id="create-account-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('admin.accounts.accountName') }}</label>
        <input
          v-model="form.name"
          type="text"
          :required="!isGrokSSOInputMethod"
          class="input"
          :placeholder="t('admin.accounts.enterAccountName')"
          data-tour="account-form-name"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.accounts.notes') }}</label>
        <textarea
          v-model="form.notes"
          rows="3"
          class="input"
          :placeholder="t('admin.accounts.notesPlaceholder')"
        ></textarea>
        <p class="input-hint">{{ t('admin.accounts.notesHint') }}</p>
      </div>

      <AccountPlatformSelector :platform="form.platform" @select="handlePlatformSelect" />

      <AnthropicAccountTypePanel
        v-if="form.platform === 'anthropic'"
        v-model="accountCategory"
      />
      <OpenAIAccountTypePanel
        v-if="form.platform === 'openai'"
        v-model="accountCategory"
      />
      <GrokAccountTypePanel
        v-if="form.platform === 'grok'"
        v-model="accountCategory"
      />

      <GeminiAccountTypePanel
        v-if="form.platform === 'gemini'"
        v-model="accountCategory"
        :oauth-type="geminiOAuthType"
        :ai-studio-o-auth-enabled="geminiAIStudioOAuthEnabled"
        v-model:show-advanced="showAdvancedOAuth"
        v-model:google-one-tier="geminiTierGoogleOne"
        v-model:gcp-tier="geminiTierGcp"
        v-model:ai-studio-tier="geminiTierAIStudio"
        :help-links="geminiHelpLinks"
        @select-oauth-type="handleSelectGeminiOAuthType"
        @open-help="showGeminiHelpDialog = true"
      />

      <AntigravityAccountTypePanel
        v-if="form.platform === 'antigravity'"
        v-model:account-kind="antigravityAccountType"
        v-model:project-id="antigravityProjectId"
        v-model:base-url="upstreamBaseUrl"
        v-model:api-key="upstreamApiKey"
      />

      <VertexServiceAccountPanel
        v-if="(form.platform === 'gemini' || form.platform === 'anthropic') && accountCategory === 'service_account'"
        :project-id="vertexProjectId"
        :client-email="vertexClientEmail"
        v-model:location="vertexLocation"
        @file-selected="handleVertexServiceAccountFile"
      />

      <AntigravityModelMappingPanel
        v-if="form.platform === 'antigravity'"
        :mappings="antigravityModelMappings"
        :presets="antigravityPresetMappings"
        @add="addAntigravityModelMapping"
        @remove="removeAntigravityModelMapping"
        @update-mapping="updateAntigravityModelMapping"
        @add-preset="addAntigravityPresetMapping"
      />

      <!-- Add Method (only for Anthropic OAuth-based type) -->
      <div v-if="form.platform === 'anthropic' && isOAuthFlow">
        <label class="input-label">{{ t('admin.accounts.addMethod') }}</label>
        <div class="mt-2 flex gap-4">
          <label class="flex cursor-pointer items-center">
            <input
              v-model="addMethod"
              type="radio"
              value="oauth"
              class="mr-2 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.types.oauth') }}</span>
          </label>
          <label class="flex cursor-pointer items-center">
            <input
              v-model="addMethod"
              type="radio"
              value="setup-token"
              class="mr-2 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{
              t('admin.accounts.setupTokenLongLived')
            }}</span>
          </label>
        </div>
      </div>

      <PlatformCredentialSettingsPanel
        :platform="form.platform"
        :account-type="form.type"
        :account-category="accountCategory"
        :openai-model-restriction-disabled="isOpenAIModelRestrictionDisabled"
        v-model:api-key-base-url="apiKeyBaseUrl"
        v-model:api-key-value="apiKeyValue"
        v-model:gemini-tier-ai-studio="geminiTierAIStudio"
        v-model:model-restriction-mode="modelRestrictionMode"
        v-model:allowed-models="allowedModels"
        v-model:model-mappings="modelMappings"
        v-model:pool-mode-enabled="poolModeEnabled"
        v-model:pool-mode-retry-count="poolModeRetryCount"
        v-model:pool-mode-retry-status-codes-input="poolModeRetryStatusCodesInput"
        v-model:custom-error-codes-enabled="customErrorCodesEnabled"
        v-model:selected-error-codes="selectedErrorCodes"
        v-model:custom-error-code-input="customErrorCodeInput"
        v-model:header-override-enabled="headerOverrideEnabled"
        v-model:header-override-rows="headerOverrideRows"
        v-model:account-mode="accountMode"
        v-model:api-protocol="apiProtocol"
        v-model:zhipu-organization="zhipuOrganization"
        v-model:zhipu-project="zhipuProject"
        v-model:bedrock-auth-mode="bedrockAuthMode"
        v-model:bedrock-access-key-id="bedrockAccessKeyId"
        v-model:bedrock-secret-access-key="bedrockSecretAccessKey"
        v-model:bedrock-session-token="bedrockSessionToken"
        v-model:bedrock-api-key-value="bedrockApiKeyValue"
        v-model:bedrock-region="bedrockRegion"
        v-model:bedrock-force-global="bedrockForceGlobal"
      />
      <!-- 配额控制 (Anthropic apikey/bedrock: 配额限制 + 亲和) -->
      <div
        v-if="form.platform === 'anthropic' && (form.type === 'apikey' || form.type === 'bedrock')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.hint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          :quotaNotifyGlobalEnabled="quotaNotifyGlobalEnabled"
          :quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled"
          :quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold"
          :quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType"
          :quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled"
          :quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold"
          :quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType"
          :quotaNotifyTotalEnabled="quotaNotifyState.total.enabled"
          :quotaNotifyTotalThreshold="quotaNotifyState.total.threshold"
          :quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType"
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
        />
      </div>

      <!-- 配额控制 (非 Anthropic apikey/bedrock) -->
      <div
        v-else-if="form.type === 'apikey' || form.type === 'bedrock'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaLimitHint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          :quotaNotifyGlobalEnabled="quotaNotifyGlobalEnabled"
          :quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled"
          :quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold"
          :quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType"
          :quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled"
          :quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold"
          :quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType"
          :quotaNotifyTotalEnabled="quotaNotifyState.total.enabled"
          :quotaNotifyTotalThreshold="quotaNotifyState.total.threshold"
          :quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType"
          :dailyResetMode="editDailyResetMode"
          :dailyResetHour="editDailyResetHour"
          :weeklyResetMode="editWeeklyResetMode"
          :weeklyResetDay="editWeeklyResetDay"
          :weeklyResetHour="editWeeklyResetHour"
          :resetTimezone="editResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:quotaNotifyDailyEnabled="quotaNotifyState.daily.enabled = $event"
          @update:quotaNotifyDailyThreshold="quotaNotifyState.daily.threshold = $event"
          @update:quotaNotifyDailyThresholdType="quotaNotifyState.daily.thresholdType = $event"
          @update:quotaNotifyWeeklyEnabled="quotaNotifyState.weekly.enabled = $event"
          @update:quotaNotifyWeeklyThreshold="quotaNotifyState.weekly.threshold = $event"
          @update:quotaNotifyWeeklyThresholdType="quotaNotifyState.weekly.thresholdType = $event"
          @update:quotaNotifyTotalEnabled="quotaNotifyState.total.enabled = $event"
          @update:quotaNotifyTotalThreshold="quotaNotifyState.total.threshold = $event"
          @update:quotaNotifyTotalThresholdType="quotaNotifyState.total.thresholdType = $event"
          @update:dailyResetMode="editDailyResetMode = $event"
          @update:dailyResetHour="editDailyResetHour = $event"
          @update:weeklyResetMode="editWeeklyResetMode = $event"
          @update:weeklyResetDay="editWeeklyResetDay = $event"
          @update:weeklyResetHour="editWeeklyResetHour = $event"
          @update:resetTimezone="editResetTimezone = $event"
        />
      </div>

      <GrokOAuthOptionsPanel
        v-if="form.platform === 'grok' && isOAuthFlow"
        v-model:base-url-enabled="grokOAuthCustomBaseUrlEnabled"
        v-model:base-url="grokOAuthBaseUrl"
        v-model:header-enabled="headerOverrideEnabled"
        v-model:rows="headerOverrideRows"
      />

      <!-- OpenAI OAuth Model Mapping (OAuth 类型没有 apikey 容器，需要独立的模型映射区域) -->
      <div
        v-if="(form.platform === 'openai' || form.platform === 'grok') && isOAuthFlow"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

        <div
          v-if="isOpenAIModelRestrictionDisabled"
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
              {{ t('admin.accounts.modelMapping') }}
            </button>
          </div>

          <!-- Whitelist Mode -->
          <div v-if="modelRestrictionMode === 'whitelist'">
            <ModelWhitelistSelector v-model="allowedModels" :platform="form.platform" :sync-credentials="syncPreviewCredentials" />
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
                {{ t('admin.accounts.mapRequestModels') }}
              </p>
            </div>

            <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
              <div
                v-for="(mapping, index) in modelMappings"
                :key="'oauth-' + getModelMappingKey(mapping)"
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
              + {{ t('admin.accounts.addMapping') }}
            </button>

            <!-- Quick Add Buttons -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="preset in presetMappings"
                :key="'oauth-' + preset.label"
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

      <TempUnschedulablePanel
        v-model:enabled="tempUnschedEnabled"
        v-model:rules="tempUnschedRules"
      />
      <AccountRuntimeSettingsPanel
        :platform="form.platform"
        :account-category="accountCategory"
        :proxies="proxies"
        :umq-mode-options="umqModeOptions"
        :tls-fingerprint-profiles="tlsFingerprintProfiles"
        v-model:intercept-warmup-requests="interceptWarmupRequests"
        v-model:window-cost-enabled="windowCostEnabled"
        v-model:window-cost-limit="windowCostLimit"
        v-model:window-cost-sticky-reserve="windowCostStickyReserve"
        v-model:session-limit-enabled="sessionLimitEnabled"
        v-model:max-sessions="maxSessions"
        v-model:session-idle-timeout="sessionIdleTimeout"
        v-model:rpm-limit-enabled="rpmLimitEnabled"
        v-model:base-rpm="baseRpm"
        v-model:rpm-strategy="rpmStrategy"
        v-model:rpm-sticky-buffer="rpmStickyBuffer"
        v-model:user-msg-queue-mode="userMsgQueueMode"
        v-model:tls-fingerprint-enabled="tlsFingerprintEnabled"
        v-model:tls-fingerprint-profile-id="tlsFingerprintProfileId"
        v-model:session-id-masking-enabled="sessionIdMaskingEnabled"
        v-model:cache-t-t-l-override-enabled="cacheTTLOverrideEnabled"
        v-model:cache-t-t-l-override-target="cacheTTLOverrideTarget"
        v-model:custom-base-url-enabled="customBaseUrlEnabled"
        v-model:custom-base-url="customBaseUrl"
        v-model:proxy-id="form.proxy_id"
        v-model:concurrency="form.concurrency"
        v-model:load-factor="form.load_factor"
        v-model:priority="form.priority"
        v-model:rate-multiplier="form.rate_multiplier"
      />
      <AccountBehaviorSettingsPanel
        :platform="form.platform"
        :account-category="accountCategory"
        :groups="groups"
        :simple-mode="authStore.isSimpleMode"
        :web-search-global-enabled="webSearchGlobalEnabled"
        v-model:expires-at-input="expiresAtInput"
        v-model:openai-passthrough-enabled="openaiPassthroughEnabled"
        v-model:openai-web-socket-mode="openaiResponsesWebSocketV2Mode"
        v-model:anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        v-model:anthropic-api-key-auth-scheme="anthropicAPIKeyAuthScheme"
        v-model:web-search-emulation-mode="webSearchEmulationMode"
        v-model:long-context-billing-enabled="openAILongContextBillingEnabled"
        v-model:long-context-billing-touched="openAILongContextBillingTouched"
        v-model:codex-cli-only-enabled="codexCLIOnlyEnabled"
        v-model:codex-cli-only-app-server-enabled="codexCLIOnlyAppServerEnabled"
        v-model:openai-compact-mode="openAICompactMode"
        v-model:openai-compact-model-mappings="openAICompactModelMappings"
        v-model:openai-responses-mode="openAIResponsesMode"
        v-model:openai-endpoint-capabilities="openAIEndpointCapabilities"
        v-model:auto-pause-on-expired="autoPauseOnExpired"
        v-model:mixed-scheduling="mixedScheduling"
        v-model:allow-overages="allowOverages"
        v-model:group-ids="form.group_ids"
      />
    </form>

    <!-- Step 2: OAuth Authorization -->
    <div v-else class="space-y-5">
      <OAuthAuthorizationFlow
        ref="oauthFlowRef"
        :add-method="form.platform === 'anthropic' ? addMethod : 'oauth'"
        :auth-url="currentAuthUrl"
        :session-id="currentSessionId"
        :loading="currentOAuthLoading"
        :error="currentOAuthError"
        :show-help="form.platform === 'anthropic'"
        :show-proxy-warning="form.platform !== 'openai' && form.platform !== 'grok' && !!form.proxy_id"
        :allow-multiple="form.platform === 'anthropic'"
        :show-cookie-option="form.platform === 'anthropic'"
        :show-refresh-token-option="form.platform === 'openai' || form.platform === 'antigravity' || form.platform === 'grok'"
        :show-mobile-refresh-token-option="form.platform === 'openai'"
        :show-session-token-option="false"
        :show-access-token-option="false"
        :show-codex-session-import-option="form.platform === 'openai'"
        :show-agent-identity-option="form.platform === 'openai'"
        :show-codex-pat-option="form.platform === 'openai'"
        :show-sso-option="form.platform === 'grok'"
        :show-manual-option="true"
        :initial-input-method="'manual'"
        :platform="form.platform"
        :show-project-id="geminiOAuthType === 'code_assist'"
        @generate-url="handleGenerateUrl"
        @cookie-auth="handleCookieAuth"
        @validate-refresh-token="handleValidateRefreshToken"
        @validate-mobile-refresh-token="handleOpenAIValidateMobileRT"
        @validate-session-token="handleValidateSessionToken"
        @import-codex-session="handleOpenAIImportCodexSession"
        @import-codex-pat="handleOpenAIImportCodexPAT"
        @import-sso="handleGrokImportSSO"
      />

    </div>

    <template #footer>
      <div v-if="step === 1" class="flex justify-end gap-3">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="create-account-form"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="account-form-submit"
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
            isOAuthFlow
              ? t('common.next')
              : submitting
                ? t('admin.accounts.creating')
                : t('common.create')
          }}
        </button>
      </div>
      <div v-else class="flex justify-between gap-3">
        <button type="button" class="btn btn-secondary" @click="goBackToBasicInfo">
          {{ t('common.back') }}
        </button>
        <button
          v-if="isManualInputMethod"
          type="button"
          :disabled="!canExchangeCode"
          class="btn btn-primary"
          @click="handleExchangeCode"
        >
          <svg
            v-if="currentOAuthLoading"
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
            currentOAuthLoading
              ? t('admin.accounts.oauth.verifying')
              : t('admin.accounts.oauth.completeAuth')
          }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- Gemini Help Dialog -->
  <BaseDialog
    :show="showGeminiHelpDialog"
    :title="t('admin.accounts.gemini.helpDialog.title')"
    @close="showGeminiHelpDialog = false"
    max-width="max-w-3xl"
  >
    <div class="space-y-6">
      <!-- Setup Guide Section -->
      <div>
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.setupGuide.title') }}
        </h3>
        <div class="space-y-4">
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.gemini.setupGuide.checklistTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.accounts.gemini.setupGuide.checklistItems.usIp') }}</li>
              <li>{{ t('admin.accounts.gemini.setupGuide.checklistItems.age') }}</li>
            </ul>
          </div>
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.gemini.setupGuide.activationTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.accounts.gemini.setupGuide.activationItems.geminiWeb') }}</li>
              <li>{{ t('admin.accounts.gemini.setupGuide.activationItems.gcpProject') }}</li>
            </ul>
            <div class="mt-2 flex flex-wrap gap-2">
              <a
                href="https://policies.google.com/terms"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.accounts.gemini.setupGuide.links.countryCheck') }}
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://policies.google.com/country-association-form"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.accounts.gemini.setupGuide.links.countryChange') }}
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://gemini.google.com/gems/create?hl=en-US&pli=1"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.accounts.gemini.setupGuide.links.geminiWebActivation') }}
              </a>
              <span class="text-gray-400">·</span>
              <a
                href="https://console.cloud.google.com"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t('admin.accounts.gemini.setupGuide.links.gcpProject') }}
              </a>
            </div>
          </div>
        </div>
      </div>

      <!-- Quota Policy Section -->
      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.quotaPolicy.title') }}
        </h3>
        <p class="mb-4 text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.accounts.gemini.quotaPolicy.note') }}
        </p>
        <div class="overflow-x-auto">
          <table class="w-full text-xs">
            <thead class="bg-gray-50 dark:bg-dark-600">
              <tr>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.channel') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.account') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.limits') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.googleOne.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Free</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsFree') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Pro</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsPro') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Ultra</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsUltra') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.gcp.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Standard</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsStandard') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Enterprise</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.gcp.limitsEnterprise') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.channel') }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Free</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsFree') }}
                </td>
              </tr>
              <tr>
                <td class="px-3 py-2 text-gray-900 dark:text-white"></td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">Paid</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                  {{ t('admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsPaid') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-4 flex flex-wrap gap-3">
          <a
            :href="geminiQuotaDocs.codeAssist"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.accounts.gemini.quotaPolicy.docs.codeAssist') }}
          </a>
          <a
            :href="geminiQuotaDocs.aiStudio"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.accounts.gemini.quotaPolicy.docs.aiStudio') }}
          </a>
          <a
            :href="geminiQuotaDocs.vertex"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.accounts.gemini.quotaPolicy.docs.vertex') }}
          </a>
        </div>
      </div>

      <!-- API Key Links Section -->
      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.helpDialog.apiKeySection') }}
        </h3>
        <div class="flex flex-wrap gap-3">
          <a
            :href="geminiHelpLinks.apiKey"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.accounts.gemini.accountType.apiKeyLink') }}
          </a>
          <a
            :href="geminiHelpLinks.aiStudioPricing"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t('admin.accounts.gemini.accountType.quotaLink') }}
          </a>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="showGeminiHelpDialog = false" type="button" class="btn btn-primary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- Mixed Channel Warning Dialog -->
  <ConfirmDialog
    :show="showMixedChannelWarning"
    :title="t('admin.accounts.mixedChannelWarningTitle')"
    :message="mixedChannelWarningMessageText"
    :confirm-text="t('common.confirm')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="handleMixedChannelConfirm"
    @cancel="handleMixedChannelCancel"
  />
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import {
  claudeModels,
  getPresetMappingsByPlatform,
  getModelsByPlatform,
  buildModelMappingObject,
  fetchAntigravityDefaultMappings
} from '@/composables/useModelWhitelist'
import { useAuthStore } from '@/stores/auth'
import { useQuotaNotifyState } from '@/composables/useQuotaNotifyState'
import type {
  Proxy,
  AdminGroup,
  AccountPlatform,
  AccountType,
  CreateAccountRequest,
  OpenAICompactMode,
  OpenAIResponsesMode
} from '@/types'
import {
  DEFAULT_OPENAI_ENDPOINT_CAPABILITIES,
  type OpenAIEndpointCapability
} from '@/components/account/openAIEndpointCapabilities'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import {
  applyHeaderOverride,
  isHeaderOverrideCapable,
  validateHeaderOverrideRows,
  type HeaderOverrideRow
} from '@/components/account/credentialsBuilder'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  OPENAI_WS_MODE_OFF,
  type OpenAIWSMode
} from '@/utils/openaiWsMode'
import OAuthAuthorizationFlow from './OAuthAuthorizationFlow.vue'
import AntigravityAccountTypePanel from './create/panels/AntigravityAccountTypePanel.vue'
import AntigravityModelMappingPanel from './create/panels/AntigravityModelMappingPanel.vue'
import AccountPlatformSelector from './create/panels/AccountPlatformSelector.vue'
import AccountBehaviorSettingsPanel from './create/panels/AccountBehaviorSettingsPanel.vue'
import AccountRuntimeSettingsPanel from './create/panels/AccountRuntimeSettingsPanel.vue'
import AnthropicAccountTypePanel from './create/panels/AnthropicAccountTypePanel.vue'
import GeminiAccountTypePanel from './create/panels/GeminiAccountTypePanel.vue'
import GrokAccountTypePanel from './create/panels/GrokAccountTypePanel.vue'
import GrokOAuthOptionsPanel from './create/panels/GrokOAuthOptionsPanel.vue'
import OpenAIAccountTypePanel from './create/panels/OpenAIAccountTypePanel.vue'
import PlatformCredentialSettingsPanel from './create/panels/PlatformCredentialSettingsPanel.vue'
import TempUnschedulablePanel from './create/panels/TempUnschedulablePanel.vue'
import VertexServiceAccountPanel from './create/panels/VertexServiceAccountPanel.vue'
import {
  buildAccountCreatePayload,
  toAccountCredentialDraft,
  type AccountBaseDraft,
} from './create/accountDraft'
import type {
  AddMethod,
  AntigravityAccountKind,
  AccountModelMappingDraft,
  AuthInputMethod,
  BedrockAuthMode,
  CreateAccountCategory,
  GeminiAIStudioTier,
  GeminiGCPTier,
  GeminiGoogleOneTier,
  GeminiOAuthType,
  ModelRestrictionMode,
} from './create/formDraft'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  buildAPIKeyCredentials,
  buildAPIKeyQuotaExtra,
  buildAntigravityExtra as composeAntigravityExtra,
  buildAntigravityUpstreamCredentials,
  buildAnthropicAPIKeyExtra,
  buildBedrockCredentials,
  buildOpenAIExtra as composeOpenAIExtra,
  buildTempUnschedulableRules,
  buildVertexServiceAccountCredentials,
  defaultAPIKeyBaseURL,
  defaultZhipuBaseURL,
  isManagedZhipuBaseURL,
  zhipuBaseURLForRouting,
  parseVertexServiceAccountJSON,
  type AnthropicAPIKeyAuthScheme,
  type TempUnschedRuleForm,
} from './create/credentialDraftBuilders'
import type { CnAccountMode, CnApiProtocol } from './create/credentialDraftBuilders'
import {
  oauthStepTitleKey,
  resolveCreateAccountType,
  resolveGeminiSelectedTier,
  usesCreateAccountOAuthFlow,
} from './create/platformFormPolicy'
import { useAnthropicOAuthControlDraft } from './create/useAnthropicOAuthControlDraft'
import { useAnthropicCreateAccountController } from './create/useAnthropicCreateAccountController'
import { useAntigravityCreateAccountController } from './create/useAntigravityCreateAccountController'
import { useCreateAccountFlowController } from './create/useCreateAccountFlowController'
import { useCreateAccountOAuthDrivers } from './create/useCreateAccountOAuthDrivers'
import { useGeminiCreateAccountController } from './create/useGeminiCreateAccountController'
import { useGrokCreateAccountController } from './create/useGrokCreateAccountController'
import { useOpenAICreateAccountController } from './create/useOpenAICreateAccountController'

// Type for exposed OAuthAuthorizationFlow component
// Note: defineExpose automatically unwraps refs, so we use the unwrapped types
interface OAuthFlowExposed {
  authCode: string
  oauthState: string
  projectId: string
  sessionKey: string
  refreshToken: string
  sessionToken: string
  codexSession: string
  codexPAT: string
  ssoCookie: string
  inputMethod: AuthInputMethod
  reset: () => void
}

const { t } = useI18n()
const authStore = useAuthStore()

const oauthStepTitle = computed(() => t(oauthStepTitleKey(form.platform)))

interface Props {
  show: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  created: []
}>()

const appStore = useAppStore()

const {
  anthropicOAuth: oauth,
  openaiOAuth,
  geminiOAuth,
  antigravityOAuth,
  grokOAuth,
  registry: oauthDrivers,
  currentAuthUrl,
  currentSessionId,
  currentLoading: currentOAuthLoading,
  currentError: currentOAuthError,
  generateAuthorization: generateCurrentOAuthAuthorization,
  exchangeAuthorizationCode: exchangeCurrentOAuthCode,
  validateRefreshToken: validateCurrentOAuthRefreshToken,
} = useCreateAccountOAuthDrivers({
  platform: () => form.platform,
  anthropicAddMethod: () => addMethod.value,
  exchangeAuthorizationCode: {
    anthropic: (code) => handleAnthropicExchange(code),
    openai: (code) => handleOpenAIExchange(code),
    gemini: (code) => handleGeminiExchange(code),
    antigravity: (code) => handleAntigravityExchange(code),
    grok: (code) => handleGrokExchange(code),
    zhipu: async () => undefined,
    deepseek: async () => undefined,
  },
  validateRefreshToken: {
    openai: async (refreshToken) => handleOpenAIValidateRT(refreshToken),
    antigravity: (refreshToken) =>
      handleAntigravityValidateRT(refreshToken),
    grok: (refreshToken) => handleGrokValidateRT(refreshToken),
  },
})

// Refs
const oauthFlowRef = ref<OAuthFlowExposed | null>(null)

// Model mapping type
type ModelMapping = AccountModelMappingDraft

// State
const step = ref(1)
const accountCategory = ref<CreateAccountCategory>('oauth-based') // UI selection for account category
const addMethod = ref<AddMethod>('oauth') // For oauth-based: 'oauth' or 'setup-token'
const apiKeyBaseUrl = ref('https://api.anthropic.com')
const apiKeyValue = ref('')
const accountMode = ref<CnAccountMode>('payg')
const apiProtocol = ref<CnApiProtocol>('chat_completions')
const zhipuOrganization = ref('')
const zhipuProject = ref('')

const syncPreviewCredentials = computed(() => {
  if (!apiKeyValue.value) return undefined
  return {
    platform: form.platform,
    type: form.type,
    base_url: apiKeyBaseUrl.value || undefined,
    api_key: apiKeyValue.value,
    ...(form.platform === 'zhipu'
      ? {
          account_mode: accountMode.value,
          api_protocol: apiProtocol.value,
          zhipu_organization: zhipuOrganization.value || undefined,
          zhipu_project: zhipuProject.value || undefined,
        }
      : {}),
  }
})

const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editDailyResetHour = ref<number | null>(null)
const editWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editWeeklyResetDay = ref<number | null>(null)
const editWeeklyResetHour = ref<number | null>(null)
const editResetTimezone = ref<string | null>(null)
const modelMappings = ref<ModelMapping[]>([])
const openAICompactModelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<ModelRestrictionMode>('whitelist')
const allowedModels = ref<string[]>([])
const poolModeEnabled = ref(false)
const poolModeRetryCount = ref(DEFAULT_POOL_MODE_RETRY_COUNT)
const poolModeRetryStatusCodesInput = ref('')
const customErrorCodesEnabled = ref(false)
const selectedErrorCodes = ref<number[]>([])
const customErrorCodeInput = ref<number | null>(null)
const headerOverrideEnabled = ref(false)
const headerOverrideRows = ref<HeaderOverrideRow[]>([])

const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(true)
const openaiPassthroughEnabled = ref(false)
const openAILongContextBillingEnabled = ref(false)
const openAILongContextBillingTouched = ref(false)
const openAICompactMode = ref<OpenAICompactMode>('auto')
const openAIResponsesMode = ref<OpenAIResponsesMode>('auto')
const openAIEndpointCapabilities = ref<OpenAIEndpointCapability[]>([
  ...DEFAULT_OPENAI_ENDPOINT_CAPABILITIES
])
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const codexCLIOnlyAppServerEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const anthropicAPIKeyAuthScheme = ref<AnthropicAPIKeyAuthScheme>('x_api_key')
const webSearchEmulationMode = ref('default')
const webSearchGlobalEnabled = ref(false)

const {
  globalEnabled: quotaNotifyGlobalEnabled,
  state: quotaNotifyState,
  loadGlobalState: loadQuotaNotifyGlobal,
  writeToExtra: writeQuotaNotifyToExtra,
} = useQuotaNotifyState()

loadQuotaNotifyGlobal()
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const allowOverages = ref(false) // For antigravity accounts: enable AI Credits overages
const antigravityAccountType = ref<AntigravityAccountKind>('oauth') // For antigravity: oauth or upstream
const antigravityProjectId = ref('')
const upstreamBaseUrl = ref('') // For upstream type: base URL
const upstreamApiKey = ref('') // For upstream type: API key
const antigravityModelRestrictionMode = ref<ModelRestrictionMode>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMapping[]>([])
const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))

// Bedrock credentials
const bedrockAuthMode = ref<BedrockAuthMode>('sigv4')
const bedrockAccessKeyId = ref('')
const bedrockSecretAccessKey = ref('')
const bedrockSessionToken = ref('')
const bedrockRegion = ref('us-east-1')
const bedrockForceGlobal = ref(false)
const bedrockApiKeyValue = ref('')
const vertexServiceAccountJson = ref('')
const vertexProjectId = ref('')
const vertexClientEmail = ref('')
const vertexLocation = ref('global')
const tempUnschedEnabled = ref(false)
const tempUnschedRules = ref<TempUnschedRuleForm[]>([])
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('create-model-mapping')
const geminiOAuthType = ref<GeminiOAuthType>('google_one')
const openAITextGenerationCapabilityEnabled = computed(() =>
  openAIEndpointCapabilities.value.includes('chat_completions')
)

const buildAntigravityExtra = () =>
  composeAntigravityExtra({
    mixedScheduling: mixedScheduling.value,
    allowOverages: allowOverages.value,
  })

const buildOpenAICompactModelMapping = () =>
  buildModelMappingObject('mapping', [], openAICompactModelMappings.value)

const showAdvancedOAuth = ref(false)
const showGeminiHelpDialog = ref(false)

// Quota control state (Anthropic OAuth/SetupToken only)
const anthropicOAuthControls = useAnthropicOAuthControlDraft()
const {
  windowCostEnabled,
  windowCostLimit,
  windowCostStickyReserve,
  sessionLimitEnabled,
  maxSessions,
  sessionIdleTimeout,
  rpmLimitEnabled,
  baseRpm,
  rpmStrategy,
  rpmStickyBuffer,
  userMsgQueueMode,
  tlsFingerprintEnabled,
  tlsFingerprintProfileId,
  sessionIdMaskingEnabled,
  cacheTTLOverrideEnabled,
  cacheTTLOverrideTarget,
  customBaseUrlEnabled,
  customBaseUrl,
} = anthropicOAuthControls
const umqModeOptions = computed(() => [
  { value: '', label: t('admin.accounts.quotaControl.rpmLimit.umqModeOff') },
  { value: 'throttle', label: t('admin.accounts.quotaControl.rpmLimit.umqModeThrottle') },
  { value: 'serialize', label: t('admin.accounts.quotaControl.rpmLimit.umqModeSerialize') },
])
const tlsFingerprintProfiles = ref<{ id: number; name: string }[]>([])

// Gemini tier selection (used as fallback when auto-detection is unavailable/fails)
const geminiTierGoogleOne = ref<GeminiGoogleOneTier>('google_one_free')
const geminiTierGcp = ref<GeminiGCPTier>('gcp_standard')
const geminiTierAIStudio = ref<GeminiAIStudioTier>('aistudio_free')

const geminiSelectedTier = computed(() =>
  resolveGeminiSelectedTier({
    platform: form.platform,
    category: accountCategory.value,
    oauthType: geminiOAuthType.value,
    googleOneTier: geminiTierGoogleOne.value,
    gcpTier: geminiTierGcp.value,
    aiStudioTier: geminiTierAIStudio.value,
  }),
)

const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (form.platform === 'openai' && accountCategory.value === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (form.platform === 'openai' && accountCategory.value === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})

const isOpenAIModelRestrictionDisabled = computed(() =>
  form.platform === 'openai' && openaiPassthroughEnabled.value
)

const geminiQuotaDocs = {
  codeAssist: 'https://developers.google.com/gemini-code-assist/resources/quotas',
  aiStudio: 'https://ai.google.dev/pricing',
  vertex: 'https://cloud.google.com/vertex-ai/generative-ai/docs/quotas'
}

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  aiStudioPricing: 'https://ai.google.dev/pricing',
  gcpProject: 'https://console.cloud.google.com/welcome/new',
  geminiWebActivation: 'https://gemini.google.com/gems/create?hl=en-US&pli=1',
  countryCheck: 'https://policies.google.com/terms',
  countryChange: 'https://policies.google.com/country-association-form'
}

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(form.platform))

const form = reactive({
  name: '',
  notes: '',
  platform: 'anthropic' as AccountPlatform,
  type: 'oauth' as AccountType, // Will be 'oauth', 'setup-token', or 'apikey'
  proxy_id: null as number | null,
  concurrency: 10,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  group_ids: [] as number[],
  expires_at: null as number | null
})

const accountBaseDraft = (
  overrides: Partial<AccountBaseDraft> = {},
): AccountBaseDraft => {
  const base: AccountBaseDraft = {
    name: form.name,
    notes: form.notes,
    proxy_id: form.proxy_id,
    concurrency: form.concurrency,
    load_factor: form.load_factor,
    priority: form.priority,
    rate_multiplier: form.rate_multiplier,
    group_ids: [...form.group_ids],
    expires_at: form.expires_at,
    auto_pause_on_expired: autoPauseOnExpired.value,
  }
  return {
    ...base,
    ...overrides,
    group_ids: [...(overrides.group_ids ?? base.group_ids)],
  }
}

const buildModalAccountPayload = (
  platform: AccountPlatform,
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>,
  baseOverrides?: Partial<AccountBaseDraft>,
): CreateAccountRequest =>
  buildAccountCreatePayload(
    accountBaseDraft(baseOverrides),
    toAccountCredentialDraft(platform, type, credentials, extra),
  )

const {
  submitting,
  showMixedChannelWarning,
  mixedChannelWarningMessage: mixedChannelWarningMessageText,
  ensureMixedChannelConfirmed: ensureAntigravityMixedChannelConfirmed,
  withConfirmFlag: withAntigravityConfirmFlag,
  createAccount: doCreateAccount,
  confirmMixedChannel: handleMixedChannelConfirm,
  cancelMixedChannel: handleMixedChannelCancel,
  resetConfirmation: resetCreateFlowConfirmation,
  loadTLSFingerprintProfiles,
  loadWebSearchEmulationAvailability,
} = useCreateAccountFlowController({
  platform: () => form.platform,
  groupIds: () => form.group_ids,
  notifyCreated: () => emit('created'),
  close: () => handleClose(),
})

void loadWebSearchEmulationAvailability().then((enabled) => {
  webSearchGlobalEnabled.value = enabled
})

// Helper to check if current type needs OAuth flow
const isOAuthFlow = computed(() => {
  return usesCreateAccountOAuthFlow({
    platform: form.platform,
    category: accountCategory.value,
    antigravityKind: antigravityAccountType.value,
  })
})

const isGrokSSOInputMethod = computed(() => form.platform === 'grok' && oauthFlowRef.value?.inputMethod === 'sso_cookie')

const isManualInputMethod = computed(() => {
  return oauthFlowRef.value?.inputMethod === 'manual'
})

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

const canExchangeCode = computed(() => {
  const authCode = oauthFlowRef.value?.authCode || ''
  return Boolean(
    authCode.trim() &&
      currentSessionId.value &&
      !currentOAuthLoading.value,
  )
})

// Watchers
watch(
  () => props.show,
  (newVal) => {
    if (newVal) {
      // Load TLS fingerprint profiles
      void loadTLSFingerprintProfiles().then((profiles) => {
        tlsFingerprintProfiles.value = profiles
      })
      // Modal opened - fill related models
      allowedModels.value = [...getModelsByPlatform(form.platform)]
      // Antigravity: 默认使用映射模式并填充默认映射
      if (form.platform === 'antigravity') {
        antigravityModelRestrictionMode.value = 'mapping'
        fetchAntigravityDefaultMappings().then(mappings => {
          antigravityModelMappings.value = [...mappings]
        })
        antigravityWhitelistModels.value = []
      } else {
        antigravityWhitelistModels.value = []
        antigravityModelMappings.value = []
        antigravityModelRestrictionMode.value = 'mapping'
      }
    } else {
      resetForm()
    }
  }
)

// Sync form.type based on accountCategory, addMethod, and platform-specific type
watch(
  [accountCategory, addMethod, antigravityAccountType, () => form.platform],
  ([category, method, agType]) => {
    form.type = resolveCreateAccountType({
      platform: form.platform,
      category,
      addMethod: method,
      antigravityKind: agType,
    })
  },
  { immediate: true }
)

// Reset platform-specific settings when platform changes
watch(
  () => form.platform,
  (newPlatform) => {
    apiKeyBaseUrl.value = defaultAPIKeyBaseURL(newPlatform)
    if (newPlatform === 'deepseek' || newPlatform === 'zhipu') {
      accountCategory.value = 'apikey'
      apiProtocol.value = 'chat_completions'
      apiKeyBaseUrl.value = newPlatform === 'zhipu'
        ? defaultZhipuBaseURL(accountMode.value, apiProtocol.value)
        : defaultAPIKeyBaseURL(newPlatform)
    }
    if (newPlatform !== 'zhipu') {
      zhipuOrganization.value = ''
      zhipuProject.value = ''
    }
    // Clear model-related settings
    allowedModels.value = []
    modelMappings.value = []
    // Antigravity: 默认使用映射模式并填充默认映射
    if (newPlatform === 'antigravity') {
      antigravityModelRestrictionMode.value = 'mapping'
      fetchAntigravityDefaultMappings().then(mappings => {
        antigravityModelMappings.value = [...mappings]
      })
      antigravityWhitelistModels.value = []
      accountCategory.value = 'oauth-based'
      antigravityAccountType.value = 'oauth'
    } else {
      allowOverages.value = false
      antigravityProjectId.value = ''
      antigravityWhitelistModels.value = []
      antigravityModelMappings.value = []
      antigravityModelRestrictionMode.value = 'mapping'
    }
    if (newPlatform === 'grok') {
      accountCategory.value = 'oauth-based'
      addMethod.value = 'oauth'
      modelRestrictionMode.value = 'mapping'
      form.concurrency = 1
      form.load_factor = null
    }
    if (newPlatform !== 'gemini' && newPlatform !== 'anthropic' && accountCategory.value === 'service_account') {
      accountCategory.value = 'oauth-based'
    }
    if (newPlatform !== 'anthropic' && accountCategory.value === 'bedrock') {
      accountCategory.value = 'oauth-based'
    }
    // Reset Bedrock fields when switching platforms
    bedrockAccessKeyId.value = ''
    bedrockSecretAccessKey.value = ''
    bedrockSessionToken.value = ''
    bedrockRegion.value = 'us-east-1'
    bedrockForceGlobal.value = false
    bedrockAuthMode.value = 'sigv4'
    bedrockApiKeyValue.value = ''
    vertexServiceAccountJson.value = ''
    vertexProjectId.value = ''
    vertexClientEmail.value = ''
    vertexLocation.value = 'global'
    // Reset Anthropic/Antigravity-specific settings when switching to other platforms
    if (newPlatform !== 'anthropic' && newPlatform !== 'antigravity') {
      interceptWarmupRequests.value = false
    }
    if (newPlatform !== 'openai') {
      openaiPassthroughEnabled.value = false
      openAIEndpointCapabilities.value = [...DEFAULT_OPENAI_ENDPOINT_CAPABILITIES]
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
      codexCLIOnlyAppServerEnabled.value = false
    }
    if (newPlatform !== 'anthropic') {
      anthropicPassthroughEnabled.value = false
      anthropicAPIKeyAuthScheme.value = 'x_api_key'
      webSearchEmulationMode.value = 'default'
    }
    // 请求头覆写为平台相关配置（常用头集合不同），切换平台时清空，
    // 避免上一平台的配置行被提交到新平台账号
    headerOverrideEnabled.value = false
    headerOverrideRows.value = []
    resetGrokOAuthUpstreamConfig()
    // Reset OAuth states through the single platform registry.
    oauthDrivers.resetAll()
  }
)

watch([accountMode, apiProtocol], ([mode, protocol]) => {
  // Preserve a custom relay URL while still keeping the official presets linked
  // to the selected mode/protocol.
  if (
    form.platform === 'zhipu' &&
    (!apiKeyBaseUrl.value.trim() || isManagedZhipuBaseURL(apiKeyBaseUrl.value))
  ) {
    apiKeyBaseUrl.value = zhipuBaseURLForRouting(
      mode,
      protocol,
      apiKeyBaseUrl.value,
    )
  }
})

// Reset options that are only valid for one platform/category.
watch(
  [accountCategory, () => form.platform],
  ([category, platform]) => {
    if (platform === 'openai' && category !== 'oauth-based') {
      codexCLIOnlyEnabled.value = false
      codexCLIOnlyAppServerEnabled.value = false
    }
    if (platform !== 'anthropic' || category !== 'apikey') {
      anthropicPassthroughEnabled.value = false
      anthropicAPIKeyAuthScheme.value = 'x_api_key'
      webSearchEmulationMode.value = 'default'
    }
  }
)

// Auto-fill related models when switching to whitelist mode or changing platform
watch(
  [modelRestrictionMode, () => form.platform],
  ([newMode]) => {
    if (newMode === 'whitelist') {
      allowedModels.value = [...getModelsByPlatform(form.platform)]
    }
  }
)

watch(
  [antigravityModelRestrictionMode, () => form.platform],
  ([, platform]) => {
    if (platform !== 'antigravity') return
    // Antigravity 默认不做限制：白名单留空表示允许所有（包含未来新增模型）。
    // 如果需要快速填充常用模型，可在组件内点“填充相关模型”。
  }
)

// Model mapping helpers
const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const updateMappingField = (
  mappings: ModelMapping[],
  index: number,
  field: keyof ModelMapping,
  value: string,
) => {
  const mapping = mappings[index]
  if (mapping) mapping[field] = value
}

const addPresetMapping = (from: string, to: string) => {
  if (modelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const addAntigravityModelMapping = () => {
  antigravityModelMappings.value.push({ from: '', to: '' })
}

const removeAntigravityModelMapping = (index: number) => {
  antigravityModelMappings.value.splice(index, 1)
}

const updateAntigravityModelMapping = (
  index: number,
  field: keyof ModelMapping,
  value: string,
) => updateMappingField(antigravityModelMappings.value, index, field, value)

const addAntigravityPresetMapping = (from: string, to: string) => {
  if (antigravityModelMappings.value.some((m) => m.from === from)) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
}

const applyTempUnschedConfig = (credentials: Record<string, unknown>) => {
  if (!tempUnschedEnabled.value) {
    delete credentials.temp_unschedulable_enabled
    delete credentials.temp_unschedulable_rules
    return true
  }

  const rules = buildTempUnschedulableRules(tempUnschedRules.value)
  if (rules.length === 0) {
    appStore.showError(t('admin.accounts.tempUnschedulable.rulesInvalid'))
    return false
  }

  credentials.temp_unschedulable_enabled = true
  credentials.temp_unschedulable_rules = rules
  return true
}

// Methods
const resetForm = () => {
  step.value = 1
  form.name = ''
  form.notes = ''
  form.platform = 'anthropic'
  form.type = 'oauth'
  form.proxy_id = null
  form.concurrency = 10
  form.load_factor = null
  form.priority = 1
  form.rate_multiplier = 1
  form.group_ids = []
  form.expires_at = null
  accountCategory.value = 'oauth-based'
  addMethod.value = 'oauth'
  apiKeyBaseUrl.value = 'https://api.anthropic.com'
  apiKeyValue.value = ''
  editQuotaLimit.value = null
  editQuotaDailyLimit.value = null
  editQuotaWeeklyLimit.value = null
  editDailyResetMode.value = null
  editDailyResetHour.value = null
  editWeeklyResetMode.value = null
  editWeeklyResetDay.value = null
  editWeeklyResetHour.value = null
  editResetTimezone.value = null
  modelMappings.value = []
  openAICompactModelMappings.value = []
  modelRestrictionMode.value = 'whitelist'
  allowedModels.value = [...claudeModels] // Default fill related models

  antigravityModelRestrictionMode.value = 'mapping'
  antigravityWhitelistModels.value = []
  fetchAntigravityDefaultMappings().then(mappings => {
    antigravityModelMappings.value = [...mappings]
  })
  poolModeEnabled.value = false
  poolModeRetryCount.value = DEFAULT_POOL_MODE_RETRY_COUNT
  poolModeRetryStatusCodesInput.value = ''
  customErrorCodesEnabled.value = false
  selectedErrorCodes.value = []
  customErrorCodeInput.value = null
  headerOverrideEnabled.value = false
  headerOverrideRows.value = []
  resetGrokOAuthUpstreamConfig()
  interceptWarmupRequests.value = false
  autoPauseOnExpired.value = true
  openaiPassthroughEnabled.value = false
  openAILongContextBillingEnabled.value = false
  openAILongContextBillingTouched.value = false
  openAICompactMode.value = 'auto'
  openAIResponsesMode.value = 'auto'
  openAIEndpointCapabilities.value = [...DEFAULT_OPENAI_ENDPOINT_CAPABILITIES]
  openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
  codexCLIOnlyEnabled.value = false
  codexCLIOnlyAppServerEnabled.value = false
  anthropicPassthroughEnabled.value = false
  anthropicAPIKeyAuthScheme.value = 'x_api_key'
  webSearchEmulationMode.value = 'default'
  anthropicOAuthControls.reset()
  allowOverages.value = false
  antigravityAccountType.value = 'oauth'
  antigravityProjectId.value = ''
  upstreamBaseUrl.value = ''
  upstreamApiKey.value = ''
  vertexServiceAccountJson.value = ''
  vertexProjectId.value = ''
  vertexClientEmail.value = ''
  vertexLocation.value = 'global'
  tempUnschedEnabled.value = false
  tempUnschedRules.value = []
  geminiOAuthType.value = 'code_assist'
  geminiTierGoogleOne.value = 'google_one_free'
  geminiTierGcp.value = 'gcp_standard'
  geminiTierAIStudio.value = 'aistudio_free'
  accountMode.value = 'payg'
  apiProtocol.value = 'chat_completions'
  zhipuOrganization.value = ''
  zhipuProject.value = ''
  oauthDrivers.resetAll()
  oauthFlowRef.value?.reset()
  resetCreateFlowConfirmation()
}

const selectDeepSeekPlatform = () => { form.platform = 'deepseek'; accountCategory.value = 'apikey'; form.type = 'apikey'; apiKeyBaseUrl.value = defaultAPIKeyBaseURL('deepseek') }
const selectZhipuPlatform = () => {
  form.platform = 'zhipu'
  accountCategory.value = 'apikey'
  form.type = 'apikey'
  accountMode.value = 'payg'
  apiProtocol.value = 'chat_completions'
  apiKeyBaseUrl.value = defaultZhipuBaseURL(accountMode.value, apiProtocol.value)
}

const handlePlatformSelect = (platform: AccountPlatform) => {
  if (platform === 'deepseek') {
    selectDeepSeekPlatform()
    return
  }
  if (platform === 'zhipu') {
    selectZhipuPlatform()
    return
  }
  form.platform = platform
}

const handleClose = () => {
  resetCreateFlowConfirmation()
  emit('close')
}

const buildOpenAIExtra = (base?: Record<string, unknown>) =>
  composeOpenAIExtra(
    {
      platform: form.platform,
      accountCategory: accountCategory.value,
      oauthWebSocketMode: openaiOAuthResponsesWebSocketV2Mode.value,
      apiKeyWebSocketMode: openaiAPIKeyResponsesWebSocketV2Mode.value,
      passthroughEnabled: openaiPassthroughEnabled.value,
      longContextBillingEnabled: openAILongContextBillingEnabled.value,
      codexCLIOnlyEnabled: codexCLIOnlyEnabled.value,
      codexCLIOnlyAppServerEnabled: codexCLIOnlyAppServerEnabled.value,
      compactMode: openAICompactMode.value,
      responsesMode: openAIResponsesMode.value,
      textGenerationCapabilityEnabled:
        openAITextGenerationCapabilityEnabled.value,
    },
    base,
  )

const buildOpenAICodexImportExtra = (): Record<string, unknown> | undefined => {
  const extra = buildOpenAIExtra()
  if (!extra) {
    return undefined
  }
  if (!openAILongContextBillingTouched.value) {
    delete extra.openai_long_context_billing_enabled
  }
  return Object.keys(extra).length > 0 ? extra : undefined
}

const prepareOpenAICredentials = (
  credentials: Record<string, unknown>,
) => {
  if (!isOpenAIModelRestrictionDisabled.value) {
    const modelMapping = buildModelMappingObject(
      modelRestrictionMode.value,
      allowedModels.value,
      modelMappings.value,
    )
    if (modelMapping) credentials.model_mapping = modelMapping
  }
  const compactModelMapping = buildOpenAICompactModelMapping()
  if (compactModelMapping) {
    credentials.compact_model_mapping = compactModelMapping
  }
  return applyTempUnschedConfig(credentials)
}

const buildAnthropicExtra = (base?: Record<string, unknown>) =>
  buildAnthropicAPIKeyExtra(
    {
      platform: form.platform,
      accountCategory: accountCategory.value,
      passthroughEnabled: anthropicPassthroughEnabled.value,
      authScheme: anthropicAPIKeyAuthScheme.value,
      webSearchEmulationMode: webSearchEmulationMode.value,
    },
    base,
  )

const buildCurrentAnthropicOAuthExtra = (
  base?: Record<string, unknown>,
) => anthropicOAuthControls.buildExtra(base)

const buildAnthropicCredentialDefaults = () => {
  const credentials: Record<string, unknown> = {}
  if (interceptWarmupRequests.value) {
    credentials.intercept_warmup_requests = true
  }
  return applyTempUnschedConfig(credentials) ? credentials : null
}

const applyVertexServiceAccountJson = (value: string) => {
  const result = parseVertexServiceAccountJSON(value)
  if (!result.ok) {
    vertexProjectId.value = ''
    vertexClientEmail.value = ''
    if (result.reason === 'missing_fields') {
      appStore.showError(t('admin.accounts.vertexSaJsonMissingFields'))
    } else if (result.reason === 'invalid_json') {
      appStore.showError(t('admin.accounts.vertexSaJsonInvalid'))
    }
    return false
  }
  vertexProjectId.value = result.projectId
  vertexClientEmail.value = result.clientEmail
  vertexServiceAccountJson.value = result.normalizedJson
  return true
}

const parseVertexServiceAccountJson = () => applyVertexServiceAccountJson(vertexServiceAccountJson.value)

const handleVertexServiceAccountFile = async (file: File) => {
  applyVertexServiceAccountJson(await file.text())
}

const handleSubmit = async () => {
  // For OAuth-based type, handle OAuth flow (goes to step 2)
  if (isOAuthFlow.value) {
    if (!isGrokSSOInputMethod.value && !form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    const canContinue = await ensureAntigravityMixedChannelConfirmed(async () => {
      step.value = 2
    })
    if (!canContinue) {
      return
    }
    step.value = 2
    return
  }

  // For Bedrock type, create directly
  if (form.platform === 'anthropic' && accountCategory.value === 'bedrock') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }

    if (bedrockAuthMode.value === 'sigv4') {
      if (!bedrockAccessKeyId.value.trim()) {
        appStore.showError(t('admin.accounts.bedrockAccessKeyIdRequired'))
        return
      }
      if (!bedrockSecretAccessKey.value.trim()) {
        appStore.showError(t('admin.accounts.bedrockSecretAccessKeyRequired'))
        return
      }
    } else {
      if (!bedrockApiKeyValue.value.trim()) {
        appStore.showError(t('admin.accounts.bedrockApiKeyRequired'))
        return
      }
    }

    const modelMapping = buildModelMappingObject(
      modelRestrictionMode.value, allowedModels.value, modelMappings.value
    )
    const credentials = buildBedrockCredentials({
      authMode: bedrockAuthMode.value,
      accessKeyId: bedrockAccessKeyId.value,
      secretAccessKey: bedrockSecretAccessKey.value,
      sessionToken: bedrockSessionToken.value,
      region: bedrockRegion.value,
      forceGlobal: bedrockForceGlobal.value,
      apiKey: bedrockApiKeyValue.value,
      modelMapping,
      enabled: poolModeEnabled.value,
      retryCount: poolModeRetryCount.value,
      retryStatusCodesInput: poolModeRetryStatusCodesInput.value,
      interceptWarmupRequests: interceptWarmupRequests.value,
    })

    await createAccountAndFinish('anthropic', 'bedrock' as AccountType, credentials)
    return
  }

  // For Antigravity upstream type, create directly
  if (form.platform === 'antigravity' && antigravityAccountType.value === 'upstream') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (!upstreamBaseUrl.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterBaseUrl'))
      return
    }
    if (!upstreamApiKey.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterApiKey'))
      return
    }

    // Antigravity 只使用映射模式
    const antigravityModelMapping = buildModelMappingObject(
      'mapping',
      [],
      antigravityModelMappings.value
    )
    const credentials = buildAntigravityUpstreamCredentials({
      baseUrl: upstreamBaseUrl.value,
      apiKey: upstreamApiKey.value,
      modelMapping: antigravityModelMapping,
      interceptWarmupRequests: interceptWarmupRequests.value,
    })

    const extra = buildAntigravityExtra()
    await createAccountAndFinish(form.platform, 'apikey', credentials, extra)
    return
  }

  if ((form.platform === 'gemini' || form.platform === 'anthropic') && accountCategory.value === 'service_account') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (!parseVertexServiceAccountJson()) {
      return
    }
    if (!vertexLocation.value.trim()) {
      appStore.showError(t('admin.accounts.vertexLocationRequired'))
      return
    }
    const credentials = buildVertexServiceAccountCredentials({
      serviceAccountJson: vertexServiceAccountJson.value,
      projectId: vertexProjectId.value,
      clientEmail: vertexClientEmail.value,
      location: vertexLocation.value,
    })
    await createAccountAndFinish(form.platform, 'service_account' as AccountType, credentials)
    return
  }

  // For apikey type, create directly
  if (!apiKeyValue.value.trim()) {
    appStore.showError(t('admin.accounts.pleaseEnterApiKey'))
    return
  }

  // Add model mapping if configured（OpenAI 开启自动透传时不应用）
  let modelMapping: Record<string, string> | null = null
  if (!isOpenAIModelRestrictionDisabled.value) {
    modelMapping = buildModelMappingObject(
      modelRestrictionMode.value,
      allowedModels.value,
      modelMappings.value,
    )
  }
  const credentials = buildAPIKeyCredentials({
    platform: form.platform,
    baseUrl: apiKeyBaseUrl.value,
    apiKey: apiKeyValue.value,
    accountMode: form.platform === 'zhipu' ? accountMode.value : undefined,
    apiProtocol: form.platform === 'zhipu' ? apiProtocol.value : undefined,
    zhipuOrganization: form.platform === 'zhipu' ? zhipuOrganization.value : undefined,
    zhipuProject: form.platform === 'zhipu' ? zhipuProject.value : undefined,
    geminiTierId: geminiTierAIStudio.value,
    modelMapping,
    compactModelMapping: buildOpenAICompactModelMapping(),
    openAIEndpointCapabilities: openAIEndpointCapabilities.value,
    enabled: poolModeEnabled.value,
    retryCount: poolModeRetryCount.value,
    retryStatusCodesInput: poolModeRetryStatusCodesInput.value,
    customErrorCodesEnabled: customErrorCodesEnabled.value,
    customErrorCodes: selectedErrorCodes.value,
    interceptWarmupRequests: interceptWarmupRequests.value,
  })

  // Add header override if enabled (anthropic/openai/grok apikey)
  if (isHeaderOverrideCapable(form.platform, 'apikey')) {
    if (headerOverrideEnabled.value) {
      const headerError = validateHeaderOverrideRows(headerOverrideRows.value)
      if (headerError) {
        appStore.showError(t(`admin.accounts.headerOverride.${headerError}`))
        return
      }
    }
    applyHeaderOverride(credentials, headerOverrideEnabled.value, headerOverrideRows.value, 'create')
  }

  if (!applyTempUnschedConfig(credentials)) {
    return
  }

  const extra = buildAnthropicExtra(buildOpenAIExtra())

  await doCreateAccount(buildModalAccountPayload(form.platform, form.type, credentials, extra))
}

const goBackToBasicInfo = () => {
  step.value = 1
  oauthDrivers.resetAll()
  oauthFlowRef.value?.reset()
}

const handleGenerateUrl = async () => {
  await generateCurrentOAuthAuthorization({
    proxyId: form.proxy_id,
    projectId: oauthFlowRef.value?.projectId,
    oauthType: geminiOAuthType.value,
    tier: geminiSelectedTier.value,
  })
}

const handleValidateRefreshToken = (rt: string) =>
  validateCurrentOAuthRefreshToken(rt)

const handleValidateSessionToken = (_sessionToken: string) => {
  // Session token validation removed
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Create account and handle success/failure
const createAccountAndFinish = async (
  platform: AccountPlatform,
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>
) => {
  if (!applyTempUnschedConfig(credentials)) {
    return
  }
  // Inject quota limits for apikey/bedrock accounts
  let finalExtra = extra
  if (type === 'apikey' || type === 'bedrock') {
    const quotaExtra = buildAPIKeyQuotaExtra(extra, {
      quotaLimit: editQuotaLimit.value,
      quotaDailyLimit: editQuotaDailyLimit.value,
      quotaWeeklyLimit: editQuotaWeeklyLimit.value,
      dailyResetMode: editDailyResetMode.value,
      dailyResetHour: editDailyResetHour.value,
      weeklyResetMode: editWeeklyResetMode.value,
      weeklyResetDay: editWeeklyResetDay.value,
      weeklyResetHour: editWeeklyResetHour.value,
      resetTimezone: editResetTimezone.value,
    })
    // Quota notify config
    writeQuotaNotifyToExtra(quotaExtra, 'create')
    if (Object.keys(quotaExtra).length > 0) {
      finalExtra = quotaExtra
    }
  }
  if (platform === 'grok') {
    if (!credentials.base_url) {
      credentials.base_url =
        apiKeyBaseUrl.value.trim() || defaultAPIKeyBaseURL('grok')
    }
    const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
    if (modelMapping) {
      credentials.model_mapping = modelMapping
    } else {
      delete credentials.model_mapping
    }
  }
  await doCreateAccount(buildModalAccountPayload(platform, type, credentials, finalExtra))
}

const {
  aiStudioOAuthEnabled: geminiAIStudioOAuthEnabled,
  handleSelectOAuthType: handleSelectGeminiOAuthType,
  handleExchangeAuthorizationCode: handleGeminiExchange,
} = useGeminiCreateAccountController({
  oauth: geminiOAuth,
  show: () => props.show,
  platform: () => form.platform,
  category: () => accountCategory.value,
  proxyId: () => form.proxy_id,
  oauthState: () => oauthFlowRef.value?.oauthState || '',
  selectedTier: () => geminiSelectedTier.value,
  oauthType: geminiOAuthType,
  createAndFinish: (credentials, extra) =>
    createAccountAndFinish('gemini', 'oauth', credentials, extra),
})

const {
  handleValidateRefreshToken: handleAntigravityValidateRT,
  handleExchangeAuthorizationCode: handleAntigravityExchange,
} = useAntigravityCreateAccountController({
  oauth: antigravityOAuth,
  proxyId: () => form.proxy_id,
  accountName: () => form.name,
  oauthState: () => oauthFlowRef.value?.oauthState || '',
  projectId: antigravityProjectId,
  interceptWarmupRequests,
  modelMappings: antigravityModelMappings,
  buildExtra: buildAntigravityExtra,
  buildPayload: (credentials, extra, baseOverrides) =>
    buildModalAccountPayload(
      'antigravity',
      'oauth',
      credentials,
      extra,
      baseOverrides,
    ),
  withConfirmFlag: withAntigravityConfirmFlag,
  createAndFinish: (credentials, extra) =>
    createAccountAndFinish('antigravity', 'oauth', credentials, extra),
  notifyCreated: () => emit('created'),
  close: handleClose,
})

const {
  customBaseUrlEnabled: grokOAuthCustomBaseUrlEnabled,
  baseUrl: grokOAuthBaseUrl,
  resetConfiguration: resetGrokOAuthUpstreamConfig,
  handleValidateRefreshToken: handleGrokValidateRT,
  handleImportSSO: handleGrokImportSSO,
  handleExchangeAuthorizationCode: handleGrokExchange,
} = useGrokCreateAccountController({
  oauth: grokOAuth,
  proxyId: () => form.proxy_id,
  accountName: () => form.name,
  oauthState: () => oauthFlowRef.value?.oauthState || '',
  headerOverrideEnabled,
  headerOverrideRows,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  buildBaseDraft: () => accountBaseDraft(),
  buildPayload: (credentials, extra, baseOverrides) =>
    buildModalAccountPayload(
      'grok',
      'oauth',
      credentials,
      extra,
      baseOverrides,
    ),
  applyTempUnschedulableConfig: applyTempUnschedConfig,
  createAndFinish: (credentials, extra) =>
    createAccountAndFinish('grok', 'oauth', credentials, extra),
  notifyCreated: () => emit('created'),
  close: handleClose,
})

const {
  handleExchangeAuthorizationCode: handleOpenAIExchange,
  handleImportCodexSession: handleOpenAIImportCodexSession,
  handleImportCodexPAT: handleOpenAIImportCodexPAT,
  handleValidateRefreshToken: handleOpenAIValidateRT,
  handleValidateMobileRefreshToken: handleOpenAIValidateMobileRT,
} = useOpenAICreateAccountController({
  oauth: openaiOAuth,
  platform: () => form.platform,
  proxyId: () => form.proxy_id,
  accountName: () => form.name,
  oauthState: () => oauthFlowRef.value?.oauthState || '',
  inputMethod: () => oauthFlowRef.value?.inputMethod,
  buildBaseDraft: () => accountBaseDraft(),
  buildPayload: (credentials, extra, baseOverrides) =>
    buildModalAccountPayload(
      'openai',
      'oauth',
      credentials,
      extra,
      baseOverrides,
    ),
  buildExtra: buildOpenAIExtra,
  buildImportExtra: buildOpenAICodexImportExtra,
  prepareCredentials: prepareOpenAICredentials,
  notifyCreated: () => emit('created'),
  close: handleClose,
})

const {
  handleExchangeAuthorizationCode: handleAnthropicExchange,
  handleCookieAuth,
} = useAnthropicCreateAccountController({
  oauth,
  platform: () => form.platform,
  addMethod: () => addMethod.value,
  proxyId: () => form.proxy_id,
  accountName: () => form.name,
  buildExtra: buildCurrentAnthropicOAuthExtra,
  buildCredentialDefaults: buildAnthropicCredentialDefaults,
  buildPayload: (credentials, extra, baseOverrides) =>
    buildModalAccountPayload(
      form.platform,
      addMethod.value as AccountType,
      credentials,
      extra,
      baseOverrides,
    ),
  createAndFinish: createAccountAndFinish,
  notifyCreated: () => emit('created'),
  close: handleClose,
})

// OAuth completion stays routed through the platform registry.
// 主入口：根据平台路由到对应处理函数
const handleExchangeCode = async () => {
  const authCode = oauthFlowRef.value?.authCode || ''
  await exchangeCurrentOAuthCode(authCode)
}

</script>
