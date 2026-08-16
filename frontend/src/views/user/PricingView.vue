<template>
  <main
    class="pricing-page font-[family-name:var(--lx-clay-font-ui)]"
    :class="activeDefinition.themeClass"
    :data-tier="activeTier"
    :data-mode="membershipMode || undefined"
    data-testid="pricing-page"
  >
    <a
      class="pricing-page__back"
      href="/dashboard"
      :aria-label="t('common.back')"
      data-testid="pricing-back"
      @click="handleBack"
    >
      <Icon name="arrowLeft" size="sm" aria-hidden="true" />
      <span>{{ t('common.back') }}</span>
    </a>

    <div class="pricing-page__module">
      <section class="pricing-page__hero" :data-tier="activeTier">
        <div class="pricing-page__atmosphere">
          <PricingFluidBackground :tier="activeTier" />
        </div>
        <div class="pricing-page__scrim" aria-hidden="true"></div>

        <div class="pricing-page__content">
          <div class="pricing-page__tier-info">
            <nav
              ref="tabListRef"
              role="tablist"
              :aria-label="t('pricing.tierSelectorLabel')"
              class="pricing-page__tabs"
              data-testid="pricing-tier-tabs"
              @keydown="handleTabKeydown"
            >
              <button
                v-for="tier in TIER_DEFINITIONS"
                :id="`pricing-tier-tab-${tier.id}`"
                :key="tier.id"
                type="button"
                role="tab"
                class="pricing-page__tab"
                :data-tier="tier.id"
                :aria-selected="tier.id === activeTier"
                :tabindex="tier.id === activeTier ? 0 : -1"
                aria-controls="pricing-tier-summary"
                @click="selectTier(tier.id)"
              >
                {{ t(tier.nameKey) }}
              </button>
            </nav>

            <div
              id="pricing-tier-summary"
              role="tabpanel"
              :aria-labelledby="activeTabId"
            >
              <h2 hidden>{{ t(activeDefinition.nameKey) }}</h2>
              <p>{{ t(activeDefinition.descriptionKey) }}</p>
            </div>
          </div>

          <section
            v-if="loading && plans.length === 0"
            class="pricing-page__state"
            data-testid="pricing-loading"
            aria-busy="true"
            :aria-label="t('pricing.loading')"
          >
            <span class="pricing-page__state-spinner" aria-hidden="true"></span>
            <p>{{ t('pricing.loading') }}</p>
          </section>

          <section
            v-else-if="loadError && plans.length === 0"
            class="pricing-page__state"
            data-testid="pricing-error"
            role="alert"
          >
            <h3>{{ t('pricing.loadErrorTitle') }}</h3>
            <p>{{ t('pricing.loadErrorDescription') }}</p>
            <button type="button" @click="loadPlans(true)">
              {{ t('pricing.retry') }}
            </button>
          </section>

          <div
            v-else-if="hasApprovedPlans"
            class="pricing-page__card-stage"
            :data-active-tier="activeTier"
            data-testid="pricing-plan-grid"
            :aria-label="t('pricing.availablePlans')"
          >
            <div
              v-for="tier in TIER_DEFINITIONS"
              :id="`pricing-panel-${tier.id}`"
              :key="tier.id"
              class="pricing-page__tier-panel"
              :class="`pricing-page__tier-panel--${tier.id}`"
              :aria-hidden="tier.id !== activeTier"
              :inert="tier.id !== activeTier"
              :data-testid="`pricing-tier-panel-${tier.id}`"
            >
              <PricingPlanCard
                v-for="plan in plansByTier[tier.id]"
                :key="plan.id"
                :plan="plan"
                :featured="isFeaturedPlan(tier.id, plan)"
                :current="isCurrentPlan(plan)"
                :action-label="planActionLabel(plan)"
                @select="selectPlan"
                @details="viewPlanDetails"
              />
            </div>
          </div>

          <section
            v-else
            class="pricing-page__state"
            data-testid="pricing-empty"
          >
            <h3>{{ t('pricing.emptyTitle') }}</h3>
            <p>{{ t('pricing.emptyDescription') }}</p>
          </section>

          <p v-if="loadError && plans.length > 0" class="sr-only" role="status">
            {{ t('pricing.staleWarning') }}
          </p>
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import PricingFluidBackground from '@/components/payment/PricingFluidBackground.vue'
import PricingPlanCard from '@/components/payment/PricingPlanCard.vue'
import { usePaymentStore } from '@/stores/payment'
import type { SubscriptionPlan } from '@/types/payment'
import {
  readMembershipMode,
  readPositiveQueryInteger,
  readSingleQueryString,
} from '@/navigation/purchaseQueryState'

