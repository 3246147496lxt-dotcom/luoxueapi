<template>
  <div
    class="wallet-settings"
    data-testid="wallet-subscription-settings"
  >
    <section
      class="wallet-settings__section"
      aria-labelledby="wallet-settings-balance-title"
    >
      <h3
        id="wallet-settings-balance-title"
        class="wallet-settings__section-title"
      >
        {{ t('personalSettings.wallet.balance') }}
      </h3>

      <dl class="wallet-settings__balance-list">
        <div class="wallet-settings__balance-row">
          <dt>{{ t('accountDock.availableBalance') }}</dt>
          <dd>
            <CreditAmount
              :value="summary.formattedAvailableBalance"
              icon-size="sm"
              :label="`${t('accountDock.availableBalance')} ${summary.formattedAvailableBalance}`"
            />
          </dd>
        </div>
        <div
          class="wallet-settings__balance-row"
          :class="{ 'wallet-settings__balance-row--frozen': summary.frozenBalance > 0 }"
          data-testid="wallet-frozen-balance"
        >
          <dt>{{ t('accountDock.frozenBalance') }}</dt>
          <dd>
            <CreditAmount
              :value="summary.formattedFrozenBalance"
              icon-size="xs"
              :label="`${t('accountDock.frozenBalance')} ${summary.formattedFrozenBalance}`"
            />
          </dd>
        </div>
      </dl>

      <nav
        class="wallet-settings__actions"
        :aria-label="t('personalSettings.wallet.actions')"
      >
        <RouterLink
          v-if="paymentEnabled"
          to="/purchase"
          class="wallet-settings__action-row"
          data-testid="wallet-purchase-link"
          @click="emitNavigate('/purchase')"
        >
          <Icon name="plus" size="sm" aria-hidden="true" />
          <span>{{ t('personalSettings.wallet.purchase') }}</span>
          <Icon name="chevronRight" size="xs" aria-hidden="true" />
        </RouterLink>
        <RouterLink
          to="/purchase#redeem"
          class="wallet-settings__action-row"
          data-testid="wallet-redeem-link"
          @click="emitNavigate('/purchase#redeem')"
        >
          <Icon name="gift" size="sm" aria-hidden="true" />
          <span>{{ t('personalSettings.wallet.redeem') }}</span>
          <Icon name="chevronRight" size="xs" aria-hidden="true" />
        </RouterLink>
      </nav>

      <p
        v-if="!paymentEnabled"
        class="wallet-settings__notice"
        data-testid="wallet-payment-disabled"
      >
        {{ t('personalSettings.wallet.paymentDisabled') }}
      </p>
    </section>

    <section
      v-if="!simpleMode"
      class="wallet-settings__section"
      aria-labelledby="wallet-settings-subscriptions-title"
    >
      <div class="wallet-settings__section-heading">
        <h3
          id="wallet-settings-subscriptions-title"
          class="wallet-settings__section-title"
        >
          {{ t('personalSettings.subscriptions.title') }}
        </h3>
        <RouterLink
          to="/subscriptions"
          class="wallet-settings__section-link"
          data-testid="wallet-subscriptions-link"
          @click="emitNavigate('/subscriptions')"
        >
          {{ t('personalSettings.subscriptions.manage') }}
        </RouterLink>
      </div>

      <div
        v-if="subscriptionsLoading"
        class="wallet-settings__loading"
        data-testid="wallet-subscriptions-loading"
        :aria-label="t('personalSettings.subscriptions.loading')"
        aria-busy="true"
      >
        <span v-for="index in 2" :key="index" class="wallet-settings__skeleton-row">
          <span></span>
          <span></span>
        </span>
      </div>

      <div
        v-else-if="subscriptionsUnavailable"
        class="wallet-settings__error"
        data-testid="wallet-subscriptions-error"
        role="alert"
      >
        <p>{{ t('personalSettings.subscriptions.loadError') }}</p>
        <button
          type="button"
          class="wallet-settings__retry"
          data-testid="wallet-subscriptions-retry"
          @click="retrySubscriptions"
        >
          <Icon name="refresh" size="xs" aria-hidden="true" />
          {{ t('personalSettings.subscriptions.retry') }}
        </button>
      </div>

      <div
        v-else-if="visibleSubscriptions.length > 0"
        class="wallet-settings__list"
        data-testid="wallet-subscription-list"
      >
        <div
          v-for="subscription in visibleSubscriptions"
          :key="subscription.id"
          class="wallet-settings__list-row"
          data-testid="wallet-subscription-row"
        >
          <Icon
            name="creditCard"
            size="sm"
            class="wallet-settings__row-icon"
            aria-hidden="true"
          />
          <div class="wallet-settings__row-copy">
            <p class="wallet-settings__row-title">
              {{ subscriptionName(subscription) }}
            </p>
            <p class="wallet-settings__row-meta">
              {{ subscriptionExpiry(subscription.expires_at) }}
            </p>
          </div>
          <RouterLink
            v-if="paymentEnabled"
            :to="renewalPath(subscription.group_id)"
            class="wallet-settings__row-link"
            :data-renew-group="String(subscription.group_id)"
            @click="emitNavigate(renewalPath(subscription.group_id))"
          >
            {{ t('personalSettings.subscriptions.renew') }}
          </RouterLink>
        </div>
      </div>

      <p
        v-else
        class="wallet-settings__empty"
        data-testid="wallet-subscriptions-empty"
      >
        {{ t('personalSettings.subscriptions.empty') }}
      </p>
    </section>

    <section
      v-if="showOrders"
      class="wallet-settings__section"
      data-testid="wallet-orders-section"
      aria-labelledby="wallet-settings-orders-title"
      :aria-busy="ordersLoading ? 'true' : undefined"
    >
      <div class="wallet-settings__section-heading">
        <h3
          id="wallet-settings-orders-title"
          class="wallet-settings__section-title"
        >
          {{ t('personalSettings.orders.title') }}
        </h3>
        <RouterLink
          to="/orders"
          class="wallet-settings__section-link"
          data-testid="wallet-orders-link"
          @click="emitNavigate('/orders')"
        >
          {{ t('personalSettings.orders.viewAll') }}
        </RouterLink>
      </div>

      <div
        v-if="ordersLoading"
        class="wallet-settings__loading"
        data-testid="wallet-orders-loading"
        :aria-label="t('personalSettings.orders.loading')"
      >
        <span v-for="index in 3" :key="index" class="wallet-settings__skeleton-row">
          <span></span>
          <span></span>
        </span>
      </div>

      <div
        v-else-if="ordersError"
        class="wallet-settings__error"
        data-testid="wallet-orders-error"
        role="alert"
      >
        <p>{{ ordersError }}</p>
        <button
          type="button"
          class="wallet-settings__retry"
          data-testid="wallet-orders-retry"
          @click="loadOrders"
        >
          <Icon name="refresh" size="xs" aria-hidden="true" />
          {{ t('personalSettings.orders.retry') }}
        </button>
      </div>

      <div
        v-else-if="orders.length > 0"
        class="wallet-settings__list"
        data-testid="wallet-order-list"
      >
        <div
          v-for="order in orders"
          :key="order.id"
          class="wallet-settings__list-row wallet-settings__order-row"
          data-testid="wallet-order-row"
        >
          <Icon
            :name="order.order_type === 'subscription' ? 'creditCard' : 'wallet'"
            size="sm"
            class="wallet-settings__row-icon"
            aria-hidden="true"
          />
          <div class="wallet-settings__row-copy">
            <p class="wallet-settings__row-title">
              {{ orderTypeLabel(order) }}
              <span class="wallet-settings__order-id">#{{ order.id }}</span>
            </p>
            <p class="wallet-settings__row-meta">
              {{ formatDate(order.created_at) }}
            </p>
          </div>
          <div class="wallet-settings__order-value">
            <span>{{ formatOrderAmount(order) }}</span>
            <OrderStatusBadge :status="order.status" />
          </div>
        </div>
      </div>

      <div
        v-else
        class="wallet-settings__empty wallet-settings__empty--orders"
        data-testid="wallet-orders-empty"
      >
        <Icon name="document" size="md" aria-hidden="true" />
        <div>
          <p>{{ t('personalSettings.orders.empty') }}</p>
          <p>{{ t('personalSettings.orders.emptyHint') }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { paymentAPI } from '@/api/payment'
import CreditAmount from '@/components/common/CreditAmount.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderStatusBadge from '@/components/payment/OrderStatusBadge.vue'
import { formatPaymentAmount } from '@/components/payment/currency'
import { useSubscriptionStore } from '@/stores/subscriptions'
import type { PaymentOrder } from '@/types/payment'
import type { UserSubscription } from '@/types'
import type { AccountPanelSummary } from './accountPanelTypes'

const props = defineProps<{
  summary: AccountPanelSummary
  paymentEnabled: boolean
  simpleMode: boolean
}>()

const emit = defineEmits<{
  navigate: [to: string]
}>()

const { locale, t } = useI18n()
const subscriptionStore = useSubscriptionStore()
const orders = ref<PaymentOrder[]>([])
const ordersLoading = ref(false)
const ordersError = ref('')
const ordersLoaded = ref(false)
let requestGeneration = 0
let mounted = true

const visibleSubscriptions = computed(
  () => subscriptionStore.activeSubscriptions.slice(0, 3),
)
const subscriptionsLoading = computed(
  () => subscriptionStore.loading,
)
const subscriptionsUnavailable = computed(
  () => (
    !subscriptionStore.loading
    && !props.summary.subscriptionsLoaded
    && !subscriptionStore.loaded
    && visibleSubscriptions.value.length === 0
  ),
)
const showOrders = computed(
  () => props.paymentEnabled && !props.simpleMode,
)

function emitNavigate(to: string) {
  emit('navigate', to)
}

function retrySubscriptions() {
  void subscriptionStore.fetchActiveSubscriptions(true).catch(() => {
    // The global store owns request state. Once loading settles, the bounded
    // inline error remains available for another explicit retry.
  })
}

function renewalPath(groupId: number) {
  return `/pricing?group=${groupId}`
}

function subscriptionName(subscription: UserSubscription) {
  return subscription.group?.name?.trim()
    || t('personalSettings.subscriptions.unnamed')
}

function formatDate(value: string) {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) {
    return t('personalSettings.common.unknownDate')
  }

  try {
    return new Intl.DateTimeFormat(locale.value, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    }).format(date)
  } catch {
    return date.toISOString().slice(0, 10)
  }
}

