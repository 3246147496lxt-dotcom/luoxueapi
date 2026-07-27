<template>
  <section
    ref="inspectorRef"
    class="account-inspector"
    :class="`account-inspector--${mode}`"
    :role="mode === 'inline' ? 'complementary' : 'dialog'"
    :aria-modal="mode === 'inline' ? undefined : true"
    :aria-labelledby="account ? titleId : undefined"
    tabindex="-1"
    data-testid="account-inspector"
    @keydown="handleKeydown"
  >
    <template v-if="account">
      <header class="account-inspector__header">
        <div class="account-inspector__identity">
          <h2 :id="titleId" :title="account.name">{{ account.name }}</h2>
          <p
            v-if="accountEmail"
            class="account-inspector__email"
            :title="accountEmail"
            data-testid="account-inspector-email"
          >
            {{ accountEmail }}
          </p>
          <div class="account-inspector__identity-meta">
            <span class="account-inspector__id">ID: {{ account.id }}</span>
            <PlatformTypeBadge
              :platform="account.platform"
              :type="account.type"
              :auth-mode="openAIAuthMode"
              :plan-type="accountPlanType"
              :privacy-mode="privacyMode"
              :subscription-expires-at="subscriptionExpiresAt"
              compact
            />
          </div>
        </div>

        <div class="account-inspector__header-actions">
          <label class="account-inspector__scheduling">
            <span>{{ t('admin.accounts.workbench.scheduling') }}</span>
            <button
              type="button"
              class="account-inspector__switch"
              role="switch"
              :aria-checked="account.schedulable"
              :aria-label="account.schedulable
                ? t('admin.accounts.schedulableEnabled')
                : t('admin.accounts.schedulableDisabled')"
              :disabled="togglingSchedulable"
              data-testid="account-inspector-scheduling"
              @click.stop="emit('toggleSchedulable', account)"
            >
              <span />
            </button>
          </label>
          <button
            type="button"
            class="account-inspector__icon-button"
            :title="t('admin.accounts.moreActions')"
            :aria-label="t('admin.accounts.moreActions')"
            data-testid="account-inspector-more"
            @click.stop="emit('more', account, $event)"
          >
            <Icon name="more" size="sm" />
          </button>
          <button
            type="button"
            class="account-inspector__icon-button"
            :title="t('admin.accounts.workbench.closeDetails')"
            :aria-label="t('admin.accounts.workbench.closeDetails')"
            data-testid="account-inspector-close"
            @click="emit('close')"
          >
            <Icon name="x" size="sm" />
          </button>
        </div>
      </header>

      <div ref="contentRef" class="account-inspector__content">
        <section
          v-if="recoveryRows.length"
          class="account-inspector__section account-inspector__section--recovery"
          data-testid="account-inspector-recovery"
        >
          <div class="account-inspector__section-title">
            <span class="account-inspector__section-icon account-inspector__section-icon--recovery">
              <Icon name="exclamationTriangle" size="sm" />
            </span>
            <h3>{{ t('admin.accounts.workbench.recovery') }}</h3>
          </div>
          <dl class="account-inspector__recovery-list">
            <div v-for="row in recoveryRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd>{{ row.value }}</dd>
            </div>
          </dl>
        </section>

        <section
          class="account-inspector__section account-inspector__section--quota"
          :aria-busy="quotaQueryLoading"
          data-testid="account-inspector-quota"
        >
          <div class="account-inspector__section-title">
            <Icon
              name="lucideBarChart3"
              size="sm"
              :stroke-width="2"
              class="account-inspector__quota-title-icon text-purple-500"
              aria-hidden="true"
            />
            <h3>{{ t('admin.accounts.workbench.quotaOverview') }}</h3>
          </div>

          <AccountUsageCell
            ref="usageCellRef"
            :key="account.id"
            :account="account"
            :today-stats="todayStats"
            :today-stats-loading="todayStatsLoading"
            :manual-refresh-token="usageRefreshToken"
            display-mode="overview"
          />

          <div class="account-inspector__concurrency" data-testid="account-inspector-concurrency">
            <span>{{ t('admin.accounts.workbench.localConcurrencyCapacity') }}</span>
            <strong>
              {{
                t('admin.accounts.workbench.concurrencySlots', {
                  current: account.current_concurrency || 0,
                  max: account.concurrency || 0
                })
              }}
            </strong>
          </div>

          <OpenAIQuotaResetCell
            :account="account"
            class="account-inspector__quota-actions"
            @quota-updated="queryQuota"
          >
            <template #pre-actions>
              <button
                type="button"
                class="account-inspector__quota-action"
                :disabled="quotaQueryLoading"
                :title="t('admin.accounts.usageWindow.activeQuery')"
                data-testid="account-inspector-query-quota"
                @click="queryQuota"
              >
                <Icon
                  name="refresh"
                  size="xs"
                  :class="{ 'animate-spin': quotaQueryLoading }"
                />
                {{ t('admin.accounts.usageWindow.activeQuery') }}
              </button>
            </template>
          </OpenAIQuotaResetCell>
        </section>

        <section class="account-inspector__section" data-testid="account-inspector-usage">
          <div class="account-inspector__section-title">
            <span class="account-inspector__section-icon account-inspector__section-icon--usage">
              <Icon name="chart" size="sm" />
            </span>
            <h3>{{ t('admin.accounts.workbench.recentUsage') }}</h3>
          </div>

          <div v-if="todayStatsLoading && !todayStats" class="account-inspector__loading" role="status">
            <span />
            <span />
            <span />
          </div>
          <p v-else-if="todayStatsError && !todayStats" class="account-inspector__error" role="alert">
            {{ todayStatsError }}
          </p>
          <dl v-else class="account-inspector__usage-grid">
            <div>
              <dt>{{ t('admin.accounts.workbench.lastActivity') }}</dt>
              <dd>{{ formatRelativeTime(account.last_used_at) }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.accounts.workbench.todayRequests') }}</dt>
              <dd>{{ todayStats ? formatNumber(todayStats.requests) : placeholder }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.accounts.workbench.todayTokens') }}</dt>
              <dd>{{ todayStats ? formatNumber(todayStats.tokens) : placeholder }}</dd>
            </div>
            <div>
              <dt>{{ t('admin.accounts.workbench.accountUserCost') }}</dt>
              <dd v-if="todayStats" class="inline-flex items-center gap-1">
                <span>{{ formatCurrency(todayStats.cost) }}</span>
                <span aria-hidden="true">/</span>
                <CreditAmount :value="(todayStats.user_cost ?? 0).toFixed(2)" icon-size="xs" />
              </dd>
              <dd v-else>{{ placeholder }}</dd>
            </div>
          </dl>
        </section>

        <section
          v-if="policyRows.length"
          class="account-inspector__section"
          data-testid="account-inspector-policies"
        >
          <div class="account-inspector__section-title">
            <span class="account-inspector__section-icon account-inspector__section-icon--policy">
              <Icon name="cog" size="sm" />
            </span>
            <h3>{{ t('admin.accounts.workbench.policies') }}</h3>
          </div>
          <dl class="account-inspector__policy-list">
            <div v-for="row in policyRows" :key="row.label">
              <dt>{{ row.label }}</dt>
              <dd :title="row.value">{{ row.value }}</dd>
            </div>
          </dl>
        </section>

        <section
          class="account-inspector__section account-inspector__section--associations"
          data-testid="account-inspector-groups-proxy"
        >
          <div class="account-inspector__section-title">
            <span class="account-inspector__section-icon account-inspector__section-icon--associations">
              <Icon name="users" size="sm" />
            </span>
            <h3>{{ t('admin.accounts.workbench.groupsAndProxy') }}</h3>
          </div>

          <div class="account-inspector__association">
            <span class="account-inspector__association-label">
              {{ t('admin.accounts.workbench.effectiveGroups') }}
            </span>
            <div
              v-if="effectiveGroups.length"
              class="account-inspector__groups"
              data-testid="account-inspector-groups"
            >
              <GroupBadge
                v-for="group in effectiveGroups"
                :key="group.id"
                class="account-inspector__group-badge"
                :name="group.name"
                :platform="group.platform"
                :subscription-type="group.subscription_type"
                :rate-multiplier="group.rate_multiplier"
                :show-rate="false"
              />
            </div>
            <span v-else class="account-inspector__association-empty">
              {{ t('admin.accounts.ungroupedGroup') }}
            </span>
          </div>

          <div class="account-inspector__association">
            <span class="account-inspector__association-label">
              {{ t('admin.accounts.workbench.proxyName') }}
            </span>
            <button
              v-if="resolvedProxy"
              type="button"
              class="account-inspector__proxy-link"
              :title="t('admin.accounts.workbench.viewProxyInManagement', { name: resolvedProxy.name })"
              :aria-label="`${t('admin.accounts.workbench.viewProxyInManagement', { name: resolvedProxy.name })}, ${proxyStatusLabel}`"
              data-testid="account-inspector-view-proxy"
              @click="emit('viewProxy', resolvedProxy.id)"
            >
              <span
                class="account-inspector__proxy-dot"
                :class="{ 'account-inspector__proxy-dot--active': resolvedProxy.status === 'active' }"
                aria-hidden="true"
              />
              <span class="account-inspector__proxy-name">{{ resolvedProxy.name }}</span>
              <span class="account-inspector__proxy-hint">
                {{ proxyStatusLabel }}
              </span>
              <Icon name="chevronRight" size="xs" aria-hidden="true" />
            </button>
            <p v-else class="account-inspector__direct-route">
              {{ t('admin.accounts.workbench.directRoute') }}
            </p>
          </div>

          <button
            v-if="account.proxy_fallback_origin_id"
            type="button"
            class="account-inspector__text-action"
            data-testid="account-inspector-revert-fallback"
            @click="emit('revertFallback', account)"
          >
            <Icon name="refresh" size="xs" />
            {{ t('admin.accounts.revertProxy') }}
          </button>
        </section>
      </div>

      <footer class="account-inspector__footer">
        <button type="button" class="account-inspector__command" data-testid="account-inspector-test" @click="emit('test', account)">
          <Icon
            name="lucidePlayCircle"
            size="xs"
            :stroke-width="2"
            class="account-inspector__command-icon text-green-500"
            aria-hidden="true"
          />
          <span>{{ t('admin.accounts.testConnection') }}</span>
        </button>
        <button type="button" class="account-inspector__command" data-testid="account-inspector-stats" @click="emit('stats', account)">
          <Icon
            name="lucideBarChartBig"
            size="xs"
            :stroke-width="2"
            class="account-inspector__command-icon text-indigo-500"
            aria-hidden="true"
          />
          <span>{{ t('admin.accounts.workbench.viewStats') }}</span>
        </button>
        <button type="button" class="account-inspector__command" data-testid="account-inspector-edit" @click="emit('edit', account)">
          <Icon
            name="lucideEdit3"
            size="xs"
            :stroke-width="2"
            class="account-inspector__command-icon text-gray-500"
            aria-hidden="true"
          />
          <span>{{ t('common.edit') }}</span>
        </button>
      </footer>
    </template>

    <div v-else class="account-inspector__empty">
      <span class="account-inspector__empty-icon">
        <Icon name="activity" size="lg" />
      </span>
      <p>{{ t('admin.accounts.workbench.selectAccount') }}</p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'
import OpenAIQuotaResetCell from '@/components/account/OpenAIQuotaResetCell.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCurrency, formatDateTime, formatNumber, formatRelativeTime } from '@/utils/format'
import type { Account, Proxy, WindowStats } from '@/types'

type InspectorMode = 'inline' | 'drawer' | 'sheet'
type ProxySummary = Pick<Proxy, 'id' | 'name' | 'status'>

const props = withDefaults(defineProps<{
  account: Account | null
  proxyTelemetry?: ProxySummary | null
  todayStats?: WindowStats | null
  todayStatsLoading?: boolean
  todayStatsError?: string | null
  usageRefreshToken?: number
  togglingSchedulable?: boolean
  mode?: InspectorMode
}>(), {
  proxyTelemetry: null,
  todayStats: null,
  todayStatsLoading: false,
  todayStatsError: null,
  usageRefreshToken: 0,
  togglingSchedulable: false,
  mode: 'inline'
})

const emit = defineEmits<{
  close: []
  test: [account: Account]
  stats: [account: Account]
  edit: [account: Account]
  more: [account: Account, event: MouseEvent]
  revertFallback: [account: Account]
  toggleSchedulable: [account: Account]
  viewProxy: [proxyId: number]
}>()

const { t } = useI18n()
const inspectorRef = ref<HTMLElement | null>(null)
const contentRef = ref<HTMLElement | null>(null)
type AccountUsageCellHandle = { queryQuota: () => Promise<void> }
const usageCellRef = ref<AccountUsageCellHandle | null>(null)
const quotaQueryLoading = ref(false)
const placeholder = '--'
const titleId = computed(() => `account-inspector-title-${props.account?.id ?? 'empty'}`)

const resolvedProxy = computed<ProxySummary | null>(() => {
  const proxy = props.proxyTelemetry || props.account?.proxy
  if (!proxy) return null
  return {
    id: proxy.id,
    name: proxy.name,
    status: proxy.status
  }
})
const proxyStatusLabel = computed(() => resolvedProxy.value
  ? t(`admin.accounts.workbench.proxyStatus.${resolvedProxy.value.status}`)
  : ''
)
const effectiveGroups = computed(() => props.account?.groups || [])

const credentials = computed<Record<string, unknown>>(() => props.account?.credentials || {})
const extra = computed<Record<string, unknown>>(() => props.account?.extra || {})
const firstString = (...values: unknown[]) => values.find(value => typeof value === 'string' && value.trim()) as string || ''
const accountEmail = computed(() => {
  const account = props.account
  if (!account) return ''

  const credentialEmail = firstString(
    credentials.value.email,
    credentials.value.client_email,
    extra.value.email,
    extra.value.email_address
  )
  if (account.parent_account_id != null || account.quota_dimension) {
    return firstString(account.parent_email, credentialEmail)
  }
  return firstString(credentialEmail, account.parent_email)
})

const openAIAuthMode = computed(() =>
  firstString(credentials.value.auth_mode, extra.value.auth_mode)
)
const accountPlanType = computed(() =>
  firstString(credentials.value.plan_type, extra.value.plan_type, props.account?.parent_plan_type)
)
const privacyMode = computed(() =>
  firstString(extra.value.privacy_mode, props.account?.parent_privacy_mode)
)
const subscriptionExpiresAt = computed(() =>
  firstString(credentials.value.subscription_expires_at, props.account?.parent_subscription_expires_at)
)

type DetailRow = { label: string; value: string }

const displayDate = (value: unknown): string => {
  if (typeof value !== 'string' || !value) return ''
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '' : formatDateTime(date)
}

const displayAccountExpiry = (value: number | null | undefined): string => {
  if (!value) return ''
  return formatDateTime(new Date(value * 1000))
}

const enabledLabel = (value: boolean | null | undefined): string =>
  value ? t('common.enabled') : t('common.disabled')

const policyRows = computed<DetailRow[]>(() => {
  const account = props.account
  if (!account) return []
  const rows: DetailRow[] = []
  const add = (label: string, value: unknown) => {
    if (value === null || value === undefined || value === '') return
    rows.push({ label, value: String(value) })
  }

  add(t('admin.accounts.workbench.notes'), account.notes)
  add(t('admin.accounts.workbench.accountExpiresAt'), displayAccountExpiry(account.expires_at))
  add(t('admin.accounts.workbench.priority'), account.priority)
  add(t('admin.accounts.workbench.sessionWindowEnd'), displayDate(account.session_window_end))
  add(t('admin.accounts.workbench.windowCostPolicy'), account.window_cost_limit != null
    ? `${account.window_cost_limit} / ${account.window_cost_sticky_reserve ?? 0}`
    : '')
  add(t('admin.accounts.workbench.sessionPolicy'), account.max_sessions != null
    ? `${account.max_sessions} / ${account.session_idle_timeout_minutes ?? '--'} min`
    : '')
  add(t('admin.accounts.workbench.rpmPolicy'), account.base_rpm != null
    ? `${account.base_rpm} · ${account.rpm_strategy || 'default'}`
    : '')
  add(t('admin.accounts.workbench.customBaseUrl'), account.custom_base_url_enabled
    ? account.custom_base_url || enabledLabel(true)
    : '')
  add(t('admin.accounts.workbench.tlsFingerprint'), account.enable_tls_fingerprint
    ? enabledLabel(true)
    : '')
  add(t('admin.accounts.workbench.sessionMasking'), account.session_id_masking_enabled
    ? enabledLabel(true)
    : '')
  add(t('admin.accounts.workbench.cacheTtlOverride'), account.cache_ttl_override_enabled
    ? account.cache_ttl_override_target || enabledLabel(true)
    : '')

  return rows
})

const queryQuota = async () => {
  if (quotaQueryLoading.value || !usageCellRef.value) return
  quotaQueryLoading.value = true
  try {
    await usageCellRef.value.queryQuota()
  } finally {
    quotaQueryLoading.value = false
  }
}

const isFutureDate = (value: string | null | undefined) => {
  if (!value) return false
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) && timestamp > Date.now()
}

const modelRateLimitCount = computed(() => {
  const limits = extra.value.model_rate_limits
  if (!limits || typeof limits !== 'object') return 0
  return Object.values(limits).filter((entry) => {
    if (!entry || typeof entry !== 'object') return false
    return isFutureDate((entry as { rate_limit_reset_at?: string }).rate_limit_reset_at)
  }).length
})

const recoveryRows = computed<DetailRow[]>(() => {
  const account = props.account
  if (!account) return []
  const rows: DetailRow[] = []
  if (account.status === 'error' || account.error_message) {
    rows.push({
      label: t('admin.accounts.workbench.errorReason'),
      value: account.error_message || t('admin.accounts.status.error')
    })
  }
  if (isFutureDate(account.rate_limit_reset_at)) {
    rows.push({
      label: t('admin.accounts.workbench.rateLimitReset'),
      value: formatDateTime(account.rate_limit_reset_at!)
    })
  }
  if (isFutureDate(account.overload_until)) {
    rows.push({
      label: t('admin.accounts.workbench.overloadUntil'),
      value: formatDateTime(account.overload_until!)
    })
  }
  if (isFutureDate(account.temp_unschedulable_until)) {
    rows.push({
      label: t('admin.accounts.workbench.tempUnschedulableUntil'),
      value: formatDateTime(account.temp_unschedulable_until!)
    })
  }
  if (modelRateLimitCount.value > 0) {
    rows.push({
      label: t('admin.accounts.workbench.modelRateLimits'),
      value: t('admin.accounts.workbench.modelRateLimitCount', { count: modelRateLimitCount.value })
    })
  }
  if (account.expires_at && account.expires_at * 1000 <= Date.now()) {
    rows.push({
      label: t('admin.accounts.workbench.accountExpiresAt'),
      value: displayAccountExpiry(account.expires_at)
    })
  }
  if (!account.schedulable) {
    rows.push({
      label: t('admin.accounts.columns.schedulable'),
      value: t('admin.accounts.schedulableDisabled')
    })
  }
  return rows
})

watch(
  () => props.account?.id,
  async () => {
    await nextTick()
    if (contentRef.value) contentRef.value.scrollTop = 0
  }
)

const handleKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    emit('close')
    return
  }
  if (props.mode === 'inline') return
  if (event.key !== 'Tab' || !inspectorRef.value) return

  const focusable = Array.from(
    inspectorRef.value.querySelectorAll<HTMLElement>(
      'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )
  ).filter(element => !element.hasAttribute('hidden'))
  if (!focusable.length) {
    event.preventDefault()
    inspectorRef.value.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (
    event.shiftKey
    && (document.activeElement === first || document.activeElement === inspectorRef.value)
  ) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

const focus = () => inspectorRef.value?.focus()
defineExpose({ focus })
</script>

<style scoped>
.account-inspector {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
}

.account-inspector:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 30%, transparent);
  outline-offset: -3px;
}

