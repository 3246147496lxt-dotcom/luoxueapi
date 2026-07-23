import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import OpsAlertEventsCard from '../OpsAlertEventsCard.vue'

const { getAlertEvent, listAlertEvents, showError, showSuccess } = vi.hoisted(() => ({
  getAlertEvent: vi.fn(),
  listAlertEvents: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  default: { getAlertEvent, listAlertEvents },
  opsAPI: { getAlertEvent, listAlertEvents },
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
  emits: ['change'],
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
    showError.mockReset()
    showSuccess.mockReset()
  })

  it('lets the filter toolbar wrap without changing the filter request contract', async () => {
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
    expect(filters.classes()).toEqual(expect.arrayContaining(['w-full', 'flex-wrap']))

    const selects = wrapper.findAllComponents(SelectStub)
    expect(selects).toHaveLength(4)
    expect(selects.every((select) => select.classes().includes('flex-1'))).toBe(true)

    listAlertEvents.mockClear()
    selects[1].vm.$emit('change', 'P0')
    await flushPromises()

    expect(listAlertEvents).toHaveBeenCalledWith({
      limit: 10,
      time_range: '24h',
      severity: 'P0',
    })
  })

  it('keeps the wide alert table keyboard-focusable and horizontally scrollable', async () => {
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

    const scrollRegion = wrapper.get('[data-testid="ops-alert-table-scroll"]')
    expect(scrollRegion.classes()).toContain('overflow-auto')
    expect(scrollRegion.classes()).not.toContain('overflow-y-auto')
    expect(scrollRegion.attributes()).toMatchObject({
      role: 'region',
      tabindex: '0',
      'aria-label': 'admin.ops.alertEvents.title',
    })
    expect(scrollRegion.find('table').exists()).toBe(true)
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
      'aria-label': 'View alert details: Upstream latency warning',
      'data-alert-id': '17',
    })

    const triggerButton = trigger.element as HTMLButtonElement
    triggerButton.focus()
    expect(document.activeElement).toBe(trigger.element)

    await trigger.trigger('click')
    await flushPromises()
    expect(getAlertEvent).toHaveBeenCalledTimes(1)
    expect(getAlertEvent).toHaveBeenCalledWith(17)

    wrapper.unmount()
  })
})
