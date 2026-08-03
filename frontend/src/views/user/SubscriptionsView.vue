<template>
  <AppLayout>
    <div class="space-y-6">
      <AdminPageHeader
        :title="t('userSubscriptions.title')"
        :description="t('userSubscriptions.description')"
      >
        <template #secondary-actions>
          <RouterLink to="/quota-viewer" class="btn btn-secondary min-h-11">
            <Icon name="download" size="sm" aria-hidden="true" />
            {{ t('quotaViewerLanding.entry.subscription') }}
          </RouterLink>
        </template>
      </AdminPageHeader>

      <!-- Loading State -->
      <div
        v-if="loading"
        class="flex justify-center py-12"
        role="status"
        :aria-label="t('userSubscriptions.loading')"
      >
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <!-- Load Error -->
      <section
        v-else-if="loadFailed"
        class="max-w-3xl rounded-2xl border border-red-200 bg-white px-6 py-7 dark:border-red-900/60 dark:bg-dark-800 sm:px-8"
        data-testid="subscriptions-load-error"
        role="alert"
      >
        <div class="flex flex-col items-center gap-4 text-center sm:flex-row sm:items-start sm:text-left">
          <span
            class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-red-50 text-red-600 dark:bg-red-950/50 dark:text-red-300"
            aria-hidden="true"
          >
            <Icon name="exclamationCircle" size="lg" />
          </span>
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('userSubscriptions.failedToLoad') }}
            </h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('userSubscriptions.failedToLoadDesc') }}
            </p>
            <button type="button" class="btn btn-secondary mt-5 !min-h-11" @click="loadPage">
              <Icon name="refresh" size="sm" aria-hidden="true" />
              {{ t('userSubscriptions.retry') }}
            </button>
          </div>
        </div>
      </section>

      <!-- Empty State -->
      <section
        v-else-if="subscriptions.length === 0"
        class="max-w-3xl rounded-2xl border border-gray-200 bg-white px-6 py-7 dark:border-dark-700 dark:bg-dark-800 sm:px-8 sm:py-8"
        data-testid="subscriptions-empty-state"
      >
        <div class="flex flex-col items-center gap-5 text-center sm:flex-row sm:items-start sm:text-left">
          <span
            class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-700 dark:bg-primary-950/50 dark:text-primary-300"
            aria-hidden="true"
          >
            <Icon name="creditCard" size="lg" />
          </span>
          <div class="min-w-0 flex-1">
            <h2
              class="text-xl font-semibold tracking-tight text-gray-900 dark:text-white"
              data-testid="subscriptions-empty-title"
            >
              {{ t(emptyStateTitleKey) }}
            </h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t(emptyStateDescriptionKey) }}
            </p>
            <p
              class="mt-4 flex items-start justify-center gap-2 text-sm leading-6 text-gray-600 dark:text-dark-300 sm:justify-start"
            >
              <Icon
                name="infoCircle"
                size="sm"
                class="mt-1 shrink-0 text-primary-600 dark:text-primary-300"
                aria-hidden="true"
              />
              <span>{{ t('userSubscriptions.payAsYouGoAvailable') }}</span>
            </p>
            <div class="mt-6 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
              <RouterLink
                v-if="hasPurchasablePlans"
                to="/pricing"
                class="btn btn-primary !min-h-11 w-full sm:w-auto"
                data-testid="subscriptions-view-plans"
              >
                {{ t('userSubscriptions.viewPlans') }}
              </RouterLink>
              <RouterLink
                v-else
                to="/purchase"
                class="btn btn-primary !min-h-11 w-full sm:w-auto"
                data-testid="subscriptions-recharge"
              >
                {{ t('userSubscriptions.rechargeOrRedeem') }}
              </RouterLink>
              <RouterLink
                v-if="hasPurchasablePlans"
                to="/purchase"
                class="btn btn-secondary !min-h-11 w-full sm:w-auto"
                data-testid="subscriptions-recharge"
              >
                {{ t('userSubscriptions.rechargeOrRedeem') }}
              </RouterLink>
              <a
                v-else
                :href="supportDestination.url"
                class="btn btn-secondary !min-h-11 w-full sm:w-auto"
                target="_blank"
                rel="noopener noreferrer"
                :aria-label="`${t(supportActionLabelKey)} · ${t('purchase.openInNewTab')}`"
                data-testid="subscriptions-support"
              >
                {{ t(supportActionLabelKey) }}
                <Icon name="externalLink" size="sm" aria-hidden="true" />
              </a>
            </div>
          </div>
        </div>
      </section>

      <!-- Subscriptions Grid -->
      <div v-else class="grid gap-6 lg:grid-cols-2">
        <div
          v-for="subscription in subscriptions"
          :key="subscription.id"
          class="overflow-hidden rounded-2xl border bg-white dark:bg-dark-800"
          :class="platformBorderClass(subscription.group?.platform || '')"
        >
          <!-- Header -->
          <div
            class="flex items-center justify-between border-b border-gray-100 p-4 dark:border-dark-700"
          >
            <div class="flex items-center gap-3">
              <div :class="['h-1.5 w-1.5 shrink-0 rounded-full', platformAccentDotClass(subscription.group?.platform || '')]" />
              <div>
                <div class="flex items-center gap-2">
                  <h3 class="font-semibold text-gray-900 dark:text-white">
                    {{ subscription.group?.name || `Group #${subscription.group_id}` }}
                  </h3>
                  <span :class="['rounded-md border px-2 py-0.5 text-[11px] font-medium', platformBadgeClass(subscription.group?.platform || '')]">
                    {{ platformLabel(subscription.group?.platform || '') }}
                  </span>
                </div>
                <p v-if="subscription.group?.description" class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ subscription.group.description }}
                </p>
                <div class="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-gray-400 dark:text-gray-500">
                  <span>{{ t('payment.planCard.rate') }}: ×{{ subscription.group?.rate_multiplier ?? 1 }}</span>
                  <span v-if="subscriptionHasPeakRate(subscription)" class="text-amber-700 dark:text-amber-300">
                    {{ t('payment.planCard.peakRate') }}: {{ subscriptionPeakRateLabel(subscription) }}
                  </span>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span
                :class="[
                  'rounded-full px-2 py-0.5 text-xs font-medium',
                  subscription.status === 'active'
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
                    : subscription.status === 'expired'
                      ? 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-400'
                      : 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
                ]"
              >
                {{ t(`userSubscriptions.status.${subscription.status}`) }}
              </span>
              <button
                v-if="subscription.status === 'active'"
                :class="['rounded-lg px-3 py-1.5 text-xs font-semibold text-white transition-colors', platformButtonClass(subscription.group?.platform || '')]"
                @click="router.push({ path: '/pricing', query: { group: String(subscription.group_id) } })"
              >
                {{ t('payment.renewNow') }}
              </button>
            </div>
          </div>

          <!-- Usage Progress -->
          <div class="space-y-4 p-4">
            <!-- Expiration Info -->
            <div v-if="subscription.expires_at" class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-dark-400">{{
                t('userSubscriptions.expires')
              }}</span>
              <span :class="getExpirationClass(subscription.expires_at)">
                {{ formatExpirationDate(subscription.expires_at) }}
              </span>
            </div>
            <div v-else class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-dark-400">{{
                t('userSubscriptions.expires')
              }}</span>
              <span class="text-gray-700 dark:text-gray-300">{{
                t('userSubscriptions.noExpiration')
              }}</span>
            </div>

            <!-- Daily Usage -->
            <div v-if="subscription.group?.daily_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('userSubscriptions.daily') }}
                </span>
                <CreditAmount
                  class="text-sm text-gray-500 dark:text-dark-400"
                  :value="formatCreditUsage(subscription.daily_usage_usd, subscription.group.daily_limit_usd)"
                  icon-size="sm"
                />
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.daily_usage_usd,
                      subscription.group.daily_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.daily_window_start"
                class="text-xs text-gray-500 dark:text-dark-400"
              >
                {{ formatDailyUsageWindow(subscription) }}
              </p>
            </div>

            <!-- Weekly Usage -->
            <div v-if="subscription.group?.weekly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('userSubscriptions.weekly') }}
                </span>
                <CreditAmount
                  class="text-sm text-gray-500 dark:text-dark-400"
                  :value="formatCreditUsage(subscription.weekly_usage_usd, subscription.group.weekly_limit_usd)"
                  icon-size="sm"
                />
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.weekly_usage_usd,
                      subscription.group.weekly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.weekly_window_start"
                class="text-xs text-gray-500 dark:text-dark-400"
              >
                {{
                  t('userSubscriptions.resetIn', {
                    time: formatResetTime(subscription.weekly_window_start, 168)
                  })
                }}
              </p>
            </div>

            <!-- Monthly Usage -->
            <div v-if="subscription.group?.monthly_limit_usd" class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('userSubscriptions.monthly') }}
                </span>
                <CreditAmount
                  class="text-sm text-gray-500 dark:text-dark-400"
                  :value="formatCreditUsage(subscription.monthly_usage_usd, subscription.group.monthly_limit_usd)"
                  icon-size="sm"
                />
              </div>
              <div class="relative h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
                <div
                  class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
                  :class="
                    getProgressBarClass(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  "
                  :style="{
                    width: getProgressWidth(
                      subscription.monthly_usage_usd,
                      subscription.group.monthly_limit_usd
                    )
                  }"
                ></div>
              </div>
              <p
                v-if="subscription.monthly_window_start"
                class="text-xs text-gray-500 dark:text-dark-400"
              >
                {{
                  t('userSubscriptions.resetIn', {
                    time: formatResetTime(subscription.monthly_window_start, 720)
                  })
                }}
              </p>
            </div>

            <!-- No limits configured - Unlimited badge -->
            <div
              v-if="
                !subscription.group?.daily_limit_usd &&
                !subscription.group?.weekly_limit_usd &&
                !subscription.group?.monthly_limit_usd
              "
              class="flex items-center justify-center rounded-xl bg-[var(--lx-clay-success-soft)] py-6"
            >
              <div class="flex items-center gap-3">
                <span class="text-4xl text-emerald-600 dark:text-emerald-400">∞</span>
                <div>
                  <p class="text-sm font-medium text-emerald-700 dark:text-emerald-300">
                    {{ t('userSubscriptions.unlimited') }}
                  </p>
                  <p class="text-xs text-emerald-600/70 dark:text-emerald-400/70">
                    {{ t('userSubscriptions.unlimitedDesc') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { usePaymentStore } from '@/stores/payment'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateOnly } from '@/utils/format'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { platformBorderClass, platformBadgeClass, platformButtonClass, platformLabel } from '@/utils/platformColors'
import { getRemainingDurationParts, isOneTimeDailyQuota, type RemainingDurationParts } from '@/utils/subscriptionQuota'
import { resolveSupportContactDestination } from '@/utils/supportUrl'

function platformAccentDotClass(p: string): string {
  switch (p) {
    case 'anthropic': return 'bg-orange-500'
    case 'openai': return 'bg-emerald-500'
    case 'antigravity': return 'bg-purple-500'
    case 'gemini': return 'bg-blue-500'
    default: return 'bg-gray-400'
  }
}

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const paymentStore = usePaymentStore()

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)
const loadFailed = ref(false)
const purchaseCapability = ref<'unknown' | 'disabled' | 'no-plans' | 'available'>('unknown')

