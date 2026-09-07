<template>
  <CreditAmount
    v-if="isCredit && hasValue && !displayCreditAsCny"
    :value="formattedValue"
    :icon-size="iconSize"
  />
  <span
    v-else-if="displayCreditAsCny && hasValue"
    class="inline-flex min-w-0 items-center whitespace-nowrap tabular-nums"
    role="group"
    :aria-label="`CNY ${formattedValue}`"
    data-testid="catalog-price-cny"
  >
    <span aria-hidden="true">¥{{ formattedValue }}</span>
  </span>
  <span v-else>{{ displayValue }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import { POINTS_PER_CNY } from '@/constants/channel'

type PointsIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'
type CreditDisplay = 'points' | 'cny'

const props = withDefaults(defineProps<{
  value: number | null | undefined
  currency?: string
  scale?: number
  emptyText?: string
  iconSize?: PointsIconSize
  creditDisplay?: CreditDisplay
}>(), {
  currency: '',
  scale: 1,
  emptyText: '-',
  iconSize: 'sm',
  creditDisplay: 'points',
})

const normalizedCurrency = computed(() => props.currency.trim().toUpperCase())
const hasValue = computed(() => typeof props.value === 'number' && Number.isFinite(props.value))
const isCredit = computed(() => normalizedCurrency.value === 'CREDIT')
const displayCreditAsCny = computed(() => isCredit.value && props.creditDisplay === 'cny')

function trimInsignificantZeros(value: number): string {
  const [coefficient, exponent] = value.toPrecision(10).split('e')
  const trimmedCoefficient = coefficient.includes('.')
    ? coefficient.replace(/0+$/, '').replace(/\.$/, '')
    : coefficient
  return exponent == null ? trimmedCoefficient : `${trimmedCoefficient}e${exponent}`
}

const formattedValue = computed(() => {
  if (!hasValue.value) return props.emptyText
  const scaledValue = (props.value as number) * props.scale
  const displayValue = displayCreditAsCny.value
    ? scaledValue / POINTS_PER_CNY
    : scaledValue
  return trimInsignificantZeros(displayValue)
})

const displayValue = computed(() => {
  if (!hasValue.value) return props.emptyText
  if (normalizedCurrency.value === 'USD') return `$${formattedValue.value}`
  if (!normalizedCurrency.value) return formattedValue.value
  return `${normalizedCurrency.value} ${formattedValue.value}`
})
</script>
