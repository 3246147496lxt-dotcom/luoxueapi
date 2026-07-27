import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import PaymentView from '../PaymentView.vue'
import CreditAmount from '@/components/common/CreditAmount.vue'
import AmountInput from '@/components/payment/AmountInput.vue'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import PaymentHelpPanel from '@/components/payment/PaymentHelpPanel.vue'
import RedeemCodePanel from '@/components/payment/RedeemCodePanel.vue'
import { PAYMENT_RECOVERY_STORAGE_KEY } from '@/components/payment/paymentFlow'
import { formatPaymentAmount } from '@/components/payment/currency'
import type { CheckoutInfoResponse, MethodLimit, SubscriptionPlan } from '@/types/payment'

const routeState = vi.hoisted(() => ({
  path: '/purchase',
  query: {} as Record<string, unknown>,
  hash: '',
}))

const routerReplace = vi.hoisted(() => vi.fn())
const routerPush = vi.hoisted(() => vi.fn())
const routerBack = vi.hoisted(() => vi.fn())
const routerResolve = vi.hoisted(() => vi.fn(() => ({ href: '/payment/stripe?mock=1' })))
const createOrder = vi.hoisted(() => vi.fn())
const refreshUser = vi.hoisted(() => vi.fn())
const fetchActiveSubscriptions = vi.hoisted(() => vi.fn().mockResolvedValue(undefined))
const showError = vi.hoisted(() => vi.fn())
const showInfo = vi.hoisted(() => vi.fn())
const showWarning = vi.hoisted(() => vi.fn())
const getCheckoutInfo = vi.hoisted(() => vi.fn())
const getDashboardStats = vi.hoisted(() => vi.fn())
const bridgeInvoke = vi.hoisted(() => vi.fn())
const publicSettings = vi.hoisted(() => ({ payment_enabled: true }))
const authUser = vi.hoisted(() => ({
  username: 'demo-user',
  email: 'demo@example.com',
  balance: 0,
  role: 'user' as 'admin' | 'user',
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => routeState,
    useRouter: () => ({
      replace: routerReplace,
      push: routerPush,
      back: routerBack,
      resolve: routerResolve,
    }),
  }
})

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: authUser,
    refreshUser,
  }),
}))

vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({
    createOrder,
    ensureCheckoutInfo: async () => (await getCheckoutInfo()).data,
  }),
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => ({
    activeSubscriptions: [],
    fetchActiveSubscriptions,
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    cachedPublicSettings: publicSettings,
    showError,
    showInfo,
    showWarning,
  }),
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getCheckoutInfo,
  },
}))

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardStats,
  },
}))

vi.mock('@/utils/device', () => ({
  isMobileDevice: () => true,
}))

