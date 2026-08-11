<template>
  <article
    class="pricing-plan-card"
    :class="{ 'pricing-plan-card--featured': featured }"
    :data-plan-id="String(plan.id)"
    :data-plan-group="String(plan.group_id)"
    :data-featured="featured ? 'true' : 'false'"
  >
    <div v-if="featured" class="pricing-plan-card__aura" aria-hidden="true"></div>

    <div class="pricing-plan-card__inner">
      <div class="pricing-plan-card__heading">
        <h3>{{ plan.name }}</h3>
        <span v-if="featured" class="pricing-plan-card__badge">
          {{ t('pricing.recommended') }}
        </span>
      </div>

      <p class="pricing-plan-card__description">
        {{ plan.description }}
      </p>

      <div class="pricing-plan-card__price">
        <span>{{ formattedPrice }}</span>
        <small>/ {{ validityLabel }}</small>
      </div>

      <button
        type="button"
        class="pricing-plan-card__action"
        :aria-label="t('pricing.choosePlanAccessible', { plan: plan.name })"
        @click="emit('select', plan)"
      >
        {{ t('pricing.choosePlan') }}
      </button>

      <div class="pricing-plan-card__divider" aria-hidden="true"></div>

      <p class="pricing-plan-card__includes">
        {{ t('pricing.includes') }}
      </p>
      <ul class="pricing-plan-card__facts">
        <li v-for="item in planFacts" :key="item">
          <Icon name="check" size="sm" aria-hidden="true" />
          <span>{{ item }}</span>
        </li>
      </ul>

      <button
        type="button"
        class="pricing-plan-card__details"
        :aria-label="t('pricing.viewQuotaDetailsAccessible', { plan: plan.name })"
        @click="emit('details', plan)"
      >
        {{ t('pricing.viewQuotaDetails') }}
        <Icon name="arrowRight" size="xs" aria-hidden="true" />
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { SubscriptionPlan } from '@/types/payment'

const props = withDefaults(defineProps<{
  plan: SubscriptionPlan
  featured?: boolean
}>(), {
  featured: false,
})

const emit = defineEmits<{
  select: [plan: SubscriptionPlan]
  details: [plan: SubscriptionPlan]
}>()

const { locale, t } = useI18n()

const formattedPrice = computed(() => {
  const currency = props.plan.currency?.trim().toUpperCase() || 'USD'
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(props.plan.price)
  } catch {
    return `${currency} ${props.plan.price.toFixed(2)}`
  }
})

const validityLabel = computed(() => {
  const unit = props.plan.validity_unit || 'day'
  if (unit === 'month') return t('payment.perMonth')
  if (unit === 'year') return t('payment.perYear')
  return t('pricing.validityDays', { days: props.plan.validity_days })
})

const planFacts = computed(() => [...new Set(
  (Array.isArray(props.plan.features) ? props.plan.features : [])
    .map((item) => item.trim())
    .filter(Boolean),
)].slice(0, 4))
</script>

<style scoped>
.pricing-plan-card {
  position: relative;
  display: flex;
  min-width: 0;
  min-height: 520px;
  overflow: hidden;
  flex-direction: column;
  padding: 24px 28px 28px;
  border: 1px solid #e5e5e5;
  border-radius: 16px;
  color: #0d0d0d;
  background: #fff;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.03);
  text-align: left;
}

.pricing-plan-card--featured {
  border-color: var(--pricing-featured-border);
  background-color: var(--pricing-featured-background);
  box-shadow: var(--pricing-featured-shadow);
  transition:
    background-color 520ms,
    border-color 520ms,
    box-shadow 520ms;
}

.pricing-plan-card__aura {
  position: absolute;
  z-index: 0;
  inset: -15%;
  background: var(--pricing-featured-aura);
  opacity: 0.6;
  pointer-events: none;
  transition: opacity 520ms;
}

.pricing-plan-card__inner {
  position: relative;
  z-index: 1;
  display: flex;
  height: 100%;
  flex: 1;
  flex-direction: column;
}

.pricing-plan-card__heading {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.pricing-plan-card__heading h3 {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.025em;
  line-height: 1.5;
}

.pricing-plan-card__badge {
  flex: none;
  padding: 4px 10px;
  border-radius: 99px;
  color: var(--pricing-badge-color);
  background: var(--pricing-badge-background);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  line-height: 1.5;
}

.pricing-plan-card__description {
  min-height: 24px;
  margin-bottom: 14px;
  color: #5d5d5d;
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
}

.pricing-plan-card__price {
  display: flex;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 24px;
  white-space: nowrap;
}

.pricing-plan-card__price span {
  font-size: 32px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  line-height: 1.5;
}

.pricing-plan-card__price small {
  flex-shrink: 0;
  color: #8e8e8e;
  font-size: 12px;
  font-weight: 400;
}

.pricing-plan-card__action {
  display: flex;
  width: 100%;
  height: 46px;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: #fff;
  background: #171717;
  font-size: 14px;
  font-weight: 500;
  transition:
    transform 150ms ease,
    opacity 150ms ease,
    box-shadow 150ms ease;
}

.pricing-plan-card--featured .pricing-plan-card__action {
  background: var(--pricing-featured-button);
  transition:
    background-color 150ms ease,
    transform 150ms ease,
    opacity 150ms ease,
    box-shadow 150ms ease;
}

.pricing-plan-card--featured .pricing-plan-card__action:hover {
  background: var(--pricing-featured-button-hover);
}

.pricing-plan-card__action:active {
  transform: scale(0.98);
}

.pricing-plan-card__action:focus-visible {
  outline: 2px solid #171717;
  outline-offset: 2px;
  box-shadow: 0 0 0 4px rgb(23 23 23 / 0.14);
}

.pricing-plan-card--featured .pricing-plan-card__action:focus-visible {
  outline-color: var(--pricing-featured-button);
  box-shadow: var(--pricing-featured-focus-ring);
}

.pricing-plan-card__divider {
  height: 1px;
  margin: 26px 0 22px;
  background: #e8e8e8;
}

.pricing-plan-card__includes {
  margin-bottom: 14px;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
}

.pricing-plan-card__facts li {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 10px;
  margin-bottom: 12px;
  color: #444;
  font-size: 14px;
  font-weight: 400;
  line-height: 1.5;
}

.pricing-plan-card__facts li :deep(svg) {
  margin-top: 2px;
  color: #171717;
  stroke-width: 2.1;
}

.pricing-plan-card__details {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: auto;
  border-radius: 2px;
  color: #525252;
  background: transparent;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.5;
  transition: color 200ms;
}

.pricing-plan-card__details:hover {
  color: #111;
}

.pricing-plan-card__details:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 4px;
}

@media (max-width: 640px) {
  .pricing-plan-card {
    padding-right: 18px;
    padding-left: 18px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .pricing-plan-card--featured,
  .pricing-plan-card__aura,
  .pricing-plan-card--featured .pricing-plan-card__action {
    transition-duration: 100ms !important;
  }
}
</style>
