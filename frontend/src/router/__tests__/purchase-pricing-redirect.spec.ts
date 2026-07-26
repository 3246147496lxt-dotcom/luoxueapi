import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import {
  isFreshSubscriptionPlanBridge,
  resolveLegacySubscriptionPricingRedirect,
} from '@/router/purchasePricingRedirect'
import { stripWechatResumeQuery } from '@/views/user/paymentWechatResume'

const AUXILIARY_RECOVERY_QUERY_KEYS = [
  'state',
  'scope',
  'payment_type',
  'amount',
  'order_type',
  'plan_id',
  'resume_token',
  'order_id',
  'out_trade_no',
] as const

function routeState(
  query: Record<string, string | string[] | undefined>,
  hash = '',
  path = '/purchase',
) {
  return {
    path,
    query,
    hash,
  } as never
}

function resolve(
  query: Record<string, string | string[] | undefined>,
  hash = '',
  path = '/purchase',
) {
  return resolveLegacySubscriptionPricingRedirect({
    path,
    query,
    hash,
  } as never)
}

describe('legacy subscription catalogue redirect', () => {
  it('moves the pure legacy catalogue URL to /pricing without requiring a group', () => {
    expect(resolve({
      tab: 'subscription',
      source: 'account-menu',
    }, '#plans')).toEqual({
      path: '/pricing',
      query: {
        source: 'account-menu',
      },
      hash: '#plans',
      replace: true,
    })
  })

  it('preserves an optional renewal group and unrelated query state', () => {
    expect(resolve({
      tab: 'subscription',
      group: '3',
      campaign: 'summer',
    })).toEqual({
      path: '/pricing',
      query: {
        group: '3',
        campaign: 'summer',
      },
      hash: '',
      replace: true,
    })
  })

  it('keeps an explicit plan checkout on /purchase', () => {
    expect(resolve({
      tab: 'subscription',
      plan: '7',
    })).toBeUndefined()
  })

  it.each([
    ['resume token', { wechat_resume: '1', wechat_resume_token: 'resume-7' }],
    ['openid', { wechat_resume: '1', openid: 'openid-7' }],
  ])('keeps a complete WeChat %s recovery on /purchase', (_name, recovery) => {
    expect(resolve({
      tab: 'subscription',
      plan: '7',
      ...recovery,
    })).toBeUndefined()
  })

  it('canonicalizes redemption independently from subscription and preserves unrelated query', () => {
    expect(resolve({
      tab: 'subscription',
      plan: '7',
      group: '3',
      source: 'wallet',
    }, '#redeem')).toEqual({
      path: '/purchase',
      query: {
        source: 'wallet',
      },
      hash: '#redeem',
      replace: true,
    })
    expect(resolve({ source: 'wallet' }, '#redeem')).toBeUndefined()
  })

  it('canonicalizes repeated tabs to ordinary purchase state', () => {
    expect(resolve({
      tab: ['subscription', 'recharge'],
      plan: '7',
      group: '3',
      source: 'account-menu',
    })).toEqual({
      path: '/purchase',
      query: {
        source: 'account-menu',
      },
      hash: '',
      replace: true,
    })
  })

  it.each([
    '',
    '0',
    '-1',
    '1.5',
    'not-a-plan',
  ])('moves a subscription URL with invalid plan %j to pricing', (plan) => {
    expect(resolve({
      tab: 'subscription',
      plan,
      source: 'account-menu',
    })).toEqual({
      path: '/pricing',
      query: {
        source: 'account-menu',
      },
      hash: '',
      replace: true,
    })
  })

  it('moves a subscription URL with a repeated plan to pricing', () => {
    expect(resolve({
      tab: 'subscription',
      plan: ['7', '8'],
      source: 'account-menu',
    })).toEqual({
      path: '/pricing',
      query: {
        source: 'account-menu',
      },
      hash: '',
      replace: true,
    })
  })

  it('keeps non-subscription tabs on /purchase', () => {
    expect(resolve({ tab: 'recharge' })).toBeUndefined()
  })

  it('canonicalizes a same-route query-only navigation through a global guard', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/purchase',
          component: { template: '<div />' },
        },
        {
          path: '/pricing',
          component: { template: '<div />' },
        },
      ],
    })
    router.beforeEach((to) => resolveLegacySubscriptionPricingRedirect(to))

    await router.push('/purchase')
    await router.isReady()
    await router.push('/purchase?tab=subscription&source=account-menu')

    expect(router.currentRoute.value.fullPath).toBe('/pricing?source=account-menu')
  })

  it('keeps a legacy WeChat return on the purchase route while consuming recovery state', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/purchase',
          name: 'Purchase',
          component: { template: '<div />' },
        },
        {
          path: '/pricing',
          name: 'Pricing',
          component: { template: '<div />' },
        },
      ],
    })
    router.beforeEach((to) => resolveLegacySubscriptionPricingRedirect(to))

    await router.push({
      path: '/purchase',
      query: {
        tab: 'subscription',
        group: '3',
        source: 'legacy-wechat',
        wechat_resume: '1',
        wechat_resume_token: 'resume-7',
        payment_type: 'wxpay',
        order_type: 'subscription',
        plan_id: '7',
      },
    })
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('Purchase')

    await router.replace({
      path: '/purchase',
      query: stripWechatResumeQuery(router.currentRoute.value.query),
    })

    expect(router.currentRoute.value.name).toBe('Purchase')
    expect(router.currentRoute.value.fullPath).toBe(
      '/purchase?source=legacy-wechat',
    )
  })
})