.account-inspector__header {
  display: flex;
  min-height: 92px;
  flex: none;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--lx-clay-border);
}

.account-inspector__identity {
  min-width: 0;
  flex: 1 1 auto;
}

.account-inspector__identity h2 {
  margin: 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1rem;
  font-weight: 850;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-inspector__identity-meta,
.account-inspector__header-actions,
.account-inspector__scheduling {
  display: flex;
  align-items: center;
}

.account-inspector__identity-meta {
  min-width: 0;
  gap: 7px;
  margin-top: 5px;
}

.account-inspector__email {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  font-size: 0.7rem;
  font-weight: 650;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-inspector__id,
.account-inspector__scheduling > span {
  color: var(--lx-clay-text-muted);
  font-size: 0.67rem;
  font-weight: 700;
}

.account-inspector__header-actions {
  flex: none;
  gap: 4px;
}

.account-inspector__scheduling {
  gap: 7px;
  margin-right: 3px;
}

.account-inspector__switch {
  position: relative;
  width: 32px;
  height: 18px;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--lx-clay-recessed-strong);
  cursor: pointer;
  transition: background-color 160ms ease-out;
}

.account-inspector__switch[aria-checked='true'] {
  background: var(--lx-clay-accent);
}

.account-inspector__switch:disabled {
  cursor: wait;
  opacity: 0.55;
}

.account-inspector__switch span {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: white;
  box-shadow: 0 1px 3px rgb(15 23 42 / 0.24);
  transition: transform 160ms ease-out;
}

.account-inspector__switch[aria-checked='true'] span {
  transform: translateX(14px);
}

.account-inspector__icon-button {
  display: inline-flex;
  width: 38px;
  height: 38px;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid transparent;
  border-radius: 10px;
  color: var(--lx-clay-text-muted);
  background: transparent;
  cursor: pointer;
}

.account-inspector__icon-button:hover,
.account-inspector__icon-button:focus-visible {
  border-color: var(--lx-clay-border);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.account-inspector__content {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 4px 18px 16px;
  scrollbar-gutter: stable;
}

.account-inspector__section {
  padding: 16px 0;
  border-bottom: 1px solid var(--lx-clay-border);
}

.account-inspector__section:last-child {
  border-bottom: 0;
}

.account-inspector__section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.account-inspector__section-title h3 {
  margin: 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.72rem;
  font-weight: 850;
  letter-spacing: 0;
}

.account-inspector__section-icon {
  display: inline-flex;
  width: 26px;
  height: 26px;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.account-inspector__section-icon--associations {
  color: #2563eb;
  background: rgb(219 234 254 / 0.72);
}

.account-inspector__quota-title-icon {
  flex: 0 0 auto;
}

.account-inspector__section-icon--policy {
  color: #7c3aed;
  background: rgb(237 233 254 / 0.8);
}

.account-inspector__section-icon--usage {
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.account-inspector__section-icon--recovery {
  color: #b45309;
  background: rgb(254 243 199 / 0.8);
}

.account-inspector__section--quota .account-inspector__section-title {
  margin-bottom: 14px;
}

.account-inspector__section--quota :deep(.account-usage-overview__rows) {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.account-inspector__section--quota :deep(.account-usage-overview__row) {
  min-width: 0;
}

.account-inspector__section--quota :deep(.account-usage-overview__heading) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 8px;
  color: var(--lx-clay-text);
  font-size: 0.78rem;
  font-weight: 760;
  line-height: 1.35;
}

.account-inspector__section--quota :deep(.account-usage-overview__heading strong) {
  flex: none;
  font-size: 0.78rem;
  font-weight: 820;
  font-variant-numeric: tabular-nums;
  transition: color 220ms ease-out;
}

.account-inspector__section--quota :deep(.account-usage-overview__track) {
  width: 100%;
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
}

.account-inspector__section--quota :deep(.account-usage-overview__track > span) {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--lx-clay-success-bright);
  transition: width 220ms ease-out;
}

.account-inspector__section--quota :deep(.account-usage-overview__row[data-tone='secondary'] .account-usage-overview__track > span) {
  background: var(--lx-clay-info);
}

.account-inspector__section--quota :deep(.account-usage-overview__row[data-level='warning'] .account-usage-overview__heading strong) {
  color: var(--lx-clay-warning);
}

.account-inspector__section--quota :deep(.account-usage-overview__row[data-level='warning'] .account-usage-overview__track > span) {
  background: var(--lx-clay-warning-bright);
}

.account-inspector__section--quota :deep(.account-usage-overview__row[data-level='danger'] .account-usage-overview__heading strong) {
  color: var(--lx-clay-danger);
}

.account-inspector__section--quota :deep(.account-usage-overview__row[data-level='danger'] .account-usage-overview__track > span) {
  background: var(--lx-clay-danger);
}

.account-inspector__section--quota :deep(.account-usage-overview__row > p) {
  margin: 8px 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
  line-height: 1.4;
}

.account-inspector__section--quota :deep(.account-usage-overview__empty) {
  margin: 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.72rem;
}

.account-inspector__quota-actions {
  margin-top: 13px;
  padding-top: 12px;
  border-top: 1px solid var(--lx-clay-border);
}

.account-inspector__quota-actions :deep(.openai-quota-reset__actions) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.account-inspector__quota-action,
.account-inspector__quota-actions :deep(.openai-quota-reset__count),
.account-inspector__quota-actions :deep(.openai-quota-reset__reset) {
  display: inline-flex;
  min-width: 0;
  min-height: 32px;
  flex: 1 1 auto;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 0 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 8px;
  color: var(--lx-clay-text-muted);
  background: transparent;
  font-size: 0.7rem;
  font-weight: 760;
  line-height: 1;
  white-space: nowrap;
  transition:
    border-color 150ms ease-out,
    color 150ms ease-out,
    background-color 150ms ease-out;
}

.account-inspector__quota-action,
.account-inspector__quota-actions :deep(.openai-quota-reset__count) {
  color: var(--lx-clay-accent-strong);
}

.account-inspector__quota-actions :deep(.openai-quota-reset__reset:not(:disabled)) {
  color: #b45309;
}

.account-inspector__quota-action:hover:not(:disabled),
.account-inspector__quota-action:focus-visible,
.account-inspector__quota-actions :deep(.openai-quota-reset__count:hover:not(:disabled)),
.account-inspector__quota-actions :deep(.openai-quota-reset__count:focus-visible),
.account-inspector__quota-actions :deep(.openai-quota-reset__reset:hover:not(:disabled)),
.account-inspector__quota-actions :deep(.openai-quota-reset__reset:focus-visible) {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 28%, var(--lx-clay-border));
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
  outline: none;
}

.account-inspector__quota-action:focus-visible,
.account-inspector__quota-actions :deep(.openai-quota-reset__count:focus-visible),
.account-inspector__quota-actions :deep(.openai-quota-reset__reset:focus-visible) {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--lx-clay-accent) 22%, transparent);
}

