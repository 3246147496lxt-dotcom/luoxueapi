<template>
  <AppLayout variant="purchase">
    <div class="mx-auto max-w-[1240px]">
      <section
        class="purchase-workbench overflow-hidden rounded-[18px] border"
        data-testid="purchase-main-card"
      >
        <header class="purchase-workbench__header flex flex-col justify-between gap-4 border-b px-6 py-6 sm:flex-row sm:items-center">
          <div class="flex min-w-0 items-center gap-4">
            <span class="purchase-workbench__icon flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl">
              <Icon name="creditCard" size="lg" />
            </span>
            <div class="min-w-0">
              <h1 class="purchase-workbench__title text-gray-950 dark:text-white">
                {{ t('payment.checkoutTitle') }}
              </h1>
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
                {{ t('payment.checkoutDescription') }}
              </p>
            </div>
          </div>

          <div class="min-w-0">
            <div
              class="purchase-account-strip flex min-w-0 items-center divide-x"
              data-testid="payment-account-strip"
              :aria-label="t('payment.rechargeAccount')"
            >
              <div class="purchase-account-strip__metric min-w-0 px-5 py-3">
                <p class="truncate text-[10px] font-bold uppercase text-gray-400 dark:text-gray-500">
                  {{ t('payment.currentBalance') }}
                </p>
                <p data-testid="payment-current-balance" class="mt-0.5 min-w-0 text-lg font-bold text-gray-950 dark:text-white">
                  <CreditAmount
                    class="purchase-account-strip__amount"
                    :value="user?.balance?.toFixed(2) || '0.00'"
                    icon-size="sm"
                    :label="`${t('payment.currentBalance')} ${user?.balance?.toFixed(2) || '0.00'}`"
                  />
                </p>
              </div>
              <div class="purchase-account-strip__metric min-w-0 px-5 py-3">
                <p class="truncate text-[10px] font-bold uppercase text-gray-400 dark:text-gray-500">
                  {{ t('dashboard.lifetimeSpend') }}
                </p>
                <p data-testid="payment-historical-spend" class="mt-0.5 min-w-0 text-lg font-bold text-gray-950 dark:text-white">
                  <CreditAmount
                    class="purchase-account-strip__amount"
                    :value="formattedHistoricalSpend"
                    icon-size="sm"
                    :label="`${t('dashboard.lifetimeSpend')} ${formattedHistoricalSpend}`"
                  />
                </p>
              </div>
            </div>
          </div>
        </header>

        <div v-if="paymentPhase === 'paying'" class="p-5 sm:p-6 lg:p-8">
          <PaymentStatusPanel
            :order-id="paymentState.orderId"
            :qr-code="paymentState.qrCode"
            :expires-at="paymentState.expiresAt"
            :payment-type="paymentState.paymentType"
            :pay-url="paymentState.payUrl"
            :order-type="paymentState.orderType"
            :currency="paymentState.currency || selectedCurrency"
            @done="onPaymentDone"
            @success="onPaymentSuccess"
            @settled="onPaymentSettled"
          />
        </div>

        <template v-else>
          <section v-if="paymentEnabled" aria-labelledby="purchase-options-title">
            <h2 id="purchase-options-title" class="sr-only">{{ t('payment.purchaseOptions') }}</h2>

            <div
              v-if="loading"
              class="space-y-4 p-5 sm:p-6 lg:p-8"
              aria-busy="true"
              data-testid="payment-loading-skeleton"
            >
              <div class="h-12 animate-pulse rounded-xl bg-gray-200/70 dark:bg-dark-700"></div>
              <div class="h-80 animate-pulse rounded-2xl bg-gray-200/70 dark:bg-dark-700"></div>
            </div>

            <template v-else>
              <section
                v-if="activeTab === 'recharge'"
                id="purchase-panel-recharge"
                :aria-label="t('payment.tabTopUp')"
              >
                <div v-if="checkout.balance_disabled || availableMethodTypes.length === 0" class="px-5 py-10 sm:px-6 lg:px-8">
                  <div class="py-4 text-center">
                    <Icon name="creditCard" size="xl" class="mx-auto text-gray-300 dark:text-gray-600" />
                    <p class="mt-3 text-sm text-gray-500 dark:text-gray-400">{{ t('payment.notAvailable') }}</p>
                  </div>
                  <PaymentHelpPanel
                    class="mt-8"
                    :text="checkout.help_text"
                    :image-url="checkout.help_image_url"
                  />
                  <div class="mx-auto mt-8 max-w-2xl space-y-5 border-t border-gray-100 pt-7 dark:border-dark-700">
                    <div class="flex items-center gap-4" aria-hidden="true">
                      <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                      <span class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                        {{ t('payment.alternativeDivider') }}
                      </span>
                      <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                    </div>
                    <RedeemCodePanel embedded />
                  </div>
                </div>

                <div
                  v-else
                  class="purchase-workbench__grid grid grid-cols-1 lg:grid-cols-12"
                  data-testid="payment-workbench"
                >
                  <div
                    class="min-w-0 space-y-10 p-6 lg:col-span-8 lg:p-8"
                    data-testid="payment-workbench-main"
                  >
                    <section aria-labelledby="payment-amount-title">
                      <AmountInput
                        v-model="amount"
                        :amounts="RECHARGE_PRESET_AMOUNTS"
                        :min="globalMinAmount"
                        :max="globalMaxAmount"
                        :currency="selectedCurrency"
                        :locale="localeCode"
                      />
                      <p v-if="amountError" role="alert" class="mt-3 text-sm text-amber-700 dark:text-amber-300">
                        {{ amountError }}
                      </p>
                    </section>

                    <PaymentMethodSelector
                      :methods="methodOptions"
                      :selected="selectedMethod"
                      @select="selectedMethod = $event"
                    />

                    <p v-if="balanceRechargeMultiplier !== 1" class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('payment.rechargeRatePreview', { credit: balanceRechargeMultiplier.toFixed(2) }) }}
                    </p>

                    <div
                      v-if="errorMessage"
                      role="alert"
                      class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-950/30 dark:text-red-300"
                    >
                      <p class="font-medium">{{ errorMessage }}</p>
                      <p v-if="errorHintMessage" class="mt-1 text-xs">{{ errorHintMessage }}</p>
                    </div>

                    <div class="pt-8">
                      <div class="mb-6 flex items-center gap-4" aria-hidden="true">
                        <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                        <span class="text-[10px] font-bold uppercase text-gray-300 dark:text-gray-500">
                          {{ t('payment.alternativeDivider') }}
                        </span>
                        <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                      </div>
                      <RedeemCodePanel embedded />
                    </div>

                    <PaymentHelpPanel
                      :text="checkout.help_text"
                      :image-url="checkout.help_image_url"
                    />
                  </div>

                  <aside
                    class="purchase-workbench__summary min-w-0 p-6 lg:col-span-4 lg:p-8"
                    data-testid="payment-workbench-summary"
                    aria-labelledby="payment-order-summary-title"
                  >
                    <div class="purchase-summary-panel">
                      <section data-testid="payment-order-summary">
                        <h3 id="payment-order-summary-title" class="text-sm font-bold uppercase text-gray-950 dark:text-white">
                          {{ t('payment.orderSummary') }}
                        </h3>
                        <dl class="mt-8 space-y-4">
                          <div class="flex items-center justify-between gap-4 text-sm">
                            <dt class="text-gray-500 dark:text-gray-400">{{ t('payment.amountLabel') }}</dt>
                            <dd class="min-w-0 break-all text-right font-bold tabular-nums text-gray-950 dark:text-white">
                              {{ formatSelectedPaymentAmount(validAmount) }}
                            </dd>
                          </div>
                          <div class="flex items-center justify-between gap-4 text-sm">
                            <dt class="text-gray-500 dark:text-gray-400">{{ t('payment.creditedBalance') }}</dt>
                            <dd data-testid="payment-credited-balance" class="min-w-0 break-all text-right font-bold text-gray-950 dark:text-white">
                              <CreditAmount
                                class="purchase-summary-credit"
                                :value="creditedAmount.toFixed(2)"
                                icon-size="xs"
                                :label="`${t('payment.creditedBalance')} ${creditedAmount.toFixed(2)}`"
                              />
                            </dd>
                          </div>
                          <div class="flex items-center justify-between gap-4 text-sm">
                            <dt class="text-gray-500 dark:text-gray-400">
                              {{ t('payment.fee') }}<span v-if="feeRate > 0"> · {{ feeRate }}%</span>
                            </dt>
                            <dd class="min-w-0 break-all text-right font-bold tabular-nums text-gray-950 dark:text-white">
                              {{ formatSelectedPaymentAmount(feeAmount) }}
                            </dd>
                          </div>
                          <div class="flex items-end justify-between gap-4 border-t border-gray-200 pt-4 dark:border-dark-700">
                            <dt class="purchase-summary-total text-sm font-bold">
                              {{ t('payment.actualPay') }}
                            </dt>
                            <dd class="purchase-summary-total min-w-0 break-all text-right leading-none tabular-nums">
                              {{ formatSelectedPaymentAmount(totalAmount) }}
                            </dd>
                          </div>
                        </dl>
                      </section>

                      <div class="mt-8" data-testid="payment-recharge-action-bar">
                        <button
                          class="purchase-payment-submit flex min-h-12 w-full min-w-0 items-center justify-center gap-2 px-3 text-base font-bold"
                          :disabled="!canSubmit || submitting"
                          @click="handleSubmitRecharge"
                        >
                          <span
                            v-if="submitting"
                            class="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-white/40 border-t-white"
                            aria-hidden="true"
                          ></span>
                          <span class="min-w-0 text-center leading-snug [overflow-wrap:anywhere]">
                            {{ submitting ? t('common.processing') : t('payment.createOrder') }}
                          </span>
                        </button>
                      </div>
                    </div>
                  </aside>
                </div>
              </section>

              <section
                v-else-if="activeTab === 'subscription' && selectedPlan && hasAvailableSubscriptionMethod"
                id="purchase-panel-subscription"
                :aria-label="t('payment.tabSubscribe')"
              >
                <div
                  class="purchase-workbench__grid grid grid-cols-1 lg:grid-cols-12"
                  data-testid="payment-workbench"
                >
                  <div
                    class="min-w-0 space-y-8 p-5 sm:p-6 lg:col-span-8 lg:p-8"
                    data-testid="payment-workbench-main"
                  >
                    <section aria-labelledby="selected-plan-title">
                      <div class="flex flex-wrap items-center gap-2">
                        <span :class="['rounded-md border px-2 py-0.5 text-xs font-medium', planBadgeClass]">
                          {{ platformLabel(selectedPlan.group_platform || '') }}
                        </span>
                        <h3 id="selected-plan-title" class="min-w-0 break-words text-lg font-bold text-gray-950 dark:text-white">
                          {{ selectedPlan.name }}
                        </h3>
                      </div>
                      <div class="mt-4 flex flex-wrap items-baseline gap-2">
                        <span v-if="selectedPlan.original_price" class="text-sm text-gray-400 line-through dark:text-gray-500">
                          {{ formatSelectedSubscriptionPaymentAmount(selectedPlan.original_price) }}
                        </span>
                        <span :class="['text-3xl font-bold tabular-nums', planTextClass]">
                          {{ formatSelectedSubscriptionPaymentAmount(selectedPlan.price) }}
                        </span>
                        <span class="text-sm text-gray-500 dark:text-gray-400">/ {{ planValiditySuffix }}</span>
                      </div>
                      <p v-if="selectedPlan.description" class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
                        {{ selectedPlan.description }}
                      </p>

                      <div class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-3">
                        <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60">
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.rate') }}</span>
                          <div :class="['mt-1 text-lg font-bold', planTextClass]">×{{ selectedPlan.rate_multiplier ?? 1 }}</div>
                        </div>
                        <div v-if="planHasPeakRate(selectedPlan)" class="rounded-xl bg-amber-50 p-3 dark:bg-amber-950/20">
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.peakRate') }}</span>
                          <div class="mt-1 text-sm font-semibold text-amber-700 dark:text-amber-300">
                            {{ planPeakRateLabel(selectedPlan) }}
                          </div>
                        </div>
                        <div v-if="selectedPlan.daily_limit_usd != null" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60">
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.dailyLimit') }}</span>
                          <div class="mt-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                            <CreditAmount :value="selectedPlan.daily_limit_usd" icon-size="sm" />
                          </div>
                        </div>
                        <div v-if="selectedPlan.weekly_limit_usd != null" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60">
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.weeklyLimit') }}</span>
                          <div class="mt-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                            <CreditAmount :value="selectedPlan.weekly_limit_usd" icon-size="sm" />
                          </div>
                        </div>
                        <div v-if="selectedPlan.monthly_limit_usd != null" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60">
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.monthlyLimit') }}</span>
                          <div class="mt-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                            <CreditAmount :value="selectedPlan.monthly_limit_usd" icon-size="sm" />
                          </div>
                        </div>
                        <div
                          v-if="selectedPlan.daily_limit_usd == null && selectedPlan.weekly_limit_usd == null && selectedPlan.monthly_limit_usd == null"
                          class="rounded-xl bg-gray-50 p-3 dark:bg-dark-900/60"
                        >
                          <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('payment.planCard.quota') }}</span>
                          <div class="mt-1 text-lg font-semibold text-gray-800 dark:text-gray-200">
                            {{ t('payment.planCard.unlimited') }}
                          </div>
                        </div>
                      </div>
                    </section>

                    <div v-if="enabledMethods.length" class="border-t border-gray-100 pt-8 dark:border-dark-700">
                      <PaymentMethodSelector
                        :methods="subMethodOptions"
                        :selected="selectedMethod"
                        @select="selectedMethod = $event"
                      />
                    </div>

                    <div
                      v-if="errorMessage"
                      role="alert"
                      class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800/60 dark:bg-red-950/30 dark:text-red-300"
                    >
                      <p class="font-medium">{{ errorMessage }}</p>
                      <p v-if="errorHintMessage" class="mt-1 text-xs">{{ errorHintMessage }}</p>
                    </div>

                    <PaymentHelpPanel
                      :text="checkout.help_text"
                      :image-url="checkout.help_image_url"
                    />
                  </div>

                  <aside
                    class="purchase-workbench__summary min-w-0 p-5 sm:p-6 lg:col-span-4 lg:p-8"
                    data-testid="payment-workbench-summary"
                    aria-labelledby="payment-subscription-summary-title"
                  >
                    <div class="purchase-summary-panel">
                      <section data-testid="payment-subscription-summary">
                        <h3 id="payment-subscription-summary-title" class="text-sm font-bold text-gray-950 dark:text-white">
                          {{ t('payment.orderSummary') }}
                        </h3>
                        <dl class="mt-6 space-y-4">
                          <div class="flex items-center justify-between gap-4 text-sm">
                            <dt class="text-gray-500 dark:text-gray-400">{{ t('payment.amountLabel') }}</dt>
                            <dd class="min-w-0 break-all text-right font-semibold tabular-nums text-gray-950 dark:text-white">
                              {{ formatSelectedPaymentAmount(subPaymentAmount) }}
                            </dd>
                          </div>
                          <div class="flex items-center justify-between gap-4 text-sm">
                            <dt class="text-gray-500 dark:text-gray-400">
                              {{ t('payment.fee') }}<span v-if="feeRate > 0"> · {{ feeRate }}%</span>
                            </dt>
                            <dd class="min-w-0 break-all text-right font-semibold tabular-nums text-gray-950 dark:text-white">
                              {{ formatSelectedPaymentAmount(subFeeAmount) }}
                            </dd>
                          </div>
                          <div class="flex items-end justify-between gap-4 border-t border-gray-200 pt-4 dark:border-dark-700">
                            <dt class="purchase-summary-total text-sm font-semibold">
                              {{ t('payment.actualPay') }}
                            </dt>
                            <dd class="purchase-summary-total min-w-0 break-all text-right leading-none tabular-nums">
                              {{ formatSelectedPaymentAmount(subTotalAmount) }}
                            </dd>
                          </div>
                        </dl>
                      </section>

                      <div
                        class="payment-mobile-action mt-8 space-y-3"
                        data-testid="payment-subscription-action-bar"
                      >
                        <button
                          class="btn btn-primary min-h-12 w-full min-w-0 justify-center px-3 text-sm font-semibold sm:text-base"
                          :disabled="!canSubmitSubscription || submitting"
                          @click="confirmSubscribe"
                        >
                          <span
                            v-if="submitting"
                            class="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-white/40 border-t-white"
                            aria-hidden="true"
                          ></span>
                          <span class="min-w-0 text-center leading-snug [overflow-wrap:anywhere]">
                            {{ submitting ? t('common.processing') : `${t('payment.createOrder')} · ${formatSelectedPaymentAmount(subTotalAmount)}` }}
                          </span>
                        </button>
                        <button
                          class="btn btn-secondary min-h-11 w-full justify-center"
                          data-testid="payment-subscription-cancel"
                          @click="cancelSubscriptionCheckout"
                        >
                          {{ t('common.cancel') }}
                        </button>
                      </div>
                    </div>
                  </aside>
                </div>
              </section>

              <div
                v-else-if="activeTab === 'subscription' && selectedPlan"
                class="px-5 py-10 sm:px-6 lg:px-8"
                data-testid="payment-subscription-unavailable"
              >
                <div class="py-4 text-center">
                  <Icon name="creditCard" size="xl" class="mx-auto text-gray-300 dark:text-gray-600" />
                  <h3 class="mt-4 break-words text-base font-semibold text-gray-950 dark:text-white">
                    {{ selectedPlan.name }}
                  </h3>
                  <p role="status" class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('payment.notAvailable') }}
                  </p>
                </div>
                <PaymentHelpPanel
                  class="mx-auto mt-8 max-w-2xl"
                  :text="checkout.help_text"
                  :image-url="checkout.help_image_url"
                />
                <div class="mx-auto mt-8 max-w-2xl space-y-5 border-t border-gray-100 pt-7 text-left dark:border-dark-700">
                  <div class="flex items-center gap-4" aria-hidden="true">
                    <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                    <span class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                      {{ t('payment.alternativeDivider') }}
                    </span>
                    <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                  </div>
                  <RedeemCodePanel embedded />
                </div>
              </div>

              <div
                v-else-if="activeTab === 'subscription'"
                class="px-5 py-10 sm:px-6 lg:px-8"
                data-testid="payment-plan-unavailable"
                role="alert"
              >
                <div class="py-4 text-center">
                  <Icon name="creditCard" size="xl" class="mx-auto text-gray-300 dark:text-gray-600" />
                  <h3 class="mt-4 text-base font-semibold text-gray-950 dark:text-white">
                    {{ t('pricing.planUnavailableTitle') }}
                  </h3>
                  <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                    {{ t('pricing.planUnavailableDescription') }}
                  </p>
                  <RouterLink to="/pricing" class="btn btn-primary mt-5">
                    {{ t('pricing.returnToPlans') }}
                  </RouterLink>
                </div>
                <PaymentHelpPanel
                  class="mx-auto mt-8 max-w-2xl"
                  :text="checkout.help_text"
                  :image-url="checkout.help_image_url"
                />
                <div class="mx-auto mt-8 max-w-2xl space-y-5 border-t border-gray-100 pt-7 text-left dark:border-dark-700">
                  <div class="flex items-center gap-4" aria-hidden="true">
                    <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                    <span class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                      {{ t('payment.alternativeDivider') }}
                    </span>
                    <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                  </div>
                  <RedeemCodePanel embedded />
                </div>
              </div>
            </template>
          </section>

          <section v-else class="p-5 sm:p-6 lg:p-8">
            <div class="py-8 text-center" data-testid="payment-disabled-notice">
              <span class="mx-auto flex h-11 w-11 items-center justify-center rounded-xl bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                <Icon name="creditCard" size="md" />
              </span>
              <h2 class="mt-3 text-base font-semibold text-gray-950 dark:text-white">
                {{ t('purchase.notEnabledTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('purchase.notEnabledDesc') }}</p>
            </div>
            <PaymentHelpPanel
              :text="checkout.help_text"
              :image-url="checkout.help_image_url"
            />
            <div class="space-y-5 border-t border-gray-100 pt-7 dark:border-dark-700">
              <div class="flex items-center gap-4" aria-hidden="true">
                <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
                <span class="text-xs font-semibold text-gray-400 dark:text-gray-500">
                  {{ t('payment.alternativeDivider') }}
                </span>
                <span class="h-px flex-1 bg-gray-100 dark:bg-dark-700"></span>
              </div>
              <RedeemCodePanel embedded />
            </div>
          </section>
        </template>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { usePaymentStore } from '@/stores/payment'
import { useUserProfileStore } from '@/stores/userProfile'
import { useAppStore } from '@/stores'
import { usageAPI } from '@/api/usage'
import { extractApiErrorMessage, extractI18nErrorMessage } from '@/utils/apiError'
import { isMobileDevice } from '@/utils/device'
import { formatCostFixed } from '@/utils/format'
import { hasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import type { SubscriptionPlan, CheckoutInfoResponse, CreateOrderResult, OrderType } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import { METHOD_ORDER, getPaymentPopupFeatures } from '@/components/payment/providerConfig'
import {
  PAYMENT_RECOVERY_STORAGE_KEY,
  buildCreateOrderPayload,
  clearPaymentRecoverySnapshot,
  decidePaymentLaunch,
  getVisibleMethods,
  normalizeVisibleMethod,
  readPaymentRecoverySnapshot,
  type PaymentRecoverySnapshot,
  writePaymentRecoverySnapshot,
} from '@/components/payment/paymentFlow'
import { platformBadgeClass, platformTextClass, platformLabel } from '@/utils/platformColors'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import RedeemCodePanel from '@/components/payment/RedeemCodePanel.vue'
import PaymentHelpPanel from '@/components/payment/PaymentHelpPanel.vue'
import Icon from '@/components/icons/Icon.vue'
import { DEFAULT_PAYMENT_CURRENCY, formatPaymentAmount, normalizePaymentCurrency } from '@/components/payment/currency'
import type { PaymentMethodOption } from '@/components/payment/PaymentMethodSelector.vue'
import { buildPaymentErrorToastMessage, describePaymentScenarioError } from './paymentUx'
import { hasWechatResumeQuery, parseWechatResumeRoute, stripWechatResumeQuery } from './paymentWechatResume'
import {
  hasCompleteWechatResumeQuery,
  readSingleQueryString,
} from '@/navigation/purchaseQueryState'

const i18n = useI18n()
const { t } = i18n
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const paymentStore = usePaymentStore()
const userProfileStore = useUserProfileStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)
const paymentEnabled = computed(() => appStore.cachedPublicSettings?.payment_enabled !== false)

const loading = ref(true)
const historicalSpend = ref(0)
const formattedHistoricalSpend = computed(() => formatCostFixed(historicalSpend.value))
const submitting = ref(false)
const errorMessage = ref('')
const errorHintMessage = ref('')
const activeTab = ref<'recharge' | 'subscription'>('recharge')
const RECHARGE_PRESET_AMOUNTS = [10, 20, 50, 100, 200, 500, 1000, 2000]
const DEFAULT_RECHARGE_AMOUNT = 50
const amount = ref<number | null>(null)
const selectedMethod = ref('')
const selectedPlan = ref<SubscriptionPlan | null>(null)

const paymentPhase = ref<'select' | 'paying'>('select')

interface CreateOrderOptions {
  openid?: string
  wechatResumeToken?: string
  paymentType?: string
  isResume?: boolean
  mobileQrFallbackAttempted?: boolean
}

interface WeixinJSBridgeLike {
  invoke(
    action: string,
    payload: Record<string, unknown>,
    callback: (result: Record<string, unknown>) => void,
  ): void
}

function emptyPaymentState(): PaymentRecoverySnapshot {
  return {
    orderId: 0,
    amount: 0,
    qrCode: '',
    expiresAt: '',
    paymentType: '',
    payUrl: '',
    outTradeNo: '',
    clientSecret: '',
    intentId: '',
    currency: '',
    countryCode: '',
    paymentEnv: '',
    payAmount: 0,
    orderType: '',
    paymentMode: '',
    resumeToken: '',
    createdAt: 0,
  }
}

function getWeixinJSBridge(): WeixinJSBridgeLike | undefined {
  return (window as Window & { WeixinJSBridge?: WeixinJSBridgeLike }).WeixinJSBridge
}

function waitForWeixinJSBridge(timeoutMs = 4000): Promise<WeixinJSBridgeLike | null> {
  const existing = getWeixinJSBridge()
  if (existing) return Promise.resolve(existing)

  return new Promise((resolve) => {
    let settled = false
    const finish = (bridge: WeixinJSBridgeLike | null) => {
      if (settled) return
      settled = true
      document.removeEventListener('WeixinJSBridgeReady', handleReady)
      document.removeEventListener('onWeixinJSBridgeReady', handleReady)
      window.clearTimeout(timer)
      resolve(bridge)
    }
    const handleReady = () => finish(getWeixinJSBridge() ?? null)
    const timer = window.setTimeout(() => finish(getWeixinJSBridge() ?? null), timeoutMs)
    document.addEventListener('WeixinJSBridgeReady', handleReady, false)
    document.addEventListener('onWeixinJSBridgeReady', handleReady, false)
  })
}

async function invokeWechatJsapiPayment(payload: Record<string, unknown>): Promise<Record<string, unknown>> {
  const bridge = await waitForWeixinJSBridge()
  if (!bridge) {
    throw new Error('WECHAT_JSAPI_UNAVAILABLE')
  }
  return new Promise((resolve) => {
    bridge.invoke('getBrandWCPayRequest', payload, (result) => resolve(result || {}))
  })
}

const paymentState = ref<PaymentRecoverySnapshot>(emptyPaymentState())

function persistRecoverySnapshot(snapshot: PaymentRecoverySnapshot) {
  if (typeof window === 'undefined' || !snapshot.orderId) return
  writePaymentRecoverySnapshot(window.localStorage, snapshot, PAYMENT_RECOVERY_STORAGE_KEY)
}

function removeRecoverySnapshot() {
  if (typeof window === 'undefined') return
  clearPaymentRecoverySnapshot(window.localStorage, PAYMENT_RECOVERY_STORAGE_KEY)
}

function resetPayment() {
  paymentPhase.value = 'select'
  paymentState.value = emptyPaymentState()
  removeRecoverySnapshot()
}

async function redirectToPaymentResult(state: PaymentRecoverySnapshot): Promise<void> {
  const query: Record<string, string | undefined> = {}
  if (state.orderId > 0) {
    query.order_id = String(state.orderId)
  }
  if (state.outTradeNo) {
    query.out_trade_no = state.outTradeNo
  }
  if (state.resumeToken) {
    query.resume_token = state.resumeToken
  }
  await router.push({
    path: '/payment/result',
    query,
  })
}

function buildWechatOAuthAuthorizeUrl(
  authorizeUrl: string,
  context: { paymentType: string; orderType: OrderType; planId?: number; orderAmount: number },
): string {
  const normalizedUrl = authorizeUrl.trim()
  if (!normalizedUrl || typeof window === 'undefined') {
    return normalizedUrl
  }

  try {
    const targetUrl = new URL(normalizedUrl, window.location.origin)
    const redirectPath = targetUrl.searchParams.get('redirect') || '/purchase'
    const redirectUrl = new URL(redirectPath, window.location.origin)
    const paymentType = normalizeVisibleMethod(context.paymentType) || context.paymentType.trim() || 'wxpay'

    redirectUrl.searchParams.set('payment_type', paymentType)
    redirectUrl.searchParams.set('order_type', context.orderType)

    if (context.planId) {
      redirectUrl.searchParams.set('plan_id', String(context.planId))
    } else {
      redirectUrl.searchParams.delete('plan_id')
    }

    if (context.orderAmount > 0) {
      redirectUrl.searchParams.set('amount', String(context.orderAmount))
    } else {
      redirectUrl.searchParams.delete('amount')
    }

    targetUrl.searchParams.set('redirect', `${redirectUrl.pathname}${redirectUrl.search}`)
    return targetUrl.toString()
  } catch {
    return normalizedUrl
  }
}

function onPaymentDone() {
  resetPayment()
  selectedPlan.value = null
  syncSubscriptionCheckoutFromRoute()
}

async function onPaymentSuccess() {
  removeRecoverySnapshot()

  try {
    if (paymentState.value.orderType === 'subscription') {
      await userProfileStore.syncAfterUpgrade()
    } else {
      await userProfileStore.refreshProfile()
    }
  } catch (error) {
    console.error('Failed to sync user profile after payment:', error)
  }
}

function onPaymentSettled() {
  removeRecoverySnapshot()
}

// All checkout data from single API call
const checkout = ref<CheckoutInfoResponse>({
  methods: {}, global_min: 0, global_max: 0,
  plans: [], balance_disabled: false, balance_recharge_multiplier: 1, subscription_usd_to_cny_rate: 0, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '',
})

const visibleMethods = computed(() => getVisibleMethods(checkout.value.methods))
const enabledMethods = computed(() => Object.keys(visibleMethods.value))
const availableMethodTypes = computed(() => {
  const order: readonly string[] = METHOD_ORDER
  return enabledMethods.value
    .filter(type => visibleMethods.value[type]?.available !== false)
    .sort((a, b) => {
      const ai = order.indexOf(a)
      const bi = order.indexOf(b)
      return (ai === -1 ? 999 : ai) - (bi === -1 ? 999 : bi)
    })
})
const validAmount = computed(() => amount.value ?? 0)
const balanceRechargeMultiplier = computed(() => {
  const multiplier = checkout.value.balance_recharge_multiplier
  return Number.isFinite(multiplier) && multiplier > 0 ? multiplier : 1
})
// 订阅 CNY 换算汇率（1 USD = X CNY）。0 = 未配置，订阅保持 price 直付（与后端 opt-in 条件严格镜像）。
const subscriptionUsdToCnyRate = computed(() => {
  const rate = checkout.value.subscription_usd_to_cny_rate
  return Number.isFinite(rate) && rate > 0 ? rate : 0
})
const creditedAmount = computed(() => Math.round((validAmount.value * balanceRechargeMultiplier.value) * 100) / 100)

// Check if an amount fits a method's [min, max]. 0 = no limit.
function amountFitsMethod(amt: number, methodType: string): boolean {
  if (amt <= 0) return true
  const ml = visibleMethods.value[methodType]
  if (!ml) return false
  if (ml.single_min > 0 && amt < ml.single_min) return false
  if (ml.single_max > 0 && amt > ml.single_max) return false
  return true
}

function initializeRechargeAmount() {
  if (amount.value !== null || checkout.value.balance_disabled) return

  const payablePresets = RECHARGE_PRESET_AMOUNTS.filter(candidate =>
    availableMethodTypes.value.some(type => amountFitsMethod(candidate, type)),
  )
  amount.value = payablePresets.includes(DEFAULT_RECHARGE_AMOUNT)
    ? DEFAULT_RECHARGE_AMOUNT
    : payablePresets[0] ?? null
}

// Visible methods decide the amount range shown to users.
const globalMinAmount = computed(() => {
  const limits = Object.values(visibleMethods.value).filter(limit => limit.available !== false)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_min <= 0)) return 0
  return Math.min(...limits.map(limit => limit.single_min))
})
const globalMaxAmount = computed(() => {
  const limits = Object.values(visibleMethods.value).filter(limit => limit.available !== false)
  if (limits.length === 0) return 0
  if (limits.some(limit => limit.single_max <= 0)) return 0
  return Math.max(...limits.map(limit => limit.single_max))
})

// Selected method's limits (for validation and error messages)
const selectedLimit = computed(() => visibleMethods.value[selectedMethod.value])
const selectedCurrency = computed(() => normalizePaymentCurrency(selectedLimit.value?.currency))
const localeCode = computed(() => {
  const raw = i18n.locale as unknown
  if (typeof raw === 'string') return raw
  if (raw && typeof raw === 'object' && 'value' in raw) {
    return String((raw as { value?: string }).value || '')
  }
  return undefined
})

function currencyFractionDigits(currency: string): number {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency,
    }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function roundPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.round(value * factor) / factor
}

