<template>
  <AppLayout>
    <div class="mx-auto w-full max-w-[1200px] space-y-6">
      <AdminPageHeader
        :title="t('userSubscriptions.title')"
        :description="t('userSubscriptions.description')"
      />

      <section
        class="grid items-stretch gap-5 lg:grid-cols-[minmax(0,0.4fr)_minmax(0,0.6fr)]"
        data-testid="balance-membership-overview"
      >
        <!-- The balance surface is deliberately neutral: it is not a membership quota. -->
        <article
          class="flex h-full min-h-[220px] flex-col rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          data-testid="balance-card"
        >
          <div class="flex items-center gap-3">
            <span
              class="flex h-10 w-10 items-center justify-center rounded-xl bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300"
              aria-hidden="true"
            >
              <Icon name="creditCard" size="md" />
            </span>
            <div>
              <h2 class="text-sm font-semibold text-gray-700 dark:text-gray-200">
                {{ t('balanceMembership.balanceTitle') }}
              </h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ t('balanceMembership.balanceHint') }}
              </p>
            </div>
          </div>

          <div class="mt-7 flex items-baseline gap-2">
            <CreditAmount
              class="text-3xl font-semibold text-gray-950 dark:text-white"
              :value="formattedBalance"
              icon-size="md"
              :label="`${t('balanceMembership.balanceTitle')} ${formattedBalance}`"
            />
            <span class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('balanceMembership.pointsUnit') }}
            </span>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ t('balanceMembership.paygNote') }}
          </p>

          <div class="mt-auto grid gap-3 pt-7 sm:grid-cols-2">
            <button
              type="button"
              class="btn btn-primary min-h-11 w-full justify-center"
              data-testid="balance-recharge"
              @click="goToPurchase()"
            >
              {{ t('balanceMembership.recharge') }}
            </button>
            <button
              type="button"
              class="btn btn-secondary min-h-11 w-full justify-center"
              data-testid="balance-redeem"
              @click="goToPurchase('#redeem')"
            >
              {{ t('balanceMembership.redeem') }}
            </button>
          </div>
        </article>

        <article
          v-if="loading"
          class="flex h-full min-h-[220px] flex-col justify-center rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          data-testid="member-loading-card"
          role="status"
          :aria-label="t('userSubscriptions.loading')"
        >
          <div class="flex items-center gap-3 text-sm text-gray-500 dark:text-dark-300">
            <span class="h-5 w-5 animate-spin rounded-full border-2 border-primary-500 border-t-transparent" aria-hidden="true"></span>
            {{ t('userSubscriptions.loading') }}
          </div>
        </article>

        <article
          v-else-if="loadFailed"
          class="flex h-full min-h-[220px] flex-col justify-center rounded-2xl border border-red-200 bg-white p-6 shadow-sm dark:border-red-900/60 dark:bg-dark-800"
          data-testid="subscriptions-load-error"
          role="alert"
        >
          <div class="flex items-start gap-3">
            <span
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-red-50 text-red-600 dark:bg-red-950/50 dark:text-red-300"
              aria-hidden="true"
            >
              <Icon name="exclamationCircle" size="md" />
            </span>
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ t('userSubscriptions.failedToLoad') }}
              </h2>
              <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ t('userSubscriptions.failedToLoadDesc') }}
              </p>
              <button type="button" class="btn btn-secondary mt-4 !min-h-10" @click="loadPage">
                <Icon name="refresh" size="sm" aria-hidden="true" />
                {{ t('userSubscriptions.retry') }}
              </button>
            </div>
          </div>
        </article>

        <!-- Only the current membership belongs here; the catalogue lives at /pricing. -->
        <article
          v-else-if="currentSubscription"
          class="membership-overview-card flex h-full min-h-[220px] flex-col overflow-hidden rounded-2xl border p-6 shadow-sm"
          :class="memberCardClass"
          data-testid="current-member-card"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <h2 class="truncate text-lg font-semibold text-gray-950 dark:text-white">
                  {{ currentMemberName }}
                </h2>
                <span class="membership-status-badge">
                  {{ t(`userSubscriptions.status.${currentSubscription.status}`) }}
                </span>
              </div>
            </div>
            <span class="shrink-0 text-xs text-gray-500 dark:text-dark-300">
              {{ t('balanceMembership.currentMember') }}
            </span>
          </div>

          <div class="mt-7">
            <div class="flex items-end justify-between gap-4">
              <div>
                <p class="text-sm font-medium text-gray-600 dark:text-dark-300">
                  {{ t('balanceMembership.monthlyQuota') }}
                </p>
                <p v-if="monthlyQuotaPercent !== null" class="mt-1 text-xl font-semibold tabular-nums text-gray-950 dark:text-white">
                  {{ formatPercent(monthlyQuotaPercent) }}% {{ t('balanceMembership.remaining') }}
                </p>
                <p v-else class="mt-1 text-sm font-medium text-gray-600 dark:text-dark-300">
                  {{ t('balanceMembership.monthlyQuotaUnavailable') }}
                </p>
              </div>
              <span class="text-xs text-gray-500 dark:text-dark-400">
                {{ monthlyResetLabel }}
              </span>
            </div>

            <div
              v-if="monthlyQuotaPercent !== null"
              class="mt-3 h-2.5 overflow-hidden rounded-full bg-black/10 dark:bg-white/10"
              role="progressbar"
              :aria-valuenow="monthlyQuotaPercent"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="t('balanceMembership.monthlyQuota')"
            >
              <div
                class="h-full rounded-full bg-current transition-[width] duration-300"
                :class="monthlyProgressClass"
                :style="{ width: `${monthlyQuotaPercent}%` }"
              ></div>
            </div>

            <div v-if="monthlyQuotaValues" class="mt-3 flex items-baseline gap-2 text-sm text-gray-600 dark:text-dark-300">
              <span>{{ t('balanceMembership.remainingLabel') }}</span>
              <span class="font-semibold tabular-nums text-gray-950 dark:text-white">
                {{ monthlyQuotaValues.remaining }}
              </span>
              <span aria-hidden="true">/</span>
              <span class="tabular-nums">{{ monthlyQuotaValues.limit }}</span>
            </div>
          </div>

          <div class="mt-auto flex flex-wrap items-center justify-between gap-3 pt-6">
            <p v-if="expirationLabel" class="text-xs text-gray-500 dark:text-dark-400">
              {{ expirationLabel }}
            </p>
            <span v-else></span>
            <div class="flex flex-wrap gap-2">
              <button
                type="button"
                class="btn btn-primary min-h-10 px-4"
                data-testid="member-renew"
                @click="goToPricing('renew')"
              >
                {{ t('balanceMembership.renew') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary min-h-10 px-4"
                data-testid="member-upgrade"
                @click="goToPricing(secondaryMemberMode)"
              >
                {{ secondaryMemberLabel }}
              </button>
            </div>
          </div>
        </article>

        <article
          v-else
          class="flex h-full min-h-[220px] flex-col rounded-2xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-800"
          data-testid="free-member-card"
        >
          <div>
            <div class="flex items-center gap-2">
              <h2 class="text-lg font-semibold text-gray-950 dark:text-white">
                {{ t('balanceMembership.freeTitle') }}
              </h2>
              <span class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-300">
                {{ t('balanceMembership.freeBadge') }}
              </span>
            </div>
            <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">
              {{ t('balanceMembership.freeDescription') }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('balanceMembership.freeHint') }}
            </p>
          </div>
          <div class="mt-auto pt-7">
            <button
              type="button"
              class="btn btn-primary min-h-11"
              data-testid="member-subscribe"
              @click="goToPricing('subscribe', 'low')"
            >
              {{ t('balanceMembership.subscribe') }}
            </button>
          </div>
        </article>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/layout/AdminPageHeader.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateOnly } from '@/utils/format'

