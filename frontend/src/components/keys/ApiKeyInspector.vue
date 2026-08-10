<template>
  <aside
    ref="inspectorRef"
    class="api-key-inspector"
    :class="`api-key-inspector--${mode}`"
    role="region"
    :aria-labelledby="mode === 'inline' ? titleId : undefined"
    :aria-label="mode === 'sheet' ? apiKey.name : undefined"
    tabindex="-1"
    data-test="api-key-inspector"
    :data-key-id="apiKey.id"
    :data-mode="mode"
  >
    <template v-if="mode === 'inline'">
      <div class="api-key-inspector__content api-key-inspector__content--inline">
        <section class="api-key-inspector__inline-overview" data-test="api-key-inspector-overview">
          <div class="api-key-inspector__inline-identity">
            <div class="min-w-0">
              <h2 :id="titleId" class="api-key-inspector__title">{{ apiKey.name }}</h2>
              <p class="api-key-inspector__id">
                KEY ID: #{{ apiKey.id }}
              </p>
            </div>
            <span class="api-key-inspector__status-pill" :data-status="apiKey.status">
              <span aria-hidden="true" />
              {{ t(`keys.status.${apiKey.status}`) }}
            </span>
          </div>

          <div class="api-key-inspector__credential-stack">
            <div data-test="api-key-inspector-key">
              <p class="api-key-inspector__field-label">{{ t('keys.workspaceTokenLabel') }}</p>
              <div class="api-key-inspector__key-field">
                <code>{{ revealKey ? apiKey.key : maskedKey }}</code>
                <button
                  type="button"
                  class="api-key-inspector__icon-button"
                  :aria-label="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
                  :title="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
                  :data-test="`key-inspector-reveal-${apiKey.id}`"
                  @click="revealKey = !revealKey"
                >
                  <KeysLucideIcon :name="revealKey ? 'eyeOff' : 'eye'" :size="18" />
                </button>
                <button
                  type="button"
                  class="api-key-inspector__icon-button api-key-inspector__icon-button--copy"
                  :class="{ 'is-copied': copied }"
                  :aria-label="copied ? t('keys.copied') : t('keys.copyToClipboard')"
                  :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
                  :data-test="`key-inspector-copy-${apiKey.id}`"
                  @click="emit('copy-key', apiKey)"
                >
                  <KeysLucideIcon :name="copied ? 'check' : 'copy'" :size="18" />
                </button>
              </div>
            </div>

            <div :data-test="hasConfiguredEndpoints ? 'api-key-inspector-endpoints' : undefined">
              <p class="api-key-inspector__field-label">{{ t('keys.workspaceEndpointLabel') }}</p>
              <div class="api-key-inspector__endpoint-well">
                <EndpointPopover
                  v-if="hasConfiguredEndpoints"
                  :api-base-url="configuredApiBaseUrl"
                  :custom-endpoints="configuredCustomEndpoints"
                />
                <p v-else class="api-key-inspector__empty-endpoint">{{ t('keys.workspaceEndpointEmpty') }}</p>
              </div>
            </div>
          </div>
        </section>

        <div class="api-key-inspector__divider" />

        <section
          class="api-key-inspector__section"
          data-test="api-key-inspector-quota"
          :data-quota-meter-class="quotaMeterClass"
        >
          <div class="api-key-inspector__section-heading">
            <h3>{{ t('keys.workspaceQuotaHeading') }}</h3>
            <button
              type="button"
              class="api-key-inspector__text-button"
              :disabled="quotaUsedValue <= 0"
              :data-test="`key-inspector-reset-quota-${apiKey.id}`"
              @click="emit('reset-quota', apiKey)"
            >
              {{ t('keys.workspaceResetQuota') }}
            </button>
          </div>

          <dl class="api-key-inspector__quota-grid">
            <div>
              <dt>{{ t('keys.workspaceCurrentAvailable') }}</dt>
              <dd>
                <CreditAmount v-if="hasQuotaLimit" :value="quotaAvailableValue.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('keys.unlimitedQuota') }}</span>
              </dd>
              <p>{{ t('keys.workspaceSnowCreditsUnit') }}</p>
            </div>
            <div>
              <dt>{{ t('keys.workspaceTotalLimit') }}</dt>
              <dd>
                <CreditAmount v-if="hasQuotaLimit" :value="quotaValue.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('keys.unlimitedQuota') }}</span>
              </dd>
            </div>
          </dl>

          <dl class="api-key-inspector__usage-rows" data-test="api-key-inspector-usage">
            <div>
              <dt>{{ t('keys.workspaceTodayUsage') }}</dt>
              <dd>
                <CreditAmount v-if="usage" :value="todayCost.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('common.notAvailable') }}</span>
              </dd>
            </div>
            <div>
              <dt>{{ t('keys.workspaceThirtyDayUsage') }}</dt>
              <dd>
                <CreditAmount v-if="usage" :value="totalCost.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('common.notAvailable') }}</span>
              </dd>
            </div>
          </dl>
        </section>

        <template v-if="isVisible('rate_limit')">
          <div class="api-key-inspector__divider" />
          <section class="api-key-inspector__section" data-test="api-key-inspector-rate-limits">
            <div class="api-key-inspector__section-heading">
              <h3>{{ t('keys.workspaceRateHeading') }}</h3>
              <button
                type="button"
                class="api-key-inspector__text-button"
                :disabled="!hasRateLimitUsage"
                :data-test="`key-inspector-reset-rate-limit-${apiKey.id}`"
                @click="emit('reset-rate-limit', apiKey)"
              >
                {{ t('keys.workspaceResetAll') }}
              </button>
            </div>
            <div class="api-key-inspector__rate-list">
              <article v-for="limit in rateLimits" :key="limit.label" class="api-key-inspector__rate-item">
                <div class="api-key-inspector__rate-topline">
                  <strong>{{ limit.inlineLabel }}</strong>
                  <span class="api-key-inspector__rate-values">
                    <CreditAmount :value="limit.used.toFixed(2)" icon-size="xs" />
                    <span aria-hidden="true">/</span>
                    <CreditAmount v-if="limit.limited" :value="limit.total.toFixed(2)" icon-size="xs" />
                    <span v-else>{{ t('keys.noRateLimit') }}</span>
                  </span>
                </div>
                <div
                  v-if="limit.limited"
                  class="api-key-inspector__progress"
                  role="progressbar"
                  :aria-label="`${limit.label} ${t('keys.rateLimitUsage')}`"
                  aria-valuemin="0"
                  aria-valuemax="100"
                  :aria-valuenow="Math.round(limit.percent)"
                >
                  <span :class="meterClass(limit.percent)" :style="{ width: `${limit.percent}%` }" />
                </div>
                <p v-if="limit.resetAt" class="api-key-inspector__reset-time">
                  {{ t('keys.workspaceResetInInline', { time: formatResetTime(limit.resetAt) }) }}
                </p>
              </article>
            </div>
          </section>
        </template>

        <div class="api-key-inspector__divider" />

        <section
          class="api-key-inspector__section"
          data-test="api-key-inspector-audit"
          :data-has-audit-fields="hasVisibleAuditFields"
        >
          <div class="api-key-inspector__section-heading">
            <h3>{{ t('keys.workspaceSecurityHeading') }}</h3>
          </div>
          <dl class="api-key-inspector__security-card">
            <div v-if="isVisible('expires_at')">
              <dt>{{ t('keys.expiresAt') }}</dt>
              <dd :class="{ 'is-danger': isExpired }">
                {{ apiKey.expires_at ? formatInspectorDateTime(apiKey.expires_at) : t('keys.noExpiration') }}
              </dd>
            </div>
            <div v-if="isVisible('current_concurrency')">
              <dt>{{ t('keys.workspaceConcurrencyLimit') }}</dt>
              <dd>{{ t('common.unlimited') }}</dd>
            </div>
            <div
              class="api-key-inspector__ip-rules"
              data-test="api-key-inspector-ip-rules"
              :data-configured="hasIpRestriction"
            >
              <div>
                <dt>{{ t('keys.ipWhitelist') }}</dt>
                <dd>
                  <span>{{ t('keys.workspaceIpConfigured', { count: whitelist.length }) }}</span>
                  <span v-if="whitelist.length" class="api-key-inspector__ip-values">
                    <code v-for="entry in whitelist" :key="`allow-${entry}`">{{ entry }}</code>
                  </span>
                </dd>
              </div>
              <div>
                <dt>{{ t('keys.ipBlacklist') }}</dt>
                <dd>
                  <span>{{ t('keys.workspaceIpConfigured', { count: blacklist.length }) }}</span>
                  <span v-if="blacklist.length" class="api-key-inspector__ip-values">
                    <code v-for="entry in blacklist" :key="`deny-${entry}`">{{ entry }}</code>
                  </span>
                </dd>
              </div>
            </div>
            <div v-if="isVisible('created_at')" class="api-key-inspector__security-divider">
              <dt>{{ t('keys.workspaceCreatedInline') }}</dt>
              <dd>{{ formatInspectorDateTime(apiKey.created_at) }}</dd>
            </div>
            <div v-if="isVisible('last_used_at')">
              <dt>{{ t('keys.workspaceLastUsedInline') }}</dt>
              <dd>{{ apiKey.last_used_at ? formatInspectorDateTime(apiKey.last_used_at) : t('common.notAvailable') }}</dd>
            </div>
            <div v-if="isVisible('last_used_ip')">
              <dt>{{ t('keys.workspaceLastIpInline') }}</dt>
              <dd class="api-key-inspector__mono">{{ apiKey.last_used_ip || t('common.notAvailable') }}</dd>
            </div>
          </dl>
        </section>
      </div>
    </template>

    <template v-else>
      <div class="api-key-inspector__content api-key-inspector__content--sheet">
        <section class="api-key-inspector__sheet-section" data-test="api-key-inspector-overview">
          <h3 class="api-key-inspector__sheet-heading">{{ t('keys.workspaceCoreAuthHeading') }}</h3>
          <div class="api-key-inspector__sheet-card api-key-inspector__core-card">
            <div class="api-key-inspector__sheet-status-row">
              <span>{{ t('keys.workspaceCurrentStatus') }}</span>
              <div class="api-key-inspector__sheet-status-control">
                <strong :data-status="apiKey.status">{{ t(`keys.workspaceStatus.${apiKey.status}`) }}</strong>
                <button
                  type="button"
                  role="switch"
                  class="api-key-inspector__switch"
                  :aria-checked="isActive"
                  :aria-busy="statusUpdating"
                  :aria-label="`${apiKey.name} · ${isActive ? t('keys.disable') : t('keys.enable')}`"
                  :disabled="statusUpdating"
                  :data-test="`key-inspector-status-switch-${apiKey.id}`"
                  @click="emit('toggle-status', apiKey)"
                >
                  <span :class="{ 'is-active': isActive }"><span /></span>
                </button>
              </div>
            </div>

            <div class="api-key-inspector__sheet-field" data-test="api-key-inspector-key">
              <p class="api-key-inspector__field-label">{{ t('keys.workspaceSheetTokenLabel') }}</p>
              <div class="api-key-inspector__key-field">
                <code>{{ revealKey ? apiKey.key : maskedKey }}</code>
                <button
                  type="button"
                  class="api-key-inspector__icon-button"
                  :aria-label="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
                  :title="revealKey ? t('keys.hideKey') : t('keys.revealKey')"
                  :data-test="`key-inspector-reveal-${apiKey.id}`"
                  @click="revealKey = !revealKey"
                >
                  <KeysLucideIcon :name="revealKey ? 'eyeOff' : 'eye'" :size="20" />
                </button>
                <button
                  type="button"
                  class="api-key-inspector__icon-button api-key-inspector__icon-button--copy"
                  :class="{ 'is-copied': copied }"
                  :aria-label="copied ? t('keys.copied') : t('keys.copyToClipboard')"
                  :title="copied ? t('keys.copied') : t('keys.copyToClipboard')"
                  :data-test="`key-inspector-copy-${apiKey.id}`"
                  @click="emit('copy-key', apiKey)"
                >
                  <KeysLucideIcon :name="copied ? 'check' : 'copy'" :size="20" />
                </button>
              </div>
            </div>

            <div
              class="api-key-inspector__sheet-field"
              :data-test="hasConfiguredEndpoints ? 'api-key-inspector-endpoints' : undefined"
            >
              <p class="api-key-inspector__field-label">{{ t('keys.workspaceSheetEndpointLabel') }}</p>
              <div class="api-key-inspector__endpoint-well">
                <EndpointPopover
                  v-if="hasConfiguredEndpoints"
                  :api-base-url="configuredApiBaseUrl"
                  :custom-endpoints="configuredCustomEndpoints"
                />
                <p v-else class="api-key-inspector__empty-endpoint">{{ t('keys.workspaceEndpointEmpty') }}</p>
              </div>
            </div>
          </div>
        </section>

        <section
          class="api-key-inspector__sheet-section"
          data-test="api-key-inspector-quota"
          :data-quota-meter-class="quotaMeterClass"
        >
          <div class="api-key-inspector__sheet-section-heading">
            <h3 class="api-key-inspector__sheet-heading">{{ t('keys.workspaceSheetQuotaHeading') }}</h3>
            <button
              type="button"
              class="api-key-inspector__text-button"
              :disabled="quotaUsedValue <= 0"
              :data-test="`key-inspector-reset-quota-${apiKey.id}`"
              @click="emit('reset-quota', apiKey)"
            >
              {{ t('keys.workspaceResetQuota') }}
            </button>
          </div>
          <dl class="api-key-inspector__quota-grid">
            <div>
              <dt>{{ t('keys.workspaceSheetCurrentAvailable') }}</dt>
              <dd>
                <CreditAmount v-if="hasQuotaLimit" :value="quotaAvailableValue.toFixed(2)" icon-size="md" />
                <span v-else>{{ t('keys.unlimitedQuota') }}</span>
              </dd>
            </div>
            <div>
              <dt>{{ t('keys.workspaceSheetTotalLimit') }}</dt>
              <dd>
                <CreditAmount v-if="hasQuotaLimit" :value="quotaValue.toFixed(2)" icon-size="md" />
                <span v-else>{{ t('keys.unlimitedQuota') }}</span>
              </dd>
            </div>
          </dl>
          <dl class="api-key-inspector__usage-card" data-test="api-key-inspector-usage">
            <div>
              <dt>{{ t('keys.workspaceSheetTodayUsage') }}</dt>
              <dd>
                <CreditAmount v-if="usage" :value="todayCost.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('common.notAvailable') }}</span>
              </dd>
            </div>
            <div>
              <dt>{{ t('keys.workspaceSheetThirtyDayUsage') }}</dt>
              <dd>
                <CreditAmount v-if="usage" :value="totalCost.toFixed(2)" icon-size="xs" />
                <span v-else>{{ t('common.notAvailable') }}</span>
              </dd>
            </div>
          </dl>
        </section>

        <section
          v-if="isVisible('rate_limit')"
          class="api-key-inspector__sheet-section"
          data-test="api-key-inspector-rate-limits"
        >
          <div class="api-key-inspector__sheet-section-heading">
            <h3 class="api-key-inspector__sheet-heading">{{ t('keys.workspaceSheetRateHeading') }}</h3>
            <button
              type="button"
              class="api-key-inspector__text-button"
              :disabled="!hasRateLimitUsage"
              :data-test="`key-inspector-reset-rate-limit-${apiKey.id}`"
              @click="emit('reset-rate-limit', apiKey)"
            >
              {{ t('keys.workspaceResetWindow') }}
            </button>
          </div>
          <div class="api-key-inspector__sheet-card api-key-inspector__rate-list">
            <article v-for="limit in rateLimits" :key="limit.label" class="api-key-inspector__rate-item">
              <div class="api-key-inspector__rate-topline">
                <strong>{{ limit.sheetLabel }}</strong>
                <span class="api-key-inspector__rate-values">
                  <CreditAmount :value="limit.used.toFixed(2)" icon-size="xs" />
                  <span aria-hidden="true">/</span>
                  <CreditAmount v-if="limit.limited" :value="limit.total.toFixed(2)" icon-size="xs" />
                  <span v-else>{{ t('keys.noRateLimit') }}</span>
                </span>
              </div>
              <div
                v-if="limit.limited"
                class="api-key-inspector__progress"
                role="progressbar"
                :aria-label="`${limit.label} ${t('keys.rateLimitUsage')}`"
                aria-valuemin="0"
                aria-valuemax="100"
                :aria-valuenow="Math.round(limit.percent)"
              >
                <span :class="meterClass(limit.percent)" :style="{ width: `${limit.percent}%` }" />
              </div>
              <p v-if="limit.resetAt" class="api-key-inspector__reset-time">
                {{ t('keys.workspaceResetInSheet', { time: formatResetTime(limit.resetAt) }) }}
              </p>
            </article>
          </div>
        </section>

        <section
          class="api-key-inspector__sheet-section"
          data-test="api-key-inspector-audit"
          :data-has-audit-fields="hasVisibleAuditFields"
        >
          <h3 class="api-key-inspector__sheet-heading">{{ t('keys.workspaceSheetSecurityHeading') }}</h3>
          <dl class="api-key-inspector__sheet-card api-key-inspector__security-card">
            <div v-if="isVisible('current_concurrency')">
              <dt>{{ t('keys.workspaceSheetConcurrencyLimit') }}</dt>
              <dd>{{ t('common.unlimited') }}</dd>
            </div>
            <div v-if="isVisible('expires_at')">
              <dt>{{ t('keys.workspaceSheetExpiration') }}</dt>
              <dd :class="{ 'is-danger': isExpired }">
                {{ apiKey.expires_at ? formatInspectorDateTime(apiKey.expires_at) : t('keys.noExpiration') }}
              </dd>
            </div>
            <div
              class="api-key-inspector__ip-rules api-key-inspector__security-divider"
              data-test="api-key-inspector-ip-rules"
              :data-configured="hasIpRestriction"
            >
              <div>
                <dt>{{ t('keys.ipWhitelist') }}</dt>
                <dd>
                  <span>{{ t('keys.workspaceIpConfigured', { count: whitelist.length }) }}</span>
                  <span v-if="whitelist.length" class="api-key-inspector__ip-values">
                    <code v-for="entry in whitelist" :key="`allow-${entry}`">{{ entry }}</code>
                  </span>
                </dd>
              </div>
              <div>
                <dt>{{ t('keys.ipBlacklist') }}</dt>
                <dd>
                  <span>{{ t('keys.workspaceIpConfigured', { count: blacklist.length }) }}</span>
                  <span v-if="blacklist.length" class="api-key-inspector__ip-values">
                    <code v-for="entry in blacklist" :key="`deny-${entry}`">{{ entry }}</code>
                  </span>
                </dd>
              </div>
            </div>
            <div v-if="isVisible('created_at')" class="api-key-inspector__security-divider">
              <dt>{{ t('keys.workspaceCreatedSheet') }}</dt>
              <dd>{{ formatInspectorDateTime(apiKey.created_at) }}</dd>
            </div>
            <div v-if="isVisible('last_used_at')">
              <dt>{{ t('keys.workspaceLastUsedSheet') }}</dt>
              <dd>{{ apiKey.last_used_at ? formatInspectorDateTime(apiKey.last_used_at) : t('common.notAvailable') }}</dd>
            </div>
            <div v-if="isVisible('last_used_ip')">
              <dt>{{ t('keys.workspaceLastIpSheet') }}</dt>
              <dd class="api-key-inspector__mono">{{ apiKey.last_used_ip || t('common.notAvailable') }}</dd>
            </div>
          </dl>
        </section>

        <div class="api-key-inspector__sheet-spacer" aria-hidden="true" />
      </div>
    </template>

    <footer class="api-key-inspector__footer" data-test="api-key-inspector-actions">
      <template v-if="mode === 'sheet'">
        <button type="button" class="api-key-inspector__command api-key-inspector__command--sheet-primary" @click="emit('edit', apiKey)">
          <span>{{ t('keys.editKey') }}</span>
        </button>
        <button type="button" class="api-key-inspector__command" @click="emit('use-key', apiKey)">
          <span>{{ t('keys.useKey') }}</span>
        </button>
        <button
          v-if="canShowCcsImport"
          type="button"
          class="api-key-inspector__command"
          @click="emit('import-ccs', apiKey)"
        >
          <span>{{ t('keys.workspaceImportCcs') }}</span>
        </button>
        <button type="button" class="api-key-inspector__command api-key-inspector__command--danger" @click="emit('delete', apiKey)">
          <span>{{ t('keys.deleteKey') }}</span>
        </button>
      </template>
      <template v-else>
        <button type="button" class="api-key-inspector__command" @click="emit('use-key', apiKey)">
          <KeysLucideIcon name="terminal" :size="14" aria-hidden="true" />
          <span>{{ t('keys.useKey') }}</span>
        </button>
        <button
          v-if="canShowCcsImport"
          type="button"
          class="api-key-inspector__command"
          @click="emit('import-ccs', apiKey)"
        >
          <KeysLucideIcon name="uploadCloud" :size="14" aria-hidden="true" />
          <span>{{ t('keys.workspaceImportCcs') }}</span>
        </button>
        <button type="button" class="api-key-inspector__command api-key-inspector__command--edit" @click="emit('edit', apiKey)">
          <KeysLucideIcon name="penLine" :size="14" aria-hidden="true" />
          <span>{{ t('keys.editKey') }}</span>
        </button>
        <button type="button" class="api-key-inspector__command api-key-inspector__command--danger" @click="emit('delete', apiKey)">
          <KeysLucideIcon name="trash2" :size="14" aria-hidden="true" />
          <span>{{ t('keys.deleteKey') }}</span>
        </button>
      </template>
    </footer>
  </aside>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import CreditAmount from '@/components/common/CreditAmount.vue'
