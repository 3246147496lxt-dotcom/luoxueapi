<template>
  <article
    class="pricing-plan-card"
    :class="{
      'pricing-plan-card--renewal': renewal,
      'pricing-plan-card--focused': focused,
    }"
    :data-plan-id="String(plan.id)"
    :data-plan-group="String(plan.group_id)"
  >
    <div class="pricing-plan-card__heading">
      <div class="min-w-0">
        <h2>{{ plan.name }}</h2>
      </div>
      <span v-if="renewal" class="pricing-plan-card__status">
        {{ t('pricing.renewalOption') }}
      </span>
    </div>

    <p v-if="plan.description" class="pricing-plan-card__description">
      {{ plan.description }}
    </p>

    <div class="pricing-plan-card__price">
      <span>{{ formattedPrice }}</span>
      <small>/ {{ validityLabel }}</small>
    </div>

    <button
      type="button"
      class="pricing-plan-card__action"
      :aria-label="t(
        renewal ? 'pricing.renewPlanAccessible' : 'pricing.choosePlanAccessible',
        { plan: plan.name },
      )"
      @click="emit('select', plan)"
    >
      {{ renewal ? t('pricing.renewPlan') : t('pricing.choosePlan') }}
    </button>

    <div class="pricing-plan-card__divider" aria-hidden="true"></div>

    <p class="pricing-plan-card__includes">
      {{ t('pricing.includes') }}
    </p>
    <ul class="pricing-plan-card__facts">
      <li v-for="item in planFacts" :key="`fact-${item}`">
        <Icon name="check" size="sm" aria-hidden="true" />
        <span>{{ item }}</span>
      </li>
      <li
        v-for="term in planTerms"
        :key="`term-${term.key}`"
        :data-metric="term.key"
      >
        <Icon name="check" size="sm" aria-hidden="true" />
        <span>{{ term.text }}</span>
      </li>
    </ul>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import type { SubscriptionPlan } from '@/types/payment'
import { formatPeakRateWindow, hasPeakRate, serverTimezoneLabel } from '@/utils/peak-rate'

const props = defineProps<{
  plan: SubscriptionPlan
  renewal: boolean
  focused?: boolean
}>()

const emit = defineEmits<{
  select: [plan: SubscriptionPlan]
}>()

const { locale, t } = useI18n()
const appStore = useAppStore()

