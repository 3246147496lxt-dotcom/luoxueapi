import { describe, expect, it } from 'vitest'
import { useAnthropicOAuthControlDraft } from '../useAnthropicOAuthControlDraft'

describe('useAnthropicOAuthControlDraft', () => {
  it('owns OAuth control state, extra composition and reset', () => {
    const draft = useAnthropicOAuthControlDraft()
    draft.windowCostEnabled.value = true
    draft.windowCostLimit.value = 25
    draft.rpmLimitEnabled.value = true
    draft.baseRpm.value = 20
    draft.customBaseUrlEnabled.value = true
    draft.customBaseUrl.value = ' https://claude.example '

    expect(draft.buildExtra({ account_uuid: 'account' })).toEqual({
      account_uuid: 'account',
      window_cost_limit: 25,
      window_cost_sticky_reserve: 10,
      base_rpm: 20,
      rpm_strategy: 'tiered',
      custom_base_url_enabled: true,
      custom_base_url: 'https://claude.example',
    })

    draft.reset()
    expect(draft.windowCostEnabled.value).toBe(false)
    expect(draft.windowCostLimit.value).toBeNull()
    expect(draft.rpmLimitEnabled.value).toBe(false)
    expect(draft.baseRpm.value).toBeNull()
    expect(draft.customBaseUrlEnabled.value).toBe(false)
    expect(draft.customBaseUrl.value).toBe('')
  })
})
