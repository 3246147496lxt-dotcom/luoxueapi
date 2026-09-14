<template>
  <CreditAmount
    v-if="isUsdWallet && hasValue"
    :value="formattedValue"
    :icon-size="iconSize"
  />
  <span v-else>{{ displayValue }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CreditAmount from '@/components/common/CreditAmount.vue'

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
const isUsdWallet = computed(() => normalizedCurrency.value === 'USD' || normalizedCurrency.value === 'CREDIT')
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
  return trimInsignificantZeros(scaledValue)
})

const displayValue = computed(() => {
  if (!hasValue.value) return props.emptyText
  if (isUsdWallet.value) return `$${formattedValue.value}`
  if (!normalizedCurrency.value) return formattedValue.value
  return `${normalizedCurrency.value} ${formattedValue.value}`
})
</script>
