<template>
  <CreditAmount
    v-if="isCredit && hasValue"
    :value="formattedValue"
    :icon-size="iconSize"
  />
  <span v-else>{{ displayValue }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import CreditAmount from '@/components/common/CreditAmount.vue'

type CreditIconSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

const props = withDefaults(defineProps<{
  value: number | null | undefined
  currency?: string
  scale?: number
  emptyText?: string
  iconSize?: CreditIconSize
}>(), {
  currency: '',
  scale: 1,
  emptyText: '-',
  iconSize: 'sm',
})

const normalizedCurrency = computed(() => props.currency.trim().toUpperCase())
const hasValue = computed(() => typeof props.value === 'number' && Number.isFinite(props.value))
const isCredit = computed(() => normalizedCurrency.value === 'CREDIT')

const formattedValue = computed(() => {
  if (!hasValue.value) return props.emptyText
  return ((props.value as number) * props.scale)
    .toPrecision(10)
    .replace(/\.?0+$/, '')
})

const displayValue = computed(() => {
  if (!hasValue.value) return props.emptyText
  if (normalizedCurrency.value === 'USD') return `$${formattedValue.value}`
  if (!normalizedCurrency.value) return formattedValue.value
  return `${normalizedCurrency.value} ${formattedValue.value}`
})
</script>
