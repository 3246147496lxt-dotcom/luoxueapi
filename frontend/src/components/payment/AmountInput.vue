<template>
  <div class="space-y-6">
    <h2 id="payment-amount-title" class="amount-section-title text-base font-bold">
      {{ t('payment.chooseAmountTitle') }}
    </h2>

    <div
      ref="presetGroupRef"
      class="grid min-w-0 grid-cols-2 gap-3 sm:grid-cols-4"
      role="radiogroup"
      :aria-label="t('payment.quickAmounts')"
      data-testid="preset-amount-grid"
    >
      <button
        v-for="(amt, index) in filteredAmounts"
        :key="amt"
        type="button"
        role="radio"
        :aria-checked="isPresetSelected(amt)"
        :aria-label="`${t('payment.selectThisAmount')} ${formatAmount(amt)}`"
        :tabindex="presetOptionTabIndex(index)"
        :class="[
          'amount-preset-option flex min-w-0 flex-col items-center justify-center rounded-2xl border p-4 text-center',
          { 'is-selected': isPresetSelected(amt) },
        ]"
        @click="selectAmount(amt)"
        @keydown="handlePresetKeydown($event, index)"
      >
        <span
          class="amount-preset-value max-w-full text-xl font-black tabular-nums [overflow-wrap:anywhere]"
          data-testid="preset-amount-value"
        >
          {{ formatAmount(amt) }}
        </span>
      </button>
    </div>

    <div class="space-y-2">
      <label
        for="custom-recharge-amount"
        class="amount-custom-label ml-1 block text-xs font-bold uppercase"
      >
        {{ t('payment.customAmount') }}
      </label>
      <div class="relative">
        <span
          class="amount-currency-symbol pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 font-bold"
          aria-hidden="true"
        >
          {{ currencySymbol }}
        </span>
        <input
          id="custom-recharge-amount"
          type="text"
          inputmode="decimal"
          autocomplete="off"
          :value="customText"
          placeholder="0.00"
          aria-describedby="custom-recharge-amount-range"
          class="amount-custom-input min-h-11 w-full pl-8 pr-4 text-base font-bold tabular-nums"
          @input="handleInput"
        />
      </div>
      <p id="custom-recharge-amount-range" class="sr-only">{{ rangeHint }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
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
const selectionSource = ref<'preset' | 'custom'>('preset')
const presetGroupRef = ref<HTMLElement | null>(null)

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((amount) =>
    (props.min <= 0 || amount >= props.min)
      && (props.max <= 0 || amount <= props.max),
  )
)

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

function isPresetSelected(amount: number): boolean {
  return selectionSource.value === 'preset' && props.modelValue === amount
}

function selectAmount(amount: number) {
  selectionSource.value = 'preset'
  customText.value = ''
  emit('update:modelValue', amount)
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

function presetOptionTabIndex(index: number): 0 | -1 {
  const selectedIndex = selectionSource.value === 'preset'
    ? filteredAmounts.value.findIndex(amount => amount === props.modelValue)
    : -1
  const focusableIndex = selectedIndex >= 0 ? selectedIndex : 0
  return index === focusableIndex ? 0 : -1
}

function handlePresetKeydown(event: KeyboardEvent, currentIndex: number) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return

  const buttons = Array.from(
    presetGroupRef.value?.querySelectorAll<HTMLButtonElement>('[role="radio"]') ?? [],
  )
  if (buttons.length === 0) return

  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = buttons.length - 1
  else if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % buttons.length
  } else {
    nextIndex = (currentIndex - 1 + buttons.length) % buttons.length
  }

  const nextAmount = filteredAmounts.value[nextIndex]
  if (!isPresetSelected(nextAmount)) selectAmount(nextAmount)
  nextTick(() => buttons[nextIndex]?.focus())
}

function customTextMatchesModel(value: number | null): boolean {
  if (customText.value === '') return value === null
  const parsed = Number.parseFloat(customText.value)
  if (!Number.isFinite(parsed) || parsed <= 0) return value === null
  return value === parsed
}

function handleInput(event: Event) {
  const input = event.target as HTMLInputElement
  const value = input.value
  if (!AMOUNT_PATTERN.test(value)) {
    input.value = customText.value
    return
  }

  selectionSource.value = 'custom'
  customText.value = value
  if (value === '') {
    emit('update:modelValue', null)
    return
  }

  const amount = Number.parseFloat(value)
  emit('update:modelValue', Number.isFinite(amount) && amount > 0 ? amount : null)
}

watch(
  [() => props.modelValue, filteredAmounts],
  ([value]) => {
    if (selectionSource.value === 'custom' && customTextMatchesModel(value)) return
    if (
      selectionSource.value === 'preset'
      && customText.value === ''
      && (value === null || filteredAmounts.value.includes(value))
    ) return

    if (value !== null && filteredAmounts.value.includes(value)) {
      selectionSource.value = 'preset'
      customText.value = ''
      return
    }

    selectionSource.value = 'custom'
    customText.value = value === null ? '' : String(value)
  },
  { immediate: true },
)
</script>

<style scoped>
.amount-section-title {
  color: #111827;
}

.amount-preset-option {
  color: #030712;
  background: var(--lx-clay-surface);
  border-color: var(--lx-clay-border);
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    color 150ms ease;
}

.amount-preset-option:hover {
  color: #6d28d9;
  border-color: #c4b5fd;
}

.amount-preset-option.is-selected {
  color: #6d28d9;
  background: color-mix(in srgb, #f5f3ff 30%, var(--lx-clay-surface));
  border-color: #8b5cf6;
}

.amount-preset-option:focus-visible,
.amount-custom-input:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.amount-custom-label {
  color: #6b7280;
}

.amount-currency-symbol {
  color: #9ca3af;
}

.amount-custom-input {
  border: 1px solid var(--lx-clay-border);
  border-radius: 13px;
  color: var(--lx-clay-text);
  background: var(--lx-clay-recessed);
  transition:
    border-color 150ms ease,
    box-shadow 150ms ease;
}

.amount-custom-input::placeholder {
  color: #9ca3af;
}

:global(.dark) .amount-section-title,
:global(.dark) .amount-preset-option {
  color: var(--lx-clay-text);
}

:global(.dark) .amount-custom-label,
:global(.dark) .amount-currency-symbol,
:global(.dark) .amount-custom-input::placeholder {
  color: var(--lx-clay-text-muted);
}

.amount-custom-input:focus {
  border-color: var(--lx-clay-accent);
  outline: none;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--lx-clay-accent) 10%, transparent);
}

@media (prefers-reduced-motion: reduce) {
  .amount-preset-option,
  .amount-custom-input {
    transition-duration: 0.01ms;
  }
}
</style>
