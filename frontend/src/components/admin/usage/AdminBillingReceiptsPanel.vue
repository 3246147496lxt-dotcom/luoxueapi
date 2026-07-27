<template>
  <section aria-labelledby="billing-receipts-heading">
    <div class="border-b border-gray-100 p-4 dark:border-dark-700/50 sm:p-6">
      <div class="flex flex-wrap items-end justify-between gap-4">
        <div class="flex flex-1 flex-wrap items-end gap-4">
          <div
            ref="userSearchRef"
            class="relative w-full sm:w-auto sm:min-w-[240px]"
          >
            <label for="billing-receipt-user" class="input-label">
              {{ t('admin.usage.billingReceipts.filters.user') }}
            </label>
            <input
              id="billing-receipt-user"
              v-model="userKeyword"
              type="text"
              class="input pr-9"
              :placeholder="t('admin.usage.billingReceipts.filters.userPlaceholder')"
              autocomplete="off"
              @focus="showUserDropdown = true"
              @input="handleUserInput"
              @keyup.enter="applyFilters"
            />
            <button
              v-if="userKeyword"
              type="button"
              class="absolute right-2 top-9 rounded p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:hover:bg-dark-700 dark:hover:text-gray-200"
              :title="t('admin.usage.billingReceipts.filters.clearUser')"
              :aria-label="t('admin.usage.billingReceipts.filters.clearUser')"
              @click="clearUser"
            >
              <Icon name="x" size="xs" />
            </button>
            <div
              v-if="showUserDropdown && userResults.length > 0"
              class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-md border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
            >
              <button
                v-for="user in userResults"
                :key="user.id"
                type="button"
                class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 focus:bg-gray-100 focus:outline-none dark:text-gray-200 dark:hover:bg-dark-700 dark:focus:bg-dark-700"
                @click="selectUser(user)"
              >
                <span class="min-w-0 truncate">{{ user.email }}</span>
                <span class="shrink-0 text-xs text-gray-400">#{{ user.id }}</span>
              </button>
            </div>
          </div>

          <div class="w-full sm:w-auto sm:min-w-[220px]">
            <label for="billing-receipt-model" class="input-label">
              {{ t('admin.usage.billingReceipts.filters.model') }}
            </label>
            <input
              id="billing-receipt-model"
              v-model.trim="filters.model"
              type="text"
              class="input"
              :placeholder="t('admin.usage.billingReceipts.filters.modelPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </div>

          <div class="w-full sm:w-auto sm:min-w-[240px]">
            <label for="billing-receipt-id" class="input-label">
              {{ t('admin.usage.billingReceipts.filters.receiptId') }}
            </label>
            <input
              id="billing-receipt-id"
              v-model.trim="filters.receipt_id"
              type="text"
              class="input font-mono"
              :placeholder="t('admin.usage.billingReceipts.filters.receiptIdPlaceholder')"
              @keyup.enter="applyFilters"
            />
          </div>

          <div class="w-full sm:w-auto sm:min-w-[180px]">
            <label class="input-label">
              {{ t('admin.usage.billingReceipts.filters.status') }}
            </label>
            <Select
              v-model="filters.status"
              :options="statusOptions"
              @change="applyFilters"
            />
          </div>
        </div>

        <div class="flex w-full items-center justify-end gap-3 sm:w-auto">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="applyFilters"
          >
            <Icon name="search" size="sm" />
            <span>{{ t('common.search') }}</span>
          </button>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="resetFilters"
          >
            {{ t('common.reset') }}
          </button>
        </div>
      </div>
    </div>

    <h2 id="billing-receipts-heading" class="sr-only">
      {{ t('admin.usage.billingReceipts.title') }}
    </h2>

    <div class="overflow-hidden">
      <DataTable
        :columns="columns"
        :data="receipts"
        :loading="loading"
        :error="loadError"
        row-key="receipt_id"
        mobile-primary-key="receipt_id"
        :mobile-visible-keys="['user', 'model', 'tokens', 'cost', 'balance', 'status', 'created_at']"
      >
        <template #cell-user="{ row }">
          <div class="max-w-[240px] text-sm">
            <div class="truncate font-medium text-gray-900 dark:text-white">
              {{ receiptUserEmail(row) || '-' }}
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">#{{ row.user_id }}</div>
          </div>
        </template>

        <template #cell-receipt_id="{ row }">
          <code
            class="block max-w-[220px] truncate text-xs text-gray-700 dark:text-gray-300"
            :title="row.receipt_id"
          >{{ row.receipt_id }}</code>
        </template>

        <template #cell-model="{ row }">
          <div class="max-w-[260px] space-y-0.5 text-xs">
            <div
              v-if="row.requested_model && row.requested_model !== row.actual_model"
              class="truncate text-gray-500 dark:text-gray-400"
              :title="row.requested_model"
            >
              {{ t('usage.requestedModel') }}: {{ row.requested_model }}
            </div>
            <div
              class="truncate font-medium text-gray-900 dark:text-white"
              :title="row.actual_model"
            >
              {{ t('usage.actualModel') }}: {{ row.actual_model || '-' }}
            </div>
          </div>
        </template>

        <template #cell-tokens="{ row }">
          <div class="space-y-1 text-xs tabular-nums">
            <div class="flex items-center gap-3">
              <span class="inline-flex items-center gap-1 text-gray-700 dark:text-gray-300">
                <Icon name="arrowDown" size="xs" class="text-emerald-500" />
                {{ row.tokens.input_tokens.toLocaleString() }}
              </span>
              <span class="inline-flex items-center gap-1 text-gray-700 dark:text-gray-300">
                <Icon name="arrowUp" size="xs" class="text-violet-500" />
                {{ row.tokens.output_tokens.toLocaleString() }}
              </span>
            </div>
            <div
              v-if="row.tokens.cache_tokens > 0"
              class="text-sky-600 dark:text-sky-400"
            >
              {{ t('usage.cacheTotal') }} {{ row.tokens.cache_tokens.toLocaleString() }}
            </div>
          </div>
        </template>

        <template #cell-cost="{ row }">
          <div class="space-y-0.5 text-sm tabular-nums">
            <CreditAmount
              class="font-medium text-green-600 dark:text-green-400"
              :value="formatCredits(row.charged_amount)"
              icon-size="xs"
              :label="`${t('usage.chargedAmount')} ${formatCredits(row.charged_amount)}`"
            />
            <div
              v-if="amountsDiffer(row.gross_cost, row.charged_amount)"
              class="text-[11px] text-gray-500 dark:text-gray-400"
            >
              {{ t('usage.grossCost') }} ${{ formatCredits(row.gross_cost) }}
            </div>
          </div>
        </template>

        <template #cell-balance="{ row }">
          <div
            v-if="row.balance_before != null && row.balance_after != null"
            class="flex items-center gap-1 text-xs tabular-nums text-gray-700 dark:text-gray-300"
          >
            <CreditAmount :value="formatCredits(row.balance_before)" icon-size="xs" />
            <Icon name="arrowRight" size="xs" class="shrink-0 text-gray-400" />
            <CreditAmount :value="formatCredits(row.balance_after)" icon-size="xs" />
          </div>
          <span v-else class="text-gray-400">-</span>
        </template>

        <template #cell-status="{ row }">
          <div class="max-w-[240px] space-y-1">
            <span
              class="inline-flex items-center rounded px-2 py-0.5 text-xs font-medium"
              :class="statusBadgeClass(row.status)"
            >
              {{ statusLabel(row.status) }}
            </span>
            <p
              v-if="row.failure_reason || row.failure_code"
              class="line-clamp-2 whitespace-normal text-xs text-red-600 dark:text-red-400"
              :title="row.failure_reason || row.failure_code || ''"
            >
              {{ row.failure_reason || row.failure_code }}
            </p>
          </div>
        </template>

        <template #cell-created_at="{ value }">
          <span class="text-xs text-gray-600 dark:text-gray-400">
            {{ formatDateTime(value) }}
          </span>
        </template>

        <template #empty>
          <EmptyState
            :title="t('admin.usage.billingReceipts.empty')"
            :description="t('admin.usage.billingReceipts.emptyDescription')"
          />
        </template>
      </DataTable>

      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import {
  adminUsageAPI,
  type AdminBillingReceipt,
  type SimpleUser,
} from '@/api/admin/usage'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatDateTime } from '@/utils/format'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'

