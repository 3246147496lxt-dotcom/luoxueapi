import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

import UsageFilters from '../UsageFilters.vue'

// --- i18n messages (only what UsageFilters needs) ---
const messages: Record<string, string> = {
  'admin.usage.userDeletedBadge': 'deleted',
  'admin.usage.userFilter': 'User',
  'admin.usage.searchUserPlaceholder': 'Search user...',
  'usage.apiKeyFilter': 'API Key',
  'admin.usage.searchApiKeyPlaceholder': 'Search API key...',
  'usage.model': 'Model',
  'admin.usage.allModels': 'All Models',
  'admin.usage.account': 'Account',
  'admin.usage.searchAccountPlaceholder': 'Search account...',
  'usage.type': 'Type',
  'admin.usage.allTypes': 'All Types',
  'usage.ws': 'WS',
  'usage.stream': 'Stream',
  'usage.sync': 'Sync',
  'admin.usage.billingType': 'Billing Type',
  'admin.usage.allBillingTypes': 'All Billing Types',
  'admin.usage.billingTypeBalance': 'Balance',
  'admin.usage.billingTypeSubscription': 'Subscription',
  'admin.usage.billingMode': 'Billing Mode',
  'admin.usage.allBillingModes': 'All Billing Modes',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per Request',
  'admin.usage.billingModeImage': 'Image',
  'admin.usage.upstreamModelAudit': 'Upstream model audit',
  'admin.usage.allUpstreamModelAudit': 'All response model states',
  'admin.usage.upstreamModelMismatchOnly': 'Mismatched only',
  'admin.usage.upstreamModelMatchedOnly': 'Matched only',
  'admin.ops.errorLog.type': 'Error phase',
  'usage.errors.category': 'Error category',
  'admin.ops.errorLog.status': 'Status',
  'admin.usage.group': 'Group',
  'admin.usage.allGroups': 'All Groups',
  'common.refresh': 'Refresh',
  'common.reset': 'Reset',
  'admin.usage.cleanup.button': 'Cleanup',
  'usage.exportExcel': 'Export',
}

// Mock vue-i18n
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

// Mock the admin API module — we control searchUsers return value per test
const mockSearchUsers = vi.fn()
const mockSearchApiKeys = vi.fn().mockResolvedValue([])
const mockGroupsList = vi.fn().mockResolvedValue({ items: [] })
const mockGetModelStats = vi.fn().mockResolvedValue({ models: [] })
const mockAccountsList = vi.fn().mockResolvedValue({ items: [] })

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      searchUsers: (...args: any[]) => mockSearchUsers(...args),
      searchApiKeys: (...args: any[]) => mockSearchApiKeys(...args),
    },
    groups: { list: (...args: any[]) => mockGroupsList(...args) },
    dashboard: { getModelStats: (...args: any[]) => mockGetModelStats(...args) },
    accounts: { list: (...args: any[]) => mockAccountsList(...args) },
  },
}))

// Default props helper
const defaultFilters = () => ({
  user_id: undefined,
  api_key_id: undefined,
  account_id: undefined,
  model: null,
  request_type: null,
  billing_type: null,
  billing_mode: null,
  upstream_model_mismatch: null,
  group_id: null,
  start_date: '',
  end_date: '',
})

function mountFilters(filters = defaultFilters()) {
  return mount(UsageFilters, {
    props: {
      modelValue: filters,
      exporting: false,
      startDate: '2026-05-01',
      endDate: '2026-05-28',
      showActions: false,
      modelOptions: [],
    },
    global: {
      stubs: {
        Select: true,
        Teleport: true,
      },
    },
  })
}