function ceilPaymentAmount(value: number, currency: string): number {
  if (!Number.isFinite(value)) return 0
  const factor = 10 ** currencyFractionDigits(currency)
  return Math.ceil(value * factor) / factor
}

function subscriptionPaymentAmountForCurrency(value: number, currency: string): number {
  const rate = subscriptionUsdToCnyRate.value
  if (rate <= 0 || currency !== DEFAULT_PAYMENT_CURRENCY) return roundPaymentAmount(value, currency)
  return roundPaymentAmount(value * rate, currency)
}

function formatSelectedPaymentAmount(value: number): string {
  return formatPaymentAmount(value, selectedCurrency.value, localeCode.value)
}

function formatSelectedSubscriptionPaymentAmount(value: number): string {
  return formatSelectedPaymentAmount(subscriptionPaymentAmountForCurrency(value, selectedCurrency.value))
}

const methodOptions = computed<PaymentMethodOption[]>(() =>
  enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(validAmount.value, type),
    }
  })
)

const feeRate = computed(() => checkout.value?.recharge_fee_rate ?? 0)
const feeAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.ceil(((validAmount.value * feeRate.value) / 100) * 100) / 100
    : 0
)
const totalAmount = computed(() =>
  feeRate.value > 0 && validAmount.value > 0
    ? Math.round((validAmount.value + feeAmount.value) * 100) / 100
    : validAmount.value
)