function checkoutInfoFixture(overrides: Partial<CheckoutInfoResponse> = {}) {
  const wxpayMethod: MethodLimit = {
    daily_limit: 0,
    daily_used: 0,
    daily_remaining: 0,
    single_min: 0,
    single_max: 0,
    fee_rate: 0,
    available: true,
  }
  const data: CheckoutInfoResponse = {
    methods: {
      wxpay: wxpayMethod,
    },
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

  return {
    data: { ...data, ...overrides },
  }
}

function checkoutInfoWithPlansFixture(options: {
  checkout?: Partial<CheckoutInfoResponse>
  method?: Partial<MethodLimit>
  methods?: Record<string, MethodLimit>
  plan?: Partial<SubscriptionPlan>
} = {}) {
  const base = checkoutInfoFixture(options.checkout).data
  const plan: SubscriptionPlan = {
    id: 7,
    group_id: 3,
    name: 'Starter',
    description: '',
    price: 128,
    original_price: 0,
    validity_days: 30,
    validity_unit: 'day',
    rate_multiplier: 1,
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    features: [],
    group_platform: 'openai',
    sort_order: 1,
    for_sale: true,
    group_name: 'OpenAI',
    ...options.plan,
  }

  return {
    data: {
      ...base,
      methods: options.methods ?? {
        ...base.methods,
        wxpay: {
          ...base.methods.wxpay,
          ...options.method,
        },
      },
      plans: [plan],
    },
  }
}

function jsapiOrderFixture(resumeToken: string) {
  return {
    order_id: 123,
    amount: 88,
    pay_amount: 88,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    out_trade_no: 'sub2_jsapi_123',
    result_type: 'jsapi_ready' as const,
    resume_token: resumeToken,
    jsapi: {
      appId: 'wx123',
      timeStamp: '1712345678',
      nonceStr: 'nonce',
      package: 'prepay_id=wx123',
      signType: 'RSA',
      paySign: 'signed',
    },
  }
}

function oauthOrderFixture() {
  return {
    order_id: 456,
    amount: 128,
    pay_amount: 128,
    fee_rate: 0,
    expires_at: '2099-01-01T00:10:00.000Z',
    payment_type: 'wxpay',
    result_type: 'oauth_required' as const,
    oauth: {
      authorize_url: '/api/v1/auth/oauth/wechat/payment/start?payment_type=wxpay&redirect=%2Fpurchase%3Ffrom%3Dwechat',
      appid: 'wx123',
      scope: 'snsapi_base',
      redirect_url: '/auth/wechat/payment/callback',
    },
  }
}

beforeEach(() => {
  getDashboardStats.mockReset().mockResolvedValue({ total_actual_cost: 25.34 })
})

describe('PaymentView integrated purchase surface', () => {
  beforeEach(() => {
    authUser.username = 'demo-user'
    authUser.role = 'user'
    publicSettings.payment_enabled = true
    routeState.path = '/purchase'
    routeState.query = {}
    routeState.hash = ''
    getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoFixture())
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showError.mockReset()
  })

  it('keeps the redeem panel available when online payment is disabled', async () => {
    publicSettings.payment_enabled = false
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          RedeemCodePanel: { template: '<div data-test="redeem-panel" />' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-disabled-notice"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="redeem-panel"]').exists()).toBe(true)
    publicSettings.payment_enabled = true
  })

  it.each([
    ['balance recharge is disabled', { balance_disabled: true }],
    ['no online payment method is available', { methods: {} }],
  ])('keeps the redeem panel available when %s', async (_label, checkoutOverride) => {
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture(checkoutOverride))
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="payment-workbench"]').exists()).toBe(false)
    expect(wrapper.findComponent(RedeemCodePanel).exists()).toBe(true)
  })

  it('keeps administrator-provided payment help when balance recharge is unavailable', async () => {
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture({
      balance_disabled: true,
      help_text: 'Contact support before using an offline payment method.',
      help_image_url: 'https://example.com/help.png',
    }))
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const helpPanel = wrapper.getComponent(PaymentHelpPanel)
    expect(helpPanel.props()).toMatchObject({
      text: 'Contact support before using an offline payment method.',
      imageUrl: 'https://example.com/help.png',
    })
    expect(wrapper.findComponent(RedeemCodePanel).exists()).toBe(true)
  })

  it('selects the first available payment method when an earlier channel is disabled', async () => {
    const method = checkoutInfoFixture().data.methods.wxpay
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture({
      methods: {
        alipay: { ...method, available: false },
        wxpay: { ...method, available: true },
      },
    }))
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const selector = wrapper.getComponent(PaymentMethodSelector)
    expect(selector.props('selected')).toBe('wxpay')
    expect(selector.props('methods')).toEqual(expect.arrayContaining([
      expect.objectContaining({ type: 'alipay', available: false }),
      expect.objectContaining({ type: 'wxpay', available: true }),
    ]))
  })

  it('keeps the default purchase surface focused on balance recharge', async () => {
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    expect(wrapper.findAll('[role="tab"]')).toHaveLength(0)
    const rechargeRegion = wrapper.get('#purchase-panel-recharge')
    expect(rechargeRegion.element.tagName).toBe('SECTION')
    expect(rechargeRegion.attributes('role')).toBeUndefined()
    expect(rechargeRegion.attributes('aria-label')).toBe('payment.tabTopUp')
    expect(wrapper.find('#purchase-panel-subscription').exists()).toBe(false)
    const workbench = wrapper.get('[data-testid="payment-workbench"]')
    expect(workbench.classes()).toEqual(expect.arrayContaining(['grid', 'lg:grid-cols-12']))
    const main = workbench.get('[data-testid="payment-workbench-main"]')
    const summary = workbench.get('[data-testid="payment-workbench-summary"]')
    expect(main.classes()).toContain('lg:col-span-8')
    expect(summary.classes()).toContain('lg:col-span-4')
    expect(main.element.nextElementSibling).toBe(summary.element)
    expect(wrapper.text()).toContain('payment.alternativeDivider')
    expect(wrapper.text()).not.toContain('redeem.dividerTitle')
  })

  it('uses snowflake credits for balances while keeping recharge amounts in CNY', async () => {
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture({
      methods: {
        wxpay: {
          ...checkoutInfoFixture().data.methods.wxpay,
          currency: 'CNY',
        },
      },
    }))
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="payment-current-balance"]').findComponent(CreditAmount).props('value')).toBe('0.00')
    expect(wrapper.get('[data-testid="payment-credited-balance"]').findComponent(CreditAmount).props('value')).toBe('50.00')
    expect(wrapper.findComponent(AmountInput).props('currency')).toBe('CNY')
    expect(wrapper.findComponent(AmountInput).props('modelValue')).toBe(50)
    expect(wrapper.text()).toContain(formatPaymentAmount(50, 'CNY'))
  })

  it('shows the Draft header metrics and real historical spend', async () => {
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('payment.checkoutTitle')
    expect(wrapper.text()).toContain('payment.checkoutDescription')
    expect(wrapper.text()).toContain('dashboard.lifetimeSpend')
    const accountStrip = wrapper.get('[data-testid="payment-account-strip"]')
    expect(accountStrip.classes()).toContain('purchase-account-strip')
    expect(accountStrip.classes()).toEqual(expect.arrayContaining(['flex', 'divide-x']))
    expect(accountStrip.get('[data-testid="payment-current-balance"]').findComponent(CreditAmount).props('value')).toBe('0.00')
    expect(accountStrip.get('[data-testid="payment-historical-spend"]').findComponent(CreditAmount).props('value')).toBe('25.3400')
    expect(getDashboardStats).toHaveBeenCalledOnce()
  })

  it('uses one row-based order summary instead of four disconnected metric cards', async () => {
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const summary = wrapper.get('[data-testid="payment-order-summary"]')
    expect(summary.findAll('dl > div')).toHaveLength(4)
    expect(summary.text()).toContain('payment.amountLabel')
    expect(summary.text()).toContain('payment.creditedBalance')
    expect(summary.text()).toContain('payment.fee')
    expect(summary.text()).toContain('payment.actualPay')
    expect(wrapper.get('[data-testid="payment-workbench-summary"]').element.tagName).toBe('ASIDE')
    expect(wrapper.get('[data-testid="payment-workbench-summary"]').find('[data-testid="payment-order-summary"]').exists()).toBe(true)
  })

  it('keeps the Draft submit action inside the right-hand summary', async () => {
    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()

    const mainCard = wrapper.get('[data-testid="purchase-main-card"]')
    const actionBar = wrapper.get('[data-testid="payment-recharge-action-bar"]')
    expect(mainCard.classes()).toContain('overflow-hidden')
    expect(actionBar.classes()).toContain('mt-8')
    expect(actionBar.classes()).not.toContain('payment-mobile-action')
    expect(actionBar.get('button').classes()).toEqual(expect.arrayContaining([
      'purchase-payment-submit',
      'min-h-12',
      'w-full',
    ]))
    expect(actionBar.get('button').text()).toBe('payment.createOrder')
    expect(actionBar.get('button').text()).not.toContain('50')
  })
})