import EndpointPopover from '@/components/keys/EndpointPopover.vue'
import KeysLucideIcon from '@/components/keys/KeysLucideIcon.vue'
import type { BatchApiKeyUsageStats } from '@/api/usage'
import type { ApiKey, CustomEndpoint, PublicSettings } from '@/types'
import { formatDateTime } from '@/utils/format'
import { maskApiKey } from '@/utils/maskApiKey'

interface Props {
  apiKey: ApiKey
  usage?: BatchApiKeyUsageStats
  userGroupRate?: number | null
  publicSettings?: PublicSettings | null
  copied?: boolean
  statusUpdating?: boolean
  now?: Date
  showCcsImport?: boolean
  mode?: 'inline' | 'sheet'
  visibleColumns?: string[]
}

const props = withDefaults(defineProps<Props>(), {
  usage: undefined,
  userGroupRate: null,
  publicSettings: null,
  copied: false,
  statusUpdating: false,
  now: () => new Date(),
  showCcsImport: true,
  mode: 'inline',
  visibleColumns: undefined
})

const emit = defineEmits<{
  (event: 'copy-key', key: ApiKey): void
  (event: 'toggle-status', key: ApiKey): void
  (event: 'change-group', key: ApiKey, mouseEvent: MouseEvent): void
  (event: 'reset-quota', key: ApiKey): void
  (event: 'reset-rate-limit', key: ApiKey): void
  (event: 'use-key', key: ApiKey): void
  (event: 'import-ccs', key: ApiKey): void
  (event: 'edit', key: ApiKey): void
  (event: 'delete', key: ApiKey): void
}>()