function subscriptionExpiry(value: string | null) {
  if (!value) return t('personalSettings.subscriptions.noExpiry')
  return t('personalSettings.subscriptions.expiresOn', {
    date: formatDate(value),
  })
}

function orderTypeLabel(order: PaymentOrder) {
  return order.order_type === 'subscription'
    ? t('personalSettings.orders.subscription')
    : t('personalSettings.orders.balance')
}

function formatOrderAmount(order: PaymentOrder) {
  return formatPaymentAmount(order.pay_amount, order.currency, locale.value)
}

async function loadOrders() {
  if (!showOrders.value || ordersLoading.value) return

  const generation = ++requestGeneration
  ordersLoading.value = true
  ordersError.value = ''

  try {
    const response = await paymentAPI.getMyOrders({
      page: 1,
      page_size: 3,
    })
    if (!mounted || generation !== requestGeneration) return
    orders.value = (response.data.items ?? []).slice(0, 3)
    ordersLoaded.value = true
  } catch {
    if (!mounted || generation !== requestGeneration) return
    orders.value = []
    ordersError.value = t('personalSettings.orders.loadError')
  } finally {
    if (mounted && generation === requestGeneration) {
      ordersLoading.value = false
    }
  }
}

onMounted(() => {
  mounted = true
  if (showOrders.value) {
    void loadOrders()
  }
})

