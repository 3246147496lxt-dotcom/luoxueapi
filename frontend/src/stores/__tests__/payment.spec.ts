import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { paymentAPI } from '@/api/payment'
import { CHECKOUT_INFO_CACHE_TTL_MS, usePaymentStore } from '@/stores/payment'
import type { CheckoutInfoResponse } from '@/types/payment'

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo: vi.fn(),
    getConfig: vi.fn(),
    getPlans: vi.fn(),
    createOrder: vi.fn(),
    getOrder: vi.fn(),
  },
}))

function checkoutInfoFixture(): CheckoutInfoResponse {
  return {
    methods: {},
    global_min: 0,
    global_max: 0,
    plans: [],
    balance_disabled: false,
    balance_recharge_multiplier: 1,
    subscription_usd_to_cny_rate: 0,
    recharge_fee_rate: 0,
    help_text: '',
    help_image_url: '',
    stripe_publishable_key: '',
  }
}

function createDeferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('usePaymentStore checkout info cache', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.mocked(paymentAPI.getCheckoutInfo).mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('deduplicates concurrent catalogue and checkout requests', async () => {
    const deferred = createDeferred<{ data: CheckoutInfoResponse }>()
    vi.mocked(paymentAPI.getCheckoutInfo).mockReturnValue(deferred.promise as never)
    const store = usePaymentStore()

    const first = store.ensureCheckoutInfo()
    const second = store.ensureCheckoutInfo()

    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledOnce()
    expect(store.checkoutInfoLoading).toBe(true)

    const checkoutInfo = checkoutInfoFixture()
    deferred.resolve({ data: checkoutInfo })

    await expect(first).resolves.toBe(checkoutInfo)
    await expect(second).resolves.toBe(checkoutInfo)
    expect(store.checkoutInfo).toEqual(checkoutInfo)
    expect(store.checkoutInfoLoading).toBe(false)
  })

  it('reuses fresh checkout info and refreshes it after the TTL', async () => {
    const firstCheckoutInfo = checkoutInfoFixture()
    const refreshedCheckoutInfo = {
      ...checkoutInfoFixture(),
      global_min: 10,
    }
    vi.mocked(paymentAPI.getCheckoutInfo)
      .mockResolvedValueOnce({ data: firstCheckoutInfo } as never)
      .mockResolvedValueOnce({ data: refreshedCheckoutInfo } as never)
    const store = usePaymentStore()

    await expect(store.ensureCheckoutInfo()).resolves.toEqual(firstCheckoutInfo)
    vi.advanceTimersByTime(CHECKOUT_INFO_CACHE_TTL_MS - 1)
    await expect(store.ensureCheckoutInfo()).resolves.toEqual(firstCheckoutInfo)
    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledOnce()

    vi.advanceTimersByTime(2)
    await expect(store.ensureCheckoutInfo()).resolves.toBe(refreshedCheckoutInfo)
    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledTimes(2)
  })

  it('allows retry after a failed request and still deduplicates forced refreshes', async () => {
    vi.mocked(paymentAPI.getCheckoutInfo)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({ data: checkoutInfoFixture() } as never)
    const store = usePaymentStore()

    await expect(store.ensureCheckoutInfo()).rejects.toThrow('network unavailable')
    expect(store.checkoutInfoLoading).toBe(false)

    const firstRetry = store.ensureCheckoutInfo(true)
    const secondRetry = store.ensureCheckoutInfo(true)
    await expect(firstRetry).resolves.toEqual(checkoutInfoFixture())
    await expect(secondRetry).resolves.toEqual(checkoutInfoFixture())
    expect(paymentAPI.getCheckoutInfo).toHaveBeenCalledTimes(2)
  })
})
