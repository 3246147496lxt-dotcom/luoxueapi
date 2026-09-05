// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import MetricStrip from './MetricStrip.vue'
import { messages } from '@/i18n'

const today = {
  requests: 4,
  tokens: 1_500,
  cost: 1.25,
  balance: 9.75,
  averageFirstTokenMs: 321,
}

function mountMetricStrip(locale: 'zh-CN' | 'en') {
  const i18n = createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'zh-CN',
    messages,
  })

  return mount(MetricStrip, {
    props: { today },
    global: { plugins: [i18n] },
  })
}

describe('MetricStrip points display', () => {
  it.each([
    ['zh-CN', '今日积分消费', '积分余额', '积分'],
    ['en', 'Points spent today', 'Points balance', 'points'],
  ] as const)('labels user billing values as points in %s', (locale, spendLabel, balanceLabel, unit) => {
    const wrapper = mountMetricStrip(locale)

    expect(wrapper.text()).toContain(spendLabel)
    expect(wrapper.text()).toContain(balanceLabel)
    expect(wrapper.text()).toContain(`1.25 ${unit}`)
    expect(wrapper.text()).toContain(`9.75 ${unit}`)
    expect(wrapper.text()).not.toContain('$')
  })
})
