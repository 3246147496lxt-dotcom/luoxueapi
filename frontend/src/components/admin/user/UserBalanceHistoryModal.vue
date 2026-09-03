<template>
  <BaseDialog :show="show" :title="t('admin.users.balanceHistoryTitle')" width="wide" :close-on-click-outside="true" :z-index="40" @close="$emit('close')">
    <div v-if="user" class="space-y-4">
      <!-- User header: two-row layout with full user info -->
      <div class="rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <!-- Row 1: avatar + email/username/created_at (left) + current balance (right) -->
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
            <span class="text-lg font-medium text-primary-700 dark:text-primary-300">
              {{ user.email.charAt(0).toUpperCase() }}
            </span>
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <p class="truncate font-medium text-gray-900 dark:text-white">{{ user.email }}</p>
              <span v-if="user.deleted_at" class="flex-shrink-0 inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-100 text-rose-600 ring-1 ring-inset ring-rose-200 dark:bg-rose-500/20 dark:text-rose-400 dark:ring-rose-500/30">
                {{ t('admin.usage.userDeletedBadge') }}
              </span>
              <span
                v-if="user.username"
                class="flex-shrink-0 rounded bg-primary-50 px-1.5 py-0.5 text-xs text-primary-600 dark:bg-primary-900/20 dark:text-primary-400"
              >
                {{ user.username }}
              </span>
            </div>
            <p class="text-xs text-gray-400 dark:text-dark-500">
              {{ t('admin.users.createdAt') }}: {{ formatDateTime(user.created_at) }}
            </p>
          </div>
          <!-- Current balance: prominent display on the right -->
          <div class="flex-shrink-0 text-right">
            <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('admin.users.currentBalance') }}</p>
            <CreditAmount
              class="text-xl font-bold text-gray-900 dark:text-white"
              :value="(user.balance ?? 0).toFixed(2)"
              icon-size="md"
            />
          </div>
        </div>
        <!-- Row 2: notes + total recharged -->
        <div class="mt-2.5 flex items-center justify-between border-t border-gray-200/60 pt-2.5 dark:border-dark-600/60">
          <p class="min-w-0 flex-1 truncate text-xs text-gray-500 dark:text-dark-400" :title="user.notes || ''">
            <template v-if="user.notes">{{ t('admin.users.notes') }}: {{ user.notes }}</template>
            <template v-else>&nbsp;</template>
          </p>
          <p class="ml-4 flex-shrink-0 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ t('admin.users.totalRecharged') }}:</span>
            <CreditAmount
              class="ml-1 font-semibold text-emerald-600 dark:text-emerald-400"
              :value="totalRecharged.toFixed(2)"
              icon-size="xs"
            />
          </p>
        </div>
      </div>

      <!-- Type filter + Action buttons -->
      <div class="flex items-center gap-3">
        <Select
          v-model="typeFilter"
          :options="typeOptions"
          class="w-56"
          @change="loadHistory(1)"
        />
        <!-- Deposit button - matches menu style -->
        <button
          v-if="!hideActions"
          @click="emit('deposit')"
          class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700"
        >
          <Icon name="plus" size="sm" class="text-emerald-500" :stroke-width="2" />
          {{ t('admin.users.deposit') }}
        </button>
        <!-- Withdraw button - matches menu style -->
        <button
          v-if="!hideActions"
          @click="emit('withdraw')"
          class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:bg-dark-700"
        >
          <svg class="h-4 w-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
          </svg>
          {{ t('admin.users.withdraw') }}
        </button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>

      <!-- Load error -->
      <div v-else-if="loadError" data-testid="history-load-error" class="py-8 text-center">
        <p class="text-sm text-gray-500 dark:text-dark-400">{{ loadErrorMessage }}</p>
        <button
          type="button"
          class="btn btn-secondary mt-3 px-3 py-1.5 text-sm"
          data-testid="history-load-retry"
          @click="loadHistory(currentPage)"
        >
          {{ t('admin.users.retry') }}
        </button>
      </div>

      <!-- Actual subscriptions, not subscription-type redeem-code history -->
      <div v-else-if="isSubscriptionFilter && subscriptions.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500 dark:text-dark-400">
          {{ t('admin.subscriptions.noSubscriptionsYet') }}
        </p>
      </div>

      <div v-else-if="isSubscriptionFilter" class="max-h-[28rem] space-y-3 overflow-y-auto">
        <div
          v-for="subscription in subscriptions"
          :key="subscription.id"
          data-testid="subscription-record"
          class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="flex min-w-0 items-start gap-3">
              <div class="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg bg-purple-100 dark:bg-purple-900/30">
                <Icon name="badge" size="sm" class="text-purple-600 dark:text-purple-400" />
              </div>
              <div class="min-w-0">
                <p
                  class="truncate text-sm font-medium text-gray-900 dark:text-white"
                  :title="getSubscriptionGroupName(subscription)"
                >
                  {{ getSubscriptionGroupName(subscription) }}
                </p>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('admin.users.subscriptionStartsAt') }}:
                  {{ formatDateTime(subscription.starts_at) }}
                </p>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                  {{ t('admin.subscriptions.columns.expires') }}:
                  {{ subscription.expires_at
                    ? formatDateTime(subscription.expires_at)
                    : t('admin.subscriptions.noExpiration') }}
                </p>
              </div>
            </div>
            <span :class="['badge flex-shrink-0', getSubscriptionStatusClass(subscription.status)]">
              {{ t(`admin.subscriptions.status.${subscription.status}`) }}
            </span>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div v-else-if="history.length === 0" class="py-8 text-center">
        <p class="text-sm text-gray-500">{{ t('admin.users.noBalanceHistory') }}</p>
      </div>

      <!-- History list -->
      <div v-else class="max-h-[28rem] space-y-3 overflow-y-auto">
        <div
          v-for="item in history"
          :key="item.id"
          class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between">
            <!-- Left: type icon + description -->
            <div class="flex items-start gap-3">
              <div
                :class="[
                  'flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-lg',
                  getIconBg(item)
                ]"
              >
                <PointsIcon v-if="isBalanceType(item.type)" size="sm" />
                <Icon v-else :name="getIconName(item)" size="sm" :class="getIconColor(item)" />
              </div>
              <div>
                <p class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ getItemTitle(item) }}
                </p>
                <!-- Notes (admin adjustment reason) -->
                <p
                  v-if="item.notes"
                  class="mt-0.5 text-xs text-gray-500 dark:text-dark-400"
                  :title="item.notes"
                >
                  {{ item.notes.length > 60 ? item.notes.substring(0, 55) + '...' : item.notes }}
                </p>
                <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                  {{ formatDateTime(item.used_at || item.created_at) }}
                </p>
              </div>
            </div>
            <!-- Right: value -->
            <div class="text-right">
              <p :class="['text-sm font-semibold', getValueColor(item)]">
                <CreditAmount
                  v-if="isBalanceType(item.type)"
                  :value="formatValue(item)"
                  icon-size="xs"
                />
                <template v-else>{{ formatValue(item) }}</template>
              </p>
              <p
                v-if="isAdminType(item.type)"
                class="text-xs text-gray-400 dark:text-dark-500"
              >
                {{ t('redeem.adminAdjustment') }}
              </p>
              <p
                v-else
                class="font-mono text-xs text-gray-400 dark:text-dark-500"
              >
                {{ item.code.slice(0, 8) }}...
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="!isSubscriptionFilter && totalPages > 1" class="flex items-center justify-center gap-2 pt-2">
        <button
          :disabled="currentPage <= 1"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadHistory(currentPage - 1)"
        >
          {{ t('pagination.previous') }}
        </button>
        <span class="text-sm text-gray-500 dark:text-dark-400">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          :disabled="currentPage >= totalPages"
          class="btn btn-secondary px-3 py-1 text-sm"
          @click="loadHistory(currentPage + 1)"
        >
          {{ t('pagination.next') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI, type BalanceHistoryItem } from '@/api/admin'
import { formatDateTime } from '@/utils/format'
import type { AdminUser, UserSubscription } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import PointsIcon from '@/components/icons/PointsIcon.vue'

const props = defineProps<{ show: boolean; user: AdminUser | null; hideActions?: boolean }>()
const emit = defineEmits(['close', 'deposit', 'withdraw'])
const { t } = useI18n()

const history = ref<BalanceHistoryItem[]>([])
const subscriptions = ref<UserSubscription[]>([])
const loading = ref(false)
const loadError = ref<'history' | 'subscriptions' | null>(null)
const currentPage = ref(1)
const total = ref(0)
const totalRecharged = ref(0)
const pageSize = 15
const typeFilter = ref('')
let loadRequestId = 0
let userContextId = 0

const totalPages = computed(() => Math.ceil(total.value / pageSize) || 1)
const isSubscriptionFilter = computed(() => typeFilter.value === 'subscription')
const loadErrorMessage = computed(() =>
  loadError.value === 'subscriptions'
    ? t('admin.subscriptions.failedToLoad')
    : t('admin.users.failedToLoadBalanceHistory')
)

// Type filter options
const typeOptions = computed(() => [
  { value: '', label: t('admin.users.allTypes') },
  { value: 'balance', label: t('admin.users.typeBalance') },
  { value: 'affiliate_balance', label: t('admin.users.typeAffiliateBalance') },
  { value: 'admin_balance', label: t('admin.users.typeAdminBalance') },
  { value: 'concurrency', label: t('admin.users.typeConcurrency') },
  { value: 'admin_concurrency', label: t('admin.users.typeAdminConcurrency') },
  { value: 'subscription', label: t('admin.users.typeSubscription') }
])

// Reload when the modal opens or changes to another user while still open.
watch(() => [props.show, props.user?.id] as const, ([show, userId]) => {
  loadRequestId += 1
  userContextId += 1
  loading.value = false
  if (show && userId) {
    typeFilter.value = ''
    history.value = []
    subscriptions.value = []
    total.value = 0
    totalRecharged.value = 0
    loadError.value = null
    loadHistory(1)
  }
})

const loadHistory = async (page: number) => {
  if (!props.user) return
  const userId = props.user.id
  const contextId = userContextId
  const requestId = ++loadRequestId
  loading.value = true
  loadError.value = null
  currentPage.value = page
  try {
    if (isSubscriptionFilter.value) {
      const items = await adminAPI.subscriptions.listByUser(userId)
      if (requestId !== loadRequestId) return
      subscriptions.value = items
      history.value = []
      total.value = items.length
      return
    }

    const res = await adminAPI.users.getUserBalanceHistory(
      userId,
      page,
      pageSize,
      typeFilter.value || undefined
    )
    // Total recharge is user-level summary data. Preserve it even if this history
    // response became stale only because the user switched filters.
    if (contextId === userContextId) {
      totalRecharged.value = res.total_recharged || 0
    }
    if (requestId !== loadRequestId) return
    history.value = res.items || []
    subscriptions.value = []
    total.value = res.total || 0
  } catch (error) {
    if (requestId !== loadRequestId) return
    console.error(
      isSubscriptionFilter.value
        ? 'Failed to load user subscriptions:'
        : 'Failed to load balance history:',
      error
    )
    history.value = []
    subscriptions.value = []
    total.value = 0
    loadError.value = isSubscriptionFilter.value ? 'subscriptions' : 'history'
  } finally {
    if (requestId === loadRequestId) loading.value = false
  }
}

const getSubscriptionGroupName = (subscription: UserSubscription) => {
  const groupName = subscription.group?.name?.trim()
  return groupName || `#${subscription.group_id}`
}

const getSubscriptionStatusClass = (status: UserSubscription['status']) => {
  if (status === 'active') return 'badge-success'
  if (status === 'expired' || status === 'suspended') return 'badge-warning'
  return 'badge-danger'
}

// Helper: check if admin type
const isAdminType = (type: string) => type === 'admin_balance' || type === 'admin_concurrency'

// Helper: check if balance type (includes admin_balance)
const isBalanceType = (type: string) => type === 'balance' || type === 'admin_balance' || type === 'affiliate_balance'

// Icon name based on type
const getIconName = (item: BalanceHistoryItem) => {
  if (isBalanceType(item.type)) return 'dollar'
  return 'bolt' // concurrency
}

// Icon background color
const getIconBg = (item: BalanceHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'bg-emerald-100 dark:bg-emerald-900/30'
      : 'bg-red-100 dark:bg-red-900/30'
  }
  return item.value >= 0
    ? 'bg-blue-100 dark:bg-blue-900/30'
    : 'bg-orange-100 dark:bg-orange-900/30'
}

// Icon text color
const getIconColor = (item: BalanceHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-red-600 dark:text-red-400'
  }
  return item.value >= 0
    ? 'text-blue-600 dark:text-blue-400'
    : 'text-orange-600 dark:text-orange-400'
}

// Value text color
const getValueColor = (item: BalanceHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : 'text-red-600 dark:text-red-400'
  }
  return item.value >= 0
    ? 'text-blue-600 dark:text-blue-400'
    : 'text-orange-600 dark:text-orange-400'
}

// Item title
const getItemTitle = (item: BalanceHistoryItem) => {
  switch (item.type) {
    case 'balance':
      return t('redeem.balanceAddedRedeem')
    case 'affiliate_balance':
      return t('redeem.balanceAddedAffiliate')
    case 'admin_balance':
      return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
    case 'concurrency':
      return t('redeem.concurrencyAddedRedeem')
    case 'admin_concurrency':
      return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
    default:
      return t('common.unknown')
  }
}

// Format display value
const formatValue = (item: BalanceHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value.toFixed(2)}`
  }
  // concurrency types
  const sign = item.value >= 0 ? '+' : ''
  return `${sign}${item.value}`
}
</script>