const TIER_DEFINITIONS = [
  {
    id: 'low',
    nameKey: 'pricing.tiers.low.name',
    descriptionKey: 'pricing.tiers.low.description',
    themeClass: 'pricing-theme-light',
    planNames: ['Try', 'Basic', 'Standard'],
    featuredPlan: 'Standard',
  },
  {
    id: 'mid',
    nameKey: 'pricing.tiers.mid.name',
    descriptionKey: 'pricing.tiers.mid.description',
    themeClass: 'pricing-theme-medium',
    planNames: ['Plus', 'Pro', 'Max'],
    featuredPlan: 'Pro',
  },
  {
    id: 'high',
    nameKey: 'pricing.tiers.high.name',
    descriptionKey: 'pricing.tiers.high.description',
    themeClass: 'pricing-theme-high',
    planNames: ['Ultra'],
    featuredPlan: 'Ultra',
  },
] as const

type PricingTier = typeof TIER_DEFINITIONS[number]['id']

const TIER_QUERY_ALIASES: Record<string, PricingTier> = {
  low: 'low',
  light: 'low',
  lightweight: 'low',
  '轻量级': 'low',
  mid: 'mid',
  middle: 'mid',
  medium: 'mid',
  '中量级': 'mid',
  high: 'high',
  heavy: 'high',
  heavyweight: 'high',
  '高量级': 'high',
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const paymentStore = usePaymentStore()

const tabListRef = ref<HTMLElement | null>(null)
const activeTier = ref<PricingTier>('low')
const loading = ref(paymentStore.checkoutInfo === null)
const loadError = ref(false)
const appliedPreferredSelection = ref('')

const normalizePlanName = (name: string) => name.trim().toLocaleLowerCase('en-US')

const activeDefinition = computed(() => (
  TIER_DEFINITIONS.find((tier) => tier.id === activeTier.value) ?? TIER_DEFINITIONS[1]
))
const activeTabId = computed(() => `pricing-tier-tab-${activeTier.value}`)
const plans = computed(() => paymentStore.checkoutInfo?.plans ?? [])
const membershipMode = computed(() => readMembershipMode(route.query))
const preferredTier = computed<PricingTier | null>(() => {
  const value = readSingleQueryString(route.query, 'tier').trim().toLocaleLowerCase('en-US')
  if (value) {
    const aliasedTier = TIER_QUERY_ALIASES[value]
    if (aliasedTier) return aliasedTier

    // Accept a plan name as a convenience for copied links while keeping the
    // canonical tier ids (`low`, `mid`, `high`) as the primary contract.
    const matchingTier = TIER_DEFINITIONS.find((tier) => (
      tier.planNames.some((name) => normalizePlanName(name) === value)
    ))
    if (matchingTier) return matchingTier.id
  }

  // Renewal/upgrade links may carry a concrete plan name or id. Resolve it
  // only after checkout data is available, without inventing a fallback tier.
  const planQuery = readSingleQueryString(route.query, 'plan').trim()
  if (!planQuery) return null
  const normalizedPlanQuery = normalizePlanName(planQuery)
  const planFromCatalogue = plans.value.find((plan) => (
    String(plan.id) === planQuery || normalizePlanName(plan.name) === normalizedPlanQuery
  ))
  return planFromCatalogue ? tierForPlan(planFromCatalogue) : TIER_DEFINITIONS.find((tier) => (
    tier.planNames.some((name) => normalizePlanName(name) === normalizedPlanQuery)
  ))?.id ?? null
})
const preferredGroupId = computed(() => {
  return readPositiveQueryInteger(route.query, 'group')
})

const plansByTier = computed<Record<PricingTier, SubscriptionPlan[]>>(() => {
  const plansByName = new Map<string, SubscriptionPlan>()
  plans.value.forEach((plan) => {
    const normalizedName = normalizePlanName(plan.name)
    if (!plansByName.has(normalizedName)) plansByName.set(normalizedName, plan)
  })

  const getPlans = (names: readonly string[]) => names
    .map((name) => plansByName.get(normalizePlanName(name)))
    .filter((plan): plan is SubscriptionPlan => plan !== undefined)

  return {
    low: getPlans(TIER_DEFINITIONS[0].planNames),
    mid: getPlans(TIER_DEFINITIONS[1].planNames),
    high: getPlans(TIER_DEFINITIONS[2].planNames),
  }
})

const hasApprovedPlans = computed(() => (
  Object.values(plansByTier.value).some((tierPlans) => tierPlans.length > 0)
))

function tierForPlan(plan: SubscriptionPlan): PricingTier | null {
  const normalizedName = normalizePlanName(plan.name)
  const definition = TIER_DEFINITIONS.find((tier) => (
    tier.planNames.some((name) => normalizePlanName(name) === normalizedName)
  ))
  return definition?.id ?? null
}

function isFeaturedPlan(tier: PricingTier, plan: SubscriptionPlan) {
  const definition = TIER_DEFINITIONS.find((item) => item.id === tier)
  return definition
    ? normalizePlanName(plan.name) === normalizePlanName(definition.featuredPlan)
    : false
}

function isCurrentPlan(plan: SubscriptionPlan): boolean {
  const planQuery = readSingleQueryString(route.query, 'plan').trim()
  if (planQuery && (
    String(plan.id) === planQuery
    || normalizePlanName(plan.name) === normalizePlanName(planQuery)
  )) return true

  // A bare `group` query is also used by legacy catalogue links. Only treat
  // it as the current membership when it carries a renewal/upgrade intent.
  if (membershipMode.value !== 'renew' && membershipMode.value !== 'upgrade') return false
  const groupId = preferredGroupId.value
  return groupId !== null && plan.group_id === groupId
}

function planActionLabel(plan: SubscriptionPlan): string {
  const mode = membershipMode.value
  const current = isCurrentPlan(plan)
  if (mode === 'renew') return current ? t('balanceMembership.renew') : ''
  if (mode === 'upgrade') return current ? t('pricing.currentPlan') : t('balanceMembership.upgrade')
  if (mode === 'subscribe') return t('balanceMembership.subscribe')
  return current ? t('pricing.currentPlan') : ''
}

function selectTier(tier: PricingTier) {
  activeTier.value = tier
}

function handleBack(event: MouseEvent) {
  if (
    event.defaultPrevented
    || event.button !== 0
    || event.metaKey
    || event.ctrlKey
    || event.shiftKey
    || event.altKey
  ) return

  const historyBack = router.options.history.state.back
  let hasSafeBackTarget = false
  if (typeof historyBack === 'string') {
    try {
      hasSafeBackTarget = new URL(historyBack, window.location.origin).origin === window.location.origin
    } catch {
      hasSafeBackTarget = false
    }
  }

  event.preventDefault()
  if (hasSafeBackTarget) router.back()
  else void router.push('/dashboard')
}

function handleTabKeydown(event: KeyboardEvent) {
  const tabs = Array.from(
    tabListRef.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]') ?? [],
  )
  const currentIndex = tabs.indexOf(document.activeElement as HTMLButtonElement)
  if (currentIndex < 0) return

  let nextIndex: number | null = null
  if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % tabs.length
  else if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
  else if (event.key === 'Home') nextIndex = 0
  else if (event.key === 'End') nextIndex = tabs.length - 1
  if (nextIndex === null) return

  event.preventDefault()
  const nextTab = tabs[nextIndex]
  const tier = nextTab.dataset.tier as PricingTier | undefined
  if (!tier) return
  nextTab.focus()
  selectTier(tier)
}