describe('fresh subscription plan bridge', () => {
  it.each(['1', '7', '01', String(Number.MAX_SAFE_INTEGER)])(
    'accepts positive integer plan id %s',
    (plan) => {
      expect(isFreshSubscriptionPlanBridge(routeState({
        tab: 'subscription',
        plan,
      }))).toBe(true)
    },
  )

  it.each([
    '',
    '0',
    '-1',
    '1.5',
    '1e2',
    ' 7',
    `${Number.MAX_SAFE_INTEGER}0`,
  ])('rejects invalid plan id %s', (plan) => {
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan,
    }))).toBe(false)
  })

  it('rejects repeated plan query values and unrelated routes', () => {
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan: ['7', '8'],
    }))).toBe(false)
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan: '7',
    }, '', '/pricing'))).toBe(false)
  })

  it.each(AUXILIARY_RECOVERY_QUERY_KEYS)(
    'does not let auxiliary recovery query %s bypass the plan bridge gate',
    (key) => {
      expect(isFreshSubscriptionPlanBridge(routeState({
        tab: 'subscription',
        plan: '7',
        [key]: '1',
      }))).toBe(true)
    },
  )

  it.each([
    ['resume marker only', { wechat_resume: '1' }],
    ['resume token only', { wechat_resume_token: 'resume-7' }],
    ['openid only', { openid: 'openid-7' }],
    ['empty resume token', { wechat_resume: '1', wechat_resume_token: '   ' }],
    ['repeated resume marker', {
      wechat_resume: ['1', '1'],
      wechat_resume_token: 'resume-7',
    }],
    ['repeated resume token', {
      wechat_resume: '1',
      wechat_resume_token: ['resume-7', 'resume-8'],
    }],
  ])('does not let %s bypass the plan bridge gate', (_name, recovery) => {
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan: '7',
      ...recovery,
    }))).toBe(true)
  })

  it.each([
    ['resume token', { wechat_resume: '1', wechat_resume_token: 'resume-7' }],
    ['openid', { wechat_resume: '1', openid: 'openid-7' }],
  ])('lets a complete WeChat %s recovery bypass the plan bridge gate', (_name, recovery) => {
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan: '7',
      ...recovery,
    }))).toBe(false)
  })

  it('does not treat the redeem anchor as a fresh plan bridge', () => {
    expect(isFreshSubscriptionPlanBridge(routeState({
      tab: 'subscription',
      plan: '7',
    }, '#redeem'))).toBe(false)
  })
})