const amountError = computed(() => {
  if (validAmount.value <= 0) return ''
  // No method can handle this amount
  if (!availableMethodTypes.value.some((m) => amountFitsMethod(validAmount.value, m))) {
    return t('payment.amountNoMethod')
  }
  // Selected method can't handle this amount (but others can)
  const ml = selectedLimit.value
  if (ml) {
    if (ml.single_min > 0 && validAmount.value < ml.single_min) return t('payment.amountTooLow', { min: formatSelectedPaymentAmount(ml.single_min) })
    if (ml.single_max > 0 && validAmount.value > ml.single_max) return t('payment.amountTooHigh', { max: formatSelectedPaymentAmount(ml.single_max) })
  }
  return ''
})

const canSubmit = computed(() =>
  validAmount.value > 0
    && amountFitsMethod(validAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

const subPaymentAmount = computed(() => {
  const price = selectedPlan.value?.price ?? 0
  return subscriptionPaymentAmountForCurrency(price, selectedCurrency.value)
})

const subFeeAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return 0
  return ceilPaymentAmount((subPaymentAmount.value * feeRate.value) / 100, selectedCurrency.value)
})

const subTotalAmount = computed(() => {
  if (feeRate.value <= 0 || subPaymentAmount.value <= 0) return subPaymentAmount.value
  return roundPaymentAmount(subPaymentAmount.value + subFeeAmount.value, selectedCurrency.value)
})