type MembershipTier = 'low' | 'mid' | 'high' | 'unknown'
type MembershipMode = 'subscribe' | 'renew' | 'upgrade'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()

const subscriptions = ref<UserSubscription[]>([])
const loading = ref(true)
const loadFailed = ref(false)

const currentSubscription = computed(() => {
  const active = subscriptions.value.filter((subscription) => subscription.status === 'active')
  return [...active].sort((a, b) => {
    const aExpiry = a.expires_at ? Date.parse(a.expires_at) : Number.POSITIVE_INFINITY
    const bExpiry = b.expires_at ? Date.parse(b.expires_at) : Number.POSITIVE_INFINITY
    return aExpiry - bExpiry
  })[0] ?? null
})

const currentMemberName = computed(() => currentSubscription.value?.group?.name?.trim() || t('balanceMembership.memberFallback'))

function tierForName(name: string | null | undefined): MembershipTier {
  const normalized = name?.trim().toLocaleLowerCase('en-US') ?? ''
  if (['ultra', 'high', 'heavy', '高量级', '高量'].includes(normalized)) return 'high'
  if (['plus', 'pro', 'max', 'mid', 'medium', '中量级', '中量'].includes(normalized)) return 'mid'
  if (['try', 'basic', 'standard', 'low', 'light', 'lightweight', '轻量级', '轻量'].includes(normalized)) return 'low'
  return 'unknown'
}