const hasPurchasablePlans = computed(() => purchaseCapability.value === 'available')
const emptyStateTitleKey = computed(() => {
  if (purchaseCapability.value === 'available') return 'userSubscriptions.emptyWithPlansTitle'
  if (purchaseCapability.value === 'disabled') return 'userSubscriptions.selfServiceDisabledTitle'
  if (purchaseCapability.value === 'no-plans') return 'userSubscriptions.noPlansTitle'
  return 'userSubscriptions.purchaseOptionsUnknownTitle'
})
const emptyStateDescriptionKey = computed(() => {
  if (purchaseCapability.value === 'available') return 'userSubscriptions.emptyWithPlansDesc'
  if (purchaseCapability.value === 'disabled') return 'userSubscriptions.selfServiceDisabledDesc'
  if (purchaseCapability.value === 'no-plans') return 'userSubscriptions.noPlansDesc'
  return 'userSubscriptions.purchaseOptionsUnknownDesc'
})
const supportDestination = computed(() => resolveSupportContactDestination(
  appStore.cachedPublicSettings?.contact_info || appStore.contactInfo,
  appStore.cachedPublicSettings?.doc_url || appStore.docUrl,
))
const supportActionLabelKey = computed(() => (
  supportDestination.value.kind === 'contact'
    ? 'userSubscriptions.contactAdmin'
    : 'userSubscriptions.viewSubscriptionHelp'
))

