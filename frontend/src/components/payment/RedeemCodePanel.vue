<template>
  <section
    id="redeem"
    :class="embedded
      ? 'redeem-panel--embedded scroll-mt-24'
      : 'scroll-mt-24 overflow-hidden rounded-2xl border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-800'"
    aria-labelledby="redeem-panel-title"
  >
    <div :class="embedded ? '' : 'border-b border-gray-100 px-5 py-5 dark:border-dark-700 sm:px-6'">
      <div :class="['flex', embedded ? 'items-center gap-2' : 'items-start gap-3']">
        <span :class="[
          'flex shrink-0 items-center justify-center',
          embedded
            ? 'redeem-panel__embedded-icon h-4 w-4'
            : 'h-10 w-10 rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300',
        ]">
          <Icon name="gift" :size="embedded ? 'sm' : 'md'" />
        </span>
        <div>
          <component
            :is="embedded ? 'h3' : 'h2'"
            id="redeem-panel-title"
            :class="embedded
              ? 'redeem-panel__embedded-title text-sm'
              : 'text-base font-semibold text-gray-950 dark:text-white'"
          >
            {{ t('redeem.quickRedeemTitle') }}
          </component>
          <p v-if="!embedded" class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">
            {{ t('redeem.quickRedeemDescription') }}
          </p>
        </div>
      </div>
    </div>

    <div :class="embedded ? 'space-y-4 pt-4' : 'space-y-5 p-5 sm:p-6'">
      <form :aria-busy="submitting" @submit.prevent="handleRedeem">
        <label for="integrated-redeem-code" :class="embedded ? 'sr-only' : 'input-label'">
          {{ t('redeem.redeemCodeLabel') }}
        </label>
        <div
          data-testid="redeem-form-row"
          :class="embedded
            ? 'flex flex-col gap-3 sm:flex-row sm:items-center'
            : 'mt-2 grid gap-3 sm:grid-cols-[minmax(0,1fr)_10rem]'"
        >
          <div :class="['relative', { 'min-w-0 flex-1': embedded }]">
            <span
              v-if="!embedded"
              class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4"
            >
              <Icon
                name="gift"
                size="sm"
                class="text-gray-400 dark:text-gray-500"
              />
            </span>
            <input
              id="integrated-redeem-code"
              v-model="redeemCode"
              type="text"
              autocomplete="off"
              autocapitalize="none"
              spellcheck="false"
              :placeholder="t('redeem.redeemCodePlaceholder')"
              :disabled="submitting"
              :class="[
                'input w-full pr-4 font-mono text-sm',
                embedded
                  ? 'redeem-panel__embedded-input min-h-11 px-4'
                  : 'min-h-12 pl-11 tracking-wide',
              ]"
            />
          </div>
          <button
            type="submit"
            :disabled="!canRedeem || submitting"
            :class="[
              'btn w-full justify-center',
              embedded
                ? 'redeem-panel__embedded-button btn-secondary min-h-11 sm:w-auto sm:px-8'
                : 'btn-primary min-h-12',
            ]"
          >
            <span
              v-if="submitting"
              :class="[
                'h-4 w-4 animate-spin rounded-full border-2',
                embedded
                  ? 'redeem-panel__embedded-spinner'
                  : 'border-white/40 border-t-white',
              ]"
              aria-hidden="true"
            ></span>
            <Icon v-else-if="!embedded" name="checkCircle" size="sm" aria-hidden="true" />
            <span>{{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}</span>
          </button>
        </div>
        <p v-if="!embedded" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t('redeem.redeemCodeHint') }}
          <span v-if="contactInfo"> · {{ t('redeem.supportContact', { contact: contactInfo }) }}</span>
        </p>
      </form>

      <div
        v-if="redeemResult"
        ref="resultRegion"
        role="status"
        aria-live="polite"
        tabindex="-1"
        :class="[
          'flex items-start gap-3 rounded-xl border px-4 py-3 outline-none',
          embedded
            ? 'redeem-panel__feedback--success'
            : 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-800/60 dark:bg-emerald-950/30 dark:text-emerald-200',
        ]"
      >
        <Icon
          name="checkCircle"
          size="md"
          :class="[
            'redeem-panel__feedback-icon mt-0.5 shrink-0',
            { 'text-emerald-600 dark:text-emerald-400': !embedded },
          ]"
        />
        <div class="min-w-0">
          <p class="text-sm font-semibold">{{ t('redeem.redeemSuccess') }}</p>
          <p v-if="redeemResult.type === 'balance'" class="mt-1 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-sm leading-6">
            <span>{{ t('redeem.balanceAddedAmount') }}</span>
            <CreditAmount :value="redeemResult.value.toFixed(2)" icon-size="xs" />
            <span>· {{ t('redeem.currentBalance') }}</span>
            <CreditAmount :value="Number(authStore.user?.balance || 0).toFixed(2)" icon-size="xs" />
          </p>
          <p v-else class="mt-1 text-sm leading-6">{{ redeemResultSummary }}</p>
        </div>
      </div>

      <div
        v-if="errorMessage"
        role="alert"
        :class="[
          'flex items-start gap-3 rounded-xl border px-4 py-3',
          embedded
            ? 'redeem-panel__feedback--error'
            : 'border-red-200 bg-red-50 text-red-800 dark:border-red-800/60 dark:bg-red-950/30 dark:text-red-200',
        ]"
      >
        <Icon
          name="exclamationCircle"
          size="md"
          :class="[
            'redeem-panel__feedback-icon mt-0.5 shrink-0',
            { 'text-red-600 dark:text-red-400': !embedded },
          ]"
        />
        <div class="min-w-0">
          <p class="text-sm font-semibold">{{ t('redeem.redeemFailed') }}</p>
          <p class="mt-1 break-words text-sm leading-6">{{ errorMessage }}</p>
        </div>
      </div>

      <details v-if="!embedded" class="group border-t border-gray-100 pt-4 dark:border-dark-700">
        <summary class="flex min-h-11 cursor-pointer list-none items-center justify-between gap-3 rounded-lg px-1 text-sm font-semibold text-gray-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-gray-200">
          <span class="flex items-center gap-2">
            <Icon name="clock" size="sm" class="text-gray-400" />
            {{ t('redeem.recentActivity') }}
            <span v-if="history.length" class="rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              {{ history.length }}
            </span>
          </span>
          <Icon name="chevronDown" size="sm" class="text-gray-400 transition-transform group-open:rotate-180" />
        </summary>

        <div class="mt-3">
          <div v-if="loadingHistory" class="flex items-center justify-center py-8" role="status">
            <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" aria-hidden="true"></span>
            <span class="sr-only">{{ t('common.loading') }}</span>
          </div>
          <div v-else-if="history.length" class="divide-y divide-gray-100 dark:divide-dark-700">
            <div
              v-for="item in history"
              :key="item.id"
              class="flex flex-wrap items-center justify-between gap-3 py-3 first:pt-1"
            >
              <div class="min-w-0">
                <p class="truncate text-sm font-medium text-gray-800 dark:text-gray-100">
                  {{ getHistoryItemTitle(item) }}
                </p>
                <p class="mt-0.5 text-xs text-gray-400 dark:text-gray-500">
                  {{ formatDateTime(item.used_at) }}
                  <span v-if="!isAdminAdjustment(item.type) && item.code"> · {{ item.code.slice(0, 8) }}…</span>
                </p>
              </div>
              <CreditAmount
                v-if="isBalanceType(item.type)"
                :class="['text-sm font-semibold', historyValueClass(item)]"
                :value="formatHistoryValue(item)"
                icon-size="xs"
                :label="`${getHistoryItemTitle(item)} ${formatHistoryValue(item)}`"
              />
              <span v-else :class="['text-sm font-semibold tabular-nums', historyValueClass(item)]">
                {{ formatHistoryValue(item) }}
              </span>
            </div>
          </div>
          <p v-else class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
            {{ t('redeem.historyWillAppear') }}
          </p>
        </div>
      </details>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useUserProfileStore } from '@/stores/userProfile'