async function loadPlans(force = false) {
  loading.value = plans.value.length === 0
  loadError.value = false
  try {
    await paymentStore.ensureCheckoutInfo(force)
  } catch {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

function openPlan(plan: SubscriptionPlan) {
  const mode = membershipMode.value
  void router.push({
    path: '/purchase',
    query: {
      tab: 'subscription',
      plan: String(plan.id),
      ...(mode ? { mode } : {}),
    },
  })
}

function selectPlan(plan: SubscriptionPlan) {
  openPlan(plan)
}

function viewPlanDetails(plan: SubscriptionPlan) {
  openPlan(plan)
}

watch([plans, preferredTier, preferredGroupId], ([availablePlans, tier, groupId]) => {
  // An explicit tier is authoritative. This is used by upgrade links, while
  // renewal links can continue to use the legacy group id to infer a tier.
  const planQuery = readSingleQueryString(route.query, 'plan').trim()
  const selectionKey = `${tier ?? ''}:${groupId ?? ''}:${planQuery}`
  if (selectionKey === '::') {
    activeTier.value = 'low'
    appliedPreferredSelection.value = ''
    return
  }
  if (appliedPreferredSelection.value === selectionKey) return

  if (tier) {
    activeTier.value = tier
    appliedPreferredSelection.value = selectionKey
    return
  }

  if (groupId === null || availablePlans.length === 0) return

  const preferredPlan = availablePlans.find((plan) => (
    plan.group_id === groupId && tierForPlan(plan) !== null
  ))
  const inferredTier = preferredPlan ? tierForPlan(preferredPlan) : null
  if (inferredTier) activeTier.value = inferredTier
  appliedPreferredSelection.value = selectionKey
}, { immediate: true })

onMounted(() => {
  void loadPlans()
})
</script>

<style scoped>
.pricing-page {
  --pricing-hero-background: radial-gradient(circle at 72% 35%, #7c3aed 0%, #eaf5ff 45%, #fff 100%);
  --pricing-featured-background: #f5f3ff;
  --pricing-featured-border: rgb(124 58 237 / 0.28);
  --pricing-featured-shadow: 0 16px 42px rgb(124 58 237 / 0.11);
  --pricing-featured-aura: radial-gradient(circle at 100% 0%, rgb(124 58 237 / 0.5) 0%, transparent 65%);
  --pricing-featured-button: #7c3aed;
  --pricing-featured-button-hover: #6d28d9;
  --pricing-featured-focus-ring: 0 0 0 4px rgb(124 58 237 / 0.15);
  --pricing-badge-background: #f5f3ff;
  --pricing-badge-color: #6d28d9;

  min-height: 100dvh;
  position: relative;
  color: #0d0d0d;
  background: #f4f1fa;
}

.pricing-theme-light {
  --pricing-hero-background: radial-gradient(circle at 72% 35%, #7dc4ff 0%, #cde8ff 42%, #fff 100%);
  --pricing-featured-background: #f8fbff;
  --pricing-featured-border: rgb(54 118 209 / 0.18);
  --pricing-featured-shadow: 0 12px 32px rgb(54 118 209 / 0.06);
  --pricing-featured-aura: radial-gradient(circle at 100% 0%, rgb(96 165 250 / 0.38) 0%, rgb(125 211 252 / 0.18) 42%, transparent 72%);
  --pricing-featured-button: #3676d1;
  --pricing-featured-button-hover: #2f67bd;
  --pricing-featured-focus-ring: 0 0 0 4px rgb(54 118 209 / 0.18);
  --pricing-badge-background: #eaf5ff;
  --pricing-badge-color: #075985;
}

.pricing-theme-medium {
  --pricing-hero-background: radial-gradient(circle at 72% 35%, #7c3aed 0%, #eaf5ff 45%, #fff 100%);
  --pricing-featured-background: #f5f3ff;
  --pricing-featured-border: rgb(124 58 237 / 0.28);
  --pricing-featured-shadow: 0 16px 42px rgb(124 58 237 / 0.11);
  --pricing-featured-aura: radial-gradient(circle at 100% 0%, rgb(124 58 237 / 0.5) 0%, transparent 65%);
  --pricing-featured-button: #7c3aed;
  --pricing-featured-button-hover: #6d28d9;
  --pricing-featured-focus-ring: 0 0 0 4px rgb(124 58 237 / 0.15);
  --pricing-badge-background: #f5f3ff;
  --pricing-badge-color: #6d28d9;
}

.pricing-theme-high {
  --pricing-hero-background: radial-gradient(circle at 72% 35%, #db2777 0%, #f59e0b 44%, #fff 100%);
  --pricing-featured-background: #fffaf5;
  --pricing-featured-border: rgb(219 39 119 / 0.18);
  --pricing-featured-shadow: 0 14px 36px rgb(245 158 11 / 0.07);
  --pricing-featured-aura: radial-gradient(circle at 100% 0%, rgb(219 39 119 / 0.35) 0%, rgb(245 158 11 / 0.25) 40%, transparent 75%);
  --pricing-featured-button: #ba5a17;
  --pricing-featured-button-hover: #a85115;
  --pricing-featured-focus-ring: 0 0 0 4px rgb(217 119 6 / 0.22);
  --pricing-badge-background: #fff4e5;
  --pricing-badge-color: #9a6700;
}

.pricing-page__module {
  position: relative;
  z-index: 10;
  display: flex;
  width: 100%;
  margin: 0 auto;
  flex-direction: column;
  gap: 80px;
  padding: 120px 0 200px;
}

.pricing-page__hero {
  width: 100%;
  display: contents;
}

.pricing-page__atmosphere {
  position: fixed;
  z-index: 0;
  overflow: hidden;
  inset: -5%;
  background: var(--pricing-hero-background);
  pointer-events: none;
}

.pricing-page__scrim {
  position: fixed;
  z-index: 1;
  inset: 0;
  background: radial-gradient(circle at 50% 25%, rgb(244 241 250 / 0.5) 0%, rgb(244 241 250 / 0.9) 100%);
  pointer-events: none;
}

.pricing-page__content {
  position: relative;
  z-index: 10;
  display: flex;
  width: 100%;
  align-items: center;
  flex-direction: column;
  padding: 0 24px;
  text-align: center;
}

.pricing-page__tier-info {
  max-width: 800px;
  margin-bottom: 56px;
}

.pricing-page__tabs {
  position: relative;
  z-index: 20;
  display: inline-flex;
  gap: 2px;
  margin-bottom: 48px;
  padding: 3px;
  border: 1px solid #e5e5e5;
  border-radius: 12px;
  background: #f7f7f8;
}

.pricing-page__tab {
  padding: 6px 18px;
  border-radius: 8px;
  color: #5d5d5d;
  background: transparent;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 160ms ease;
}

.pricing-page__tab:hover:not([aria-selected="true"]) {
  color: #0d0d0d;
  background: rgb(0 0 0 / 0.04);
}

.pricing-page__tab[aria-selected="true"] {
  color: #0d0d0d;
  background: #fff;
  box-shadow: 0 1px 2px rgb(17 17 17 / 0.08);
}

.pricing-page__tab:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
}

.pricing-page__tier-info h2 {
  margin-bottom: 12px;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.025em;
  line-height: 32px;
}

.pricing-page__tier-info p {
  color: #6b7280;
  font-size: 14px;
  line-height: 1.625;
}

.pricing-page__card-stage {
  position: relative;
  z-index: 20;
  display: grid;
  width: 100%;
  max-width: 932px;
  margin: 0 auto;
  grid-template-columns: 1fr;
  text-align: left;
}

.pricing-page__tier-panel {
  position: relative;
  display: grid;
  width: 100%;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 26px;
  transition:
    opacity 520ms cubic-bezier(0.22, 1, 0.36, 1),
    visibility 520ms;
}

.pricing-page__tier-panel--high > article {
  grid-column: 2;
}

.pricing-page__tier-panel[aria-hidden="true"] {
  position: absolute;
  z-index: -1;
  inset: 0;
  visibility: hidden;
  opacity: 0;
  pointer-events: none;
}

.pricing-page__tier-panel[aria-hidden="false"] {
  position: relative;
  z-index: 1;
}

.pricing-page__state {
  display: flex;
  width: min(420px, 100%);
  min-height: 240px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 32px;
  border: 1px solid #e5e5e5;
  border-radius: 16px;
  background: rgb(255 255 255 / 0.88);
  text-align: center;
}

.pricing-page__state h3 {
  font-size: 18px;
  font-weight: 700;
}

.pricing-page__state p {
  margin-top: 10px;
  color: #5d5d5d;
  font-size: 14px;
  line-height: 1.5;
}

.pricing-page__state button {
  min-height: 42px;
  margin-top: 20px;
  padding: 0 18px;
  border-radius: 999px;
  color: #fff;
  background: #171717;
  font-size: 14px;
  font-weight: 500;
}

.pricing-page__state-spinner {
  width: 28px;
  height: 28px;
  border: 2px solid #dedede;
  border-top-color: #171717;
  border-radius: 999px;
  animation: pricing-spin 700ms linear infinite;
}

@keyframes pricing-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 900px) {
  .pricing-page__tier-panel {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  .pricing-page__tier-panel > article:nth-child(3) {
    width: calc((100% - 16px) / 2);
    max-width: none;
    grid-column: 1 / -1;
    justify-self: center;
  }

  .pricing-page__tier-panel--high {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pricing-page__tier-panel--high > article {
    width: calc((100% - 16px) / 2);
    max-width: none;
    grid-column: 1 / -1;
    justify-self: center;
  }
}

.pricing-page__back {
  position: absolute;
  z-index: 30;
  top: calc(env(safe-area-inset-top, 0px) + 32px);
  left: calc(env(safe-area-inset-left, 0px) + clamp(24px, 2.35vw, 48px));
  display: inline-flex;
  min-width: 72px;
  height: 36px;
  align-items: center;
  gap: 6px;
  padding: 0 11px;
  border: 1px solid transparent;
  border-radius: 10px;
  color: rgb(13 13 13 / 0.62);
  background: transparent;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  text-decoration: none;
  cursor: pointer;
  transition:
    color 155ms ease,
    background-color 155ms ease,
    border-color 155ms ease;
}

.pricing-page__back :deep(svg) {
  width: 16px;
  height: 16px;
  transition: transform 155ms ease;
}

.pricing-page__back:hover {
  border-color: rgb(13 13 13 / 0.08);
  color: #171717;
  background: rgb(255 255 255 / 0.55);
}

.pricing-page__back:hover :deep(svg) {
  transform: translateX(-1px);
}

.pricing-page__back:active {
  background: rgb(255 255 255 / 0.7);
}

.pricing-page__back:focus-visible {
  outline: 2px solid #7c3aed;
  outline-offset: 2px;
  color: #171717;
  background: rgb(255 255 255 / 0.72);
  box-shadow: 0 0 0 4px #fff;
}

@media (max-width: 640px) {
  .pricing-page__back {
    top: calc(env(safe-area-inset-top, 0px) + 12px);
    left: calc(env(safe-area-inset-left, 0px) + 12px);
    height: 44px;
    padding: 0 10px;
  }

  .pricing-page__back :deep(svg) {
    width: 18px;
    height: 18px;
  }

  .pricing-page__content {
    min-height: 320px;
    padding: 48px 16px 32px;
  }

  .pricing-page__tier-panel,
  .pricing-page__tier-panel--high {
    grid-template-columns: minmax(0, 1fr);
  }

  .pricing-page__tier-panel > article:nth-child(3),
  .pricing-page__tier-panel--high > article {
    width: 100%;
    max-width: none;
    grid-column: 1;
    justify-self: stretch;
  }
}

@media (prefers-reduced-motion: reduce) {
  .pricing-page__back,
  .pricing-page__back :deep(svg) {
    transition: none !important;
    transform: none !important;
  }

  .pricing-page__tier-panel {
    transition-duration: 100ms !important;
  }

  .pricing-page__state-spinner {
    animation: none;
  }
}
</style>