describe('UsageFilters — layout variants', () => {
  it('keeps toolbar as the default and removes card framing in rail mode', async () => {
    const wrapper = mountFilters()

    expect(wrapper.classes()).toContain('usage-filters--toolbar')
    expect(wrapper.classes()).toContain('usage-filters--card')
    expect(wrapper.classes()).toContain('card')

    await wrapper.setProps({ layout: 'rail' })

    expect(wrapper.classes()).toContain('usage-filters--rail')
    expect(wrapper.classes()).not.toContain('usage-filters--toolbar')
    expect(wrapper.classes()).not.toContain('usage-filters--card')
    expect(wrapper.classes()).not.toContain('card')
  })

  it('keeps search input ids unique and labels scoped to each instance', () => {
    const firstFilters = defaultFilters()
    const secondFilters = defaultFilters()
    const Host = defineComponent({
      setup() {
        return () => h('div', [
          h(UsageFilters, {
            modelValue: firstFilters,
            exporting: false,
            startDate: '2026-05-01',
            endDate: '2026-05-28',
            showActions: false,
          }),
          h(UsageFilters, {
            modelValue: secondFilters,
            exporting: false,
            startDate: '2026-05-01',
            endDate: '2026-05-28',
            showActions: false,
          }),
        ])
      },
    })
    const wrapper = mount(Host, {
      global: {
        stubs: {
          Select: true,
          Teleport: true,
        },
      },
    })
    const instances = wrapper.findAllComponents(UsageFilters)
    const allInputIds: string[] = []

    expect(instances).toHaveLength(2)
    instances.forEach((instance) => {
      const dropdowns = instance.findAll('.usage-filter-dropdown')
      expect(dropdowns).toHaveLength(3)

      dropdowns.forEach((dropdown) => {
        const label = dropdown.get('label').element as HTMLLabelElement
        const input = dropdown.get('input').element as HTMLInputElement
        const instanceElement = instance.element as HTMLElement
        const target = Array.from(instanceElement.querySelectorAll<HTMLInputElement>('input'))
          .find((candidate) => candidate.id === label.htmlFor)

        expect(input.id).not.toBe('')
        expect(label.htmlFor).toBe(input.id)
        expect(target).toBe(input)
        allInputIds.push(input.id)
      })
    })

    expect(allInputIds).toHaveLength(6)
    expect(new Set(allInputIds).size).toBe(allInputIds.length)
  })

  it('gives every custom select a field-level accessible name', () => {
    const SelectAriaStub = defineComponent({
      props: {
        ariaLabel: String,
      },
      setup(props) {
        return () => h('button', {
          class: 'select-aria-stub',
          'aria-label': props.ariaLabel,
        })
      },
    })
    const expectedUsageLabels = ['Model', 'Type', 'Billing Type', 'Billing Mode', 'Upstream model audit', 'Group']
    const expectedErrorLabels = ['Model', 'Error phase', 'Error category', 'Status', 'Group']

    const usage = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
      },
      global: { stubs: { Select: SelectAriaStub, Teleport: true } },
    })
    const errors = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        mode: 'errors',
      },
      global: { stubs: { Select: SelectAriaStub, Teleport: true } },
    })

    expect(usage.findAll('.select-aria-stub').map((select) => select.attributes('aria-label')))
      .toEqual(expectedUsageLabels)
    expect(errors.findAll('.select-aria-stub').map((select) => select.attributes('aria-label')))
      .toEqual(expectedErrorLabels)
  })

  it('keeps response-model audit as an explicit three-state filter', () => {
    const wrapper = mountFilters()
    const options = (wrapper.vm as any).upstreamModelMismatchOptions as Array<{
      value: boolean | null
      label: string
    }>

    expect(options).toEqual([
      { value: null, label: 'All response model states' },
      { value: true, label: 'Mismatched only' },
      { value: false, label: 'Matched only' },
    ])
  })
})

describe('UsageFilters — user search dropdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockSearchUsers.mockReset()
    mockSearchApiKeys.mockResolvedValue([])
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('(a) labels deleted users with the i18n badge and (b) sorts active users before deleted ones, (c) selection sets user_id', async () => {
    // Arrange: mock returns deleted FIRST (proves sorting re-orders to active-first)
    mockSearchUsers.mockResolvedValue([
      { id: 2, email: 'gone@test.com', deleted: true },
      { id: 1, email: 'active@test.com', deleted: false },
    ])

    const wrapper = mountFilters()

    // Trigger focus (sets showUserDropdown = true) then input (fires debounceUserSearch)
    const input = wrapper.find('input[type="text"]')
    await input.trigger('focus')
    await input.setValue('test')
    await input.trigger('input')

    // Advance debounce timer (300ms) then flush the resolved promise
    vi.advanceTimersByTime(300)
    await flushPromises()

    // --- (b) Sort: active user should appear BEFORE deleted user ---
    // Check the underlying component state via rendered DOM order
    const buttons = wrapper.findAll('.usage-filter-dropdown button[type="button"]')
    const emailTexts = buttons.map((b) => b.text())

    // active@test.com should be listed first
    const activeIdx = emailTexts.findIndex((t) => t.includes('active@test.com'))
    const deletedIdx = emailTexts.findIndex((t) => t.includes('gone@test.com'))
    expect(activeIdx).toBeGreaterThanOrEqual(0)
    expect(deletedIdx).toBeGreaterThanOrEqual(0)
    expect(activeIdx).toBeLessThan(deletedIdx)

    // --- (a) Label: deleted user's button shows the badge text ---
    const deletedButton = buttons[deletedIdx]
    expect(deletedButton.text()).toContain('deleted')

    // active user's button does NOT show the badge text
    const activeButton = buttons[activeIdx]
    expect(activeButton.text()).not.toContain('deleted')

    // --- (c) Selection: clicking active user button sets filters.user_id ---
    await activeButton.trigger('click')
    await flushPromises()

    // The component emits 'update:modelValue' or modifies filters.user_id via toRef
    // selectUser sets filters.value.user_id = u.id and emits 'change'
    const changeEmits = wrapper.emitted('change')
    expect(changeEmits).toBeTruthy()
    expect(changeEmits!.length).toBeGreaterThan(0)

    // Also confirm user_id was set by checking the emitted change came through
    // (the component uses toRef so modelValue is mutated in place and 'change' is emitted)
    expect(wrapper.props('modelValue').user_id).toBe(1)
  })
})

describe('UsageFilters — model options come from prop (no dup request)', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetModelStats.mockClear()
    mockGroupsList.mockClear()
  })
  afterEach(() => { vi.useRealTimers() })

  it('does not call dashboard.getModelStats on mount and renders model options from prop', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: defaultFilters(),
        exporting: false,
        startDate: '2026-05-01',
        endDate: '2026-05-28',
        showActions: false,
        modelOptions: ['claude-3', 'gpt-4o'],
      },
      global: { stubs: { Select: true, Teleport: true } },
    })
    await flushPromises()

    expect(mockGetModelStats).not.toHaveBeenCalled()

    const opts = (wrapper.vm as any).modelOptions as Array<{ value: string | null; label: string }>
    expect(opts.map((o) => o.value)).toEqual([null, 'claude-3', 'gpt-4o'])
  })
})
