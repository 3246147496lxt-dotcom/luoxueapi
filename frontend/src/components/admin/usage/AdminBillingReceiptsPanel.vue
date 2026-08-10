<template>
  <section class="billing-receipts" aria-labelledby="billing-receipts-heading">
    <header class="billing-receipts__header">
      <div class="min-w-0">
        <div class="billing-receipts__title-row">
          <h2 id="billing-receipts-heading" class="billing-receipts__title">
            {{ t('admin.usage.billingReceipts.title') }}
          </h2>
          <span class="billing-receipts__readonly-badge">
            <Icon name="lock" size="xs" aria-hidden="true" />
            {{ t('admin.usage.billingReceipts.readOnly') }}
          </span>
        </div>
        <p class="billing-receipts__subtitle">
          {{ t('admin.usage.billingReceipts.subtitle') }}
        </p>
      </div>
      <div class="billing-receipts__header-note">
        <Icon name="shield" size="sm" aria-hidden="true" />
        <span>{{ t('admin.usage.billingReceipts.readOnlyDescription') }}</span>
      </div>
    </header>

    <div class="billing-receipts__filters">
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
              class="input pr-11"
              :placeholder="t('admin.usage.billingReceipts.filters.userPlaceholder')"
              autocomplete="off"
              @focus="showUserDropdown = true"
              @input="handleUserInput"
              @keyup.enter="applyFilters"
            />
            <button
              v-if="userKeyword"
              type="button"
              class="billing-receipts__input-clear"
              :title="t('admin.usage.billingReceipts.filters.clearUser')"
              :aria-label="t('admin.usage.billingReceipts.filters.clearUser')"
              @click="clearUser"
            >
              <Icon name="x" size="xs" />
            </button>
            <div
              v-if="showUserDropdown && userResults.length > 0"
              class="billing-receipts__user-results"
            >
              <button
                v-for="user in userResults"
                :key="user.id"
                type="button"
                class="billing-receipts__user-option"
                @click="selectUser(user)"
              >
                <span class="min-w-0">
                  <span
                    v-if="simpleUserName(user)"
                    class="block truncate font-semibold"
                  >{{ simpleUserName(user) }}</span>
                  <span class="block truncate" :class="simpleUserName(user) ? 'text-xs opacity-70' : ''">
                    {{ user.email }}
                  </span>
                </span>
                <span class="shrink-0 text-xs opacity-60">#{{ user.id }}</span>
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
            <label for="billing-request-id" class="input-label">
              {{ t('admin.usage.billingReceipts.identifiers.requestId') }}
            </label>
            <input
              id="billing-request-id"
              v-model.trim="filters.request_id"
              type="text"
              class="input font-mono"
              :placeholder="t('admin.usage.billingReceipts.identifiers.requestId')"
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

    <div class="overflow-hidden">
      <DataTable
        :columns="columns"
        :data="receipts"
        :loading="loading"
        :error="loadError"
        row-key="row_key"
        mobile-primary-key="receipt_id"
        :mobile-visible-keys="['user', 'model', 'tokens', 'cost', 'balance', 'status', 'created_at']"
        clickable-rows
        :row-aria-label="receiptRowAriaLabel"
        :selected-row-key="selectedReceipt?.row_key ?? null"
        @row-click="openReceipt"
      >
        <template #cell-user="{ row }">
          <div class="billing-receipts__user-cell">
            <div class="billing-receipts__user-primary" :title="receiptUserPrimary(row)">
              {{ receiptUserPrimary(row) }}
            </div>
            <div v-if="receiptUserSecondary(row)" class="billing-receipts__user-secondary">
              {{ receiptUserSecondary(row) }}
            </div>
            <div class="billing-receipts__user-id">
              #{{ row.user_id }}
              <span v-if="row.user?.deleted">· {{ t('admin.usage.userDeletedBadge') }}</span>
            </div>
          </div>
        </template>

        <template #cell-receipt_id="{ row }">
          <div class="billing-receipts__identifier-stack">
            <div v-if="hasReceiptId(row)" class="billing-receipts__identifier-line">
              <div class="min-w-0">
                <span class="billing-receipts__identifier-label">
                  {{ t('admin.usage.billingReceipts.identifiers.receiptId') }}
                </span>
                <code class="billing-receipts__identifier" :title="row.receipt_id">
                  {{ row.receipt_id }}
                </code>
              </div>
              <button
                type="button"
                class="billing-receipts__copy-button"
                :title="t('admin.usage.billingReceipts.copyReceiptId')"
                :aria-label="t('admin.usage.billingReceipts.copyReceiptId')"
                :data-test="`copy-receipt-${row.row_key}`"
                @click.stop="copyReceiptValue(row.receipt_id, `receipt-${row.row_key}`)"
              >
                <Icon :name="isCopied(`receipt-${row.row_key}`) ? 'check' : 'copy'" size="sm" />
              </button>
            </div>
            <div v-else class="billing-receipts__identifier-line">
              <div class="min-w-0">
                <span class="billing-receipts__identifier-label">
                  {{ t('admin.usage.billingReceipts.identifiers.receiptId') }}
                </span>
                <span class="billing-receipts__muted-value">
                  {{ t('admin.usage.billingReceipts.identifiers.receiptNotGenerated') }}
                </span>
              </div>
            </div>
            <div
              v-if="hasRequestId(row)"
              class="billing-receipts__identifier-line billing-receipts__identifier-line--secondary"
            >
              <div class="min-w-0">
                <span class="billing-receipts__identifier-label">
                  {{ t('admin.usage.billingReceipts.identifiers.requestId') }}
                </span>
                <code class="billing-receipts__identifier" :title="row.request_id || ''">
                  {{ row.request_id }}
                </code>
              </div>
              <button
                type="button"
                class="billing-receipts__copy-button"
                :title="t('admin.usage.billingReceipts.copyRequestId')"
                :aria-label="t('admin.usage.billingReceipts.copyRequestId')"
                :data-test="`copy-request-${row.row_key}`"
                @click.stop="copyReceiptValue(row.request_id || '', `request-${row.row_key}`)"
              >
                <Icon :name="isCopied(`request-${row.row_key}`) ? 'check' : 'copy'" size="sm" />
              </button>
            </div>
          </div>
        </template>

        <template #cell-model="{ row }">
          <div class="billing-receipts__model-cell">
            <div
              v-if="row.requested_model && row.requested_model !== row.actual_model"
              class="billing-receipts__model-requested"
              :title="row.requested_model"
            >
              {{ t('usage.requestedModel') }} · {{ row.requested_model }}
            </div>
            <div class="billing-receipts__model-actual" :title="row.actual_model">
              {{ row.actual_model || '-' }}
            </div>
          </div>
        </template>

        <template #cell-tokens="{ row }">
          <div class="billing-receipts__tokens-cell">
            <strong>{{ formatCount(totalTokens(row)) }}</strong>
            <span>
              {{ formatCount(row.tokens.input_tokens) }} / {{ formatCount(row.tokens.output_tokens) }}
              <template v-if="cacheTokens(row) > 0"> / {{ formatCount(cacheTokens(row)) }}</template>
            </span>
          </div>
        </template>

        <template #cell-cost="{ row }">
          <div class="billing-receipts__cost-cell">
            <CreditAmount
              class="billing-receipts__charged-amount"
              :value="formatCredits(row.charged_amount)"
              icon-size="xs"
              :label="`${t('usage.chargedAmount')} ${formatCredits(row.charged_amount)}`"
            />
            <span v-if="amountsDiffer(row.gross_cost, row.charged_amount)">
              {{ t('usage.grossCost') }} ${{ formatCredits(row.gross_cost) }}
            </span>
          </div>
        </template>

        <template #cell-balance="{ row }">
          <div
            v-if="row.balance_before != null && row.balance_after != null"
            class="billing-receipts__balance-cell"
          >
            <CreditAmount :value="formatCredits(row.balance_before)" icon-size="xs" />
            <Icon name="arrowRight" size="xs" class="shrink-0" aria-hidden="true" />
            <CreditAmount :value="formatCredits(row.balance_after)" icon-size="xs" />
          </div>
          <span v-else class="billing-receipts__muted-value">-</span>
        </template>

        <template #cell-status="{ row }">
          <div class="billing-receipts__status-cell">
            <span class="billing-receipts__status-badge" :class="statusBadgeClass(row.status)">
              <span class="billing-receipts__status-dot" aria-hidden="true"></span>
              {{ statusLabel(row.status) }}
            </span>
            <p
              v-if="row.failure_reason || row.failure_code"
              class="billing-receipts__failure-preview"
              :title="row.failure_reason || row.failure_code || ''"
            >
              <span v-if="row.failure_code" class="font-semibold">{{ row.failure_code }}</span>
              <span v-if="row.failure_reason">{{ row.failure_reason }}</span>
            </p>
          </div>
        </template>

        <template #cell-created_at="{ value }">
          <span class="billing-receipts__time-cell">
            {{ formatDateTime(value) }}
          </span>
        </template>

        <template #cell-actions="{ row }">
          <button
            type="button"
            class="billing-receipts__details-button"
            :title="t('admin.usage.billingReceipts.viewDetails')"
            :aria-label="receiptRowAriaLabel(row)"
            @click.stop="openReceipt(row)"
          >
            <Icon name="chevronRight" size="sm" aria-hidden="true" />
          </button>
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

  <Teleport to="body">
    <Transition name="billing-receipt-drawer">
      <div
        v-if="selectedReceipt"
        class="billing-receipts__drawer-overlay"
        data-test="billing-receipt-drawer-overlay"
        @click.self="closeReceipt"
      >
        <aside
          ref="drawerRef"
          class="billing-receipts__drawer"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="drawerTitleId"
          tabindex="-1"
          data-test="billing-receipt-drawer"
          @click.stop
        >
          <header class="billing-receipts__drawer-header">
            <div class="min-w-0">
              <div class="billing-receipts__drawer-title-row">
                <h3 :id="drawerTitleId" class="billing-receipts__drawer-title">
                  {{ t('admin.usage.billingReceipts.drawer.title') }}
                </h3>
                <span class="billing-receipts__readonly-badge">
                  <Icon name="lock" size="xs" aria-hidden="true" />
                  {{ t('admin.usage.billingReceipts.readOnly') }}
                </span>
              </div>
              <p class="billing-receipts__drawer-subtitle">
                {{ t('admin.usage.billingReceipts.drawer.subtitle') }}
              </p>
            </div>
            <button
              ref="drawerCloseRef"
              type="button"
              class="billing-receipts__drawer-close"
              :title="t('common.close')"
              :aria-label="t('common.close')"
              data-test="billing-receipt-drawer-close"
              @click="closeReceipt"
            >
              <Icon name="x" size="md" />
            </button>
          </header>

          <div class="billing-receipts__drawer-scroll">
            <div class="billing-receipts__drawer-summary">
              <span
                class="billing-receipts__status-badge"
                :class="statusBadgeClass(selectedReceipt.status)"
              >
                <span class="billing-receipts__status-dot" aria-hidden="true"></span>
                {{ statusLabel(selectedReceipt.status) }}
              </span>
              <div class="min-w-0">
                <span class="billing-receipts__drawer-summary-label">
                  {{ hasReceiptId(selectedReceipt)
                    ? t('admin.usage.billingReceipts.identifiers.receiptId')
                    : t('admin.usage.billingReceipts.identifiers.requestId') }}
                </span>
                <code class="billing-receipts__drawer-summary-id">
                  {{ receiptAuditIdentifier(selectedReceipt) }}
                </code>
              </div>
            </div>

            <section class="billing-receipts__drawer-section" aria-labelledby="billing-audit-identifiers">
              <h4 id="billing-audit-identifiers" class="billing-receipts__section-title">
                {{ t('admin.usage.billingReceipts.drawer.sections.identifiers') }}
              </h4>
              <dl class="billing-receipts__detail-list">
                <div class="billing-receipts__detail-row">
                  <dt>
                    {{ t('admin.usage.billingReceipts.identifiers.receiptId') }}
                  </dt>
                  <dd v-if="hasReceiptId(selectedReceipt)" class="billing-receipts__detail-with-action">
                    <code :title="selectedReceipt.receipt_id">{{ selectedReceipt.receipt_id }}</code>
                    <button
                      type="button"
                      class="billing-receipts__copy-button"
                      :title="t('admin.usage.billingReceipts.copyReceiptId')"
                      :aria-label="t('admin.usage.billingReceipts.copyReceiptId')"
                      data-test="drawer-copy-receipt-id"
                      @click="copyReceiptValue(selectedReceipt.receipt_id, 'drawer-receipt')"
                    >
                      <Icon :name="isCopied('drawer-receipt') ? 'check' : 'copy'" size="sm" />
                    </button>
                  </dd>
                  <dd v-else class="billing-receipts__muted-value">
                    {{ t('admin.usage.billingReceipts.identifiers.receiptNotGenerated') }}
                  </dd>
                </div>
                <div v-if="hasRequestId(selectedReceipt)" class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.identifiers.requestId') }}</dt>
                  <dd class="billing-receipts__detail-with-action">
                    <code :title="selectedReceipt.request_id || ''">{{ selectedReceipt.request_id }}</code>
                    <button
                      type="button"
                      class="billing-receipts__copy-button"
                      :title="t('admin.usage.billingReceipts.copyRequestId')"
                      :aria-label="t('admin.usage.billingReceipts.copyRequestId')"
                      data-test="drawer-copy-request-id"
                      @click="copyReceiptValue(selectedReceipt.request_id || '', 'drawer-request')"
                    >
                      <Icon :name="isCopied('drawer-request') ? 'check' : 'copy'" size="sm" />
                    </button>
                  </dd>
                </div>
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.user') }}</dt>
                  <dd>
                    <strong>{{ receiptUserPrimary(selectedReceipt) }}</strong>
                    <span v-if="receiptUserSecondary(selectedReceipt)">
                      {{ receiptUserSecondary(selectedReceipt) }}
                    </span>
                    <span>#{{ selectedReceipt.user_id }}</span>
                  </dd>
                </div>
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.source') }}</dt>
                  <dd>{{ sourceLabel(selectedReceipt.source) }}</dd>
                </div>
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.createdAt') }}</dt>
                  <dd>{{ formatDateTime(selectedReceipt.created_at) }}</dd>
                </div>
              </dl>
            </section>

            <section class="billing-receipts__drawer-section" aria-labelledby="billing-model-routing">
              <h4 id="billing-model-routing" class="billing-receipts__section-title">
                {{ t('admin.usage.billingReceipts.drawer.sections.models') }}
              </h4>
              <dl class="billing-receipts__detail-list">
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('usage.requestedModel') }}</dt>
                  <dd><code>{{ selectedReceipt.requested_model || '-' }}</code></dd>
                </div>
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('usage.actualModel') }}</dt>
                  <dd><code>{{ selectedReceipt.actual_model || '-' }}</code></dd>
                </div>
              </dl>
            </section>

            <section class="billing-receipts__drawer-section" aria-labelledby="billing-token-breakdown">
              <div class="billing-receipts__section-heading-row">
                <h4 id="billing-token-breakdown" class="billing-receipts__section-title">
                  {{ t('admin.usage.billingReceipts.drawer.sections.tokens') }}
                </h4>
                <strong class="billing-receipts__section-total">
                  {{ formatCount(totalTokens(selectedReceipt)) }}
                </strong>
              </div>
              <dl class="billing-receipts__metric-grid">
                <div>
                  <dt>{{ t('admin.usage.inputTokens') }}</dt>
                  <dd>{{ formatCount(selectedReceipt.tokens.input_tokens) }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.outputTokens') }}</dt>
                  <dd>{{ formatCount(selectedReceipt.tokens.output_tokens) }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.cacheReadTokens') }}</dt>
                  <dd>{{ formatCount(selectedReceipt.tokens.cache_read_tokens || 0) }}</dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.cacheCreationTokens') }}</dt>
                  <dd>{{ formatCount(selectedReceipt.tokens.cache_creation_tokens || 0) }}</dd>
                </div>
                <div class="billing-receipts__metric-grid-total">
                  <dt>{{ t('usage.cacheTotal') }}</dt>
                  <dd>{{ formatCount(cacheTokens(selectedReceipt)) }}</dd>
                </div>
              </dl>
            </section>

            <section class="billing-receipts__drawer-section" aria-labelledby="billing-settlement-math">
              <h4 id="billing-settlement-math" class="billing-receipts__section-title">
                {{ t('admin.usage.billingReceipts.drawer.sections.settlement') }}
              </h4>
              <dl class="billing-receipts__settlement-list">
                <div>
                  <dt>{{ t('usage.grossCost') }}</dt>
                  <dd><CreditAmount :value="formatCredits(selectedReceipt.gross_cost)" icon-size="sm" /></dd>
                </div>
                <div class="billing-receipts__settlement-list-emphasis">
                  <dt>{{ t('usage.chargedAmount') }}</dt>
                  <dd><CreditAmount :value="formatCredits(selectedReceipt.charged_amount)" icon-size="sm" /></dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.balanceBefore') }}</dt>
                  <dd>
                    <CreditAmount
                      v-if="selectedReceipt.balance_before != null"
                      :value="formatCredits(selectedReceipt.balance_before)"
                      icon-size="sm"
                    />
                    <span v-else>-</span>
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.balanceAfter') }}</dt>
                  <dd>
                    <CreditAmount
                      v-if="selectedReceipt.balance_after != null"
                      :value="formatCredits(selectedReceipt.balance_after)"
                      icon-size="sm"
                    />
                    <span v-else>-</span>
                  </dd>
                </div>
                <div>
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.overdraft') }}</dt>
                  <dd :class="selectedReceipt.overdraft ? 'billing-receipts__danger-text' : ''">
                    {{ selectedReceipt.overdraft
                      ? t('admin.usage.billingReceipts.drawer.values.yes')
                      : t('admin.usage.billingReceipts.drawer.values.no') }}
                  </dd>
                </div>
              </dl>
            </section>

            <section
              v-if="selectedReceipt.failure_code || selectedReceipt.failure_reason"
              class="billing-receipts__drawer-section billing-receipts__drawer-section--failure"
              aria-labelledby="billing-failure-context"
            >
              <h4 id="billing-failure-context" class="billing-receipts__section-title">
                {{ t('admin.usage.billingReceipts.drawer.sections.failure') }}
              </h4>
              <dl class="billing-receipts__detail-list">
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.failureCode') }}</dt>
                  <dd><code>{{ selectedReceipt.failure_code || '-' }}</code></dd>
                </div>
                <div class="billing-receipts__detail-row">
                  <dt>{{ t('admin.usage.billingReceipts.drawer.fields.failureReason') }}</dt>
                  <dd class="billing-receipts__failure-reason">
                    {{ selectedReceipt.failure_reason || '-' }}
                  </dd>
                </div>
              </dl>
            </section>

            <p class="billing-receipts__drawer-readonly-note">
              <Icon name="lock" size="sm" aria-hidden="true" />
              {{ t('admin.usage.billingReceipts.drawer.readOnlyNote') }}
            </p>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import {
  adminUsageAPI,
  type AdminBillingReceipt,
  type SimpleUser,
} from '@/api/admin/usage'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useClipboard } from '@/composables/useClipboard'
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
const { copyToClipboard } = useClipboard()

const receipts = ref<AdminBillingReceipt[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)
const filters = reactive({
  user_id: undefined as number | undefined,
  model: '',
  request_id: '',
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
const selectedReceipt = ref<AdminBillingReceipt | null>(null)
const drawerRef = ref<HTMLElement | null>(null)
const drawerCloseRef = ref<HTMLButtonElement | null>(null)
const copiedKey = ref<string | null>(null)
const drawerTitleId = `billing-receipt-drawer-title-${Math.random().toString(36).slice(2, 9)}`
let userSearchTimer: ReturnType<typeof setTimeout> | null = null
let copyFeedbackTimer: ReturnType<typeof setTimeout> | null = null
let abortController: AbortController | null = null
let requestSequence = 0
let previousActiveElement: HTMLElement | null = null

const BODY_LOCK_CLASS = 'billing-receipt-drawer-open'
const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const columns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.usage.billingReceipts.columns.user') },
  { key: 'receipt_id', label: t('admin.usage.billingReceipts.columns.receiptId') },
  { key: 'model', label: t('admin.usage.billingReceipts.columns.model') },
  { key: 'tokens', label: t('admin.usage.billingReceipts.columns.tokens') },
  { key: 'cost', label: t('admin.usage.billingReceipts.columns.cost') },
  { key: 'balance', label: t('admin.usage.billingReceipts.columns.balance') },
  { key: 'status', label: t('admin.usage.billingReceipts.columns.status') },
  { key: 'created_at', label: t('admin.usage.billingReceipts.columns.time') },
  { key: 'actions', label: t('admin.usage.billingReceipts.columns.details'), class: 'w-16' },
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
  filters.request_id = singleQueryValue(route.query.request_id)
    || singleQueryValue(route.query.receipt_id)
  filters.model = singleQueryValue(route.query.model)
  filters.status = singleQueryValue(route.query.status)

  const userId = Number(singleQueryValue(route.query.user_id))
  if (Number.isFinite(userId) && userId > 0) {
    filters.user_id = userId
    userKeyword.value = `#${userId}`
  }
}

const firstNonEmptyString = (...values: unknown[]) => {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

const receiptUserEmail = (receipt: AdminBillingReceipt) =>
  firstNonEmptyString(receipt.user_email, receipt.user?.email)

const receiptUsername = (receipt: AdminBillingReceipt) => {
  return firstNonEmptyString(
    receipt.username,
    receipt.user_username,
    receipt.user?.username,
  )
}

const receiptUserPrimary = (receipt: AdminBillingReceipt) =>
  receiptUsername(receipt) || receiptUserEmail(receipt) || `#${receipt.user_id}`

const receiptUserSecondary = (receipt: AdminBillingReceipt) =>
  receiptUsername(receipt) ? receiptUserEmail(receipt) : ''

const simpleUserName = (user: SimpleUser) =>
  firstNonEmptyString((user as SimpleUser & { username?: string | null }).username)

const formatCredits = (value: number) => value.toFixed(6)
const formatCount = (value: number) => value.toLocaleString()
const amountsDiffer = (left: number, right: number) => Math.abs(left - right) > 0.0000005

const cacheTokens = (receipt: AdminBillingReceipt) => {
  if (receipt.tokens.cache_tokens > 0) return receipt.tokens.cache_tokens
  return (receipt.tokens.cache_read_tokens || 0) + (receipt.tokens.cache_creation_tokens || 0)
}

const totalTokens = (receipt: AdminBillingReceipt) =>
  receipt.tokens.input_tokens + receipt.tokens.output_tokens + cacheTokens(receipt)

const hasRequestId = (receipt: AdminBillingReceipt) =>
  Boolean(firstNonEmptyString(receipt.request_id))

const hasReceiptId = (receipt: AdminBillingReceipt) =>
  Boolean(firstNonEmptyString(receipt.receipt_id))

const receiptAuditIdentifier = (receipt: AdminBillingReceipt) =>
  firstNonEmptyString(receipt.receipt_id, receipt.request_id, receipt.row_key)

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

const statusBadgeClass = (status: string) =>
  `billing-receipts__status-badge--${statusKind(status)}`

const sourceLabel = (source: AdminBillingReceipt['source']) => {
  if (source === 'web_chat' || !source) return t('admin.usage.billingReceipts.sourceWebChat')
  return source
}

const receiptRowAriaLabel = (receipt: AdminBillingReceipt) =>
  t('admin.usage.billingReceipts.openDetails', { id: receiptAuditIdentifier(receipt) })

const isCopied = (key: string) => copiedKey.value === key

const copyReceiptValue = async (value: string, key: string) => {
  if (!value) return
  const copied = await copyToClipboard(value, t('admin.usage.billingReceipts.copySuccess'))
  if (!copied) return
  copiedKey.value = key
  if (copyFeedbackTimer) clearTimeout(copyFeedbackTimer)
  copyFeedbackTimer = setTimeout(() => {
    copiedKey.value = null
  }, 2000)
}

const openReceipt = (receipt: AdminBillingReceipt) => {
  selectedReceipt.value = receipt
}

const closeReceipt = () => {
  selectedReceipt.value = null
}

const focusableDrawerElements = () => {
  if (!drawerRef.value) return []
  return Array.from(drawerRef.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
}

const handleDocumentKeydown = (event: KeyboardEvent) => {
  if (!selectedReceipt.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    closeReceipt()
    return
  }
  if (event.key !== 'Tab') return

  const focusable = focusableDrawerElements()
  if (focusable.length === 0) {
    event.preventDefault()
    drawerRef.value?.focus()
    return
  }

  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  const active = document.activeElement as HTMLElement | null
  if (!drawerRef.value?.contains(active)) {
    event.preventDefault()
    ;(event.shiftKey ? last : first).focus()
    return
  }
  if (event.shiftKey && active === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && active === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(selectedReceipt, async (receipt, previousReceipt) => {
  if (receipt) {
    if (!previousReceipt) previousActiveElement = document.activeElement as HTMLElement | null
    document.body.classList.add(BODY_LOCK_CLASS)
    await nextTick()
    drawerCloseRef.value?.focus({ preventScroll: true })
    return
  }

  document.body.classList.remove(BODY_LOCK_CLASS)
  const focusTarget = previousActiveElement
  previousActiveElement = null
  if (focusTarget?.isConnected) focusTarget.focus({ preventScroll: true })
})

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
      receipt_id: filters.request_id.trim() || undefined,
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
  closeReceipt()
  pagination.page = 1
  showUserDropdown.value = false
  void loadReceipts()
}

const resetFilters = () => {
  filters.user_id = undefined
  filters.model = ''
  filters.request_id = ''
  filters.status = ''
  userKeyword.value = ''
  userResults.value = []
  applyFilters()
}

const handlePageChange = (page: number) => {
  closeReceipt()
  pagination.page = page
  void loadReceipts()
}

const handlePageSizeChange = (pageSize: number) => {
  closeReceipt()
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
  userKeyword.value = simpleUserName(user) || user.email
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
    closeReceipt()
    pagination.page = 1
    void loadReceipts()
  }
)

onMounted(() => {
  applyRouteQuery()
  document.addEventListener('click', handleDocumentClick)
  document.addEventListener('keydown', handleDocumentKeydown, true)
  void loadReceipts()
})

onUnmounted(() => {
  abortController?.abort()
  if (userSearchTimer) clearTimeout(userSearchTimer)
  if (copyFeedbackTimer) clearTimeout(copyFeedbackTimer)
  document.removeEventListener('click', handleDocumentClick)
  document.removeEventListener('keydown', handleDocumentKeydown, true)
  document.body.classList.remove(BODY_LOCK_CLASS)
})

defineExpose({ reload: loadReceipts })
</script>

<style scoped>
:global(body.billing-receipt-drawer-open) {
  overflow: hidden;
}

.billing-receipts {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-ui);
}

.billing-receipts__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.5rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 1.25rem 1.5rem;
  background: var(--lx-clay-surface);
}

.billing-receipts__title-row,
.billing-receipts__drawer-title-row,
.billing-receipts__section-heading-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.65rem;
}

.billing-receipts__title,
.billing-receipts__drawer-title {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 1.05rem;
  font-weight: 850;
  letter-spacing: -0.015em;
}

.billing-receipts__subtitle,
.billing-receipts__drawer-subtitle {
  max-width: 68ch;
  margin: 0.3rem 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.8125rem;
  line-height: 1.55;
}

.billing-receipts__readonly-badge {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 0.35rem;
  border-radius: 999px;
  padding: 0.2rem 0.65rem;
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
  font-size: 0.6875rem;
  font-weight: 800;
  line-height: 1;
}

.billing-receipts__header-note {
  display: flex;
  max-width: 28rem;
  align-items: flex-start;
  gap: 0.5rem;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.billing-receipts__header-note svg {
  margin-top: 0.1rem;
  flex: 0 0 auto;
  color: var(--lx-clay-info);
}

.billing-receipts__filters {
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 1rem 1.5rem 1.25rem;
  background: var(--lx-clay-surface-soft);
}

.billing-receipts__input-clear {
  position: absolute;
  top: 1.55rem;
  right: 0;
  display: inline-flex;
  width: 44px;
  height: 44px;
  align-items: center;
  justify-content: center;
  border-radius: var(--lx-clay-radius-control);
  color: var(--lx-clay-text-muted);
}

.billing-receipts__input-clear:hover,
.billing-receipts__input-clear:focus-visible {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.billing-receipts__input-clear:focus-visible,
.billing-receipts__copy-button:focus-visible,
.billing-receipts__details-button:focus-visible,
.billing-receipts__drawer-close:focus-visible,
.billing-receipts__user-option:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 32%, transparent);
  outline-offset: 2px;
}

.billing-receipts__user-results {
  position: absolute;
  z-index: 50;
  width: 100%;
  max-height: 16rem;
  margin-top: 0.3rem;
  overflow: auto;
  border: 1px solid var(--lx-clay-border);
  border-radius: var(--lx-clay-radius-control);
  padding: 0.3rem;
  background: var(--lx-clay-surface);
  box-shadow: var(--lx-clay-shadow-overlay);
}

.billing-receipts__user-option {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: 10px;
  padding: 0.5rem 0.65rem;
  color: var(--lx-clay-text-secondary);
  text-align: start;
  font-size: 0.8125rem;
}

.billing-receipts__user-option:hover,
.billing-receipts__user-option:focus-visible {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.billing-receipts__user-cell,
.billing-receipts__model-cell,
.billing-receipts__status-cell {
  max-width: 15rem;
  min-width: 0;
}

.billing-receipts__user-primary,
.billing-receipts__model-actual {
  overflow: hidden;
  color: var(--lx-clay-text);
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.billing-receipts__user-secondary,
.billing-receipts__user-id,
.billing-receipts__model-requested,
.billing-receipts__time-cell,
.billing-receipts__muted-value {
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
}

.billing-receipts__user-secondary {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.billing-receipts__user-id {
  margin-top: 0.1rem;
}

.billing-receipts__identifier-stack {
  width: min(18rem, 30vw);
  min-width: 12rem;
}

.billing-receipts__identifier-line {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.35rem;
}

.billing-receipts__identifier-line--secondary {
  margin-top: 0.2rem;
}

.billing-receipts__identifier-label,
.billing-receipts__drawer-summary-label {
  display: block;
  color: var(--lx-clay-text-muted);
  font-size: 0.625rem;
  font-weight: 750;
  line-height: 1.2;
  text-transform: uppercase;
}

.billing-receipts__identifier,
.billing-receipts__drawer-summary-id {
  display: block;
  min-width: 0;
  overflow: hidden;
  color: var(--lx-clay-text-secondary);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  line-height: 1.45;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.billing-receipts__copy-button,
.billing-receipts__details-button,
.billing-receipts__drawer-close {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: var(--lx-clay-text-muted);
  transition: color 150ms ease, background-color 150ms ease, transform 150ms ease;
}

.billing-receipts__copy-button:hover,
.billing-receipts__details-button:hover,
.billing-receipts__drawer-close:hover {
  color: var(--lx-clay-accent-deep);
  background: var(--lx-clay-accent-soft);
}

.billing-receipts__copy-button:active,
.billing-receipts__details-button:active,
.billing-receipts__drawer-close:active {
  transform: scale(0.96);
}

.billing-receipts__model-requested,
.billing-receipts__model-actual {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.billing-receipts__tokens-cell,
.billing-receipts__cost-cell {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  font-variant-numeric: tabular-nums;
}

.billing-receipts__tokens-cell strong,
.billing-receipts__charged-amount {
  color: var(--lx-clay-success-text);
  font-weight: 750;
}

.billing-receipts__tokens-cell span,
.billing-receipts__cost-cell > span {
  color: var(--lx-clay-text-muted);
  font-size: 0.6875rem;
}

.billing-receipts__balance-cell {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--lx-clay-text-secondary);
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
}

.billing-receipts__status-badge {
  display: inline-flex;
  min-height: 26px;
  align-items: center;
  gap: 0.4rem;
  border-radius: 999px;
  padding: 0.25rem 0.65rem;
  font-size: 0.6875rem;
  font-weight: 800;
  line-height: 1;
  white-space: nowrap;
}

.billing-receipts__status-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  background: currentColor;
}

.billing-receipts__status-badge--charged {
  color: var(--lx-clay-success-text);
  background: var(--lx-clay-success-soft);
}

.billing-receipts__status-badge--subscription {
  color: var(--lx-clay-info-deep);
  background: var(--lx-clay-info-soft);
}

.billing-receipts__status-badge--pending {
  color: var(--lx-clay-warning);
  background: var(--lx-clay-warning-soft);
}

.billing-receipts__status-badge--failed {
  color: var(--lx-clay-danger);
  background: var(--lx-clay-danger-soft);
}

.billing-receipts__status-badge--notCharged,
.billing-receipts__status-badge--unknown {
  color: var(--lx-clay-text-secondary);
  background: var(--lx-clay-recessed);
}

.billing-receipts__failure-preview {
  display: -webkit-box;
  margin: 0.25rem 0 0;
  overflow: hidden;
  color: var(--lx-clay-danger);
  font-size: 0.6875rem;
  line-height: 1.35;
  white-space: normal;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.billing-receipts__failure-preview span + span::before {
  content: " · ";
}

.billing-receipts__drawer-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  justify-content: flex-end;
  background: rgb(23 19 31 / 0.38);
}

.billing-receipts__drawer {
  display: flex;
  width: min(36rem, 100vw);
  height: 100%;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border-inline-start: 1px solid var(--lx-clay-border);
  outline: none;
  color: var(--lx-clay-text);
  background: var(--lx-clay-canvas);
  box-shadow: var(--lx-clay-shadow-overlay);
  font-family: var(--lx-clay-font-ui);
}

.billing-receipts__drawer:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 36%, transparent);
  outline-offset: -3px;
}

.billing-receipts__drawer-header {
  display: flex;
  flex: 0 0 auto;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 1.25rem 1.25rem 1rem;
  background: var(--lx-clay-surface);
}

.billing-receipts__drawer-scroll {
  min-height: 0;
  flex: 1 1 auto;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 1.25rem max(1.5rem, env(safe-area-inset-bottom));
}

.billing-receipts__drawer-summary {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 0.9rem;
  border-bottom: 1px solid var(--lx-clay-border);
  padding: 1rem 0;
}

.billing-receipts__drawer-summary-id {
  margin-top: 0.2rem;
  color: var(--lx-clay-text);
}

.billing-receipts__drawer-section {
  padding: 1.2rem 0;
}

.billing-receipts__drawer-section + .billing-receipts__drawer-section {
  border-top: 1px solid var(--lx-clay-border);
}

.billing-receipts__drawer-section--failure {
  color: var(--lx-clay-danger);
}

.billing-receipts__section-title {
  margin: 0;
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-display);
  font-size: 0.875rem;
  font-weight: 850;
  letter-spacing: -0.01em;
}

.billing-receipts__section-heading-row {
  justify-content: space-between;
}

.billing-receipts__section-total {
  color: var(--lx-clay-accent-deep);
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
}

.billing-receipts__detail-list,
.billing-receipts__metric-grid,
.billing-receipts__settlement-list {
  margin: 0.8rem 0 0;
}

.billing-receipts__detail-row {
  display: grid;
  grid-template-columns: minmax(7rem, 0.38fr) minmax(0, 1fr);
  gap: 1rem;
  padding: 0.65rem 0;
}

.billing-receipts__detail-row + .billing-receipts__detail-row {
  border-top: 1px solid color-mix(in srgb, var(--lx-clay-border) 72%, transparent);
}

.billing-receipts__detail-row dt,
.billing-receipts__metric-grid dt,
.billing-receipts__settlement-list dt {
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  font-weight: 650;
}

.billing-receipts__detail-row dd,
.billing-receipts__metric-grid dd,
.billing-receipts__settlement-list dd {
  min-width: 0;
  margin: 0;
  color: var(--lx-clay-text-secondary);
  font-size: 0.8125rem;
  text-align: end;
  overflow-wrap: anywhere;
}

.billing-receipts__detail-row dd > * {
  display: block;
}

.billing-receipts__detail-row code {
  color: var(--lx-clay-text);
  font-family: var(--lx-clay-font-mono);
  font-size: 0.75rem;
  overflow-wrap: anywhere;
  white-space: normal;
}

.billing-receipts__detail-with-action {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 0.4rem;
}

.billing-receipts__detail-with-action code {
  min-width: 0;
}

.billing-receipts__metric-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  border-top: 1px solid var(--lx-clay-border);
}

