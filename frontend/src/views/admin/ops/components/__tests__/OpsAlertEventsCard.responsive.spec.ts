import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import OpsAlertEventsCard from '../OpsAlertEventsCard.vue'

const { getAlertEvent, listAlertEvents, getGroups, showError, showSuccess } = vi.hoisted(() => ({
  getAlertEvent: vi.fn(),
  listAlertEvents: vi.fn(),
  getGroups: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  default: { getAlertEvent, listAlertEvents },
  opsAPI: { getAlertEvent, listAlertEvents },
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { groups: { getAll: getGroups } },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => (
        key === 'admin.ops.alertEvents.detail.open'
          ? `View alert details: ${params?.title ?? ''}`
          : key
      ),
    }),
  }
})

const SelectStub = defineComponent({
  name: 'OpsSelectStub',
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number, Boolean], default: '' },
    options: { type: Array, default: () => [] },
  },
  emits: ['change', 'update:modelValue'],
  template: '<div data-testid="select-stub" :class="$attrs.class" />',
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: Boolean },
  template: '<div v-if="show"><slot /></div>',
})

const alertEvent = {
  id: 17,
  rule_id: 3,
  severity: 'P1',
  status: 'firing',
  title: 'Upstream latency warning',
  description: 'P99 crossed the warning threshold',
  fired_at: '2026-07-21T03:00:00Z',
  created_at: '2026-07-21T03:00:00Z',
  dimensions: { platform: 'openai' },
  email_sent: true,
}

describe('OpsAlertEventsCard responsive controls', () => {
  beforeEach(() => {
    getAlertEvent.mockReset().mockResolvedValue(alertEvent)
    listAlertEvents.mockReset().mockResolvedValue([alertEvent])
    getGroups.mockReset().mockResolvedValue([])
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('uses a compact two-column filter rail without changing the filter request contract', async () => {
    const wrapper = mount(OpsAlertEventsCard, {
      global: {
        stubs: {
          Select: SelectStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
        },
      },
    })
    await flushPromises()

    const filters = wrapper.get('[data-testid="ops-alert-filters"]')
    expect(filters.classes()).toEqual(expect.arrayContaining(['grid', 'grid-cols-2']))

    const selects = filters.findAllComponents(SelectStub)
    expect(selects).toHaveLength(6)
    for (const select of selects) {
      expect(select.classes()).toContain('min-w-0')
    }

    listAlertEvents.mockClear()
    selects[3].vm.$emit('change', 'P0')
    await flushPromises()

    expect(listAlertEvents).toHaveBeenCalledWith(
      {
        limit: 10,
        time_range: '1h',
        severity: 'P0',
      },
      { signal: expect.any(AbortSignal) },
    )
  })

  it('keeps the alert queue keyboard-focusable and exposes native row buttons', async () => {
    const wrapper = mount(OpsAlertEventsCard, {
      global: {
        stubs: {
          Select: SelectStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
        },
      },
    })
    await flushPromises()

    const scrollRegion = wrapper.get('[data-testid="ops-alert-queue-scroll"]')
    expect(scrollRegion.classes()).toContain('ops-alert-queue-scroll')
    expect(scrollRegion.attributes()).toMatchObject({
      role: 'region',
      tabindex: '0',
      'aria-label': 'admin.ops.alertEvents.title',
    })
    expect(scrollRegion.find('ul[role="list"]').exists()).toBe(true)
    expect(scrollRegion.get('[data-testid="ops-alert-detail-trigger"]').element.tagName).toBe('BUTTON')
  })

  it('uses a native detail button so Enter and Space retain their standard activation behavior', async () => {
    const wrapper = mount(OpsAlertEventsCard, {
      attachTo: document.body,
      global: {
        stubs: {
          Select: SelectStub,
          BaseDialog: BaseDialogStub,
          Icon: true,
        },
      },
    })
    await flushPromises()

    const trigger = wrapper.get('[data-testid="ops-alert-detail-trigger"]')
    expect(trigger.element.tagName).toBe('BUTTON')
    expect(trigger.attributes()).toMatchObject({
      type: 'button',
      'aria-labelledby': 'ops-alert-title-17',
      'aria-describedby': 'ops-alert-status-17 ops-alert-description-17 ops-alert-context-17',
      'aria-controls': 'ops-alert-inspector',
      'data-alert-id': '17',
    })
    expect(wrapper.get('#ops-alert-title-17').text()).toBe('Upstream latency warning')

    const triggerButton = trigger.element as HTMLButtonElement
    triggerButton.focus()
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('click')
    await flushPromises()
    expect(getAlertEvent).toHaveBeenCalledTimes(1)
    expect(getAlertEvent).toHaveBeenCalledWith(17)

    wrapper.unmount()
  })

  it('stays mounted but suppresses requests when alert display is disabled', async () => {
    const wrapper = mount(OpsAlertEventsCard, {
      props: { enabled: false },
      global: { stubs: { Select: SelectStub, Icon: true } },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="ops-alert-disabled"]').attributes('role')).toBe('status')
    expect(listAlertEvents).not.toHaveBeenCalled()

    await wrapper.setProps({ enabled: true })
    await flushPromises()
    expect(listAlertEvents).toHaveBeenCalledTimes(1)
  })
})