const currentTier = computed(() => tierForName(currentSubscription.value?.group?.name))
const memberCardClass = computed(() => `membership-overview-card--${currentTier.value}`)
const secondaryMemberMode = computed<MembershipMode | null>(() => (
  currentTier.value === 'low' || currentTier.value === 'mid' ? 'upgrade' : null
))
const secondaryMemberLabel = computed(() => (
  currentTier.value === 'low' || currentTier.value === 'mid'
    ? t('balanceMembership.upgrade')
    : t('balanceMembership.viewPlans')
))

const upgradeTargetTier = computed<MembershipTier>(() => {
  if (currentTier.value === 'low') return 'mid'
  if (currentTier.value === 'mid') return 'high'
  return currentTier.value
})

const monthlyQuota = computed(() => {
  const subscription = currentSubscription.value
  const rawLimit = subscription?.group?.monthly_limit_usd
  const rawUsage = subscription?.monthly_usage_usd
  if (rawLimit === null || rawLimit === undefined || rawUsage === null || rawUsage === undefined) {
    return null
  }

  const limit = Number(rawLimit)
  if (!subscription || !Number.isFinite(limit) || limit <= 0) return null

  const used = Number(rawUsage)
  if (!Number.isFinite(used)) return null
  const safeUsed = Math.max(0, used)
  const remaining = Math.max(limit - safeUsed, 0)
  return { limit, remaining, percent: Math.min((remaining / limit) * 100, 100) }
})

const monthlyQuotaPercent = computed(() => monthlyQuota.value?.percent ?? null)
const monthlyQuotaValues = computed(() => {
  if (!monthlyQuota.value) return null
  return {
    remaining: formatQuota(monthlyQuota.value.remaining),
    limit: formatQuota(monthlyQuota.value.limit),
  }
})

const monthlyProgressClass = computed(() => {
  if (monthlyQuotaPercent.value === null) return ''
  if (monthlyQuotaPercent.value <= 10) return 'text-red-500'
  if (monthlyQuotaPercent.value <= 30) return 'text-amber-500'
  return 'text-emerald-500'
})

const monthlyResetLabel = computed(() => {
  const start = currentSubscription.value?.monthly_window_start
  if (!start) return t('balanceMembership.monthlyResetPending')
  const date = new Date(start)
  if (Number.isNaN(date.getTime())) return t('balanceMembership.monthlyResetPending')
  date.setHours(date.getHours() + 30 * 24)
  return t('balanceMembership.monthlyReset', { date: formatDateOnly(date) })
})