import { redeemAPI, type RedeemHistoryItem, type RedeemResult } from '@/api/redeem'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  embedded?: boolean
}>(), {
  embedded: false,
})

const emit = defineEmits<{
  redeemed: [result: RedeemResult]
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const userProfileStore = useUserProfileStore()

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<RedeemResult | null>(null)
const errorMessage = ref('')
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const resultRegion = ref<HTMLElement | null>(null)

const contactInfo = computed(() => appStore.contactInfo || appStore.cachedPublicSettings?.contact_info || '')
const canRedeem = computed(() => redeemCode.value.trim().length > 0)
const redeemResultSummary = computed(() => {
  const result = redeemResult.value
  if (!result) return ''
  if (result.type === 'balance') {
    return t('redeem.balanceRedeemSummary', {
      added: result.value.toFixed(2),
      balance: Number(authStore.user?.balance || 0).toFixed(2),
    })
  }
  if (result.type === 'concurrency') {
    return t('redeem.concurrencyRedeemSummary', {
      added: result.value,
      concurrency: authStore.user?.concurrency ?? result.value,
    })
  }
  if (result.type === 'subscription') {
    return t('redeem.subscriptionRedeemSummary', {
      group: result.group?.name || t('redeem.subscriptionAssigned'),
      days: result.validity_days || Math.round(result.value),
    })
  }
  return t('redeem.redeemSuccess')
})

function isBalanceType(type: string) {
  return type === 'balance' || type === 'admin_balance'
}

function isSubscriptionType(type: string) {
  return type === 'subscription'
}

function isAdminAdjustment(type: string) {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

function getHistoryItemTitle(item: RedeemHistoryItem) {
  if (item.type === 'balance') return t('redeem.balanceAddedRedeem')
  if (item.type === 'admin_balance') return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  if (item.type === 'concurrency') return t('redeem.concurrencyAddedRedeem')
  if (item.type === 'admin_concurrency') return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  if (item.type === 'subscription') return t('redeem.subscriptionAssigned')
  return t('common.unknown')
}

function formatHistoryValue(item: RedeemHistoryItem) {
  if (isBalanceType(item.type)) return `${item.value >= 0 ? '+' : ''}${item.value.toFixed(2)}`
  if (isSubscriptionType(item.type)) {
    const days = item.validity_days || Math.round(item.value)
    return item.group?.name ? `${days}${t('redeem.days')} · ${item.group.name}` : `${days}${t('redeem.days')}`
  }
  return `${item.value >= 0 ? '+' : ''}${item.value} ${t('redeem.requests')}`
}

function historyValueClass(item: RedeemHistoryItem) {
  if (isSubscriptionType(item.type)) return 'text-purple-600 dark:text-purple-400'
  if (item.value < 0) return 'text-red-600 dark:text-red-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

async function fetchHistory() {
  loadingHistory.value = true
  try {
    history.value = await redeemAPI.getHistory()
  } catch (error) {
    console.error('Failed to fetch redeem history:', error)
  } finally {
    loadingHistory.value = false
  }
}

async function handleRedeem() {
  const code = redeemCode.value.trim()
  if (!code || submitting.value) return

  submitting.value = true
  redeemResult.value = null
  errorMessage.value = ''
  try {
    const result = await redeemAPI.redeem(code)
    redeemResult.value = result
    redeemCode.value = ''
    try {
      if (result.type === 'subscription') {
        await userProfileStore.syncAfterUpgrade()
      } else {
        await userProfileStore.refreshProfile()
      }
    } catch (error) {
      console.error('Failed to refresh account after redeem:', error)
      appStore.showWarning(t('redeem.accountRefreshFailed'))
    }
    if (!props.embedded) await fetchHistory()
    emit('redeemed', result)
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
    await nextTick()
    resultRegion.value?.focus()
  } catch (error: unknown) {
    errorMessage.value = extractI18nErrorMessage(error, t, 'redeem.errors', t('redeem.failedToRedeem'))
    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  if (!props.embedded) void fetchHistory()
})
</script>

<style scoped>
.redeem-panel--embedded {
  color: var(--lx-clay-text);
  background: transparent;
}

.redeem-panel__embedded-icon {
  color: var(--lx-clay-accent);
}

.redeem-panel__embedded-title {
  color: var(--lx-clay-text);
  font-weight: 700;
  letter-spacing: 0;
}

.redeem-panel__embedded-input {
  border-radius: 13px;
  background: var(--lx-clay-recessed);
  box-shadow: none;
}

.redeem-panel__embedded-button {
  border-radius: 12px;
  font-weight: 700;
  box-shadow: none;
}

.redeem-panel__embedded-spinner {
  border-color: var(--lx-clay-border-strong);
  border-top-color: var(--lx-clay-accent);
}

.redeem-panel__feedback--success {
  border-color: color-mix(in srgb, var(--lx-clay-success) 24%, transparent);
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.redeem-panel__feedback--success .redeem-panel__feedback-icon {
  color: var(--lx-clay-success);
}

.redeem-panel__feedback--error {
  border-color: color-mix(in srgb, var(--lx-clay-danger) 24%, transparent);
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.redeem-panel__feedback--error .redeem-panel__feedback-icon {
  color: var(--lx-clay-danger);
}
</style>
