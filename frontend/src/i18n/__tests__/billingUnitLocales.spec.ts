import { describe, expect, it } from 'vitest'

import enAdminAccounts from '../locales/en/admin/accounts'
import enAdminOverview from '../locales/en/admin/overview'
import enAdminResources from '../locales/en/admin/resources'
import enAdminSettings from '../locales/en/admin/settings'
import enChat from '../locales/en/chat'
import enCommon from '../locales/en/common'
import enDashboard from '../locales/en/dashboard'
import enLanding from '../locales/en/landing'
import enMisc from '../locales/en/misc'
import zhAdminOverview from '../locales/zh/admin/overview'
import zhAdminResources from '../locales/zh/admin/resources'
import zhAdminSettings from '../locales/zh/admin/settings'
import zhChat from '../locales/zh/chat'
import zhCommon from '../locales/zh/common'
import zhDashboard from '../locales/zh/dashboard'
import zhLanding from '../locales/zh/landing'
import zhMisc from '../locales/zh/misc'

const currencyMarkers = /\bUSD\b|美元|\$\d/

function expectPointsCopy(values: string[], unit: RegExp) {
  for (const value of values) {
    expect(value).toMatch(unit)
    expect(value).not.toMatch(currencyMarkers)
  }
}

describe('billing unit locale contract', () => {
  it('labels user balances and quota limits as Points', () => {
    expectPointsCopy([
      enDashboard.keys.workspaceRateHeading,
      enDashboard.keys.rateLimit5h,
      enDashboard.keys.rateLimit1d,
      enDashboard.keys.rateLimit7d,
      enAdminSettings.settings.platformQuota.daily,
      enAdminSettings.settings.platformQuota.weekly,
      enAdminSettings.settings.platformQuota.monthly,
      enAdminOverview.users.platformQuota.subtitle,
      enAdminOverview.users.platformQuota.columns.daily,
      enAdminOverview.users.platformQuota.columns.weekly,
      enAdminOverview.users.platformQuota.columns.monthly,
      enAdminOverview.groups.subscription.dailyLimit,
      enAdminOverview.groups.subscription.weeklyLimit,
      enAdminOverview.groups.subscription.monthlyLimit,
      enAdminResources.redeem.form.balanceHint,
      enAdminResources.redeem.amount,
      enAdminResources.promo.bonusAmount,
      enMisc.payment.admin.insufficientBalance,
      enChat.chat.receipt.lowBalance,
    ], /Points?/)

    expectPointsCopy([
      zhDashboard.keys.workspaceRateHeading,
      zhDashboard.keys.rateLimit5h,
      zhDashboard.keys.rateLimit1d,
      zhDashboard.keys.rateLimit7d,
      zhAdminSettings.settings.platformQuota.daily,
      zhAdminSettings.settings.platformQuota.weekly,
      zhAdminSettings.settings.platformQuota.monthly,
      zhAdminOverview.users.platformQuota.subtitle,
      zhAdminOverview.users.platformQuota.columns.daily,
      zhAdminOverview.users.platformQuota.columns.weekly,
      zhAdminOverview.users.platformQuota.columns.monthly,
      zhAdminOverview.groups.subscription.dailyLimit,
      zhAdminOverview.groups.subscription.weeklyLimit,
      zhAdminOverview.groups.subscription.monthlyLimit,
      zhAdminResources.redeem.form.balanceHint,
      zhAdminResources.redeem.amount,
      zhAdminResources.promo.bonusAmount,
      zhMisc.payment.admin.insufficientBalance,
      zhChat.chat.receipt.lowBalance,
    ], /积分/)
  })

  it('describes CNY recharge as Points granted', () => {
    const enPayment = enAdminSettings.settings.payment
    const zhPayment = zhAdminSettings.settings.payment

    expect(enPayment.balanceRechargeMultiplierHint).toContain('Points')
    expect(enPayment.balanceRechargePreview).toBe('Preview: 1 CNY = {credit} Points')
    expect(zhPayment.balanceRechargeMultiplierHint).toContain('积分')
    expect(zhPayment.balanceRechargePreview).toBe('预览：1 CNY = {credit} 积分')
  })

  it('describes group billing as USD source price converted to Points', () => {
    expect(enAdminOverview.groups.form.rateMultiplierHint).toContain('5 USD/MTok × 70 = 350 Points/MTok')
    expect(zhAdminOverview.groups.form.rateMultiplierHint).toContain('5 USD/MTok × 70 = 350 积分/MTok')
    expect(enMisc.onboarding.admin.groupMultiplier.description).not.toContain('charged $')
    expect(zhMisc.onboarding.admin.groupMultiplier.description).not.toContain('扣除 $')
  })

  it('does not describe promotional balance as dollars', () => {
    expect(enCommon.auth.promoCodeValid).not.toMatch(currencyMarkers)
    expect(zhCommon.auth.promoCodeValid).not.toMatch(currencyMarkers)
  })

  it('explains that public catalog prices are converted from Points to CNY', () => {
    expect(enLanding.modelCatalog.publicPriceNote).toContain('¥1 = 10 Points')
    expect(enLanding.modelCatalog.publicPriceNote).toContain('converted to CNY')
    expect(enLanding.modelCatalog.pricing.dialogDescription).toContain('¥1 = 10 Points')

    expect(zhLanding.modelCatalog.publicPriceNote).toContain('¥1 = 10 积分')
    expect(zhLanding.modelCatalog.publicPriceNote).toContain('人民币')
    expect(zhLanding.modelCatalog.pricing.dialogDescription).toContain('¥1 = 10 积分')
  })

  it('keeps USD where it is the source price, conversion input, or upstream limit', () => {
    expect(enAdminSettings.settings.payment.subscriptionUsdToCnyRateHint).toContain('USD')
    expect(enAdminAccounts.accounts.quotaLimitHint).toContain('USD')
    expect(enAdminOverview.groups.videoPricing.description).toContain('USD')
    expect(enMisc.payment.admin.currencyPlaceholder).toContain('USD')
  })
})