const expirationLabel = computed(() => {
  const expiresAt = currentSubscription.value?.expires_at
  if (!expiresAt) return ''
  const formatted = formatDateOnly(expiresAt)
  return formatted ? t('balanceMembership.expires', { date: formatted }) : ''
})

const formattedBalance = computed(() => {
  const rawBalance: unknown = authStore.user?.balance
  if (rawBalance === null || rawBalance === undefined || rawBalance === '') return '—'

  const value = Number(rawBalance)
  return Number.isFinite(value)
    ? new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value)
    : '—'
})

function formatQuota(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 }).format(value)
}

function formatPercent(value: number): string {
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(value)
}

function goToPurchase(hash?: string): void {
  void router.push({ path: '/purchase', ...(hash ? { hash } : {}) })
}

function goToPricing(mode: MembershipMode | null, tier?: MembershipTier): void {
  const subscription = currentSubscription.value
  const resolvedTier = tier && tier !== 'unknown'
    ? tier
    : mode === 'upgrade'
      ? upgradeTargetTier.value
      : currentTier.value
  const query: Record<string, string> = {}
  if (mode) query.mode = mode
  if (resolvedTier !== 'unknown') query.tier = resolvedTier
  const currentPlanName = subscription?.group?.name?.trim()
  if (currentPlanName && (mode === 'renew' || mode === null)) query.plan = currentPlanName
  if (subscription?.group_id) query.group = String(subscription.group_id)
  void router.push({ path: '/pricing', query })
}

async function loadPage(): Promise<void> {
  loading.value = true
  loadFailed.value = false
  try {
    const result = await subscriptionsAPI.getMySubscriptions()
    if (!Array.isArray(result)) throw new Error('Invalid subscriptions response')
    subscriptions.value = result
  } catch (error) {
    loadFailed.value = true
    console.error('Failed to load membership overview:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadPage()
})
</script>

<style scoped>
.membership-overview-card {
  border-color: rgb(229 231 235 / 90%);
  background-color: #fff;
}

.membership-overview-card--low {
  border-color: rgb(125 196 255 / 28%);
  background:
    radial-gradient(circle at 92% 8%, rgb(125 196 255 / 18%), transparent 48%),
    #f8fbff;
}

.membership-overview-card--mid {
  border-color: rgb(124 58 237 / 24%);
  background:
    radial-gradient(circle at 92% 8%, rgb(124 58 237 / 13%), transparent 48%),
    #f7f5ff;
}

.membership-overview-card--high {
  border-color: rgb(219 39 119 / 22%);
  background:
    radial-gradient(circle at 92% 8%, rgb(219 39 119 / 11%), rgb(245 158 11 / 8%) 36%, transparent 64%),
    #fffaf5;
}

.membership-overview-card--unknown {
  background: #fff;
}

.membership-status-badge {
  border-radius: 999px;
  padding: 3px 9px;
  color: #047857;
  background: rgb(209 250 229 / 80%);
  font-size: 11px;
  font-weight: 600;
}

/* The app theme is class-driven; keep the local tier aura in sync when the
 * user switches themes without changing the neutral balance surface. */
:global(.dark) .membership-overview-card--unknown,
:global(.dark) .membership-overview-card {
  border-color: rgb(75 85 99 / 80%);
  background: rgb(31 41 55 / 90%);
}

:global(.dark) .membership-overview-card--low {
  background:
    radial-gradient(circle at 92% 8%, rgb(96 165 250 / 20%), transparent 48%),
    rgb(17 34 52 / 92%);
}

:global(.dark) .membership-overview-card--mid {
  background:
    radial-gradient(circle at 92% 8%, rgb(139 92 246 / 20%), transparent 48%),
    rgb(34 27 60 / 92%);
}

:global(.dark) .membership-overview-card--high {
  background:
    radial-gradient(circle at 92% 8%, rgb(236 72 153 / 18%), rgb(245 158 11 / 12%) 36%, transparent 64%),
    rgb(61 35 27 / 92%);
}
</style>