.account-inspector__quota-action:disabled,
.account-inspector__quota-actions :deep(.openai-quota-reset__count:disabled),
.account-inspector__quota-actions :deep(.openai-quota-reset__reset:disabled) {
  cursor: not-allowed;
  opacity: 0.46;
}

.account-inspector__section--quota :deep(.account-usage-overview__skeleton) {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.account-inspector__section--quota :deep(.account-usage-overview__skeleton-row) {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px 16px;
}

.account-inspector__section--quota :deep(.account-usage-overview__skeleton-row span) {
  width: 7rem;
  height: 13px;
  border-radius: 4px;
  background: var(--lx-clay-recessed);
  animation: account-inspector-pulse 1.2s ease-in-out infinite alternate;
}

.account-inspector__section--quota :deep(.account-usage-overview__skeleton-row span:nth-child(2)) {
  width: 3.2rem;
}

.account-inspector__section--quota :deep(.account-usage-overview__skeleton-row i) {
  grid-column: 1 / -1;
  height: 8px;
  border-radius: 999px;
  background: var(--lx-clay-recessed);
  animation: account-inspector-pulse 1.2s ease-in-out infinite alternate;
}

.account-inspector__concurrency {
  display: flex;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 18px;
  padding: 0 14px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 12px;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-surface-soft);
  font-size: 0.72rem;
  font-weight: 760;
}