const { t } = useI18n()
const inspectorRef = ref<HTMLElement | null>(null)
const revealKey = ref(false)

const titleId = computed(() => `api-key-inspector-title-${props.apiKey.id}`)
const visibleColumnSet = computed(() =>
  props.visibleColumns ? new Set(props.visibleColumns) : null
)
const isVisible = (key: string) => visibleColumnSet.value?.has(key) ?? true
const hasVisibleAuditFields = computed(() =>
  ['expires_at', 'created_at', 'last_used_at', 'last_used_ip'].some(isVisible)
)
const maskedKey = computed(() => maskApiKey(props.apiKey.key))
const isActive = computed(() => props.apiKey.status === 'active')
const isExpired = computed(() => Boolean(
  props.apiKey.expires_at && new Date(props.apiKey.expires_at).getTime() < props.now.getTime()
))

const finiteNumber = (value: unknown) =>
  typeof value === 'number' && Number.isFinite(value) ? value : 0

const quotaValue = computed(() => finiteNumber(props.apiKey.quota))
const quotaUsedValue = computed(() => finiteNumber(props.apiKey.quota_used))
const hasQuotaLimit = computed(() => quotaValue.value > 0)
const quotaAvailableValue = computed(() => Math.max(quotaValue.value - quotaUsedValue.value, 0))
const quotaPercent = computed(() => {
  if (!hasQuotaLimit.value) return 0
  return Math.min(Math.max((quotaUsedValue.value / quotaValue.value) * 100, 0), 100)
})
const quotaMeterClass = computed(() => meterClass(quotaPercent.value))