watch(showOrders, (enabled) => {
  if (enabled && !ordersLoaded.value) {
    void loadOrders()
    return
  }
  if (!enabled && ordersLoading.value) {
    requestGeneration += 1
    ordersLoading.value = false
  }
})

onBeforeUnmount(() => {
  mounted = false
  requestGeneration += 1
})
</script>

<style scoped>
.wallet-settings {
  color: rgb(51 65 85);
}

.wallet-settings__section {
  padding: 12px 10px;
  border-top: 1px solid rgb(226 232 240 / 0.86);
}

.wallet-settings__section:first-child {
  border-top: 0;
}

.wallet-settings__section-heading {
  display: flex;
  min-height: 28px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.wallet-settings__section-title {
  color: rgb(51 65 85);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.25rem;
}

.wallet-settings__section-link,
.wallet-settings__row-link {
  border-radius: 6px;
  color: rgb(79 70 229);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-decoration: none;
}

.wallet-settings__section-link {
  padding: 4px 2px;
}

.wallet-settings__balance-list {
  margin-top: 4px;
}

.wallet-settings__balance-row {
  display: flex;
  min-height: 42px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.wallet-settings__balance-row dt {
  color: rgb(71 85 105);
  font-size: var(--workspace-type-body-size);
  font-weight: var(--workspace-type-body-weight);
}

.wallet-settings__balance-row dd {
  color: rgb(15 23 42);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.wallet-settings__balance-row--frozen dt,
.wallet-settings__balance-row--frozen dd {
  color: rgb(161 98 7);
}

.wallet-settings__actions {
  margin-top: 5px;
  border-top: 1px solid rgb(226 232 240 / 0.72);
}

.wallet-settings__action-row {
  display: grid;
  min-height: 42px;
  grid-template-columns: 18px minmax(0, 1fr) 14px;
  align-items: center;
  gap: 9px;
  border-radius: 9px;
  color: rgb(51 65 85);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  text-decoration: none;
  transition:
    color 150ms ease,
    background-color 150ms ease;
}

.wallet-settings__action-row:hover {
  color: rgb(15 23 42);
  background: rgb(15 23 42 / 0.045);
}

.wallet-settings__notice {
  margin-top: 7px;
  padding: 8px 9px;
  border-radius: 8px;
  color: rgb(120 53 15);
  background: rgb(255 247 237);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.45;
}

.wallet-settings__list {
  margin-top: 3px;
}

.wallet-settings__list-row {
  display: grid;
  min-height: 54px;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  border-top: 1px solid rgb(226 232 240 / 0.64);
}

.wallet-settings__list-row:first-child {
  border-top: 0;
}

.wallet-settings__row-icon {
  color: rgb(100 116 139);
}

.wallet-settings__row-copy {
  min-width: 0;
}

.wallet-settings__row-title {
  overflow: hidden;
  color: rgb(30 41 59);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
  line-height: 1.2rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wallet-settings__row-meta {
  overflow: hidden;
  margin-top: 1px;
  color: rgb(71 85 105);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wallet-settings__row-link {
  padding: 6px;
}

.wallet-settings__order-id {
  margin-left: 3px;
  color: rgb(100 116 139);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.wallet-settings__order-value {
  display: flex;
  min-width: 92px;
  flex-direction: column;
  align-items: flex-end;
  gap: 3px;
  color: rgb(15 23 42);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
}

.wallet-settings__loading {
  padding-top: 5px;
}

.wallet-settings__skeleton-row {
  display: flex;
  min-height: 52px;
  flex-direction: column;
  justify-content: center;
  gap: 7px;
  border-top: 1px solid rgb(226 232 240 / 0.6);
}

.wallet-settings__skeleton-row:first-child {
  border-top: 0;
}

.wallet-settings__skeleton-row span {
  display: block;
  width: 58%;
  height: 8px;
  border-radius: 4px;
  background: rgb(226 232 240);
  animation: wallet-settings-pulse 1.4s ease-in-out infinite;
}

.wallet-settings__skeleton-row span:last-child {
  width: 34%;
  height: 7px;
}

.wallet-settings__error,
.wallet-settings__empty {
  margin-top: 5px;
  padding: 12px 0 2px;
  color: rgb(71 85 105);
  font-size: var(--workspace-type-secondary-size);
  font-weight: var(--workspace-type-secondary-weight);
  line-height: 1.45;
}

.wallet-settings__error {
  display: flex;
  min-height: 50px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: rgb(153 27 27);
}

.wallet-settings__retry {
  display: inline-flex;
  min-height: 34px;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  padding: 6px 9px;
  border-radius: 8px;
  color: rgb(185 28 28);
  background: rgb(254 226 226 / 0.72);
  font-size: var(--workspace-type-navigation-size);
  font-weight: var(--workspace-type-navigation-weight);
}

.wallet-settings__empty--orders {
  display: flex;
  align-items: flex-start;
  gap: 9px;
}

.wallet-settings__empty--orders p:first-child {
  color: rgb(51 65 85);
  font-weight: var(--workspace-type-navigation-weight);
}

.wallet-settings__empty--orders p:last-child {
  margin-top: 2px;
}

.wallet-settings__action-row:focus-visible,
.wallet-settings__section-link:focus-visible,
.wallet-settings__row-link:focus-visible,
.wallet-settings__retry:focus-visible {
  outline: 2px solid var(--app-shell-sidebar-focus, rgb(0 132 255 / 0.5));
  outline-offset: 2px;
}

:global(html.dark .wallet-settings) {
  color: rgb(203 213 225);
}

:global(html.dark .wallet-settings__balance-row dd),
:global(html.dark .wallet-settings__order-value) {
  color: rgb(248 250 252);
}

:global(html.dark .wallet-settings__balance-row dt),
:global(html.dark .wallet-settings__row-meta),
:global(html.dark .wallet-settings__empty) {
  color: rgb(148 163 184);
}

:global(html.dark .wallet-settings__section),
:global(html.dark .wallet-settings__actions),
:global(html.dark .wallet-settings__list-row),
:global(html.dark .wallet-settings__skeleton-row) {
  border-color: rgb(255 255 255 / 0.09);
}

:global(html.dark .wallet-settings__section-title),
:global(html.dark .wallet-settings__action-row),
:global(html.dark .wallet-settings__row-title),
:global(html.dark .wallet-settings__empty--orders p:first-child) {
  color: rgb(226 232 240);
}

:global(html.dark .wallet-settings__action-row:hover) {
  color: #fff;
  background: rgb(255 255 255 / 0.07);
}

:global(html.dark .wallet-settings__section-link),
:global(html.dark .wallet-settings__row-link) {
  color: rgb(165 180 252);
}

:global(html.dark .wallet-settings__balance-row--frozen dt),
:global(html.dark .wallet-settings__balance-row--frozen dd) {
  color: rgb(252 211 77);
}

:global(html.dark .wallet-settings__notice) {
  color: rgb(253 230 138);
  background: rgb(120 53 15 / 0.24);
}

:global(html.dark .wallet-settings__skeleton-row span) {
  background: rgb(51 65 85);
}

:global(html.dark .wallet-settings__error) {
  color: rgb(254 202 202);
}

:global(html.dark .wallet-settings__retry) {
  color: rgb(254 202 202);
  background: rgb(127 29 29 / 0.34);
}

@keyframes wallet-settings-pulse {
  0%,
  100% {
    opacity: 0.52;
  }

  50% {
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .wallet-settings__action-row {
    transition-duration: 0.01ms;
  }

  .wallet-settings__skeleton-row span {
    animation: none;
  }
}
</style>