function subscriptionHasPeakRate(subscription: UserSubscription): boolean {
  return hasPeakRate(subscription.group)
}

function subscriptionPeakRateLabel(subscription: UserSubscription): string {
  return formatPeakRateWindow(subscription.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

async function loadPurchaseOptions(): Promise<void> {
  purchaseCapability.value = 'unknown'

  const settings = await appStore.fetchPublicSettings()
  const paymentEnabled = settings?.payment_enabled
    ?? appStore.cachedPublicSettings?.payment_enabled

  if (paymentEnabled === false) {
    purchaseCapability.value = 'disabled'
    return
  }

  if (paymentEnabled !== true) {
    return
  }

  try {
    const checkoutInfo = await paymentStore.ensureCheckoutInfo()
    purchaseCapability.value = checkoutInfo.plans.length > 0 ? 'available' : 'no-plans'
  } catch (error) {
    console.warn('Failed to load subscription purchase options:', error)
  }
}

async function loadPage(): Promise<void> {
  loading.value = true
  loadFailed.value = false

  const [subscriptionsResult] = await Promise.allSettled([
    subscriptionsAPI.getMySubscriptions(),
    loadPurchaseOptions(),
  ])

  if (subscriptionsResult.status === 'fulfilled') {
    subscriptions.value = subscriptionsResult.value
  } else {
    loadFailed.value = true
    console.error('Failed to load subscriptions:', subscriptionsResult.reason)
  }

  loading.value = false
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function formatCreditUsage(used: number | undefined, limit: number): string {
  return `${(used || 0).toFixed(2)} / ${limit.toFixed(2)}`
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return 'bg-gray-400'
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'bg-red-500'
  if (percentage >= 70) return 'bg-orange-500'
  return 'bg-green-500'
}

function formatExpirationDate(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (days < 0) {
    return t('userSubscriptions.status.expired')
  }

  const dateStr = formatDateOnly(expires)

  if (days === 0) {
    return `${dateStr} (${t('common.today')})`
  }
  if (days === 1) {
    return `${dateStr} (${t('common.tomorrow')})`
  }

  return t('userSubscriptions.daysRemaining', { days }) + ` (${dateStr})`
}

function getExpirationClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))

  if (days <= 0) return 'text-red-600 dark:text-red-400 font-medium'
  if (days <= 3) return 'text-red-600 dark:text-red-400'
  if (days <= 7) return 'text-orange-600 dark:text-orange-400'
  return 'text-gray-700 dark:text-gray-300'
}

function formatDurationParts(parts: RemainingDurationParts): string {
  if (parts.days > 0) {
    return `${parts.days}d ${parts.hours}h`
  }

  if (parts.hours > 0) {
    return `${parts.hours}h ${parts.minutes}m`
  }

  return `${parts.minutes}m`
}

function formatDailyUsageWindow(subscription: UserSubscription): string {
  if (isOneTimeDailyQuota(subscription) && subscription.expires_at) {
    const parts = getRemainingDurationParts(subscription.expires_at)
    if (!parts) return t('userSubscriptions.windowNotActive')
    return t('userSubscriptions.quotaEndsIn', { time: formatDurationParts(parts) })
  }

  return t('userSubscriptions.resetIn', {
    time: formatResetTime(subscription.daily_window_start, 24)
  })
}

function formatResetTime(windowStart: string | null, windowHours: number): string {
  if (!windowStart) return t('userSubscriptions.windowNotActive')

  const start = new Date(windowStart)
  const end = new Date(start.getTime() + windowHours * 60 * 60 * 1000)
  const parts = getRemainingDurationParts(end)

  return parts ? formatDurationParts(parts) : t('userSubscriptions.windowNotActive')
}

onMounted(() => {
  loadPage()
})
</script>
