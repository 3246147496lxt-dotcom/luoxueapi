<template>
  <main class="pricing-page" data-testid="pricing-page">
    <button
      type="button"
      class="pricing-page__close"
      :aria-label="t('pricing.close')"
      data-testid="pricing-close"
      @click="closePricing"
    >
      <Icon name="x" size="md" aria-hidden="true" />
    </button>

      <div class="pricing-page__inner">
        <header class="pricing-page__hero">
          <h1>{{ t('pricing.title') }}</h1>
          <p>{{ t('pricing.description') }}</p>

          <RouterLink
            v-if="hasActiveSubscriptions"
            to="/subscriptions"
            class="pricing-page__manage-link"
          >
            {{ t('pricing.manageSubscriptions') }}
            <Icon name="arrowRight" size="xs" aria-hidden="true" />
          </RouterLink>
        </header>

        <div
          v-if="loading && plans.length === 0"
          class="pricing-page__grid"
          data-testid="pricing-loading"
          aria-busy="true"
          :aria-label="t('pricing.loading')"
        >
          <div v-for="index in 3" :key="index" class="pricing-page__skeleton">
            <span></span>
            <span></span>
            <span></span>
            <span></span>
          </div>
        </div>

        <section
          v-else-if="loadError && plans.length === 0"
          class="pricing-page__state"
          data-testid="pricing-error"
          role="alert"
        >
          <span class="pricing-page__state-icon">
            <Icon name="exclamationCircle" size="lg" aria-hidden="true" />
          </span>
          <h2>{{ t('pricing.loadErrorTitle') }}</h2>
          <p>{{ t('pricing.loadErrorDescription') }}</p>
          <button type="button" @click="loadPlans(true)">
            <Icon name="refresh" size="sm" aria-hidden="true" />
            {{ t('pricing.retry') }}
          </button>
        </section>

        <template v-else>
          <div
            v-if="loadError"
            class="pricing-page__warning"
            data-testid="pricing-stale-warning"
            role="status"
          >
            <span>{{ t('pricing.staleWarning') }}</span>
            <button type="button" @click="loadPlans(true)">
              {{ t('pricing.retry') }}
            </button>
          </div>

          <div
            v-if="preferredGroupHasPlans"
            class="pricing-page__filter"
            data-testid="pricing-group-filter"
          >
            <span>{{ t('pricing.groupFilter') }}</span>
            <button type="button" @click="clearGroupFilter">
              {{ t('pricing.viewAllPlans') }}
            </button>
          </div>

          <section
            v-if="visiblePlans.length > 0"
            class="pricing-page__grid"
            data-testid="pricing-plan-grid"
            :aria-label="t('pricing.availablePlans')"
          >
            <PricingPlanCard
              v-for="plan in visiblePlans"
              :key="plan.id"
              :plan="plan"
              :renewal="renewalGroupIds.has(plan.group_id)"
              :focused="preferredGroupId === plan.group_id"
              @select="selectPlan"
            />
          </section>

          <section
            v-else
            class="pricing-page__state"
            data-testid="pricing-empty"
          >
            <span class="pricing-page__state-icon">
              <Icon name="creditCard" size="lg" aria-hidden="true" />
            </span>
            <h2>{{ t('pricing.emptyTitle') }}</h2>
            <p>{{ t('pricing.emptyDescription') }}</p>
          </section>
        </template>
      </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'
import PricingPlanCard from '@/components/payment/PricingPlanCard.vue'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { SubscriptionPlan } from '@/types/payment'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const subscriptionStore = useSubscriptionStore()

const loading = ref(paymentStore.checkoutInfo === null)
const loadError = ref(false)

const plans = computed(() => paymentStore.checkoutInfo?.plans ?? [])
const renewalGroupIds = computed(() => new Set(
  subscriptionStore.activeSubscriptions
    .filter((subscription) => subscription.status === 'active')
    .map((subscription) => subscription.group_id),
))
const hasActiveSubscriptions = computed(() => renewalGroupIds.value.size > 0)

const preferredGroupId = computed(() => {
  const value = Array.isArray(route.query.group)
    ? route.query.group[0]
    : route.query.group
  const parsed = Number(value)
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
})

const preferredGroupPlans = computed(() => (
  preferredGroupId.value === null
    ? []
    : plans.value.filter((plan) => plan.group_id === preferredGroupId.value)
))
const preferredGroupHasPlans = computed(() => preferredGroupPlans.value.length > 0)
const visiblePlans = computed(() => (
  preferredGroupHasPlans.value ? preferredGroupPlans.value : plans.value
))

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

function clearGroupFilter() {
  const query = { ...route.query }
  delete query.group
  void router.replace({
    path: route.path,
    query,
    hash: route.hash,
  })
}

function selectPlan(plan: SubscriptionPlan) {
  void router.push({
    path: '/purchase',
    query: {
      tab: 'subscription',
      plan: String(plan.id),
    },
  })
}

function closePricing() {
  const historyBack = typeof window !== 'undefined'
    ? window.history.state?.back
    : null
  const hasInternalHistory = typeof historyBack === 'string'
    && historyBack.length > 0
    && new URL(historyBack, window.location.origin).origin === window.location.origin

  if (hasInternalHistory) {
    router.back()
    return
  }

  void router.replace(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
}

onMounted(() => {
  void loadPlans()
})
</script>

<style scoped>
.pricing-page {
  position: relative;
  min-height: 100dvh;
  padding: 68px 32px 72px;
  color: #171717;
  background: #f3f3f3;
}

.pricing-page__inner {
  width: min(1200px, 100%);
  margin: 0 auto;
}

.pricing-page__close {
  position: fixed;
  z-index: 10;
  top: max(22px, env(safe-area-inset-top));
  right: max(24px, env(safe-area-inset-right));
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: #404040;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.pricing-page__close:hover {
  color: #171717;
  background: rgb(0 0 0 / 0.06);
}

.pricing-page__close:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 38%, transparent);
  outline-offset: 2px;
}