const formattedPrice = computed(() => {
  const currency = props.plan.currency?.trim().toUpperCase() || 'USD'
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency,
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

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const peakRateFact = computed(() => {
  if (!hasPeakRate(props.plan)) return null
  return t('pricing.peakRateWindow', {
    window: formatPeakRateWindow(
      props.plan,
      serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset),
    ),
  })
})

const modelScopesFact = computed(() => {
  const scopes = [...new Set(
    (props.plan.supported_model_scopes ?? [])
      .map((scope) => scope.trim())
      .filter(Boolean),
  )]
  if (scopes.length === 0) return null
  return t('pricing.modelScopes', {
    models: scopes.map((scope) => MODEL_SCOPE_LABELS[scope] || scope).join(', '),
  })
})

interface PlanTerm {
  key: 'rate' | 'daily' | 'weekly' | 'monthly'
  text: string
}

const quotaTerm = (
  key: PlanTerm['key'],
  translationKey: 'pricing.dailyQuota' | 'pricing.weeklyQuota' | 'pricing.monthlyQuota',
  value: number | null | undefined,
): PlanTerm => ({
  key,
  text: t(translationKey, {
    amount: value == null ? t('payment.planCard.unlimited') : value.toFixed(2),
  }),
})

const planTerms = computed<PlanTerm[]>(() => [
  {
    key: 'rate',
    text: t('pricing.rateMultiplier', {
      rate: Number((props.plan.rate_multiplier ?? 1).toPrecision(10)),
    }),
  },
  quotaTerm('daily', 'pricing.dailyQuota', props.plan.daily_limit_usd),
  quotaTerm('weekly', 'pricing.weeklyQuota', props.plan.weekly_limit_usd),
  quotaTerm('monthly', 'pricing.monthlyQuota', props.plan.monthly_limit_usd),
])

const planFacts = computed(() => {
  const facts = [
    ...(Array.isArray(props.plan.features) ? props.plan.features : []),
    peakRateFact.value,
    modelScopesFact.value,
  ].filter((item): item is string => typeof item === 'string' && item.trim().length > 0)

  return [...new Set(facts)]
})
</script>

<style scoped>
.pricing-plan-card {
  display: flex;
  width: 100%;
  min-width: 0;
  min-height: 540px;
  flex-direction: column;
  padding: 30px;
  border: 1px solid #d9d9d9;
  border-radius: 16px;
  color: #171717;
  background: #fff;
  box-shadow: 0 1px 2px rgb(0 0 0 / 0.025);
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    transform 160ms ease;
}

.pricing-plan-card:hover {
  border-color: #b7b7b7;
  box-shadow: 0 10px 28px rgb(0 0 0 / 0.07);
  transform: translateY(-2px);
}

.pricing-plan-card--renewal,
.pricing-plan-card--focused {
  border-color: color-mix(in srgb, var(--lx-clay-accent) 68%, #a3a3a3);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--lx-clay-accent) 22%, transparent);
}

.pricing-plan-card__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.pricing-plan-card h2 {
  overflow-wrap: anywhere;
  font-size: 24px;
  font-weight: 650;
  letter-spacing: -0.025em;
  line-height: 1.2;
}

.pricing-plan-card__status {
  flex: none;
  padding: 5px 9px;
  border-radius: 999px;
  color: #17633a;
  background: #e8f7ee;
  font-size: 11px;
  font-weight: 650;
}

.pricing-plan-card__description {
  min-height: 48px;
  margin-top: 15px;
  color: #666;
  font-size: 14px;
  line-height: 1.65;
}

.pricing-plan-card__price {
  display: flex;
  min-height: 52px;
  align-items: baseline;
  gap: 8px;
  margin-top: 24px;
}

.pricing-plan-card__price span {
  font-size: 34px;
  font-weight: 650;
  letter-spacing: -0.04em;
  line-height: 1;
}

.pricing-plan-card__price small {
  color: #737373;
  font-size: 13px;
}

.pricing-plan-card__action {
  width: 100%;
  min-height: 46px;
  margin-top: 24px;
  border-radius: 999px;
  color: #fff;
  background: var(--lx-clay-accent-deep);
  font-size: 14px;
  font-weight: 650;
  transition:
    background-color 150ms ease,
    transform 150ms ease;
}

.pricing-plan-card__action:hover {
  background: var(--lx-clay-accent);
}

.pricing-plan-card__action:active {
  transform: scale(0.99);
}

.pricing-plan-card__action:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 38%, transparent);
  outline-offset: 2px;
}

.pricing-plan-card__divider {
  height: 1px;
  margin: 26px 0 22px;
  background: #e8e8e8;
}

.pricing-plan-card__includes {
  margin-bottom: 14px;
  font-size: 13px;
  font-weight: 650;
}

.pricing-plan-card__facts {
  display: grid;
  gap: 12px;
}

.pricing-plan-card__facts li {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: start;
  gap: 10px;
  color: #444;
  font-size: 13px;
  line-height: 1.5;
}

.pricing-plan-card__facts li svg {
  margin-top: 1px;
  color: #171717;
  stroke-width: 2.1;
}

:global(html.dark) .pricing-plan-card {
  border-color: #393939;
  color: #f4f4f4;
  background: #242424;
  box-shadow: none;
}

:global(html.dark) .pricing-plan-card:hover {
  border-color: #555;
  box-shadow: 0 12px 30px rgb(0 0 0 / 0.22);
}

:global(html.dark) .pricing-plan-card__description,
:global(html.dark) .pricing-plan-card__price small {
  color: #a3a3a3;
}

:global(html.dark) .pricing-plan-card__status {
  color: #91e6b5;
  background: #173a28;
}

:global(html.dark) .pricing-plan-card__action {
  color: var(--lx-clay-on-accent);
  background: var(--lx-clay-accent-deep);
}

:global(html.dark) .pricing-plan-card__action:hover {
  background: var(--lx-clay-accent);
}

:global(html.dark) .pricing-plan-card__divider {
  background: #3d3d3d;
}

:global(html.dark) .pricing-plan-card__facts li {
  color: #d0d0d0;
}

:global(html.dark) .pricing-plan-card__facts li svg {
  color: #f4f4f4;
}

@media (prefers-reduced-motion: reduce) {
  .pricing-plan-card,
  .pricing-plan-card__action {
    transition: none;
  }

  .pricing-plan-card:hover {
    transform: none;
  }
}

@media (max-width: 639px) {
  .pricing-plan-card {
    min-height: auto;
    padding: 22px;
    border-radius: 16px;
  }

  .pricing-plan-card__description {
    min-height: 0;
  }
}
</style>