.account-inspector__concurrency strong {
  flex: none;
  color: var(--lx-clay-text);
  font-size: 0.78rem;
  font-weight: 830;
  font-variant-numeric: tabular-nums;
}

.account-inspector__usage-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 11px 18px;
  margin: 0;
}

.account-inspector__usage-grid > div {
  min-width: 0;
}

.account-inspector dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.65rem;
  font-weight: 750;
  line-height: 1.35;
}

.account-inspector dd {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--lx-clay-text);
  font-size: 0.75rem;
  font-weight: 720;
  line-height: 1.4;
  text-overflow: ellipsis;
}

.account-inspector__section--associations {
  padding-bottom: 4px;
}

.account-inspector__association + .account-inspector__association {
  margin-top: 15px;
}

.account-inspector__association-label {
  display: block;
  margin-bottom: 7px;
  color: var(--lx-clay-text-muted);
  font-size: 0.65rem;
  font-weight: 750;
  line-height: 1.35;
}

.account-inspector__groups {
  display: flex;
  min-width: 0;
  flex-wrap: wrap;
  gap: 6px;
}

.account-inspector__groups :deep(.account-inspector__group-badge) {
  max-width: 100%;
  padding: 3px 8px;
  font-size: 0.68rem;
}

.account-inspector__association-empty,
.account-inspector__direct-route {
  color: var(--lx-clay-text-muted);
  font-size: 0.72rem;
  font-weight: 650;
}