const todayCost = computed(() => finiteNumber(props.usage?.today_actual_cost))
const totalCost = computed(() => finiteNumber(props.usage?.total_actual_cost))
const rateLimits = computed(() => [
  {
    label: '5h',
    inlineLabel: t('keys.workspaceRate5hInline'),
    sheetLabel: t('keys.workspaceRate5hSheet'),
    used: finiteNumber(props.apiKey.usage_5h),
    total: finiteNumber(props.apiKey.rate_limit_5h),
    resetAt: props.apiKey.reset_5h_at
  },
  {
    label: '1d',
    inlineLabel: t('keys.workspaceRate1dInline'),
    sheetLabel: t('keys.workspaceRate1dSheet'),
    used: finiteNumber(props.apiKey.usage_1d),
    total: finiteNumber(props.apiKey.rate_limit_1d),
    resetAt: props.apiKey.reset_1d_at
  },
  {
    label: '7d',
    inlineLabel: t('keys.workspaceRate7dInline'),
    sheetLabel: t('keys.workspaceRate7dSheet'),
    used: finiteNumber(props.apiKey.usage_7d),
    total: finiteNumber(props.apiKey.rate_limit_7d),
    resetAt: props.apiKey.reset_7d_at
  }
].map((limit) => ({
  ...limit,
  limited: limit.total > 0,
  percent: limit.total > 0
    ? Math.min(Math.max((limit.used / limit.total) * 100, 0), 100)
    : 0
})))

const hasRateLimitUsage = computed(() => rateLimits.value.some((limit) => limit.used > 0))
const whitelist = computed(() => props.apiKey.ip_whitelist?.filter(Boolean) ?? [])
const blacklist = computed(() => props.apiKey.ip_blacklist?.filter(Boolean) ?? [])
const hasIpRestriction = computed(() => whitelist.value.length > 0 || blacklist.value.length > 0)

const configuredApiBaseUrl = computed(() => props.publicSettings?.api_base_url?.trim() ?? '')
const configuredCustomEndpoints = computed<CustomEndpoint[]>(() =>
  (props.publicSettings?.custom_endpoints ?? []).filter((endpoint) => endpoint.endpoint?.trim())
)
const hasConfiguredEndpoints = computed(() =>
  configuredApiBaseUrl.value.length > 0 || configuredCustomEndpoints.value.length > 0
)
const canShowCcsImport = computed(() =>
  props.showCcsImport && !props.publicSettings?.hide_ccs_import_button
)

function meterClass(percent: number) {
  if (percent >= 100) return 'api-key-inspector__meter api-key-inspector__meter--danger'
  if (percent >= 80) return 'api-key-inspector__meter api-key-inspector__meter--warning'
  return 'api-key-inspector__meter'
}

function formatResetTime(resetAt: string) {
  const diff = new Date(resetAt).getTime() - props.now.getTime()
  if (!Number.isFinite(diff) || diff <= 0) return t('keys.resetNow')
  const days = Math.floor(diff / 86_400_000)
  const hours = Math.floor((diff % 86_400_000) / 3_600_000)
  const minutes = Math.floor((diff % 3_600_000) / 60_000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function formatInspectorDateTime(value: string | Date) {
  return formatDateTime(value, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }, 'sv-SE')
}

function focus() {
  inspectorRef.value?.focus()
}

watch(() => props.apiKey.id, () => {
  revealKey.value = false
})

defineExpose({ focus })
</script>

<style scoped>
.api-key-inspector {
  display: flex;
  min-width: 0;
  height: 100%;
  max-height: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-surface);
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-flat);
  font-family: var(--lx-clay-font-ui);
  letter-spacing: 0;
}

.api-key-inspector--sheet {
  width: 100%;
  min-height: 0;
  height: auto;
  max-height: 100%;
  flex: 0 1 auto;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.api-key-inspector:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: -3px;
}

.api-key-inspector__header,
.api-key-inspector__footer {
  flex: 0 0 auto;
  background: color-mix(in srgb, var(--lx-clay-surface) 90%, var(--lx-clay-accent-soft));
}

.api-key-inspector__header {
  display: flex;
  min-height: 4.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 0.875rem 1rem 0.875rem 1.25rem;
}

.api-key-inspector__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}

.api-key-inspector__identity-icon {
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  box-shadow: var(--lx-clay-shadow-inset);
}

.api-key-inspector__identity-copy {
  min-width: 0;
}

.api-key-inspector__title {
  overflow: hidden;
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1.05rem;
  font-weight: 750;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__id {
  margin: 0.2rem 0 0;
  color: var(--lx-clay-text-muted);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  line-height: 1.25rem;
}

.api-key-inspector__icon-button {
  display: inline-grid;
  width: 2.75rem;
  height: 2.75rem;
  flex: 0 0 auto;
  place-items: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-muted);
  transition: color 150ms ease, background-color 150ms ease;
}

.api-key-inspector__icon-button:hover {
  color: var(--lx-clay-text);
  background: color-mix(in srgb, var(--lx-clay-text) 7%, transparent);
}

.api-key-inspector__icon-button:focus-visible,
.api-key-inspector__group-button:focus-visible,
.api-key-inspector__switch:focus-visible,
.api-key-inspector__text-button:focus-visible,
.api-key-inspector__command:focus-visible {
  outline: 2px solid var(--lx-clay-accent);
  outline-offset: 2px;
}

.api-key-inspector__content {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-color: var(--lx-clay-border-strong) transparent;
  scrollbar-width: thin;
}

.api-key-inspector__lead,
.api-key-inspector__section {
  padding: 1rem 1.25rem;
}

.api-key-inspector__section {
  border-top: 1px solid var(--lx-clay-border);
}

.api-key-inspector__status-row,
.api-key-inspector__section-heading,
.api-key-inspector__rate-topline,
.api-key-inspector__ip-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.api-key-inspector__status-copy {
  display: inline-flex;
  min-height: 2.75rem;
  align-items: center;
  gap: 0.55rem;
}

.api-key-inspector__status-dot {
  width: 0.5rem;
  height: 0.5rem;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--lx-clay-text-muted);
}

.api-key-inspector__status-dot[data-status='active'] {
  background: var(--lx-clay-success-bright);
}

.api-key-inspector__status-dot[data-status='quota_exhausted'] {
  background: var(--lx-clay-warning-bright);
}

.api-key-inspector__status-dot[data-status='expired'] {
  background: var(--lx-clay-danger);
}

.api-key-inspector__status-label {
  color: var(--lx-clay-text-secondary);
  font-size: 0.875rem;
  font-weight: 700;
}

.api-key-inspector__status-label[data-status='active'] {
  color: var(--lx-clay-success-text);
}

.api-key-inspector__status-label[data-status='quota_exhausted'] {
  color: var(--lx-clay-warning);
}

.api-key-inspector__status-label[data-status='expired'] {
  color: var(--lx-clay-danger);
}

.api-key-inspector__switch {
  display: grid;
  width: 3.5rem;
  height: 2.75rem;
  place-items: center;
  border-radius: 999px;
}

.api-key-inspector__switch:disabled {
  cursor: wait;
  opacity: 0.58;
}