function subscriptionTotalAmountForCurrency(value: number, currency: string): number {
  const paymentAmount = subscriptionPaymentAmountForCurrency(value, currency)
  if (feeRate.value <= 0 || paymentAmount <= 0) return paymentAmount
  const fee = ceilPaymentAmount((paymentAmount * feeRate.value) / 100, currency)
  return roundPaymentAmount(paymentAmount + fee, currency)
}

// Subscription-specific: method options based on gateway pay amount
const subMethodOptions = computed<PaymentMethodOption[]>(() => {
  const price = selectedPlan.value?.price ?? 0
  return enabledMethods.value.map((type) => {
    const ml = visibleMethods.value[type]
    const currency = normalizePaymentCurrency(ml?.currency)
    return {
      type,
      display_name: ml?.display_name,
      fee_rate: ml?.fee_rate ?? 0,
      available: ml?.available !== false && amountFitsMethod(subscriptionTotalAmountForCurrency(price, currency), type),
    }
  })
})

const availableSubscriptionMethodTypes = computed(() =>
  availableMethodTypes.value.filter(type =>
    subMethodOptions.value.some(method => method.type === type && method.available),
  ),
)
const hasAvailableSubscriptionMethod = computed(() => availableSubscriptionMethodTypes.value.length > 0)