async function mountSubscriptionConfirm(options: Parameters<typeof checkoutInfoWithPlansFixture>[0] = {}) {
  vi.useRealTimers()
  routeState.path = '/purchase'
  routeState.query = {
    tab: 'subscription',
    plan: '7',
  }
  routeState.hash = ''
  routerReplace.mockReset().mockResolvedValue(undefined)
  routerPush.mockReset().mockResolvedValue(undefined)
  routerBack.mockReset()
  routerResolve.mockClear()
  createOrder.mockReset()
  refreshUser.mockReset()
  fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
  showError.mockReset()
  showInfo.mockReset()
  showWarning.mockReset()
  getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoWithPlansFixture(options))
  bridgeInvoke.mockReset()
  window.localStorage.clear()
  ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined

  const wrapper = shallowMount(PaymentView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div><slot /></div>',
        },
        Teleport: true,
        Transition: false,
      },
    },
  })
  await flushPromises()
  await flushPromises()
  return wrapper
}

describe('PaymentView subscription confirmation amounts', () => {
  it('shows converted CNY pay amount using the subscription rate, not the balance multiplier', async () => {
    const wrapper = await mountSubscriptionConfirm({
      checkout: {
        balance_recharge_multiplier: 0.14,
        subscription_usd_to_cny_rate: 7.15,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 9.99,
        original_price: 12.99,
      },
    })

    const text = wrapper.text()
    const convertedPrice = formatPaymentAmount(71.43, 'CNY')
    const convertedOriginalPrice = formatPaymentAmount(92.88, 'CNY')

    expect(text).toContain(convertedPrice)
    expect(text).toContain(convertedOriginalPrice)
    expect(text).not.toContain(formatPaymentAmount(9.99, 'CNY'))
    // 换算必须使用订阅汇率（×7.15），而不是余额倍率（÷0.14 = 71.36）
    expect(text).not.toContain(formatPaymentAmount(71.36, 'CNY'))
    expect(wrapper.findAll('button').some(button => button.text().includes(convertedPrice))).toBe(true)
    const actionBar = wrapper.get('[data-testid="payment-subscription-action-bar"]')
    expect(actionBar.classes()).toEqual(expect.arrayContaining([
      'payment-mobile-action',
      'space-y-3',
    ]))
    expect(actionBar.get('button').classes()).toContain('btn-primary')
    expect(actionBar.get('[data-testid="payment-subscription-cancel"]').classes()).toEqual(expect.arrayContaining([
      'btn-secondary',
      'w-full',
    ]))
    expect(wrapper.get('[data-testid="payment-workbench-main"]').classes()).toContain('lg:col-span-8')
    expect(wrapper.get('[data-testid="payment-workbench-summary"]').classes()).toContain('lg:col-span-4')
    expect(wrapper.get('[data-testid="payment-subscription-summary"]').findAll('dl > div')).toHaveLength(3)
    const subscriptionRegion = wrapper.get('#purchase-panel-subscription')
    expect(subscriptionRegion.element.tagName).toBe('SECTION')
    expect(subscriptionRegion.attributes('role')).toBeUndefined()
    expect(subscriptionRegion.attributes('aria-label')).toBe('payment.tabSubscribe')
  })

  it('keeps administrator-provided payment help in a valid subscription checkout', async () => {
    const wrapper = await mountSubscriptionConfirm({
      checkout: {
        help_text: 'Scan this guide before paying.',
        help_image_url: 'https://example.com/subscription-help.png',
      },
    })

    expect(wrapper.getComponent(PaymentHelpPanel).props()).toMatchObject({
      text: 'Scan this guide before paying.',
      imageUrl: 'https://example.com/subscription-help.png',
    })
  })

  it('keeps redemption and help available when a valid plan has no payable channel', async () => {
    const wrapper = await mountSubscriptionConfirm({
      methods: {},
      checkout: {
        help_text: 'Contact support for this plan.',
        help_image_url: 'https://example.com/plan-help.png',
      },
    })

    expect(wrapper.find('[data-testid="payment-workbench"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="payment-subscription-action-bar"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="payment-subscription-unavailable"]').text()).toContain('payment.notAvailable')
    expect(wrapper.findComponent(RedeemCodePanel).exists()).toBe(true)
    expect(wrapper.getComponent(PaymentHelpPanel).props()).toMatchObject({
      text: 'Contact support for this plan.',
      imageUrl: 'https://example.com/plan-help.png',
    })
  })

  it('selects an available subscription channel when the first configured channel is disabled', async () => {
    const method = checkoutInfoFixture().data.methods.wxpay
    const wrapper = await mountSubscriptionConfirm({
      methods: {
        alipay: { ...method, available: false },
        wxpay: { ...method, available: true },
      },
    })

    expect(wrapper.getComponent(PaymentMethodSelector).props('selected')).toBe('wxpay')
    expect(wrapper.get('[data-testid="payment-subscription-action-bar"] button').attributes('disabled')).toBeUndefined()
  })

  it('keeps plan price when the subscription rate is not configured or payment currency is not CNY', async () => {
    // opt-in 回归锁：即使余额倍率已配置，未配置订阅汇率时 CNY 订阅仍按 price 直付
    const cnyWrapper = await mountSubscriptionConfirm({
      checkout: {
        balance_recharge_multiplier: 0.14,
        subscription_usd_to_cny_rate: 0,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 7.99,
      },
    })

    expect(cnyWrapper.text()).toContain(formatPaymentAmount(7.99, 'CNY'))
    expect(cnyWrapper.text()).not.toContain(formatPaymentAmount(57.07, 'CNY'))
    expect(cnyWrapper.text()).not.toContain(formatPaymentAmount(57.13, 'CNY'))

    const usdWrapper = await mountSubscriptionConfirm({
      checkout: {
        subscription_usd_to_cny_rate: 7.15,
      },
      method: {
        currency: 'USD',
      },
      plan: {
        price: 7.99,
        original_price: 9.99,
      },
    })

    expect(usdWrapper.text()).toContain(formatPaymentAmount(7.99, 'USD'))
    expect(usdWrapper.text()).toContain(formatPaymentAmount(9.99, 'USD'))
  })

  it('adds fee rate after CNY rate conversion to match backend pay_amount', async () => {
    const wrapper = await mountSubscriptionConfirm({
      checkout: {
        subscription_usd_to_cny_rate: 7.15,
        recharge_fee_rate: 2.5,
      },
      method: {
        currency: 'CNY',
      },
      plan: {
        price: 9.99,
      },
    })

    const text = wrapper.text()
    const convertedPrice = formatPaymentAmount(71.43, 'CNY')
    const fee = formatPaymentAmount(1.79, 'CNY')
    const total = formatPaymentAmount(73.22, 'CNY')

    expect(text).toContain(convertedPrice)
    expect(text).toContain(fee)
    expect(text).toContain(total)
    expect(wrapper.findAll('button').some(button => button.text().includes(total))).toBe(true)
  })

  it('returns through pricing history when cancelling a selected plan', async () => {
    const historyState = vi.spyOn(window.history, 'state', 'get').mockReturnValue({
      back: '/pricing?group=3',
    })
    try {
      const wrapper = await mountSubscriptionConfirm()
      await wrapper.get('[data-testid="payment-subscription-cancel"]').trigger('click')

      expect(routerBack).toHaveBeenCalledOnce()
      expect(routerReplace).not.toHaveBeenCalledWith('/pricing')
      expect(routerPush).not.toHaveBeenCalledWith('/pricing')
    } finally {
      historyState.mockRestore()
    }
  })

  it('replaces with pricing when cancellation has no pricing history entry', async () => {
    const historyState = vi.spyOn(window.history, 'state', 'get').mockReturnValue({
      back: '/dashboard',
    })
    try {
      const wrapper = await mountSubscriptionConfirm()
      await wrapper.get('[data-testid="payment-subscription-cancel"]').trigger('click')

      expect(routerBack).not.toHaveBeenCalled()
      expect(routerReplace).toHaveBeenCalledWith('/pricing')
      expect(routerPush).not.toHaveBeenCalledWith('/pricing')
    } finally {
      historyState.mockRestore()
    }
  })

  it.each([
    ['repeated tab', { tab: ['subscription', 'subscription'], plan: '7' }, ''],
    ['repeated plan', { tab: 'subscription', plan: ['7', '7'] }, ''],
    ['redeem anchor', { tab: 'subscription', plan: '7' }, '#redeem'],
  ])('does not select a subscription plan for %s state', async (_label, query, hash) => {
    routeState.path = '/purchase'
    routeState.query = query
    routeState.hash = hash
    getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoWithPlansFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(wrapper.find('#purchase-panel-subscription').exists()).toBe(false)
    expect(wrapper.find('[data-testid="payment-subscription-action-bar"]').exists()).toBe(false)
    expect(wrapper.findComponent(RedeemCodePanel).exists()).toBe(true)
  })
})