.pricing-page__hero {
  width: min(660px, 100%);
  margin: 0 auto 38px;
  text-align: center;
}

.pricing-page__hero h1 {
  color: #171717;
  font-size: clamp(30px, 3vw, 34px);
  font-weight: 650;
  letter-spacing: -0.025em;
  line-height: 1.2;
  text-wrap: balance;
}

.pricing-page__hero > p {
  margin: 12px auto 0;
  color: #666;
  font-size: 15px;
  line-height: 1.65;
  text-wrap: pretty;
}

.pricing-page__manage-link {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 20px;
  border-radius: 8px;
  color: #3f3f3f;
  font-size: 13px;
  font-weight: 600;
}

.pricing-page__manage-link:hover {
  color: #171717;
}

.pricing-page__manage-link:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 35%, transparent);
  outline-offset: 4px;
}

.pricing-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 320px), 384px));
  justify-content: center;
  gap: 24px;
  align-items: stretch;
}

.pricing-page__skeleton {
  display: flex;
  min-height: 540px;
  flex-direction: column;
  gap: 16px;
  padding: 28px;
  border: 1px solid #dedede;
  border-radius: 16px;
  background: #fff;
}

.pricing-page__skeleton span {
  height: 16px;
  border-radius: 999px;
  background: linear-gradient(90deg, #ececec 25%, #f5f5f5 45%, #ececec 65%);
  background-size: 220% 100%;
  animation: pricing-shimmer 1.3s ease-in-out infinite;
}

.pricing-page__skeleton span:first-child {
  width: 46%;
  height: 28px;
}

.pricing-page__skeleton span:nth-child(2) {
  width: 72%;
}

.pricing-page__skeleton span:nth-child(3) {
  width: 58%;
  height: 42px;
  margin-top: 10px;
}

.pricing-page__skeleton span:last-child {
  width: 100%;
  height: 46px;
}

.pricing-page__state {
  display: flex;
  min-height: 330px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  padding: 42px 24px;
  border: 1px solid #dedede;
  border-radius: 20px;
  background: #fff;
  text-align: center;
}

.pricing-page__state-icon {
  display: flex;
  width: 54px;
  height: 54px;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  color: #686868;
  background: #efefef;
}

.pricing-page__state h2 {
  margin-top: 18px;
  font-size: 19px;
  font-weight: 650;
}

.pricing-page__state p {
  width: min(440px, 100%);
  margin-top: 8px;
  color: #707070;
  font-size: 14px;
  line-height: 1.6;
}

.pricing-page__state button {
  display: inline-flex;
  min-height: 42px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin-top: 20px;
  padding: 0 18px;
  border-radius: 999px;
  color: #fff;
  background: #171717;
  font-size: 13px;
  font-weight: 650;
}

.pricing-page__filter,
.pricing-page__warning {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  padding: 12px 15px;
  border: 1px solid #dedede;
  border-radius: 12px;
  color: #555;
  background: rgb(255 255 255 / 0.75);
  font-size: 13px;
}

.pricing-page__filter button,
.pricing-page__warning button {
  flex: none;
  color: #171717;
  font-weight: 650;
}

@keyframes pricing-shimmer {
  to {
    background-position-x: -220%;
  }
}

:global(html.dark) .pricing-page {
  color: #f4f4f4;
  background: #171717;
}

:global(html.dark) .pricing-page__close {
  color: #d4d4d4;
}

:global(html.dark) .pricing-page__close:hover {
  color: #fff;
  background: rgb(255 255 255 / 0.1);
}

:global(html.dark) .pricing-page__hero h1 {
  color: #f4f4f4;
}

:global(html.dark) .pricing-page__hero > p,
:global(html.dark) .pricing-page__manage-link {
  color: #aaa;
}

:global(html.dark) .pricing-page__manage-link:hover {
  color: #fff;
}

:global(html.dark) .pricing-page__skeleton,
:global(html.dark) .pricing-page__state {
  border-color: #393939;
  background: #242424;
}

:global(html.dark) .pricing-page__skeleton span {
  background: linear-gradient(90deg, #323232 25%, #3b3b3b 45%, #323232 65%);
  background-size: 220% 100%;
}

:global(html.dark) .pricing-page__state-icon {
  color: #bbb;
  background: #333;
}

:global(html.dark) .pricing-page__state p {
  color: #aaa;
}

:global(html.dark) .pricing-page__state button {
  color: #171717;
  background: #f4f4f4;
}

:global(html.dark) .pricing-page__filter,
:global(html.dark) .pricing-page__warning {
  border-color: #3a3a3a;
  color: #bbb;
  background: rgb(36 36 36 / 0.8);
}

:global(html.dark) .pricing-page__filter button,
:global(html.dark) .pricing-page__warning button {
  color: #fff;
}

@media (max-width: 639px) {
  .pricing-page {
    padding: max(76px, calc(env(safe-area-inset-top) + 58px)) 16px 48px;
  }

  .pricing-page__hero {
    margin-bottom: 30px;
  }

  .pricing-page__hero h1 {
    font-size: 30px;
  }

  .pricing-page__hero > p {
    font-size: 14px;
  }

  .pricing-page__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .pricing-page__skeleton {
    min-height: 430px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .pricing-page__close {
    transition: none;
  }

  .pricing-page__skeleton span {
    animation: none;
  }
}
</style>
