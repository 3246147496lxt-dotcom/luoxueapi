import { describe, expect, it } from 'vitest'
import {
  hasWechatResumeQuery,
  parseWechatResumeRoute,
  stripWechatResumeQuery,
} from '../paymentWechatResume'

describe('parseWechatResumeRoute', () => {
  it('prefers the opaque resume token over legacy openid query params', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
    }, [], 88)).toEqual({
      wechatResumeToken: 'resume-token-123',
      paymentType: 'wxpay',
      orderType: 'subscription',
      orderAmount: 0,
      planId: 7,
    })
  })

  it('falls back to legacy openid-based resume when opaque token is absent', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'balance',
    }, [], 88)).toEqual({
      openid: 'openid-123',
      paymentType: 'wxpay',
      orderType: 'balance',
      orderAmount: 12.5,
      planId: undefined,
    })
  })

  it.each([
    ['marker only', { wechat_resume: '1' }],
    ['token only', { wechat_resume_token: 'resume-token-123' }],
    ['openid only', { openid: 'openid-123' }],
    ['repeated marker', {
      wechat_resume: ['1', '1'],
      wechat_resume_token: 'resume-token-123',
    }],
    ['repeated token', {
      wechat_resume: '1',
      wechat_resume_token: ['resume-token-123', 'resume-token-456'],
    }],
    ['repeated openid', {
      wechat_resume: '1',
      openid: ['openid-123', 'openid-456'],
    }],
  ])('rejects incomplete or ambiguous %s resume params', (_label, query) => {
    expect(hasWechatResumeQuery(query)).toBe(false)
    expect(parseWechatResumeRoute(query, [], 88)).toBeNull()
  })

  it('does not take the first repeated metadata value during a valid resume', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      payment_type: ['alipay', 'wxpay'],
      order_type: ['subscription', 'balance'],
      plan_id: ['7', '8'],
    }, [], 88)).toEqual({
      wechatResumeToken: 'resume-token-123',
      paymentType: 'wxpay',
      orderType: 'balance',
      orderAmount: 0,
      planId: undefined,
    })
  })
})

describe('stripWechatResumeQuery', () => {
  it('removes resume and prior subscription checkout params while preserving unrelated query', () => {
    expect(stripWechatResumeQuery({
      foo: 'bar',
      tab: 'subscription',
      plan: '7',
      group: '3',
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
      state: 'state-123',
      scope: 'snsapi_base',
    })).toEqual({
      foo: 'bar',
    })
  })

  it('also consumes stale subscription checkout params when cleaning a balance resume', () => {
    expect(stripWechatResumeQuery({
      foo: 'bar',
      tab: 'subscription',
      plan: '7',
      group: '3',
      wechat_resume: '1',
      openid: 'openid-123',
    })).toEqual({
      foo: 'bar',
    })
  })
})
