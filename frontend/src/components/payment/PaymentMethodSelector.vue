<template>
  <div>
    <h2 class="payment-method-label mb-6 text-base font-bold">
      {{ t('payment.choosePaymentMethod') }}
    </h2>
    <div
      ref="methodGroupRef"
      class="grid grid-cols-1 gap-4 sm:grid-cols-2"
      role="radiogroup"
      :aria-label="t('payment.choosePaymentMethod')"
    >
      <button
        v-for="(method, index) in sortedMethods"
        :key="method.type"
        type="button"
        role="radio"
        :aria-checked="selected === method.type"
        :aria-disabled="!method.available"
        :aria-describedby="method.available ? undefined : unavailableDescriptionId(index)"
        :disabled="!method.available"
        :tabindex="methodTabIndex(method)"
        :data-method-type="method.type"
        :class="[
          'payment-method-option flex w-full items-center gap-3 rounded-xl border p-4 text-left',
          {
            'is-selected': method.available && selected === method.type,
            'is-unavailable': !method.available,
          },
        ]"
        @click="method.available && emit('select', method.type)"
        @keydown="handleMethodKeydown"
      >
        <img :src="methodIcon(method.type)" alt="" aria-hidden="true" class="h-6 w-6 shrink-0 object-contain" />
        <span class="flex min-w-0 flex-1 flex-col items-start">
          <span class="payment-method-name max-w-full break-words text-sm font-bold leading-5">{{ methodLabel(method) }}</span>
          <span
            v-if="!method.available"
            :id="unavailableDescriptionId(index)"
            class="payment-method-meta text-[10px]"
            data-testid="payment-method-unavailable"
          >
            {{ t('payment.amountUnavailable') }}
          </span>
        </span>
        <Icon
          v-if="method.available && selected === method.type"
          name="checkCircle"
          size="md"
          class="payment-method-check ml-auto shrink-0"
          data-testid="payment-method-selected"
          aria-hidden="true"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { METHOD_ORDER, isBuiltInAlipayMethod, isBuiltInWxpayMethod } from './providerConfig'
import Icon from '@/components/icons/Icon.vue'
import alipayIcon from '@/assets/icons/alipay.svg'
import wxpayIcon from '@/assets/icons/wxpay.svg'
import stripeIcon from '@/assets/icons/stripe.svg'
import airwallexIcon from '@/assets/icons/airwallex.svg'
import paymentIcon from '@/assets/icons/payment.svg'

export interface PaymentMethodOption {
  type: string
  display_name?: string
  fee_rate: number
  available: boolean
}

const props = defineProps<{
  methods: PaymentMethodOption[]
  selected: string
}>()

const emit = defineEmits<{
  select: [type: string]
}>()

const { t } = useI18n()
const methodGroupRef = ref<HTMLElement | null>(null)

const METHOD_ICONS: Record<string, string> = {
  alipay: alipayIcon,
  wxpay: wxpayIcon,
  stripe: stripeIcon,
  airwallex: airwallexIcon,
  credit_card: paymentIcon,
}

const sortedMethods = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return [...props.methods].sort((a, b) => {
    const ai = order.indexOf(a.type)
    const bi = order.indexOf(b.type)
    return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
  })
})

function methodIcon(type: string): string {
  if (isBuiltInAlipayMethod(type)) return METHOD_ICONS.alipay
  if (isBuiltInWxpayMethod(type)) return METHOD_ICONS.wxpay
  if (type === 'airwallex') return METHOD_ICONS.airwallex
  return METHOD_ICONS[type] || paymentIcon
}

function methodLabel(method: PaymentMethodOption): string {
  return method.display_name || t(`payment.methods.${method.type}`, method.type)
}

function unavailableDescriptionId(index: number): string {
  return `payment-method-unavailable-${index}`
}

const focusableMethodType = computed(() => {
  const selectedMethod = sortedMethods.value.find(
    method => method.available && method.type === props.selected,
  )
  return selectedMethod?.type ?? sortedMethods.value.find(method => method.available)?.type ?? ''
})

function methodTabIndex(method: PaymentMethodOption): 0 | -1 {
  return method.available && method.type === focusableMethodType.value ? 0 : -1
}

function handleMethodKeydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'Home', 'End'].includes(event.key)) return

  const buttons = Array.from(
    methodGroupRef.value?.querySelectorAll<HTMLButtonElement>('[role="radio"]:not(:disabled)') ?? [],
  )
  if (buttons.length === 0) return

  const currentIndex = buttons.indexOf(event.currentTarget as HTMLButtonElement)
  if (currentIndex < 0) return

  event.preventDefault()
  let nextIndex = currentIndex
  if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = buttons.length - 1
  else if (event.key === 'ArrowRight' || event.key === 'ArrowDown') {
    nextIndex = (currentIndex + 1) % buttons.length
  } else {
    nextIndex = (currentIndex - 1 + buttons.length) % buttons.length
  }

  const target = buttons[nextIndex]
  const methodType = target?.dataset.methodType
  if (!target || !methodType) return

  if (methodType !== props.selected) emit('select', methodType)
  nextTick(() => target.focus())
}
</script>

<style scoped>
.payment-method-label {
  color: var(--lx-clay-text);
}

.payment-method-option {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  border-color: var(--lx-clay-border);
  transition:
    border-color 150ms ease,
    background-color 150ms ease,
    box-shadow 150ms ease,
    color 150ms ease;
}

.payment-method-option:hover:not(:disabled) {
  background: var(--lx-clay-hover);
  border-color: var(--lx-clay-border-strong);
}

.payment-method-option.is-selected {
  background: color-mix(in srgb, var(--lx-clay-accent) 8%, var(--lx-clay-surface));
  border-color: var(--lx-clay-accent);
  border-width: 2px;
}

.payment-method-option:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.payment-method-option.is-unavailable {
  cursor: not-allowed;
  color: var(--lx-clay-text-muted);
  background: var(--lx-clay-surface);
  border-color: var(--lx-clay-border);
  box-shadow: none;
  opacity: 0.5;
}

.payment-method-meta {
  color: var(--lx-clay-text-muted);
}

.payment-method-name {
  color: var(--lx-clay-text);
}

.payment-method-check {
  color: var(--lx-clay-accent);
}

:global(.dark) .payment-method-label,
:global(.dark) .payment-method-name {
  color: var(--lx-clay-text);
}

:global(.dark) .payment-method-meta {
  color: var(--lx-clay-text-muted);
}

@media (prefers-reduced-motion: reduce) {
  .payment-method-option {
    transition-duration: 0.01ms;
  }
}
</style>