const props = defineProps<{
  startDate: string
  endDate: string
}>()

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

const receipts = ref<AdminBillingReceipt[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)
const filters = reactive({
  user_id: undefined as number | undefined,
  model: '',
  receipt_id: '',
  status: '',
})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
})

const userKeyword = ref('')
const userResults = ref<SimpleUser[]>([])
const showUserDropdown = ref(false)
const userSearchRef = ref<HTMLElement | null>(null)
let userSearchTimer: ReturnType<typeof setTimeout> | null = null
let abortController: AbortController | null = null
let requestSequence = 0

const columns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.usage.billingReceipts.columns.user') },
  { key: 'receipt_id', label: t('admin.usage.billingReceipts.columns.receiptId') },
  { key: 'model', label: t('admin.usage.billingReceipts.columns.model') },
  { key: 'tokens', label: t('admin.usage.billingReceipts.columns.tokens') },
  { key: 'cost', label: t('admin.usage.billingReceipts.columns.cost') },
  { key: 'balance', label: t('admin.usage.billingReceipts.columns.balance') },
  { key: 'status', label: t('admin.usage.billingReceipts.columns.status') },
  { key: 'created_at', label: t('admin.usage.billingReceipts.columns.time') },
])

const statusOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('admin.usage.billingReceipts.statuses.all') },
  { value: 'pending', label: t('admin.usage.billingReceipts.statuses.pending') },
  { value: 'charged', label: t('admin.usage.billingReceipts.statuses.charged') },
  { value: 'subscription', label: t('admin.usage.billingReceipts.statuses.subscription') },
  { value: 'failed', label: t('admin.usage.billingReceipts.statuses.failed') },
  { value: 'not_charged', label: t('admin.usage.billingReceipts.statuses.notCharged') },
])