.api-key-inspector__switch > span {
  position: relative;
  display: block;
  width: 2.5rem;
  height: 1.375rem;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  transition: background-color 180ms ease;
}

.api-key-inspector__switch > .api-key-inspector__switch-track--active {
  background: var(--lx-clay-accent);
}

.api-key-inspector__switch > span > span {
  position: absolute;
  top: 0.1875rem;
  left: 0.1875rem;
  width: 1rem;
  height: 1rem;
  border-radius: 50%;
  background: var(--workspace-light-surface);
  box-shadow: 0 1px 3px rgb(25 18 35 / 24%);
  transition: transform 180ms ease;
}

.api-key-inspector__switch > span > .api-key-inspector__switch-thumb--active {
  transform: translateX(1.125rem);
}

.api-key-inspector__group-button {
  display: flex;
  width: 100%;
  min-height: 2.75rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.45rem;
  border-radius: var(--lx-clay-radius-control);
  padding: 0.35rem 0.5rem;
  color: var(--lx-clay-text-secondary);
  text-align: left;
  transition: color 150ms ease, background-color 150ms ease;
}

.api-key-inspector__group-button:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.api-key-inspector__group-label {
  flex: 0 0 auto;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  font-weight: 700;
  text-transform: uppercase;
}

.api-key-inspector__group-value {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.4rem;
  font-size: 0.875rem;
}

.api-key-inspector__section-heading {
  min-height: 2.75rem;
}

.api-key-inspector__eyebrow {
  margin: 0;
  color: var(--lx-clay-text);
  font-size: 0.825rem;
  font-weight: 750;
  line-height: 1.3;
}

.api-key-inspector__section-note {
  margin: 0.2rem 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.72rem;
  line-height: 1.35;
}

.api-key-inspector__key-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 2.75rem 2.75rem;
  align-items: center;
  margin-top: 0.55rem;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: color-mix(in srgb, var(--lx-clay-recessed) 62%, var(--lx-clay-surface));
  box-shadow: var(--lx-clay-shadow-inset);
}

.api-key-inspector__key-field code {
  min-width: 0;
  overflow: hidden;
  padding: 0.75rem 0.85rem;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.78rem;
  line-height: 1.25rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__key-action {
  border-left: 1px solid var(--lx-clay-border);
  border-radius: 0;
}

.api-key-inspector__key-action:last-child {
  border-radius: 0 var(--lx-clay-radius-control) var(--lx-clay-radius-control) 0;
}

.api-key-inspector__key-action--copied {
  color: var(--lx-clay-success-text);
}

.api-key-inspector__endpoints {
  margin-top: 0.75rem;
}

.api-key-inspector__endpoints :deep(> div) {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.api-key-inspector__endpoints :deep(> div > div) {
  min-width: 0;
  overflow: hidden;
}

.api-key-inspector__endpoints :deep(code) {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__text-button {
  display: inline-flex;
  min-height: 2.75rem;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.4rem;
  border-radius: var(--lx-clay-radius-control);
  padding: 0 0.7rem;
  color: var(--lx-clay-accent-deep);
  font-size: 0.78rem;
  font-weight: 700;
  transition: color 150ms ease, background-color 150ms ease;
}

.api-key-inspector__text-button:hover:not(:disabled) {
  background: var(--lx-clay-accent-soft);
}

.api-key-inspector__text-button:disabled {
  cursor: not-allowed;
  color: var(--lx-clay-text-subtle);
  opacity: 0.62;
}

.api-key-inspector__metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin: 0.6rem 0 0;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  background: color-mix(in srgb, var(--lx-clay-recessed) 48%, var(--lx-clay-surface));
}

.api-key-inspector__metric-grid--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.api-key-inspector__metric-grid > div {
  min-width: 0;
  padding: 0.72rem;
}

.api-key-inspector__metric-grid > div + div {
  border-left: 1px solid var(--lx-clay-border);
}

.api-key-inspector__metric-grid dt,
.api-key-inspector__rate-item dt,
.api-key-inspector__detail-list dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.69rem;
  font-weight: 650;
  line-height: 1.2;
}

.api-key-inspector__metric-grid dd,
.api-key-inspector__rate-item dd,
.api-key-inspector__detail-list dd {
  min-width: 0;
  margin: 0.35rem 0 0;
  color: var(--lx-clay-text);
  font-size: 0.78rem;
  font-weight: 700;
  line-height: 1.35;
}

.api-key-inspector__metric-grid dd {
  overflow: hidden;
  text-overflow: ellipsis;
}

.api-key-inspector__number {
  font-size: 1rem !important;
  font-variant-numeric: tabular-nums;
}

.api-key-inspector__progress {
  height: 0.4rem;
  overflow: hidden;
  margin-top: 0.65rem;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
}

.api-key-inspector__progress--small {
  height: 0.3rem;
  margin-top: 0.65rem;
}

.api-key-inspector__meter {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--lx-clay-accent);
  transition: width 220ms ease;
}

.api-key-inspector__meter--warning {
  background: var(--lx-clay-warning-bright);
}

.api-key-inspector__meter--danger {
  background: var(--lx-clay-danger);
}

.api-key-inspector__rate-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.55rem;
  margin-top: 0.6rem;
}

.api-key-inspector__rate-item {
  min-width: 0;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  padding: 0.75rem;
  background: color-mix(in srgb, var(--lx-clay-recessed) 45%, var(--lx-clay-surface));
}

.api-key-inspector__rate-topline strong {
  color: var(--lx-clay-text);
  font-size: 0.82rem;
}

.api-key-inspector__rate-percent,
.api-key-inspector__unlimited {
  color: var(--lx-clay-text-muted);
  font-size: 0.68rem;
  font-weight: 650;
}

.api-key-inspector__rate-item dl {
  display: grid;
  gap: 0.55rem;
  margin: 0.7rem 0 0;
}

.api-key-inspector__rate-item dl > div {
  min-width: 0;
}

.api-key-inspector__detail-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0;
  margin: 0.55rem 0 0;
}

.api-key-inspector__detail-list > div {
  min-width: 0;
  border-top: 1px dashed var(--lx-clay-border);
  padding: 0.7rem 0;
}

.api-key-inspector__detail-list > div:nth-child(odd) {
  padding-right: 0.75rem;
}

.api-key-inspector__detail-list > div:nth-child(even) {
  border-left: 1px dashed var(--lx-clay-border);
  padding-left: 0.75rem;
}

.api-key-inspector__detail-list dd {
  overflow-wrap: anywhere;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.api-key-inspector__mono {
  font-family: var(--lx-clay-font-mono);
}

.api-key-inspector__danger-text {
  color: var(--lx-clay-danger) !important;
}

.api-key-inspector__ip-groups {
  display: grid;
  gap: 0.9rem;
  margin-top: 0.55rem;
}

.api-key-inspector__ip-heading {
  justify-content: flex-start;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-weight: 700;
}

.api-key-inspector__ip-heading > span:last-child {
  margin-left: auto;
  color: var(--lx-clay-text-muted);
  font-variant-numeric: tabular-nums;
}

.api-key-inspector__ip-dot {
  width: 0.45rem;
  height: 0.45rem;
  flex: 0 0 auto;
  border-radius: 50%;
}

.api-key-inspector__ip-dot--allow {
  background: var(--lx-clay-success-bright);
}

.api-key-inspector__ip-dot--deny {
  background: var(--lx-clay-danger);
}

.api-key-inspector__ip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.5rem;
}

.api-key-inspector__ip-list code {
  max-width: 100%;
  overflow-wrap: anywhere;
  border: 1px solid var(--lx-clay-border);
  border-radius: 0.45rem;
  padding: 0.3rem 0.48rem;
  color: var(--lx-clay-text-secondary);
  background: color-mix(in srgb, var(--lx-clay-recessed) 52%, var(--lx-clay-surface));
  font-family: var(--lx-clay-font-mono);
  font-size: 0.7rem;
}