describe('PaymentView payment recovery', () => {
  beforeEach(() => {
    vi.useRealTimers()
    routeState.path = '/purchase'
    routeState.query = {}
    routeState.hash = ''
    routerReplace.mockReset().mockResolvedValue(undefined)
    routerPush.mockReset().mockResolvedValue(undefined)
    routerResolve.mockClear()
    createOrder.mockReset()
    refreshUser.mockReset()
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showError.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    bridgeInvoke.mockReset()
    window.localStorage.clear()
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined
  })

  it('restores a custom EasyPay method as the selected payment method', async () => {
    getCheckoutInfo.mockResolvedValue(checkoutInfoFixture({
      methods: {
        wxpay: checkoutInfoFixture().data.methods.wxpay,
        ldc: {
          daily_limit: 0,
          daily_used: 0,
          daily_remaining: 0,
          single_min: 0,
          single_max: 0,
          fee_rate: 0,
          available: true,
          display_name: 'LDC Pay',
        },
      },
    }))
    window.localStorage.setItem(PAYMENT_RECOVERY_STORAGE_KEY, JSON.stringify({
      orderId: 888,
      amount: 66,
      qrCode: 'ldc-qr',
      expiresAt: '2099-01-01T00:10:00.000Z',
      paymentType: 'ldc',
      payUrl: 'https://pay.example.com/ldc',
      outTradeNo: 'sub2_ldc_888',
      clientSecret: '',
      intentId: '',
      currency: '',
      countryCode: '',
      paymentEnv: '',
      payAmount: 66,
      orderType: 'balance',
      paymentMode: 'popup',
      resumeToken: '',
      createdAt: Date.now(),
    }))

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: {
            template: '<div><slot /></div>',
          },
          PaymentStatusPanel: {
            template: '<button data-test="payment-done" @click="$emit(\'done\')" />',
          },
          PaymentMethodSelector: {
            props: ['selected'],
            template: '<div data-test="method-selector">{{ selected }}</div>',
          },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()
    await wrapper.find('[data-test="payment-done"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="method-selector"]').text()).toBe('ldc')
  })
})