.billing-receipts__metric-grid > div {
  padding: 0.75rem 0;
}

.billing-receipts__metric-grid > div:nth-child(even) {
  padding-inline-start: 1rem;
}

.billing-receipts__metric-grid > div:nth-child(n + 3) {
  border-top: 1px solid color-mix(in srgb, var(--lx-clay-border) 72%, transparent);
}

.billing-receipts__metric-grid dd {
  margin-top: 0.2rem;
  color: var(--lx-clay-text);
  font-size: 1rem;
  font-weight: 750;
  text-align: start;
  font-variant-numeric: tabular-nums;
}

.billing-receipts__metric-grid-total {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.billing-receipts__metric-grid-total dd {
  color: var(--lx-clay-accent-deep);
}

.billing-receipts__settlement-list > div {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid color-mix(in srgb, var(--lx-clay-border) 72%, transparent);
}

.billing-receipts__settlement-list-emphasis dd {
  color: var(--lx-clay-success-text);
  font-weight: 800;
}

.billing-receipts__danger-text,
.billing-receipts__failure-reason {
  color: var(--lx-clay-danger) !important;
}

.billing-receipts__drawer-readonly-note {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  margin: 0.25rem 0 0;
  border-top: 1px solid var(--lx-clay-border);
  padding: 1rem 0 0;
  color: var(--lx-clay-text-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.billing-receipt-drawer-enter-active,
.billing-receipt-drawer-leave-active {
  transition: opacity 180ms ease;
}

.billing-receipt-drawer-enter-active .billing-receipts__drawer,
.billing-receipt-drawer-leave-active .billing-receipts__drawer {
  transition: transform 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

.billing-receipt-drawer-enter-from,
.billing-receipt-drawer-leave-to {
  opacity: 0;
}

.billing-receipt-drawer-enter-from .billing-receipts__drawer,
.billing-receipt-drawer-leave-to .billing-receipts__drawer {
  transform: translateX(100%);
}

@media (max-width: 767px) {
  .billing-receipts__header {
    display: block;
    padding: 1rem;
  }

  .billing-receipts__header-note {
    margin-top: 0.75rem;
  }

  .billing-receipts__filters {
    padding: 1rem;
  }

  .billing-receipts__identifier-stack {
    width: 100%;
    min-width: 0;
  }

  .billing-receipts__drawer {
    width: 100%;
    border-inline-start: 0;
  }

  .billing-receipts__drawer-header,
  .billing-receipts__drawer-scroll {
    padding-inline: 1rem;
  }

  .billing-receipts__detail-row {
    grid-template-columns: 1fr;
    gap: 0.3rem;
  }

  .billing-receipts__detail-row dd {
    text-align: start;
  }

  .billing-receipts__detail-with-action {
    justify-content: space-between;
  }
}

@media (prefers-reduced-motion: reduce) {
  .billing-receipts__copy-button,
  .billing-receipts__details-button,
  .billing-receipts__drawer-close,
  .billing-receipt-drawer-enter-active,
  .billing-receipt-drawer-leave-active,
  .billing-receipt-drawer-enter-active .billing-receipts__drawer,
  .billing-receipt-drawer-leave-active .billing-receipts__drawer {
    transition-duration: 0.01ms;
  }
}
</style>