.api-key-inspector__empty-value {
  margin: 0.45rem 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

.api-key-inspector__footer {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.35rem;
  border-top: 1px solid var(--lx-clay-border);
  padding: 0.7rem;
}

.api-key-inspector__command {
  display: inline-flex;
  min-width: 0;
  min-height: 2.75rem;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  border-radius: var(--lx-clay-radius-button);
  padding: 0.45rem 0.5rem;
  color: var(--lx-clay-text-secondary);
  font-size: 0.76rem;
  font-weight: 700;
  line-height: 1.15;
  transition: color 150ms ease, background-color 150ms ease, box-shadow 150ms ease;
}

.api-key-inspector__command span {
  min-width: 0;
  overflow-wrap: anywhere;
}

.api-key-inspector__command:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.api-key-inspector__command--primary {
  color: var(--lx-clay-on-accent);
  background: linear-gradient(145deg, var(--lx-clay-accent-gradient-start), var(--lx-clay-accent-deep));
  box-shadow: var(--lx-clay-shadow-primary);
}

.api-key-inspector__command--primary:hover {
  color: var(--lx-clay-on-accent);
  background: linear-gradient(145deg, var(--lx-clay-accent), var(--lx-clay-accent-deepest));
}

.api-key-inspector__command--danger:hover {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

@media (max-width: 639px) {
  .api-key-inspector__header {
    padding-left: 1rem;
  }

  .api-key-inspector__lead,
  .api-key-inspector__section {
    padding-right: 1rem;
    padding-left: 1rem;
  }

  .api-key-inspector__rate-grid,
  .api-key-inspector__metric-grid--quota {
    grid-template-columns: 1fr;
  }

  .api-key-inspector__metric-grid--quota > div + div {
    border-top: 1px solid var(--lx-clay-border);
    border-left: 0;
  }

  .api-key-inspector__rate-item dl {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .api-key-inspector__footer {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (prefers-reduced-motion: reduce) {
  .api-key-inspector__icon-button,
  .api-key-inspector__group-button,
  .api-key-inspector__switch > span,
  .api-key-inspector__switch > span > span,
  .api-key-inspector__text-button,
  .api-key-inspector__meter,
  .api-key-inspector__command {
    transition: none;
  }
}
</style>

<style scoped>
/* Superdesign scheme B: persistent desktop inspector and mobile detail sheet. */
.api-key-inspector {
  display: flex;
  min-width: 0;
  min-height: 0;
  height: 100%;
  max-height: 100%;
  flex-direction: column;
  overflow: hidden;
  border: 0;
  border-radius: 0;
  outline: none;
  color: var(--workspace-text);
  background: var(--workspace-canvas);
  box-shadow: none;
  font-family: "DM Sans", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0;
  -webkit-font-smoothing: antialiased;
}

.api-key-inspector--sheet {
  width: 100%;
  height: auto;
  max-height: 100%;
  flex: 1 1 0;
  border: 0;
  border-radius: 0;
  background: var(--workspace-canvas);
  box-shadow: none;
}

.api-key-inspector:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: -2px;
}

.api-key-inspector__content {
  display: flex;
  min-height: 0;
  flex: 1 1 auto;
  flex-direction: column;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-color: var(--workspace-border-strong) transparent;
  scrollbar-width: thin;
}

.api-key-inspector__content::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}

.api-key-inspector__content::-webkit-scrollbar-thumb {
  border-radius: 10px;
  background: var(--workspace-border-strong);
}

.api-key-inspector__content--inline {
  gap: 32px;
  padding: 32px;
  background: var(--workspace-canvas);
}

.api-key-inspector__content--sheet {
  gap: 32px;
  padding: 24px;
  background: var(--workspace-canvas);
}

.api-key-inspector__inline-overview {
  display: grid;
  gap: 24px;
}

.api-key-inspector__inline-identity {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0;
}

.api-key-inspector__title {
  overflow: hidden;
  margin: 0;
  color: var(--workspace-text);
  font-family: inherit;
  font-size: 24px;
  font-weight: 900;
  line-height: 24px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__id {
  margin: 8px 0 0;
  color: var(--workspace-text-muted);
  font-family: inherit;
  font-size: 11px;
  font-weight: 900;
  line-height: 16.5px;
  text-transform: uppercase;
}

.api-key-inspector__status-pill {
  display: inline-flex;
  min-height: 0;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--workspace-border);
  border-radius: 999px;
  padding: 8px 16px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-size: 11px;
  font-weight: 900;
  line-height: 16.5px;
}

.api-key-inspector__status-pill > span {
  width: 6px;
  height: 6px;
  flex: 0 0 6px;
  border-radius: 50%;
  background: var(--workspace-text-muted);
}

.api-key-inspector__status-pill[data-status='active'] {
  border-color: #a7f3d0;
  color: #065f46;
  background: #d1fae5;
}

.api-key-inspector__status-pill[data-status='active'] > span {
  background: #10b981;
  animation: api-key-inspector-pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.api-key-inspector__status-pill[data-status='quota_exhausted'] {
  border-color: #fde68a;
  color: #92400e;
  background: #fef3c7;
}

.api-key-inspector__status-pill[data-status='quota_exhausted'] > span {
  background: #f59e0b;
}

.api-key-inspector__status-pill[data-status='expired'] {
  border-color: #fecaca;
  color: #b91c1c;
  background: #fee2e2;
}

.api-key-inspector__status-pill[data-status='expired'] > span {
  background: #ef4444;
}

.api-key-inspector__credential-stack {
  display: grid;
  gap: 16px;
}

.api-key-inspector__field-label {
  display: block;
  margin: 0 0 8px;
  color: var(--workspace-text-muted);
  font-size: 10px;
  font-weight: 900;
  line-height: 15px;
  text-transform: uppercase;
}

.api-key-inspector__key-field {
  display: grid;
  height: 56px;
  grid-template-columns: minmax(0, 1fr) 40px 40px;
  align-items: center;
  gap: 4px;
  margin: 0;
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  padding: 0 20px;
  background: var(--workspace-card-surface);
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
}

.api-key-inspector__key-field code {
  min-width: 0;
  overflow: hidden;
  padding: 0 8px 0 0;
  color: var(--workspace-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 14px;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__icon-button {
  display: inline-grid;
  width: 40px;
  height: 40px;
  min-height: 40px;
  flex: 0 0 40px;
  place-items: center;
  border: 0;
  border-radius: 8px;
  padding: 0;
  color: var(--workspace-text-muted);
  background: transparent;
  transition: color 150ms ease, background-color 150ms ease;
}

.api-key-inspector__icon-button:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.api-key-inspector__icon-button--copy:hover {
  color: #7c3aed;
}

.api-key-inspector__icon-button.is-copied {
  color: #059669;
}

.api-key-inspector__icon-button:focus-visible,
.api-key-inspector__switch:focus-visible,
.api-key-inspector__text-button:focus-visible,
.api-key-inspector__command:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.api-key-inspector__endpoint-well {
  min-width: 0;
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  padding: 20px;
  background: var(--workspace-surface-subtle);
  box-shadow:
    inset 6px 6px 12px rgb(91 80 112 / 0.08),
    inset -6px -6px 12px rgb(255 255 255 / 0.7);
}

.api-key-inspector__endpoint-well :deep(> div) {
  min-width: 0;
}

.api-key-inspector__endpoint-well :deep(code) {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-inspector__empty-endpoint {
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: 12px;
  font-style: italic;
  line-height: 16px;
}

.api-key-inspector__divider {
  width: 100%;
  height: 1px;
  flex: 0 0 1px;
  background: var(--workspace-border);
}

.api-key-inspector__section {
  display: grid;
  gap: 24px;
  border: 0;
  padding: 0;
}

.api-key-inspector__section-heading,
.api-key-inspector__sheet-section-heading,
.api-key-inspector__rate-topline {
  display: flex;
  min-height: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.api-key-inspector__section-heading h3,
.api-key-inspector__sheet-heading {
  margin: 0;
  color: var(--workspace-text-secondary);
  font-size: 11px;
  font-weight: 900;
  line-height: 16.5px;
  text-transform: uppercase;
}

.api-key-inspector__text-button {
  display: inline-flex;
  min-height: auto;
  flex: 0 0 auto;
  align-items: center;
  border: 0;
  border-radius: 0;
  padding: 0;
  color: #7c3aed;
  background: transparent;
  font-size: 10px;
  font-weight: 900;
  line-height: 15px;
  text-transform: none;
  transition: color 150ms ease;
}

.api-key-inspector__text-button:hover:not(:disabled) {
  color: #6d28d9;
  background: transparent;
  text-decoration: underline;
}

.api-key-inspector__text-button:disabled {
  cursor: not-allowed;
  color: var(--workspace-text-muted);
  opacity: 0.55;
}

.api-key-inspector__quota-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin: 0;
}

.api-key-inspector__quota-grid > div {
  min-width: 0;
  container-type: inline-size;
  border: 1px solid var(--workspace-border);
  border-radius: 22px;
  padding: 20px;
  background: var(--workspace-card-surface);
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
}

.api-key-inspector__quota-grid dt {
  color: var(--workspace-text-muted);
  font-size: 10px;
  font-weight: 700;
  line-height: 15px;
  text-transform: uppercase;
}

.api-key-inspector__quota-grid dd {
  min-width: 0;
  overflow: hidden;
  margin: 4px 0 0;
  color: var(--workspace-text);
  font-size: 24px;
  font-weight: 900;
  line-height: 32px;
}

.api-key-inspector__quota-grid dd :deep([data-testid='credit-amount']) {
  max-width: 100%;
}

.api-key-inspector__quota-grid > div > p {
  margin: 6px 0 0;
  color: var(--workspace-text-muted);
  font-size: 9px;
  font-weight: 900;
  line-height: 13.5px;
  text-transform: uppercase;
}

@container (max-width: 150px) {
  .api-key-inspector--inline .api-key-inspector__quota-grid dd {
    font-size: 18px;
    line-height: 24px;
  }

  .api-key-inspector--inline .api-key-inspector__quota-grid dd :deep([data-testid='credit-amount']) {
    column-gap: 2px;
  }
}

@container (max-width: 120px) {
  .api-key-inspector--inline .api-key-inspector__quota-grid dd {
    font-size: 16px;
  }
}

.api-key-inspector__usage-rows,
.api-key-inspector__usage-card {
  display: grid;
  margin: 0;
}

.api-key-inspector__usage-rows {
  gap: 16px;
  padding: 0 4px;
}

.api-key-inspector__usage-card {
  gap: 12px;
}

.api-key-inspector__usage-rows > div,
.api-key-inspector__usage-card > div {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.api-key-inspector__usage-rows dt,
.api-key-inspector__usage-card dt {
  color: var(--workspace-text-secondary);
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
  text-transform: uppercase;
}

.api-key-inspector__usage-rows dd,
.api-key-inspector__usage-card dd {
  min-width: 0;
  margin: 0;
  color: var(--workspace-text);
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
}

.api-key-inspector__usage-card dd {
  font-weight: 900;
}

.api-key-inspector__rate-list {
  display: grid;
  gap: 24px;
}

.api-key-inspector__rate-item {
  display: grid;
  min-width: 0;
  gap: 8px;
  border: 0;
  border-radius: 0;
  padding: 0;
  background: transparent;
}

.api-key-inspector__rate-topline strong,
.api-key-inspector__rate-values {
  font-size: 10px;
  font-weight: 900;
  line-height: 15px;
  text-transform: uppercase;
}

.api-key-inspector__rate-topline strong {
  color: var(--workspace-text-muted);
}

.api-key-inspector__rate-values {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  color: var(--workspace-text);
}

.api-key-inspector__progress {
  height: 8px;
  overflow: hidden;
  margin: 0;
  border-radius: 999px;
  background: var(--workspace-surface-subtle);
}

.api-key-inspector__meter,
.api-key-inspector__meter--warning,
.api-key-inspector__meter--danger {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: #10b981;
  transition: width 220ms ease;
}

.api-key-inspector__reset-time {
  margin: 0;
  color: var(--workspace-text-muted);
  font-size: 9px;
  line-height: 13.5px;
  text-align: right;
}

.api-key-inspector__security-card {
  display: grid;
  gap: 20px;
  margin: 0;
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  padding: 24px;
  background: var(--workspace-card-surface);
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
}

.api-key-inspector__security-card > div:not(.api-key-inspector__ip-rules),
.api-key-inspector__ip-rules > div {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.api-key-inspector__security-card dt {
  flex: 0 0 auto;
  color: var(--workspace-text-secondary);
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
}

.api-key-inspector__security-card dd {
  min-width: 0;
  max-width: 64%;
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--workspace-text);
  font-size: 12px;
  font-weight: 900;
  line-height: 16px;
  text-align: right;
}

.api-key-inspector__security-card dd.is-danger {
  color: #dc2626;
}

.api-key-inspector__ip-rules {
  display: grid;
  gap: 20px;
}

.api-key-inspector__ip-values {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
}

.api-key-inspector__ip-values code {
  max-width: 100%;
  overflow-wrap: anywhere;
  color: inherit;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 10px;
  font-weight: 700;
  line-height: 15px;
}

.api-key-inspector__security-divider {
  border-top: 1px solid var(--workspace-border);
  padding-top: 20px;
}

.api-key-inspector__mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.api-key-inspector__sheet-section {
  display: grid;
  gap: 16px;
}

.api-key-inspector__sheet-section[data-test='api-key-inspector-quota'],
.api-key-inspector__sheet-section[data-test='api-key-inspector-rate-limits'] {
  gap: 20px;
}

.api-key-inspector__sheet-heading {
  color: var(--workspace-text-muted);
}

.api-key-inspector__sheet-section-heading .api-key-inspector__text-button {
  text-transform: uppercase;
  text-decoration: underline;
}

.api-key-inspector__sheet-card {
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  background: var(--workspace-card-surface);
}

.api-key-inspector__core-card {
  display: grid;
  gap: 20px;
  padding: 20px;
}

.api-key-inspector__sheet-status-row,
.api-key-inspector__sheet-status-control {
  display: flex;
  align-items: center;
}

.api-key-inspector__sheet-status-row {
  justify-content: space-between;
  gap: 16px;
  color: var(--workspace-text-secondary);
  font-size: 12px;
  font-weight: 700;
  line-height: 16px;
}

.api-key-inspector__sheet-status-control {
  gap: 8px;
}

.api-key-inspector__sheet-status-control strong {
  color: var(--workspace-text-secondary);
  font-size: 11px;
  font-weight: 900;
  line-height: 16.5px;
  text-transform: uppercase;
}

.api-key-inspector__sheet-status-control strong[data-status='active'] {
  color: #059669;
}

.api-key-inspector__sheet-status-control strong[data-status='quota_exhausted'] {
  color: #d97706;
}

.api-key-inspector__sheet-status-control strong[data-status='expired'] {
  color: #dc2626;
}

.api-key-inspector__switch {
  display: grid;
  width: 44px;
  height: 44px;
  min-height: 44px;
  flex: 0 0 44px;
  place-items: center;
  border: 0;
  border-radius: 999px;
  padding: 0;
  background: transparent;
}

.api-key-inspector__switch:disabled {
  cursor: wait;
  opacity: 0.58;
}

.api-key-inspector__switch > span {
  position: relative;
  display: block;
  width: 40px;
  height: 22px;
  border-radius: 999px;
  background: var(--workspace-border-strong);
  transition: background-color 180ms ease;
}

.api-key-inspector__switch > span.is-active {
  background: #7c3aed;
}

.api-key-inspector__switch > span > span {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--workspace-light-surface);
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.12);
  transition: transform 180ms ease;
}

.api-key-inspector__switch > span.is-active > span {
  transform: translateX(18px);
}

.api-key-inspector__sheet-field {
  min-width: 0;
}

.api-key-inspector--sheet .api-key-inspector__key-field {
  height: 48px;
  grid-template-columns: minmax(0, 1fr) 44px 44px;
  gap: 0;
  border: 0;
  border-radius: 12px;
  padding: 0 16px;
  background: var(--workspace-surface-subtle);
  box-shadow:
    inset 6px 6px 12px rgb(91 80 112 / 0.08),
    inset -6px -6px 12px rgb(255 255 255 / 0.7);
}

.api-key-inspector--sheet .api-key-inspector__key-field code {
  padding-right: 12px;
  font-size: 12px;
  line-height: 16px;
}

.api-key-inspector--sheet .api-key-inspector__icon-button {
  width: 44px;
  height: 44px;
  min-height: 44px;
  flex-basis: 44px;
}

.api-key-inspector--sheet .api-key-inspector__icon-button:active {
  color: #7c3aed;
}

.api-key-inspector--sheet .api-key-inspector__endpoint-well {
  border: 0;
  border-radius: 12px;
  padding: 16px;
}

.api-key-inspector--sheet .api-key-inspector__quota-grid {
  gap: 12px;
}

.api-key-inspector--sheet .api-key-inspector__quota-grid > div {
  border-radius: 16px;
  padding: 16px;
}

.api-key-inspector--sheet .api-key-inspector__quota-grid dt {
  font-size: 9px;
  line-height: 13.5px;
}

.api-key-inspector--sheet .api-key-inspector__quota-grid dd {
  margin-top: 2px;
  font-size: 20px;
  line-height: 28px;
}

.api-key-inspector--sheet .api-key-inspector__quota-grid > div:first-child dd {
  color: #6d28d9;
}

.api-key-inspector__usage-card {
  border: 1px solid var(--workspace-border);
  border-radius: 16px;
  padding: 20px;
  background: var(--workspace-card-surface);
}

.api-key-inspector--sheet .api-key-inspector__rate-list {
  gap: 24px;
  padding: 24px;
}

.api-key-inspector--sheet .api-key-inspector__rate-item {
  gap: 10px;
}

.api-key-inspector--sheet .api-key-inspector__progress {
  height: 6px;
}

.api-key-inspector--sheet .api-key-inspector__security-card {
  gap: 16px;
  padding: 20px;
  box-shadow: none;
}

.api-key-inspector--sheet .api-key-inspector__ip-rules {
  gap: 16px;
}

.api-key-inspector--sheet .api-key-inspector__security-divider {
  padding-top: 16px;
}

.api-key-inspector__sheet-spacer {
  height: 40px;
  flex: 0 0 40px;
}

.api-key-inspector__footer {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  flex: 0 0 auto;
  border-top: 1px solid var(--workspace-border);
  padding: 24px;
  background: var(--workspace-card-surface);
}

.api-key-inspector__command {
  display: inline-flex;
  min-width: 0;
  min-height: 48px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid var(--workspace-border);
  border-radius: 12px;
  padding: 0 12px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
  white-space: nowrap;
  transition: color 150ms ease, background-color 150ms ease, transform 150ms ease;
}

.api-key-inspector__command span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.api-key-inspector__command:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

.api-key-inspector__command--edit {
  border-color: transparent;
  color: #6d28d9;
  background: #f5f3ff;
}

.api-key-inspector__command--edit:hover {
  color: #6d28d9;
  background: #ede9fe;
}

.api-key-inspector__command--danger {
  border-color: transparent;
  color: #dc2626;
  background: #fef2f2;
}

.api-key-inspector__command--danger:hover {
  color: #dc2626;
  background: #fee2e2;
}

.api-key-inspector--sheet .api-key-inspector__command {
  border: 0;
  border-radius: 16px;
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
  font-weight: 900;
}

.api-key-inspector--sheet .api-key-inspector__command:active {
  transform: scale(0.95);
}

.api-key-inspector--sheet .api-key-inspector__command--sheet-primary {
  color: #fff;
  background: #7c3aed;
  box-shadow:
    0 10px 15px -3px rgb(124 58 237 / 0.3),
    0 4px 6px -4px rgb(124 58 237 / 0.3);
}

.api-key-inspector--sheet .api-key-inspector__command--sheet-primary:hover {
  color: #fff;
  background: #6d28d9;
}

.api-key-inspector--sheet .api-key-inspector__command--danger {
  color: #dc2626;
  background: #fef2f2;
}

:global(.dark) .api-key-inspector {
  color: var(--workspace-text);
  background: transparent;
}

:global(.dark) .api-key-inspector--sheet,
:global(.dark) .api-key-inspector__content--sheet {
  background: var(--workspace-canvas);
}

:global(.dark) .api-key-inspector__content--inline {
  background: transparent;
}

:global(.dark) .api-key-inspector__title,
:global(.dark) .api-key-inspector__quota-grid dd,
:global(.dark) .api-key-inspector__usage-rows dd,
:global(.dark) .api-key-inspector__usage-card dd,
:global(.dark) .api-key-inspector__rate-values,
:global(.dark) .api-key-inspector__security-card dd {
  color: var(--workspace-text);
}

:global(.dark) .api-key-inspector__key-field,
:global(.dark) .api-key-inspector__endpoint-well {
  border-color: var(--workspace-border);
  background: var(--workspace-surface-subtle);
}

:global(.dark) .api-key-inspector__key-field code {
  color: var(--workspace-text-secondary);
}

:global(.dark) .api-key-inspector__icon-button:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

:global(.dark) .api-key-inspector__endpoint-well,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__key-field {
  box-shadow:
    inset 4px 4px 10px rgb(0 0 0 / 0.3),
    inset -4px -4px 10px rgb(255 255 255 / 0.02);
}

:global(.dark) .api-key-inspector__divider {
  background: var(--workspace-border);
}

:global(.dark) .api-key-inspector__security-divider {
  border-color: var(--workspace-border);
  background: transparent;
}

:global(.dark) .api-key-inspector__quota-grid > div,
:global(.dark) .api-key-inspector__sheet-card,
:global(.dark) .api-key-inspector__usage-card {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
}

:global(.dark) .api-key-inspector--inline .api-key-inspector__security-card {
  border-color: var(--workspace-border);
  background: var(--workspace-surface-subtle);
}

:global(.dark) .api-key-inspector--sheet .api-key-inspector__core-card,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__rate-list,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__usage-card {
  border-color: var(--workspace-border);
}

:global(.dark) .api-key-inspector--sheet .api-key-inspector__security-card {
  border-color: var(--workspace-border);
}

:global(.dark) .api-key-inspector--sheet .api-key-inspector__quota-grid > div {
  border-color: var(--workspace-border);
}

:global(.dark) .api-key-inspector--sheet .api-key-inspector__quota-grid > div:first-child dd {
  color: #a78bfa;
}

:global(.dark) .api-key-inspector__progress,
:global(.dark) .api-key-inspector__switch > span {
  background: var(--workspace-surface-subtle);
}

:global(.dark) .api-key-inspector__switch > span.is-active {
  background: #7c3aed;
}

:global(.dark) .api-key-inspector__footer {
  border-color: var(--workspace-border);
  background: var(--workspace-card-surface);
}

:global(.dark) .api-key-inspector__command,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__command {
  border-color: var(--workspace-border);
  color: var(--workspace-text-secondary);
  background: var(--workspace-surface-subtle);
}

:global(.dark) .api-key-inspector__command:hover,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__command:hover {
  color: var(--workspace-text);
  background: var(--workspace-hover);
}

:global(.dark) .api-key-inspector__command--edit {
  border-color: transparent;
  color: #6d28d9;
  background: #f5f3ff;
}

:global(.dark) .api-key-inspector__command--danger,
:global(.dark) .api-key-inspector--sheet .api-key-inspector__command--danger {
  border-color: transparent;
  color: #dc2626;
  background: #fef2f2;
}

:global(.dark) .api-key-inspector--sheet .api-key-inspector__command--sheet-primary {
  color: #fff;
  background: #7c3aed;
}

@keyframes api-key-inspector-pulse {
  50% {
    opacity: 0.5;
  }
}

@media (prefers-reduced-motion: reduce) {
  .api-key-inspector__status-pill[data-status='active'] > span {
    animation: none;
  }

  .api-key-inspector__icon-button,
  .api-key-inspector__switch > span,
  .api-key-inspector__switch > span > span,
  .api-key-inspector__text-button,
  .api-key-inspector__meter,
  .api-key-inspector__command {
    transition: none;
  }
}
</style>
