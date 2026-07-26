/**
 * Payment Store
 * Manages payment configuration, current order state, and subscription plans
 */

import { defineStore } from 'pinia'
import { ref } from 'vue'
import { paymentAPI } from '@/api/payment'
import type {
  CheckoutInfoResponse,
  CreateOrderRequest,
  PaymentConfig,
  PaymentOrder,
  SubscriptionPlan,
} from '@/types/payment'

export const CHECKOUT_INFO_CACHE_TTL_MS = 60_000

export const usePaymentStore = defineStore('payment', () => {
  // ==================== State ====================

  /** Payment configuration from backend */
  const config = ref<PaymentConfig | null>(null)
  /** Currently active order (for payment flow) */
  const currentOrder = ref<PaymentOrder | null>(null)
  /** Available subscription plans */
  const plans = ref<SubscriptionPlan[]>([])
  /** Shared checkout catalogue and payment-method snapshot */
  const checkoutInfo = ref<CheckoutInfoResponse | null>(null)

  const configLoading = ref(false)
  const configLoaded = ref(false)
  const checkoutInfoLoading = ref(false)
  const checkoutInfoLastFetchedAt = ref<number | null>(null)
  let checkoutInfoPromise: Promise<CheckoutInfoResponse> | null = null

  // ==================== Actions ====================

  /** Fetch payment configuration */
  async function fetchConfig(force = false): Promise<PaymentConfig | null> {
    if (configLoaded.value && !force) return config.value
    if (configLoading.value) return config.value

    configLoading.value = true
    try {
      const response = await paymentAPI.getConfig()
      config.value = response.data
      configLoaded.value = true
      return config.value
    } catch (error: unknown) {
      console.error('[payment] Failed to fetch config:', error)
      return null
    } finally {
      configLoading.value = false
    }
  }

  /** Fetch available subscription plans */
  async function fetchPlans(): Promise<SubscriptionPlan[]> {
    try {
      const response = await paymentAPI.getPlans()
      // Backend returns features as newline-separated string; parse to array
      plans.value = (response.data || []).map((p: Omit<SubscriptionPlan, 'features'> & { features: string | string[] }) => ({
        ...p,
        features: typeof p.features === 'string'
          ? p.features.split('\n').map((f: string) => f.trim()).filter(Boolean)
          : (p.features || []),
      }))
      return plans.value
    } catch (error: unknown) {
      console.error('[payment] Failed to fetch plans:', error)
      return []
    }
  }

  /**
   * Ensure checkout data is available without issuing duplicate page-level
   * requests. A short TTL keeps prices and provider limits reasonably fresh
   * while allowing the pricing catalogue and checkout surface to share data.
   */
  async function ensureCheckoutInfo(force = false): Promise<CheckoutInfoResponse> {
    const now = Date.now()
    if (
      !force
      && checkoutInfo.value
      && checkoutInfoLastFetchedAt.value !== null
      && now - checkoutInfoLastFetchedAt.value < CHECKOUT_INFO_CACHE_TTL_MS
    ) {
      return checkoutInfo.value
    }

    if (checkoutInfoPromise) {
      return checkoutInfoPromise
    }

    checkoutInfoLoading.value = true
    const request = paymentAPI
      .getCheckoutInfo()
      .then((response) => {
        checkoutInfo.value = response.data
        checkoutInfoLastFetchedAt.value = Date.now()
        return response.data
      })
      .finally(() => {
        if (checkoutInfoPromise === request) {
          checkoutInfoPromise = null
          checkoutInfoLoading.value = false
        }
      })

    checkoutInfoPromise = request
    return request
  }

  function invalidateCheckoutInfo() {
    checkoutInfoLastFetchedAt.value = null
  }

  /** Create a new order and set it as current */
  async function createOrder(params: CreateOrderRequest) {
    const response = await paymentAPI.createOrder(params)
    return response.data
  }

  /** Poll order status by ID (read-only, no upstream check) */
  async function pollOrderStatus(orderId: number): Promise<PaymentOrder | null> {
    try {
      const response = await paymentAPI.getOrder(orderId)
      const order = response.data
      if (currentOrder.value?.id === orderId) {
        currentOrder.value = order
      }
      return order
    } catch (error: unknown) {
      console.error('[payment] Failed to poll order status:', error)
      return null
    }
  }

  /** Clear current order state */
  function clearCurrentOrder() {
    currentOrder.value = null
  }

  return {
    config,
    currentOrder,
    plans,
    checkoutInfo,
    configLoading,
    configLoaded,
    checkoutInfoLoading,
    fetchConfig,
    fetchPlans,
    ensureCheckoutInfo,
    invalidateCheckoutInfo,
    createOrder,
    pollOrderStatus,
    clearCurrentOrder
  }
})
