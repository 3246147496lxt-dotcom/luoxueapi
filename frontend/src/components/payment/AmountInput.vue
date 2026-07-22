<template>
  <div class="space-y-5">
    <div class="space-y-3">
      <p class="text-sm font-semibold text-gray-800 dark:text-gray-200">
        {{ t('payment.amountType') }}
      </p>
      <div
        class="flex flex-wrap items-center gap-2"
        role="tablist"
        :aria-label="t('payment.amountType')"
        @keydown="handleModeKeydown"
      >
        <button
          id="amount-mode-preset"
          type="button"
          role="tab"
          :aria-selected="mode === 'preset'"
          :tabindex="mode === 'preset' ? 0 : -1"
          class="min-h-9 rounded-lg px-4 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
          :class="mode === 'preset'
            ? 'bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300'
            : 'bg-gray-100 text-gray-500 hover:text-gray-800 dark:bg-dark-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="setMode('preset')"
        >
          {{ t('payment.fixedAmount') }}
        </button>
        <button
          id="amount-mode-custom"
          type="button"
          role="tab"
          :aria-selected="mode === 'custom'"
          :tabindex="mode === 'custom' ? 0 : -1"
          class="min-h-9 rounded-lg px-4 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
          :class="mode === 'custom'
            ? 'bg-primary-50 text-primary-700 dark:bg-primary-950/60 dark:text-primary-300'
            : 'bg-gray-100 text-gray-500 hover:text-gray-800 dark:bg-dark-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="setMode('custom')"
        >
          {{ t('payment.customAmount') }}
        </button>
      </div>
    </div>

    <div
      v-if="mode === 'preset'"
      id="amount-panel-preset"
      role="tabpanel"
      aria-labelledby="amount-mode-preset"
    >
      <p class="mb-3 text-sm font-semibold text-gray-800 dark:text-gray-200">
        {{ t('payment.chooseAmountTitle') }}
      </p>
      <div
        class="grid min-w-0 grid-cols-2 gap-3 sm:grid-cols-4"
        role="radiogroup"
        :aria-label="t('payment.quickAmounts')"
        data-testid="preset-amount-grid"
      >
        <button
          v-for="amt in filteredAmounts"
          :key="amt"
          type="button"
          role="radio"
          :aria-checked="modelValue === amt"
          :class="[
            'flex min-h-[76px] min-w-0 flex-col items-center justify-center rounded-2xl border px-2 py-3 text-center transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 focus-visible:ring-offset-2 sm:min-h-[92px] sm:px-3 dark:focus-visible:ring-offset-dark-900',
            modelValue === amt
              ? 'border-primary-500 bg-primary-50 text-primary-800 dark:border-primary-400 dark:bg-primary-950/50 dark:text-primary-200'
              : 'border-gray-200 bg-white text-gray-700 hover:border-primary-200 hover:bg-primary-50/40 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-800 dark:hover:bg-primary-950/20',
          ]"
          @click="selectAmount(amt)"
        >
          <span
            class="max-w-full text-base font-bold tabular-nums [overflow-wrap:anywhere] sm:text-xl"
            data-testid="preset-amount-value"
          >
            {{ formatAmount(amt) }}
          </span>
          <span class="mt-1 text-xs font-medium text-gray-400 dark:text-gray-500">
            {{ t('payment.selectThisAmount') }}
          </span>
        </button>
      </div>
    </div>

    <div
      v-else
      id="amount-panel-custom"
      role="tabpanel"
      aria-labelledby="amount-mode-custom"
    >
      <label for="custom-recharge-amount" class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('payment.customAmountLabel') }}
      </label>
      <div class="relative">
        <span class="pointer-events-none absolute inset-y-0 left-4 flex items-center text-sm font-semibold text-gray-400 dark:text-gray-500">
          {{ currencySymbol }}
        </span>
        <input
          id="custom-recharge-amount"
          ref="customInputRef"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          :value="customText"
          :placeholder="placeholderText"
          class="input min-h-12 w-full pl-10 pr-4 text-base tabular-nums"
          @input="handleInput"
        />
      </div>
      <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ rangeHint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  currency?: string
  locale?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  currency: 'CNY',
  locale: 'zh-CN',
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')
const mode = ref<'preset' | 'custom'>('preset')
const customInputRef = ref<HTMLInputElement | null>(null)

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const rangeHint = computed(() => {
  if (props.min > 0 && props.max > 0) {
    return t('payment.amountRange', {
      min: formatAmount(props.min),
      max: formatAmount(props.max),
    })
  }
  if (props.min > 0) return t('payment.minimumAmount', { amount: formatAmount(props.min) })
  if (props.max > 0) return t('payment.maximumAmount', { amount: formatAmount(props.max) })
  return t('payment.customAmountHint')
})

const currencySymbol = computed(() => {
  try {
    const parts = new Intl.NumberFormat(props.locale, {
      style: 'currency',
      currency: props.currency,
      currencyDisplay: 'narrowSymbol',
    }).formatToParts(0)
    return parts.find(part => part.type === 'currency')?.value || props.currency
  } catch {
    return props.currency
  }
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function formatAmount(value: number): string {
  try {
    return new Intl.NumberFormat(props.locale, {
      style: 'currency',
      currency: props.currency,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    }).format(value)
  } catch {
    return `${props.currency} ${value}`
  }
}

function setMode(nextMode: 'preset' | 'custom') {
  mode.value = nextMode
  if (nextMode === 'custom') {
    nextTick(() => customInputRef.value?.focus())
  } else {
    nextTick(() => document.getElementById('amount-mode-preset')?.focus())
  }
}

function handleModeKeydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const nextMode = event.key === 'ArrowLeft' || event.key === 'Home' ? 'preset' : 'custom'
  setMode(nextMode)
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  if (!AMOUNT_PATTERN.test(val)) return
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
  if (v !== null && !filteredAmounts.value.includes(v)) {
    mode.value = 'custom'
  }
}, { immediate: true })

watch(filteredAmounts, (values) => {
  if (values.length === 0) mode.value = 'custom'
}, { immediate: true })
</script>
