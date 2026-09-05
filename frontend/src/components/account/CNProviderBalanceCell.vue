<template>
  <div v-if="visible" class="space-y-1">
    <!-- Keep the current balance as static data; the separate action below
         makes the manual refresh discoverable without changing the label. -->
    <div class="flex flex-wrap items-center gap-1.5">
      <span
        data-test="cn-provider-balance-value"
        :class="['text-[10px] font-medium leading-4', platformTextClass(account.platform)]"
        :title="t('admin.accounts.cnProviders.balanceProbeTooltip')"
      >
        {{ balanceLabel }}
      </span>
      <span v-if="balanceLow" class="inline-flex items-center rounded bg-red-100 px-1 py-0.5 text-[10px] font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300">
        {{ t('admin.accounts.cnProviders.balanceLow') }}
      </span>
    </div>

    <div class="flex flex-wrap items-center gap-1.5">
      <button
        type="button"
        data-test="cn-provider-balance-probe"
        class="inline-flex items-center gap-0.5 whitespace-nowrap rounded px-1.5 py-0.5 text-[10px] font-medium leading-4 text-blue-600 transition-colors hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-blue-400 dark:hover:bg-blue-900/30"
        :disabled="loading"
        :title="t('admin.accounts.cnProviders.balanceProbeTooltip')"
        @click="handleProbe"
      >
        <svg class="h-2.5 w-2.5" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {{ t('admin.accounts.cnProviders.probe') }}
      </button>
    </div>

    <div v-if="error" class="truncate text-[10px] text-red-600 dark:text-red-400" :title="error">{{ truncatedError }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { CNProviderBalanceEntry, CNProviderBalanceResult } from '@/api/admin/cnProviders'
import type { Account } from '@/types'
import { platformTextClass } from '@/utils/platformColors'
import { cnBalanceCellVisible } from './credentialsBuilder'

const props = defineProps<{ account: Account }>()
const { t } = useI18n()
const readMode = () => {
  const mode = props.account.credentials?.account_mode
  return typeof mode === 'string' ? mode : ''
}
const accountMode = computed(readMode)
const visible = computed(() => cnBalanceCellVisible(props.account.platform, accountMode.value))
const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<CNProviderBalanceResult | null>(null)
const extraKey = (suffix: string) => `${props.account.platform}_${suffix}`

const readSnapshotNumber = (value: unknown): number | null => {
  if (typeof value === 'number') return Number.isFinite(value) ? value : null
  if (typeof value !== 'string' || value.trim() === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

const snapshotBalance = computed(() => {
  const value = props.account.extra?.[extraKey('balance')]
  return readSnapshotNumber(value)
})
const snapshotCurrency = computed(() => {
  const value = props.account.extra?.[extraKey('balance_currency')]
  return typeof value === 'string' ? value : ''
})
const snapshotBalances = computed<CNProviderBalanceEntry[]>(() => {
  const value = props.account.extra?.[extraKey('balances')]
  if (!Array.isArray(value)) return []
  return value.flatMap((item): CNProviderBalanceEntry[] => {
    if (!item || typeof item !== 'object') return []
    const row = item as Record<string, unknown>
    const balance = readSnapshotNumber(row.balance)
    return typeof row.currency === 'string' && balance != null
      ? [{ currency: row.currency, balance }]
      : []
  })
})
const balanceLow = computed(() => props.account.extra?.[extraKey('balance_low')] === true)
const currentEntries = computed<CNProviderBalanceEntry[]>(() => {
  if (data.value?.success) {
    if (data.value.balances?.length) return data.value.balances
    return [{ currency: data.value.currency || '', balance: data.value.balance }]
  }
  if (snapshotBalances.value.length) return snapshotBalances.value
  return snapshotBalance.value == null ? [] : [{ currency: snapshotCurrency.value, balance: snapshotBalance.value }]
})
const formatEntry = (entry: CNProviderBalanceEntry) => `${entry.currency || '¥'} ${entry.balance >= 100 ? entry.balance.toFixed(0) : entry.balance.toFixed(2)}`
const balanceLabel = computed(() => currentEntries.value.length ? currentEntries.value.map(formatEntry).join(' · ') : (t('admin.accounts.cnProviders.balance') || 'Balance'))
const extractErrorMessage = (e: unknown) => {
  const err = e as { message?: string; reason?: string; response?: { data?: { message?: string; error?: string } } }
  return err?.message || err?.reason || err?.response?.data?.message || err?.response?.data?.error || t('common.error')
}
const truncatedError = computed(() => error.value && error.value.length > 80 ? `${error.value.slice(0, 80)}...` : error.value || '')
const handleProbe = async () => {
  if (loading.value) return
  loading.value = true
  error.value = null
  try {
    const result = await adminAPI.cnProviders.queryBalance(props.account.id)
    if (result.success) data.value = result
    else error.value = result.error || t('common.error')
  } catch (e) {
    error.value = extractErrorMessage(e)
  } finally {
    loading.value = false
  }
}
watch(
  [() => props.account.id, () => props.account.platform, accountMode],
  () => {
    data.value = null
    error.value = null
    loading.value = false
  }
)
</script>