const canSubmitSubscription = computed(() =>
  selectedPlan.value !== null
    && amountFitsMethod(subTotalAmount.value, selectedMethod.value)
    && selectedLimit.value?.available !== false
)

// Keep the selected channel payable as route, amount, and provider availability change.
watch(
  [activeTab, validAmount, selectedMethod, availableMethodTypes],
  ([tab, amt, method, available]) => {
    if (tab !== 'recharge') return
    if (available.includes(method) && amountFitsMethod(amt, method)) return
    selectedMethod.value = available.find(type => amountFitsMethod(amt, type)) ?? ''
  },
  { immediate: true },
)

watch(
  [activeTab, selectedPlan, selectedMethod, availableSubscriptionMethodTypes],
  ([tab, plan, method, available]) => {
    if (tab !== 'subscription' || !plan) return
    if (available.includes(method)) return
    selectedMethod.value = available[0] ?? ''
  },
)

// Subscription confirm: platform accent colors (clean card, no gradient)
const planBadgeClass = computed(() => platformBadgeClass(selectedPlan.value?.group_platform || ''))
const planTextClass = computed(() => platformTextClass(selectedPlan.value?.group_platform || ''))

const planValiditySuffix = computed(() => {
  if (!selectedPlan.value) return ''
  const u = selectedPlan.value.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${selectedPlan.value.validity_days}${t('payment.days')}`
})

function planHasPeakRate(plan: SubscriptionPlan): boolean {
  return hasPeakRate(plan)
}

function planPeakRateLabel(plan: SubscriptionPlan): string {
  return formatPeakRateWindow(plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

async function handleSubmitRecharge() {
  if (!canSubmit.value || submitting.value) return
  await createOrder(validAmount.value, 'balance')
}

async function confirmSubscribe() {
  if (!selectedPlan.value || submitting.value) return
  await createOrder(selectedPlan.value.price, 'subscription', selectedPlan.value.id)
}

function cancelSubscriptionCheckout() {
  const historyBack = typeof window !== 'undefined'
    ? window.history.state?.back
    : null

  if (typeof historyBack === 'string') {
    try {
      const backUrl = new URL(historyBack, window.location.origin)
      if (
        backUrl.origin === window.location.origin
        && backUrl.pathname === '/pricing'
      ) {
        router.back()
        return
      }
    } catch {
      // Invalid history state falls through to the safe catalogue fallback.
    }
  }

  void router.replace('/pricing')
}

async function createOrder(orderAmount: number, orderType: OrderType, planId?: number, options: CreateOrderOptions = {}) {
  submitting.value = true
  errorMessage.value = ''
  errorHintMessage.value = ''
  const requestType = normalizeVisibleMethod(options.paymentType || selectedMethod.value) || options.paymentType || selectedMethod.value
  try {
    const payload = buildCreateOrderPayload({
      amount: orderAmount,
      paymentType: requestType,
      orderType,
      planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && normalizeVisibleMethod(requestType) === 'alipay'),
    })
    if (options.openid) {
      payload.openid = options.openid
    }
    if (options.wechatResumeToken) {
      payload.wechat_resume_token = options.wechatResumeToken
    }

    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const openWindow = (url: string) => {
      const win = window.open(url, 'paymentPopup', getPaymentPopupFeatures())
      if (!win || win.closed) {
        window.location.href = url
      }
    }
    const visibleMethod = normalizeVisibleMethod(requestType) || requestType
    // When user clicks the dedicated Stripe button, leave method blank so the
    // landing page renders Stripe's full Payment Element (card/link/alipay/wxpay).
    const stripeMethod = visibleMethod === 'stripe'
      ? ''
      : visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret && visibleMethod !== 'airwallex'
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const airwallexRouteUrl = result.client_secret && result.intent_id
      ? router.resolve({
        path: '/payment/airwallex',
        query: {
          order_id: String(result.order_id),
          out_trade_no: result.out_trade_no || undefined,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType,
      isMobile: isMobileDevice(),
      isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
      forceQRCode: !!(checkout.value.alipay_force_qrcode && visibleMethod === 'alipay'),
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
      airwallexRouteUrl,
    })

    if (decision.kind === 'wechat_oauth' && decision.oauth?.authorize_url) {
      window.location.href = buildWechatOAuthAuthorizeUrl(decision.oauth.authorize_url, {
        paymentType: visibleMethod,
        orderType,
        planId,
        orderAmount,
      })
      return
    }

    if (decision.kind === 'unhandled') {
      applyScenarioError({ reason: 'UNHANDLED_PAYMENT_SCENARIO' }, visibleMethod)
      return
    }

    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)

    if (decision.kind === 'stripe_popup') {
      openWindow(decision.paymentState.payUrl)
      return
    }
    if (decision.kind === 'stripe_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'airwallex_route') {
      window.location.href = decision.paymentState.payUrl
      return
    }
    if (decision.kind === 'wechat_jsapi' && decision.jsapi) {
      try {
        const jsapiResult = await invokeWechatJsapiPayment(decision.jsapi as Record<string, unknown>)
        const errMsg = String(jsapiResult.err_msg || '').toLowerCase()
        if (errMsg.includes('cancel')) {
          appStore.showInfo(t('payment.qr.cancelled'))
          resetPayment()
        } else if (errMsg && !errMsg.includes('ok')) {
          resetPayment()
          const fallbackApplied = await attemptMobileQrFallback(
            { reason: 'WECHAT_JSAPI_FAILED', message: errMsg },
            {
              orderAmount,
              orderType,
              planId,
              paymentType: visibleMethod,
              attempted: options.mobileQrFallbackAttempted === true,
            },
          )
          if (!fallbackApplied) {
            applyScenarioError({ reason: 'WECHAT_JSAPI_FAILED', message: errMsg }, visibleMethod)
          }
        } else {
          const resultState = { ...decision.paymentState }
          resetPayment()
          await redirectToPaymentResult(resultState)
        }
      } catch (err: unknown) {
        resetPayment()
        const fallbackApplied = await attemptMobileQrFallback(err, {
          orderAmount,
          orderType,
          planId,
          paymentType: visibleMethod,
          attempted: options.mobileQrFallbackAttempted === true,
        })
        if (!fallbackApplied) {
          throw err
        }
      }
      return
    }
    if (decision.kind === 'redirect_waiting' && decision.paymentState.payUrl) {
      if (isMobileDevice()) {
        window.location.href = decision.paymentState.payUrl
        return
      }
      openWindow(decision.paymentState.payUrl)
    }
  } catch (err: unknown) {
    const apiErr = err as Record<string, unknown>
    if (apiErr.reason === 'TOO_MANY_PENDING') {
      const metadata = apiErr.metadata as Record<string, unknown> | undefined
      errorMessage.value = t('payment.errors.tooManyPending', { max: metadata?.max || '' })
      errorHintMessage.value = ''
    } else if (apiErr.reason === 'CANCEL_RATE_LIMITED') {
      errorMessage.value = t('payment.errors.cancelRateLimited')
      errorHintMessage.value = ''
    } else if (await attemptMobileQrFallback(err, {
      orderAmount,
      orderType,
      planId,
      paymentType: requestType,
      attempted: options.mobileQrFallbackAttempted === true,
    })) {
      return
    } else {
      const handled = applyScenarioError(
        err,
        normalizeVisibleMethod(options.paymentType || selectedMethod.value) || selectedMethod.value,
      )
      if (!handled) {
        errorMessage.value = extractI18nErrorMessage(err, t, 'payment.errors', extractApiErrorMessage(err, t('payment.result.failed')))
        errorHintMessage.value = ''
      }
      if (handled) {
        return
      }
    }
    appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  } finally {
    submitting.value = false
  }
}

interface MobileQrFallbackContext {
  orderAmount: number
  orderType: OrderType
  planId?: number
  paymentType: string
  attempted: boolean
}

function shouldFallbackToDesktopQr(err: unknown, paymentMethod: string, attempted: boolean): boolean {
  if (attempted || !isMobileDevice()) {
    return false
  }

  const normalizedMethod = normalizeVisibleMethod(paymentMethod) || paymentMethod
  const reason = typeof err === 'object' && err && 'reason' in err && typeof err.reason === 'string'
    ? err.reason
    : ''
  const message = err instanceof Error
    ? err.message
    : (typeof err === 'object' && err && 'message' in err && typeof err.message === 'string'
      ? err.message
      : '')
  const normalizedMessage = message.toLowerCase()

  if (normalizedMethod === 'wxpay') {
    return reason === 'WECHAT_H5_NOT_AUTHORIZED'
      || reason === 'WECHAT_PAYMENT_MP_NOT_CONFIGURED'
      || reason === 'WECHAT_JSAPI_FAILED'
      || reason === 'PAYMENT_GATEWAY_ERROR'
      || reason === 'UNHANDLED_PAYMENT_SCENARIO'
      || normalizedMessage.includes('weixinjsbridge is unavailable')
      || normalizedMessage.includes('wechat_jsapi_unavailable')
  }

  if (normalizedMethod === 'alipay') {
    return reason === 'PAYMENT_GATEWAY_ERROR' || reason === 'UNHANDLED_PAYMENT_SCENARIO'
  }

  return false
}

async function attemptMobileQrFallback(err: unknown, context: MobileQrFallbackContext): Promise<boolean> {
  if (!shouldFallbackToDesktopQr(err, context.paymentType, context.attempted)) {
    return false
  }

  try {
    const visibleMethod = normalizeVisibleMethod(context.paymentType) || context.paymentType
    const payload = buildCreateOrderPayload({
      amount: context.orderAmount,
      paymentType: visibleMethod,
      orderType: context.orderType,
      planId: context.planId,
      origin: typeof window !== 'undefined' ? window.location.origin : '',
      isMobile: false,
      isWechatBrowser: false,
    })
    const result = await paymentStore.createOrder(payload) as CreateOrderResult & { resume_token?: string }
    const stripeMethod = visibleMethod === 'wxpay' ? 'wechat_pay' : 'alipay'
    const stripeRouteUrl = result.client_secret
      ? router.resolve({
        path: '/payment/stripe',
        query: {
          order_id: String(result.order_id),
          client_secret: result.client_secret,
          method: stripeMethod,
          resume_token: result.resume_token || undefined,
        },
      }).href
      : ''
    const decision = decidePaymentLaunch(result, {
      visibleMethod,
      orderType: context.orderType,
      isMobile: false,
      isWechatBrowser: false,
      stripePopupUrl: stripeRouteUrl,
      stripeRouteUrl,
    })

    if (decision.kind !== 'qr_waiting' || !decision.paymentState.qrCode) {
      return false
    }

    errorMessage.value = ''
    errorHintMessage.value = ''
    paymentState.value = decision.paymentState
    paymentPhase.value = 'paying'
    persistRecoverySnapshot(decision.recovery)
    appStore.showWarning(t('payment.errors.mobilePaymentFallbackToQr'))
    return true
  } catch {
    return false
  }
}

function applyScenarioError(err: unknown, paymentMethod: string): boolean {
  const descriptor = describePaymentScenarioError(err, {
    paymentMethod,
    isMobile: isMobileDevice(),
    isWechatBrowser: typeof window !== 'undefined' && /MicroMessenger/i.test(window.navigator.userAgent),
  })
  if (!descriptor) {
    errorMessage.value = ''
    errorHintMessage.value = ''
    return false
  }
  errorMessage.value = t(descriptor.messageKey)
  errorHintMessage.value = descriptor.hintKey ? t(descriptor.hintKey) : ''
  appStore.showError(buildPaymentErrorToastMessage(errorMessage.value, errorHintMessage.value))
  return true
}

async function resumeWechatPaymentFromQuery() {
  const resume = parseWechatResumeRoute(route.query, checkout.value.plans, validAmount.value)
  if (!resume) {
    return
  }

  selectedMethod.value = resume.paymentType
  activeTab.value = 'recharge'
  selectedPlan.value = null
  if (resume.orderType === 'balance' && resume.orderAmount > 0) {
    amount.value = resume.orderAmount
  }

  await router.replace({
    path: '/purchase',
    query: stripWechatResumeQuery(route.query),
  })

  if (resume.wechatResumeToken) {
    await createOrder(0, resume.orderType, resume.planId, {
      wechatResumeToken: resume.wechatResumeToken,
      paymentType: resume.paymentType,
      isResume: true,
    })
    return
  }

  if (resume.orderAmount > 0 && resume.openid) {
    await createOrder(resume.orderAmount, resume.orderType, resume.planId, {
      openid: resume.openid,
      paymentType: resume.paymentType,
      isResume: true,
    })
  }
}

function hasWechatResumeRouteState(): boolean {
  return hasCompleteWechatResumeQuery(route.query)
    || ['wechat_resume', 'wechat_resume_token', 'openid'].some(
      key => Object.prototype.hasOwnProperty.call(route.query, key),
    )
}

function syncSubscriptionCheckoutFromRoute() {
  if (paymentPhase.value === 'paying') return

  if (
    route.hash === '#redeem'
    || hasWechatResumeRouteState()
    || readSingleQueryString(route.query, 'tab') !== 'subscription'
  ) {
    activeTab.value = 'recharge'
    selectedPlan.value = null
    return
  }

  const planId = Number(readSingleQueryString(route.query, 'plan'))
  activeTab.value = 'subscription'
  selectedPlan.value = Number.isSafeInteger(planId) && planId > 0
    ? checkout.value.plans.find((plan) => plan.id === planId) ?? null
    : null
  errorMessage.value = ''
  errorHintMessage.value = ''
}

onMounted(async () => {
  const historicalSpendRequest = usageAPI.getDashboardStats()
    .then((stats) => {
      historicalSpend.value = Number.isFinite(stats.total_actual_cost)
        ? Math.max(0, stats.total_actual_cost)
        : 0
    })
    .catch((error) => {
      console.error('Failed to load historical spend:', error)
  })

  try {
    checkout.value = await paymentStore.ensureCheckoutInfo()
    initializeRechargeAmount()
    syncSubscriptionCheckoutFromRoute()
    if (typeof window !== 'undefined') {
      const hasCompleteWechatResume = hasWechatResumeQuery(route.query)
      if (hasCompleteWechatResume) {
        removeRecoverySnapshot()
      }
      const paymentResumeToken = readSingleQueryString(route.query, 'resume_token')
      const wechatResumeToken = hasCompleteWechatResume
        ? readSingleQueryString(route.query, 'wechat_resume_token')
        : ''
      const routeResumeToken = paymentResumeToken || wechatResumeToken || undefined
      const restored = readPaymentRecoverySnapshot(
        window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY),
        { resumeToken: routeResumeToken },
      )
      if (restored) {
        paymentState.value = restored
        paymentPhase.value = 'paying'
        const restoredMethod = normalizeVisibleMethod(restored.paymentType)
          || (visibleMethods.value[restored.paymentType] ? restored.paymentType : '')
        if (restoredMethod) {
          selectedMethod.value = restoredMethod
        }
      } else {
        removeRecoverySnapshot()
      }
    }
    await resumeWechatPaymentFromQuery()
  } catch (err: unknown) { appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error'))) }
  finally { loading.value = false }
  await nextTick()
  if (route.hash === '#redeem') {
    document.getElementById('redeem')?.scrollIntoView({ block: 'start' })
  }
  await historicalSpendRequest
})

watch(
  () => [
    route.query.tab,
    route.query.plan,
    route.query.wechat_resume,
    route.query.wechat_resume_token,
    route.query.openid,
    route.hash,
  ] as const,
  () => {
    if (!loading.value) {
      syncSubscriptionCheckoutFromRoute()
    }
  },
)
</script>

<style scoped>
.purchase-workbench {
  color: var(--lx-clay-text);
  background: var(--lx-clay-surface);
  border-color: var(--lx-clay-border);
  box-shadow: 0 15px 38px rgb(70 55 96 / 8.5%);
}

.purchase-workbench__header {
  border-color: rgb(243 244 246 / 60%);
}

.purchase-workbench__title {
  font-size: var(--workspace-type-page-title-size);
  font-weight: var(--workspace-type-page-title-weight);
  letter-spacing: 0;
}

.purchase-workbench__icon {
  color: #6d28d9;
  background: #ede9fe;
}

.purchase-account-strip {
  overflow: hidden;
  border: 1px solid rgb(243 244 246 / 50%);
  border-radius: 1rem;
  background: rgb(249 250 251 / 80%);
}

.purchase-account-strip__amount {
  max-width: 100%;
  align-items: center;
  gap: 0.375rem;
  white-space: normal;
}

.purchase-account-strip__amount :deep([data-testid='credit-amount-value']),
.purchase-summary-credit :deep([data-testid='credit-amount-value']) {
  overflow: visible;
  text-overflow: clip;
  white-space: normal;
  overflow-wrap: anywhere;
}

.purchase-summary-credit {
  max-width: 100%;
  justify-content: flex-end;
  white-space: normal;
}

.purchase-summary-total {
  color: #6d28d9;
}

dd.purchase-summary-total {
  font-size: var(--workspace-type-numeric-size);
  font-weight: var(--workspace-type-numeric-weight);
}

.purchase-workbench__summary {
  border-top: 1px solid rgb(243 244 246 / 60%);
  background: rgb(249 250 251 / 50%);
}

.purchase-payment-submit {
  color: white;
  background: linear-gradient(145deg, #8b5cf6, #5b21b6);
  border-radius: 14px;
  font-weight: var(--workspace-type-navigation-weight);
  box-shadow: 0 10px 20px rgb(91 33 182 / 22%);
  transition:
    filter 150ms ease,
    transform 150ms ease,
    box-shadow 150ms ease;
}

.purchase-payment-submit:hover:not(:disabled) {
  filter: brightness(1.06);
  transform: translateY(-1px);
}

.purchase-payment-submit:active:not(:disabled) {
  transform: translateY(0);
}

.purchase-payment-submit:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--lx-clay-accent) 34%, transparent);
  outline-offset: 2px;
}

.purchase-payment-submit:disabled {
  cursor: not-allowed;
  opacity: 0.5;
  box-shadow: none;
}

.purchase-summary-panel {
  position: sticky;
  top: 2rem;
}

@media (min-width: 1024px) {
  .purchase-workbench__summary {
    border-top: 0;
    border-left: 1px solid rgb(243 244 246 / 60%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .purchase-payment-submit {
    transition-duration: 0.01ms;
  }

  .purchase-payment-submit:hover:not(:disabled) {
    transform: none;
  }
}
</style>