.account-inspector__direct-route {
  margin: 0;
}

.account-inspector__proxy-link {
  display: grid;
  width: 100%;
  min-width: 0;
  min-height: 42px;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
  padding: 0 11px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 9px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface-soft);
  font: inherit;
  cursor: pointer;
  transition:
    border-color 150ms ease-out,
    background-color 150ms ease-out;
}

.account-inspector__proxy-link:hover,
.account-inspector__proxy-link:focus-visible {
  border-color: var(--lx-clay-border-strong);
  background: var(--lx-clay-recessed);
  outline: none;
}

.account-inspector__proxy-link:focus-visible {
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--lx-clay-accent) 22%, transparent);
}

.account-inspector__proxy-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--lx-clay-text-muted);
}

.account-inspector__proxy-dot--active {
  background: var(--lx-clay-success-bright);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-success-bright) 14%, transparent);
}

.account-inspector__proxy-name {
  overflow: hidden;
  font-size: 0.73rem;
  font-weight: 780;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-inspector__proxy-hint {
  color: var(--lx-clay-text-muted);
  font-size: 0.66rem;
  font-weight: 700;
  white-space: nowrap;
}

.account-inspector__proxy-link > svg {
  color: var(--lx-clay-text-muted);
}

.account-inspector__text-action {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  padding: 0 10px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 9px;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-surface-soft);
  font: inherit;
  font-size: 0.7rem;
  font-weight: 800;
  cursor: pointer;
}