const singleQueryValue = (value: unknown): string => {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? ''
  }
  return typeof value === 'string' ? value : ''
}

const applyRouteQuery = () => {
  filters.receipt_id = singleQueryValue(route.query.receipt_id)
  filters.model = singleQueryValue(route.query.model)
  filters.status = singleQueryValue(route.query.status)

  const userId = Number(singleQueryValue(route.query.user_id))
  if (Number.isFinite(userId) && userId > 0) {
    filters.user_id = userId
    userKeyword.value = `#${userId}`
  }
}

const receiptUserEmail = (receipt: AdminBillingReceipt) =>
  receipt.user_email || receipt.user?.email || ''

const formatCredits = (value: number) => value.toFixed(6)
const amountsDiffer = (left: number, right: number) => Math.abs(left - right) > 0.0000005

const normalizedStatus = (status: string) => status.trim().toLowerCase()

const statusKind = (
  status: string
): 'charged' | 'subscription' | 'pending' | 'failed' | 'notCharged' | 'unknown' => {
  const value = normalizedStatus(status)
  if (['charged', 'success', 'succeeded', 'completed'].includes(value)) return 'charged'
  if (value === 'subscription') return 'subscription'
  if (['pending', 'processing', 'recording'].includes(value)) return 'pending'
  if (['failed', 'error'].includes(value)) return 'failed'
  if (['not_charged', 'skipped', 'cancelled', 'canceled'].includes(value)) return 'notCharged'
  return 'unknown'
}

const statusLabel = (status: string) => {
  const kind = statusKind(status)
  if (kind === 'charged') return t('admin.usage.billingReceipts.statuses.charged')
  if (kind === 'subscription') return t('admin.usage.billingReceipts.statuses.subscription')
  if (kind === 'pending') return t('admin.usage.billingReceipts.statuses.pending')
  if (kind === 'failed') return t('admin.usage.billingReceipts.statuses.failed')
  if (kind === 'notCharged') return t('admin.usage.billingReceipts.statuses.notCharged')
  return status || t('usage.unknown')
}

const statusBadgeClass = (status: string) => {
  const kind = statusKind(status)
  if (kind === 'charged') {
    return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-200'
  }
  if (kind === 'pending') {
    return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200'
  }
  if (kind === 'subscription') {
    return 'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-200'
  }
  if (kind === 'failed') {
    return 'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-200'
  }
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
}

const loadReceipts = async () => {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  const sequence = ++requestSequence
  loading.value = true
  loadError.value = null

  try {
    const response = await adminUsageAPI.listBillingReceipts({
      page: pagination.page,
      page_size: pagination.page_size,
      source: 'web_chat',
      start_date: props.startDate,
      end_date: props.endDate,
      user_id: filters.user_id,
      model: filters.model.trim() || undefined,
      receipt_id: filters.receipt_id.trim() || undefined,
      status: filters.status || undefined,
    }, {
      signal: controller.signal,
    })

    if (controller.signal.aborted || sequence !== requestSequence) return
    receipts.value = response.items
    pagination.total = response.total
  } catch (error: any) {
    if (controller.signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') return
    loadError.value = t('admin.usage.billingReceipts.loadFailed')
    appStore.showError(loadError.value)
  } finally {
    if (abortController === controller) loading.value = false
  }
}

const applyFilters = () => {
  pagination.page = 1
  showUserDropdown.value = false
  void loadReceipts()
}

const resetFilters = () => {
  filters.user_id = undefined
  filters.model = ''
  filters.receipt_id = ''
  filters.status = ''
  userKeyword.value = ''
  userResults.value = []
  applyFilters()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  void loadReceipts()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadReceipts()
}

const clearUser = () => {
  filters.user_id = undefined
  userKeyword.value = ''
  userResults.value = []
  showUserDropdown.value = false
}

const selectUser = (user: SimpleUser) => {
  filters.user_id = user.id
  userKeyword.value = user.email
  userResults.value = []
  showUserDropdown.value = false
  applyFilters()
}

const handleUserInput = () => {
  filters.user_id = undefined
  if (userSearchTimer) clearTimeout(userSearchTimer)
  const keyword = userKeyword.value.trim()
  if (keyword.length < 2) {
    userResults.value = []
    return
  }

  userSearchTimer = setTimeout(async () => {
    try {
      userResults.value = await adminUsageAPI.searchUsers(keyword)
      showUserDropdown.value = true
    } catch {
      userResults.value = []
    }
  }, 250)
}

const handleDocumentClick = (event: MouseEvent) => {
  if (userSearchRef.value && !userSearchRef.value.contains(event.target as Node)) {
    showUserDropdown.value = false
  }
}

watch(
  () => [props.startDate, props.endDate],
  () => {
    pagination.page = 1
    void loadReceipts()
  }
)

onMounted(() => {
  applyRouteQuery()
  document.addEventListener('click', handleDocumentClick)
  void loadReceipts()
})

onUnmounted(() => {
  abortController?.abort()
  if (userSearchTimer) clearTimeout(userSearchTimer)
  document.removeEventListener('click', handleDocumentClick)
})

defineExpose({ reload: loadReceipts })
</script>