describe('PaymentView WeChat JSAPI flow', () => {
  beforeEach(() => {
    routeState.path = '/purchase'
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
    }
    routeState.hash = ''
    routerReplace.mockReset().mockResolvedValue(undefined)
    routerPush.mockReset().mockResolvedValue(undefined)
    routerResolve.mockClear()
    createOrder.mockReset()
    refreshUser.mockReset()
    fetchActiveSubscriptions.mockReset().mockResolvedValue(undefined)
    showError.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    getCheckoutInfo.mockReset().mockResolvedValue(checkoutInfoFixture())
    bridgeInvoke.mockReset()
    window.localStorage.clear()
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = {
      invoke: bridgeInvoke,
    }
  })

  it('resets payment state and redirects to /payment/result after JSAPI reports success', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-123'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:ok' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(routerPush).toHaveBeenCalledWith({
      path: '/payment/result',
      query: {
        order_id: '123',
        out_trade_no: 'sub2_jsapi_123',
        resume_token: 'resume-token-123',
      },
    })
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('resets payment state when JSAPI reports cancellation', async () => {
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-cancel'))
    bridgeInvoke.mockImplementation((_action, _payload, callback) => {
      callback({ err_msg: 'get_brand_wcpay_request:cancel' })
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(showInfo).toHaveBeenCalledWith('payment.qr.cancelled')
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('clears stale recovery state when JSAPI never becomes available', async () => {
    vi.useFakeTimers()
    createOrder.mockResolvedValue(jsapiOrderFixture('resume-token-missing-bridge'))
    ;(window as Window & { WeixinJSBridge?: { invoke: typeof bridgeInvoke } }).WeixinJSBridge = undefined

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })

    await flushPromises()
    await vi.advanceTimersByTimeAsync(4000)
    await flushPromises()
    await flushPromises()

    expect(showError).toHaveBeenCalledWith(
      'payment.errors.wechatJsapiUnavailable payment.errors.wechatOpenInWeChatHint',
    )
    expect(routerPush).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
    expect(wrapper.html()).not.toContain('payment-status-panel-stub')
  })

  it('clears a stale recovery snapshot before handling wechat resume callback params', async () => {
    createOrder.mockRejectedValueOnce(new Error('resume failed'))
    window.localStorage.setItem(PAYMENT_RECOVERY_STORAGE_KEY, JSON.stringify({
      orderId: 999,
      amount: 66,
      qrCode: 'stale-qr',
      expiresAt: '2099-01-01T00:10:00.000Z',
      paymentType: 'alipay',
      payUrl: 'https://pay.example.com/stale',
      outTradeNo: 'stale-out-trade-no',
      clientSecret: '',
      intentId: '',
      currency: '',
      countryCode: '',
      paymentEnv: '',
      payAmount: 66,
      orderType: 'balance',
      paymentMode: 'popup',
      resumeToken: '',
      createdAt: Date.UTC(2099, 0, 1, 0, 0, 0),
    }))

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      wechat_resume_token: 'resume-token-123',
    }))
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toBeNull()
  })

  it('keeps subscription resume context for token-only WeChat callbacks', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-subscription-7',
      payment_type: 'wxpay_direct',
      order_type: 'subscription',
      plan_id: '7',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockResolvedValue(oauthOrderFixture())

    const originalLocation = window.location
    const locationState = {
      href: 'http://localhost/purchase',
      origin: 'http://localhost',
    }
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: locationState,
    })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({ path: '/purchase', query: {} })
    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 7,
      wechat_resume_token: 'resume-subscription-7',
    }))
    expect(locationState.href).toContain('/api/v1/auth/oauth/wechat/payment/start?')
    expect(new URL(locationState.href, 'http://localhost').searchParams.get('redirect')).toBe(
      '/purchase?from=wechat&payment_type=wxpay&order_type=subscription&plan_id=7',
    )

    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    })
  })

  it('consumes legacy subscription checkout state before resuming an openid callback', async () => {
    routeState.query = {
      from: 'wechat',
      tab: 'subscription',
      plan: '7',
      group: '3',
      wechat_resume: '1',
      openid: 'openid-123',
      payment_type: 'wxpay_direct',
      amount: '128',
      order_type: 'subscription',
      plan_id: '7',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockRejectedValueOnce(new Error('resume failed'))

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({
      path: '/purchase',
      query: {
        from: 'wechat',
      },
    })
    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      payment_type: 'wxpay',
      order_type: 'subscription',
      plan_id: 7,
      openid: 'openid-123',
    }))
    expect(routerReplace.mock.invocationCallOrder[0]).toBeLessThan(
      createOrder.mock.invocationCallOrder[0],
    )
    expect(wrapper.find('#purchase-panel-subscription').exists()).toBe(false)
    expect(wrapper.find('[data-testid="payment-subscription-action-bar"]').exists()).toBe(false)
  })

  it('consumes injected subscription checkout state before resuming a balance callback', async () => {
    routeState.query = {
      from: 'wechat',
      tab: 'subscription',
      plan: '7',
      group: '3',
      wechat_resume: '1',
      openid: 'openid-balance-123',
      payment_type: 'wxpay_direct',
      amount: '88',
      order_type: 'balance',
    }
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())
    createOrder.mockRejectedValueOnce(new Error('resume failed'))

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(routerReplace).toHaveBeenCalledWith({
      path: '/purchase',
      query: {
        from: 'wechat',
      },
    })
    expect(createOrder).toHaveBeenCalledWith(expect.objectContaining({
      amount: 88,
      payment_type: 'wxpay',
      order_type: 'balance',
      openid: 'openid-balance-123',
    }))
    expect(routerReplace.mock.invocationCallOrder[0]).toBeLessThan(
      createOrder.mock.invocationCallOrder[0],
    )
    expect(wrapper.find('#purchase-panel-subscription').exists()).toBe(false)
    expect(wrapper.find('[data-testid="payment-subscription-action-bar"]').exists()).toBe(false)
  })

  it.each([
    ['marker only', {
      tab: 'subscription',
      plan: '7',
      wechat_resume: '1',
    }],
    ['token only', {
      tab: 'subscription',
      plan: '7',
      wechat_resume_token: 'resume-token-123',
    }],
    ['repeated token', {
      tab: 'subscription',
      plan: '7',
      wechat_resume: '1',
      wechat_resume_token: ['resume-token-123', 'resume-token-456'],
    }],
  ])('does not expose a new subscription purchase for invalid %s recovery state', async (_label, query) => {
    routeState.query = query
    getCheckoutInfo.mockResolvedValue(checkoutInfoWithPlansFixture())

    const wrapper = shallowMount(PaymentView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).not.toHaveBeenCalled()
    expect(wrapper.find('#purchase-panel-subscription').exists()).toBe(false)
    expect(wrapper.find('[data-testid="payment-subscription-action-bar"]').exists()).toBe(false)
  })

  it('falls back to QR flow when mobile WeChat payment is unavailable', async () => {
    routeState.query = {
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-h5',
      payment_type: 'wxpay_direct',
    }
    createOrder
      .mockRejectedValueOnce({ reason: 'WECHAT_H5_NOT_AUTHORIZED' })
      .mockResolvedValueOnce({
        order_id: 778,
        amount: 88,
        pay_amount: 88,
        fee_rate: 0,
        expires_at: '2099-01-01T00:10:00.000Z',
        payment_type: 'wxpay',
        qr_code: 'weixin://wxpay/bizpayurl?pr=fallback-native',
        out_trade_no: 'sub2_qr_778',
      })

    shallowMount(PaymentView, {
      global: {
        stubs: {
          Teleport: true,
          Transition: false,
        },
      },
    })
    await flushPromises()
    await flushPromises()

    expect(createOrder).toHaveBeenNthCalledWith(1, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: true,
      wechat_resume_token: 'resume-token-h5',
    }))
    expect(createOrder).toHaveBeenNthCalledWith(2, expect.objectContaining({
      payment_type: 'wxpay',
      is_mobile: false,
      payment_source: 'hosted_redirect',
    }))
    expect(showWarning).toHaveBeenCalledWith('payment.errors.mobilePaymentFallbackToQr')
    expect(showError).not.toHaveBeenCalled()
    expect(window.localStorage.getItem(PAYMENT_RECOVERY_STORAGE_KEY)).toContain('weixin://wxpay/bizpayurl?pr=fallback-native')
  })
})