.account-inspector__policy-list,
.account-inspector__recovery-list {
  margin: 0;
}

.account-inspector__policy-list > div {
  display: grid;
  grid-template-columns: minmax(7.5rem, 0.8fr) minmax(0, 1.2fr);
  gap: 12px;
  padding: 7px 0;
}

.account-inspector__policy-list dd {
  margin-top: 0;
  text-align: right;
}

.account-inspector__recovery-list > div + div {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid rgb(245 158 11 / 0.18);
}

.account-inspector__section--recovery {
  margin: 12px 0 4px;
  padding: 14px;
  border: 1px solid rgb(245 158 11 / 0.24);
  border-radius: 10px;
  background: rgb(255 251 235 / 0.72);
}

.account-inspector__recovery-list dd {
  margin-top: 4px;
  color: #92400e;
  line-height: 1.5;
  text-align: left;
  white-space: normal;
  overflow-wrap: anywhere;
}

.account-inspector__loading {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.account-inspector__loading span {
  height: 44px;
  border-radius: 8px;
  background: var(--lx-clay-recessed);
  animation: account-inspector-pulse 1.2s ease-in-out infinite alternate;
}

.account-inspector__error {
  margin: 0;
  color: #b91c1c;
  font-size: 0.72rem;
}

.account-inspector__footer {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  flex: none;
  gap: 8px;
  padding: 12px 14px;
  border-top: 1px solid var(--lx-clay-border);
  background: var(--lx-clay-surface);
}

.account-inspector__command {
  display: inline-flex;
  min-width: 0;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 0 8px;
  border: 1px solid var(--lx-clay-border);
  border-radius: 10px;
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-surface-soft);
  font: inherit;
  font-size: 0.71rem;
  font-weight: 800;
  cursor: pointer;
  transition:
    border-color 150ms ease-out,
    color 150ms ease-out,
    background-color 150ms ease-out;
}

