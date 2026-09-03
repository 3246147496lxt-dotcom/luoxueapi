import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsSystemLogTable from '../OpsSystemLogTable.vue'
import enLocale from '@/i18n/locales/en'
import zhLocale from '@/i18n/locales/zh'

const mockListSystemLogs = vi.fn()
const mockCleanupSystemLogs = vi.fn()
const mockGetSystemLogSinkHealth = vi.fn()
const mockGetRuntimeLogConfig = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listSystemLogs: (...args: any[]) => mockListSystemLogs(...args),
    cleanupSystemLogs: (...args: any[]) => mockCleanupSystemLogs(...args),
    getSystemLogSinkHealth: (...args: any[]) => mockGetSystemLogSinkHealth(...args),
    getRuntimeLogConfig: (...args: any[]) => mockGetRuntimeLogConfig(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

const SelectStub = defineComponent({
  name: 'SelectControlStub',
  props: {
    modelValue: {
      type: [String, Number],
      default: '',
    },
  },
  emits: ['update:modelValue'],
  template: '<div class="select-stub" />',
})

const PaginationStub = defineComponent({
  name: 'PaginationStub',
  template: '<div class="pagination-stub" />',
})

const runtimeConfig = {
  level: 'info',
  enable_sampling: false,
  sampling_initial: 100,
  sampling_thereafter: 100,
  caller: true,
  stacktrace_level: 'error',
  retention_days: 30,
}

const sinkHealth = {
  queue_depth: 0,
  queue_capacity: 5000,
  dropped_count: 0,
  write_failed_count: 0,
  written_count: 1,
  avg_write_delay_ms: 0,
}

describe('OpsSystemLogTable host support', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    mockListSystemLogs.mockResolvedValue({
      items: [
        {
          id: 1,
          created_at: '2026-07-14T00:10:01Z',
          host: 'api-node-1',
          level: 'warn',
          component: 'app',
          message: 'request failed',
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
    })
    mockCleanupSystemLogs.mockResolvedValue({ deleted: 1 })
    mockGetSystemLogSinkHealth.mockResolvedValue(sinkHealth)
    mockGetRuntimeLogConfig.mockResolvedValue(runtimeConfig)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('renders the host and sends it with list and cleanup filters', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('api-node-1')

    expect(wrapper.find('[data-testid="ops-log-advanced-filters"]').exists()).toBe(false)
    await wrapper.get('[data-testid="ops-log-advanced-toggle"]').trigger('click')
    await wrapper.get('[data-testid="system-log-host"]').setValue(' api-node-2 ')

    await wrapper.get('[data-testid="system-log-search"]').trigger('click')
    await flushPromises()

    expect(mockListSystemLogs).toHaveBeenLastCalledWith(expect.objectContaining({ host: 'api-node-2' }))

    await wrapper.get('[data-testid="ops-log-management-toggle"]').trigger('click')
    await wrapper.get('[data-testid="system-log-cleanup"]').trigger('click')
    await flushPromises()

    expect(mockCleanupSystemLogs).toHaveBeenCalledWith(expect.objectContaining({ host: 'api-node-2' }))
  })

  it.each([
    ['zh', zhLocale],
    ['en', enLocale],
  ])('defines the Host translation for %s', (_name, locale) => {
    expect(locale.admin.ops.systemLogs.host).toBe('Host')
  })

  it('applies the exact alert investigation filters while keeping group and region as context only', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      props: {
        platformFilter: 'anthropic'
      },
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub
        }
      }
    })
    await flushPromises()

    await wrapper.get('[data-testid="system-log-time-range"]').setValue('7d')
    await wrapper.get('[data-testid="system-log-keyword"]').setValue('stale search')
    await wrapper.get('[data-testid="system-log-component"]').setValue('stale-component')
    wrapper.findComponent(SelectStub).vm.$emit('update:modelValue', 'warn')
    await wrapper.get('[data-testid="ops-log-advanced-toggle"]').trigger('click')
    await wrapper.get('[data-testid="system-log-host"]').setValue('stale-host')
    await wrapper.get('[data-testid="system-log-platform"]').setValue('stale-platform')
    await wrapper.get('[data-testid="system-log-model"]').setValue('stale-model')
    await wrapper.get('[data-testid="system-log-request-id"]').setValue('stale-request')
    await wrapper.get('[data-testid="system-log-client-request-id"]').setValue('stale-client-request')
    await wrapper.get('[data-testid="system-log-user-id"]').setValue('101')
    await wrapper.get('[data-testid="system-log-api-key-id"]').setValue('202')
    await wrapper.get('[data-testid="system-log-account-id"]').setValue('303')

    await wrapper.setProps({
      investigationPreset: {
        key: 1,
        alertId: 27,
        platform: ' openai ',
        startTime: '2026-07-14T00:00:00.000Z',
        endTime: '2026-07-14T01:00:00.000Z',
        requestId: ' req-alert-27 ',
        groupId: 42,
        region: 'us-east-1'
      }
    })
    await flushPromises()

    const query = mockListSystemLogs.mock.calls.at(-1)?.[0]
    expect(query).toEqual({
      page: 1,
      page_size: 50,
      time_range: '1h',
      platform: 'openai',
      request_id: 'req-alert-27',
      start_time: '2026-07-14T00:00:00.000Z',
      end_time: '2026-07-14T01:00:00.000Z'
    })
    expect(query).not.toHaveProperty('group_id')
    expect(query).not.toHaveProperty('groupId')
    expect(query).not.toHaveProperty('region')

    const context = wrapper.get('[data-testid="ops-log-investigation-context"]')
    expect(context.text()).toContain('#27')
    expect(context.text()).toContain('platform=openai')
    expect(context.text()).toContain('request_id=req-alert-27')
    expect(context.text()).toContain('group_id=42')
    expect(context.text()).toContain('region=us-east-1')
  })

  it('materializes the selected relative range into an immutable cleanup window', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-14T12:00:00.000Z'))

    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    await wrapper.get('[data-testid="system-log-time-range"]').setValue('6h')

    await wrapper.get('[data-testid="ops-log-management-toggle"]').trigger('click')
    await wrapper.get('[data-testid="system-log-cleanup"]').trigger('click')
    await flushPromises()

    expect(mockCleanupSystemLogs).toHaveBeenCalledTimes(1)
    const cleanupSnapshot = mockCleanupSystemLogs.mock.calls[0]?.[0]
    expect(cleanupSnapshot).toEqual({
      start_time: '2026-07-14T06:00:00.000Z',
      end_time: '2026-07-14T12:00:00.000Z',
    })
    expect(cleanupSnapshot).not.toHaveProperty('time_range')

    const confirmation = vi.mocked(window.confirm).mock.calls[0]?.[0]
    expect(confirmation).toContain('start_time=2026-07-14T06:00:00.000Z')
    expect(confirmation).toContain('end_time=2026-07-14T12:00:00.000Z')
    expect(confirmation).not.toContain('time_range=')
  })

  it('selects rows by pointer or keyboard and renders the selected log in the inline inspector', async () => {
    const firstLog = {
      id: 9,
      created_at: '2026-07-14T00:10:01Z',
      host: 'api-node-1',
      level: 'warn',
      component: 'gateway',
      message: 'upstream latency exceeded',
      request_id: 'req-first',
      platform: 'openai',
      model: 'gpt-test',
      extra: {
        method: 'POST',
        path: '/v1/chat/completions',
        latency_ms: 4200,
        error: 'upstream timeout',
      },
    }
    const secondLog = {
      id: 10,
      created_at: '2026-07-14T00:11:01Z',
      host: 'api-node-2',
      level: 'error',
      component: 'database',
      message: 'database timeout',
      request_id: 'req-second',
      extra: { error: { code: 'DB_TIMEOUT' } },
    }
    mockListSystemLogs.mockResolvedValueOnce({
      items: [firstLog, secondLog],
      total: 2,
      page: 1,
      page_size: 20,
    })

    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="ops-log-inspector"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="ops-log-split"]').classes()).not.toContain('ops-log-split--inspecting')

    const firstRow = wrapper.get('[data-testid="system-log-row-9"]')
    await firstRow.trigger('click')

    expect(firstRow.attributes('aria-selected')).toBe('true')
    expect(wrapper.get('[data-testid="ops-log-split"]').classes()).toContain('ops-log-split--inspecting')
    let inspector = wrapper.get('[data-testid="ops-log-inspector-content"]')
    expect(inspector.text()).toContain('upstream latency exceeded')
    expect(inspector.text()).toContain('req-first')
    expect(inspector.text()).toContain('/v1/chat/completions')
    expect(inspector.text()).toContain('upstream timeout')

    const secondRow = wrapper.get('[data-testid="system-log-row-10"]')
    await secondRow.trigger('keydown', { key: 'Enter' })

    expect(firstRow.attributes('aria-selected')).toBe('false')
    expect(secondRow.attributes('aria-selected')).toBe('true')
    inspector = wrapper.get('[data-testid="ops-log-inspector-content"]')
    expect(inspector.text()).toContain('database timeout')
    expect(inspector.text()).toContain('req-second')
    expect(inspector.text()).toContain('DB_TIMEOUT')

    await wrapper.get('[data-testid="ops-log-open-request-details"]').trigger('click')
    expect(wrapper.emitted('open-request-details')?.[0]?.[0]).toEqual(secondLog)

    await wrapper.get('[data-testid="ops-log-inspector-close"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="ops-log-inspector"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="ops-log-split"]').classes()).not.toContain('ops-log-split--inspecting')
    expect(secondRow.attributes('aria-selected')).toBe('false')
  })

  it('keeps optional investigation controls collapsed by default', async () => {
    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="ops-log-advanced-filters"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-log-management"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-log-inspector"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ops-log-query-toolbar"]').exists()).toBe(true)
  })

  it.each([
    ['healthy', sinkHealth],
    ['warning', { ...sinkHealth, dropped_count: 2 }],
    ['danger', { ...sinkHealth, write_failed_count: 1 }],
    ['danger', { ...sinkHealth, last_error: 'disk unavailable' }],
  ])('derives the %s collection health tone from sink counters', async (tone, healthResponse) => {
    mockGetSystemLogSinkHealth.mockResolvedValueOnce(healthResponse)
    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="ops-log-health-status"]').attributes('data-health-tone')).toBe(tone)
  })

  it('uses an unknown health tone when sink health is unavailable', async () => {
    mockGetSystemLogSinkHealth.mockRejectedValueOnce(new Error('unavailable'))
    const wrapper = mount(OpsSystemLogTable, {
      global: {
        stubs: {
          Select: SelectStub,
          Pagination: PaginationStub,
        },
      },
    })
    await flushPromises()

    expect(wrapper.get('[data-testid="ops-log-health-status"]').attributes('data-health-tone')).toBe('unknown')
  })
})