.account-inspector__command-icon {
  width: 11px;
  height: 11px;
  flex: 0 0 auto;
}

.account-inspector__command:hover,
.account-inspector__command:focus-visible {
  border-color: var(--lx-clay-border-strong);
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
}

.account-inspector__command span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-inspector__empty {
  display: flex;
  min-height: 18rem;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 28px;
  color: var(--lx-clay-text-muted);
  text-align: center;
}

.account-inspector__empty-icon {
  display: inline-flex;
  width: 48px;
  height: 48px;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  color: var(--lx-clay-accent);
  background: var(--lx-clay-accent-soft);
}

.account-inspector__empty p {
  max-width: 15rem;
  margin: 0;
  font-size: 0.78rem;
  line-height: 1.55;
}

@keyframes account-inspector-pulse {
  to {
    opacity: 0.52;
  }
}

@media (max-width: 639px) {
  .account-inspector__header {
    align-items: flex-start;
  }

  .account-inspector__scheduling > span {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }

  .account-inspector__usage-grid {
    gap: 10px 12px;
  }

  .account-inspector__footer {
    position: sticky;
    bottom: 0;
  }
}

:global(.dark) .account-inspector__section-icon--associations {
  color: #93c5fd;
  background: rgb(30 58 138 / 0.35);
}

:global(.dark) .account-inspector__section-icon--policy {
  color: #c4b5fd;
  background: rgb(76 29 149 / 0.35);
}

:global(.dark) .account-inspector__section--recovery {
  background: rgb(120 53 15 / 0.16);
}

:global(.dark) .account-inspector__recovery-list dd {
  color: #fcd34d;
}

@media (prefers-reduced-motion: reduce) {
  .account-inspector__switch,
  .account-inspector__switch span,
  .account-inspector__command {
    transition-duration: 0.01ms;
  }

  .account-inspector__loading span {
    animation: none;
  }
}
</style>
