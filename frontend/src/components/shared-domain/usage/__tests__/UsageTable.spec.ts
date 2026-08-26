type MockIpGeoEntry = {
  status: 'idle' | 'success'
  label?: string
  detail?: Record<string, unknown>
}

const ipGeoMocks = vi.hoisted(() => ({
  getEntry: vi.fn((): MockIpGeoEntry => ({ status: 'idle' })),
  fetchOne: vi.fn(),
  fetchBatch: vi.fn(),
}))

vi.mock('@/utils/ipGeoLookup', () => ipGeoMocks)

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import UsageTable from '@/components/shared-domain/usage/UsageTable.vue'
import type { AdminUsageLog } from '@/types'

const messages: Record<string, string> = {
  'admin.usage.userDeletedBadge': 'Deleted',
  'usage.costDetails': 'Cost Breakdown',
  'admin.usage.inputCost': 'Input Cost',
  'admin.usage.outputCost': 'Output Cost',
  'admin.usage.cacheCreationCost': 'Cache Creation Cost',
  'admin.usage.cacheReadCost': 'Cache Read Cost',
  'usage.inputTokenPrice': 'Input price',
  'usage.outputTokenPrice': 'Output price',
  'usage.perMillionTokens': '/ 1M tokens',
  'usage.serviceTier': 'Service tier',
  'usage.serviceTierPriority': 'Fast',
  'usage.serviceTierFlex': 'Flex',
  'usage.serviceTierStandard': 'Standard',
  'usage.rate': 'Rate',
  'usage.accountMultiplier': 'Account rate',
  'usage.original': 'Original',
  'usage.userBilled': 'User billed',
  'usage.chargedAmount': 'Charged',
  'usage.sourceWebChat': 'GPT Chat',
  'usage.sourceApi': 'API',
  'usage.requestedModel': 'Requested',
  'usage.sentUpstreamModel': 'Sent upstream',
  'usage.upstreamResponseModel': 'Upstream response',
  'usage.modelVariant': 'Possible version variant',
  'usage.modelMismatch': 'Different model',
  'usage.accountBilled': 'Account billed',
  'usage.imageUnit': ' images',
  'usage.imageCount': 'Image count',
  'usage.imageBillingSize': 'Billing size',
  'usage.imageInputSize': 'Input size',
  'usage.imageOutputSize': 'Output size',
  'usage.imageSizeSource': 'Size source',
  'usage.imageSizeBreakdown': 'Size breakdown',
  'usage.imageSizeSourceOutput': 'Upstream output',
  'usage.imageSizeSourceInput': 'Request input',
  'usage.imageSizeSourceDefault': 'Default billing tier',
  'usage.imageSizeSourceLegacy': 'Legacy record',
  'usage.imageSizeSourceMissing': 'Not recorded',
  'usage.imageSizeNotRecorded': 'not recorded',
  'usage.imageSizeLegacyUnstandardized': 'legacy unstandardized',
  'usage.imageSizeUnknown': 'unknown',
  'usage.imageUnitPrice': 'Per-image price',
  'usage.imageTotalPrice': 'Image total price',
  'admin.usage.billingModeToken': 'Token',
  'admin.usage.billingModePerRequest': 'Per request',
  'admin.usage.billingModeImage': 'Image',
  'admin.usage.workspace.table.deactivated': 'Deactivated',
  'admin.usage.workspace.table.input': 'In',
  'admin.usage.workspace.table.output': 'Out',
  'admin.usage.workspace.table.total': 'Total',
  'admin.usage.workspace.table.cost': 'Cost',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.request_id">
        <slot name="cell-source" :row="row" />
        <slot name="cell-request_id" :row="row" />
        <slot name="cell-model" :row="row" :value="row.model" />
        <slot name="cell-billing_mode" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
      </div>
    </div>
  `,
}

function makeUsageRow(overrides: Partial<AdminUsageLog> = {}): AdminUsageLog {
  return {
    id: 1,
    user_id: 1,
    api_key_id: 1,
    account_id: null,
    request_id: 'req-admin-default',
    model: 'claude-3',
    group_id: null,
    subscription_id: null,
    input_tokens: 0,
    output_tokens: 0,
    cache_creation_tokens: 0,
    cache_read_tokens: 0,
    cache_creation_5m_tokens: 0,
    cache_creation_1h_tokens: 0,
    input_cost: 0,
    output_cost: 0,
    cache_creation_cost: 0,
    cache_read_cost: 0,
    total_cost: 0,
    actual_cost: 0,
    rate_multiplier: 1,
    long_context_billing_applied: false,
    billing_type: 0,
    stream: false,
    duration_ms: null,
    first_token_ms: null,
    image_count: 0,
    image_size: null,
    image_input_size: null,
    image_output_size: null,
    image_size_source: null,
    image_size_breakdown: null,
    image_input_tokens: 0,
    image_input_cost: 0,
    image_output_tokens: 0,
    image_output_cost: 0,
    user_agent: null,
    cache_ttl_overridden: false,
    created_at: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

const baseImageRow = makeUsageRow({
  request_id: 'req-admin-image',
  model: 'gpt-image-2',
  actual_cost: 0.4,
  total_cost: 0.4,
  account_rate_multiplier: 1,
  rate_multiplier: 1,
  service_tier: null,
  input_cost: 0,
  output_cost: 0,
  cache_creation_cost: 0,
  cache_read_cost: 0,
  input_tokens: 0,
  output_tokens: 0,
  cache_creation_tokens: 0,
  cache_read_tokens: 0,
  cache_creation_5m_tokens: 0,
  cache_creation_1h_tokens: 0,
  cache_ttl_overridden: false,
  billing_mode: 'image',
  image_count: 2,
  image_size: '2K',
  image_input_size: null,
  image_output_size: null,
  image_size_source: null,
  image_size_breakdown: null,
})

describe('admin UsageTable tooltip', () => {
  beforeEach(() => {
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      x: 0,
      y: 0,
      top: 20,
      left: 20,
      right: 120,
      bottom: 40,
      width: 100,
      height: 20,
      toJSON: () => ({}),
    } as DOMRect)
  })

  it('marks only usage rows that actually applied long-context billing', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [
          {
            ...baseImageRow,
            request_id: 'req-long-context-enabled',
            long_context_billing_applied: true,
          },
          {
            ...baseImageRow,
            request_id: 'req-long-context-disabled',
            long_context_billing_applied: false,
          },
        ],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.findAll('[data-testid="long-context-billing-marker"]')).toHaveLength(1)
    expect(wrapper.get('[data-testid="long-context-billing-marker"]').text()).toBe('x2')
  })

  it('shows service tier and billing breakdown in cost tooltip', async () => {
    const row = makeUsageRow({
      request_id: 'req-admin-1',
      actual_cost: 0.092883,
      total_cost: 0.092883,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      service_tier: 'priority',
      input_cost: 0.020285,
      output_cost: 0.00303,
      cache_creation_cost: 0,
      cache_read_cost: 0.069568,
      input_tokens: 4057,
      output_tokens: 101,
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    const tooltipTriggers = wrapper.findAll('.group.relative')
    await tooltipTriggers[tooltipTriggers.length - 1].trigger('mouseenter')
    await nextTick()

    const text = wrapper.text()
    expect(text).toContain('Service tier')
    expect(text).toContain('Fast')
    expect(text).toContain('Rate')
    expect(text).toContain('1.00x')
    expect(text).toContain('Account rate')
    expect(text).toContain('Charged')
    expect(text).toContain('Account billed')
    expect(text).toContain('$0.092883')
    expect(text).toContain('$5.0000 / 1M tokens')
    expect(text).toContain('$30.0000 / 1M tokens')
    expect(text).toContain('$0.069568')
  })

  it('shows requested and upstream models separately for admin rows', () => {
    const row = makeUsageRow({
      request_id: 'req-admin-model-1',
      model: 'claude-sonnet-4',
      upstream_model: 'claude-sonnet-4-20250514',
      actual_cost: 0,
      total_cost: 0,
      account_rate_multiplier: 1,
      rate_multiplier: 1,
      input_cost: 0,
      output_cost: 0,
      cache_creation_cost: 0,
      cache_read_cost: 0,
      input_tokens: 0,
      output_tokens: 0,
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('claude-sonnet-4')
    expect(text).toContain('claude-sonnet-4-20250514')
  })

  it('shows response-model audit evidence only for explicit mismatches', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [
          makeUsageRow({
            request_id: 'req-model-audit-mismatch',
            model: 'gpt-5.4',
            upstream_model: 'gpt-5.4',
            upstream_response_model: 'gpt-5.4-2026-08-01',
            upstream_model_mismatch: true,
          }),
          makeUsageRow({
            request_id: 'req-model-audit-match',
            model: 'gpt-5.4',
            upstream_response_model: 'gpt-5.4',
            upstream_model_mismatch: false,
          }),
          makeUsageRow({
            request_id: 'req-model-audit-unobserved',
            model: 'gpt-5.4',
            upstream_response_model: 'private-unobserved-value',
            upstream_model_mismatch: null,
          }),
        ],
        loading: false,
        columns: [],
        showUpstreamModelAudit: true,
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    const auditRows = wrapper.findAll('[data-testid="upstream-response-model-audit"]')
    expect(auditRows).toHaveLength(1)
    expect(auditRows[0].text()).toContain('gpt-5.4-2026-08-01')
    expect(auditRows[0].text()).toContain('Possible version variant')
    expect(wrapper.text()).not.toContain('private-unobserved-value')
  })

  it('distinguishes a different response model from a dated model variant', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [makeUsageRow({
          request_id: 'req-model-audit-different',
          model: 'claude-sonnet-4',
          upstream_model: 'claude-sonnet-4',
          upstream_response_model: 'claude-opus-4',
          upstream_model_mismatch: true,
        })],
        loading: false,
        columns: [],
        showUpstreamModelAudit: true,
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.get('[data-testid="upstream-response-model-audit"]').text()).toContain('Different model')
  })

  it('keeps raw upstream response models hidden unless the admin audit switch is explicit', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [makeUsageRow({
          request_id: 'req-user-facing-private',
          model: 'gpt-5.4',
          upstream_response_model: 'private-upstream-response-model',
          upstream_model_mismatch: true,
        })],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.find('[data-testid="upstream-response-model-audit"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('private-upstream-response-model')
  })

  it('renders Web Chat source, request ID, actual model, and server-reported usage cost', () => {
    const row = makeUsageRow({
      request_id: 'req-chat-usage-facts',
      source: 'web_chat',
      model: 'gpt-5.5',
      upstream_model: 'gpt-5.5-2026-07-01',
      total_cost: 0.024,
      actual_cost: 0.02,
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [],
        creditMode: true,
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('GPT Chat')
    expect(text).toContain('req-chat-usage-facts')
    expect(text).toContain('gpt-5.5')
    expect(text).toContain('gpt-5.5-2026-07-01')
    expect(text).toContain('0.020000')
    expect(wrapper.findAll('[data-testid="credit-amount-value"]').map((value) => value.text()))
      .toEqual(['0.020000'])
    expect(text).not.toContain('$0.020000')
  })

  it.each([
    {
      name: 'defaulted row',
      row: makeUsageRow({
        ...baseImageRow,
        request_id: 'req-admin-default-image',
        image_size: '2K',
        image_input_size: 'auto',
        image_output_size: null,
        image_size_source: 'default',
      }),
      expected: ['2K', 'Default billing tier', 'auto', 'unknown'],
    },
    {
      name: 'output-sourced row',
      row: makeUsageRow({
        ...baseImageRow,
        request_id: 'req-admin-output-image',
        image_size: '4K',
        image_input_size: '1024x1024',
        image_output_size: '3840x2160',
        image_size_source: 'output',
        image_size_breakdown: { '4K': 1 },
      }),
      expected: ['4K', 'Upstream output', '1024x1024', '3840x2160', '4K x 1'],
    },
    {
      name: 'input-sourced row',
      row: makeUsageRow({
        ...baseImageRow,
        request_id: 'req-admin-input-image',
        image_size: '1K',
        image_input_size: '1024x1024',
        image_output_size: null,
        image_size_source: 'input',
      }),
      expected: ['1K', 'Request input', '1024x1024', 'unknown'],
    },
    {
      name: 'legacy unstandardized row',
      row: makeUsageRow({
        ...baseImageRow,
        request_id: 'req-admin-legacy-unstandardized-image',
        image_size: '512x512',
        image_input_size: null,
        image_output_size: null,
        image_size_source: null,
      }),
      expected: ['legacy unstandardized: 512x512', 'Legacy record', 'unknown'],
    },
  ])('shows image usage metadata for $name', async ({ row, expected }) => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    await wrapper.find('.group.relative').trigger('mouseenter')
    await nextTick()

    const text = wrapper.text()
    expect(text).toContain('Image count')
    expect(text).toContain('Billing size')
    expect(text).toContain('Size source')
    expect(text).toContain('Input size')
    expect(text).toContain('Output size')
    expect(text).toContain('Per-image price')
    expect(text).toContain('Image total price')
    for (const value of expected) {
      expect(text).toContain(value)
    }
  })

  it('displays historical image rows with missing billing_mode as image usage without a 2K fallback', async () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [
          {
            ...baseImageRow,
            request_id: 'req-admin-legacy-missing-image',
            billing_mode: null,
            image_size: null,
            image_input_size: null,
            image_output_size: null,
            image_size_source: null,
            image_size_breakdown: null,
          },
        ],
        loading: false,
        columns: [],
      },
      global: {
        stubs: {
          DataTable: DataTableStub,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    await wrapper.find('.group.relative').trigger('mouseenter')
    await nextTick()

    const text = wrapper.text()
    expect(text).toContain('Image')
    expect(text).toContain('Image count')
    expect(text).toContain('Per-image price')
    expect(text).toContain('not recorded')
    expect(text).not.toContain('(2K)')
  })
})

describe('admin UsageTable IP geolocation batch toolbar', () => {
  const DataTableStubWithIp = {
    props: ['data'],
    template: `
      <div>
        <div v-for="row in data" :key="row.request_id">
          <slot name="cell-ip_address" :row="row" />
        </div>
      </div>
    `,
  }

  beforeEach(() => {
    ipGeoMocks.getEntry.mockReset()
    ipGeoMocks.fetchOne.mockReset()
    ipGeoMocks.fetchBatch.mockReset()
    ipGeoMocks.getEntry.mockReturnValue({ status: 'idle' })
  })

  it('does not render the batch toolbar when the ip_address column is not visible', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [makeUsageRow({ request_id: 'r1', ip_address: '8.8.8.8' })],
        loading: false,
        columns: [],
      },
      global: { stubs: { DataTable: DataTableStubWithIp, EmptyState: true, Teleport: true } },
    })
    expect(wrapper.text()).not.toContain('usage.ipGeo.batchFetch')
  })

  it('renders the batch toolbar with a pending count when the ip_address column is visible', () => {
    const wrapper = mount(UsageTable, {
      props: {
        data: [
          makeUsageRow({ request_id: 'r1', ip_address: '8.8.8.8' }),
          makeUsageRow({ request_id: 'r2', ip_address: '8.8.8.8' }),
          makeUsageRow({ request_id: 'r3', ip_address: '1.1.1.1' }),
        ],
        loading: false,
        columns: [{ key: 'ip_address', label: 'IP' }],
      },
      global: { stubs: { DataTable: DataTableStubWithIp, EmptyState: true, Teleport: true } },
    })
    expect(wrapper.text()).toContain('usage.ipGeo.pending')
    const button = wrapper.find('button')
    expect(button.exists()).toBe(true)
    expect((button.element as HTMLButtonElement).disabled).toBe(false)
  })

  it('fetches deduplicated IPs from the current page when the batch button is clicked', async () => {
    ipGeoMocks.fetchBatch.mockResolvedValue(true)
    const wrapper = mount(UsageTable, {
      props: {
        data: [
          makeUsageRow({ request_id: 'r1', ip_address: '8.8.8.8' }),
          makeUsageRow({ request_id: 'r2', ip_address: '8.8.8.8' }),
          makeUsageRow({ request_id: 'r3', ip_address: '1.1.1.1' }),
        ],
        loading: false,
        columns: [{ key: 'ip_address', label: 'IP' }],
      },
      global: { stubs: { DataTable: DataTableStubWithIp, EmptyState: true, Teleport: true } },
    })
    await wrapper.find('button').trigger('click')
    expect(ipGeoMocks.fetchBatch).toHaveBeenCalledWith(['8.8.8.8', '1.1.1.1'])
    expect(wrapper.emitted('ipGeoBatchFailed')).toBeUndefined()
  })

  it('emits ipGeoBatchFailed when the batch request reports a network-level failure', async () => {
    ipGeoMocks.fetchBatch.mockResolvedValue(false)
    const wrapper = mount(UsageTable, {
      props: {
        data: [makeUsageRow({ request_id: 'r1', ip_address: '8.8.8.8' })],
        loading: false,
        columns: [{ key: 'ip_address', label: 'IP' }],
      },
      global: { stubs: { DataTable: DataTableStubWithIp, EmptyState: true, Teleport: true } },
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('ipGeoBatchFailed')).toHaveLength(1)
  })

  it('renders IpGeoCell content for ip_address cells', () => {
    ipGeoMocks.getEntry.mockReturnValue({ status: 'success', label: 'CN · Guangdong · Shenzhen', detail: {} })
    const wrapper = mount(UsageTable, {
      props: {
        data: [makeUsageRow({ request_id: 'r1', ip_address: '121.35.47.43' })],
        loading: false,
        columns: [{ key: 'ip_address', label: 'IP' }],
      },
      global: { stubs: { DataTable: DataTableStubWithIp, EmptyState: true, Teleport: true } },
    })
    expect(wrapper.text()).toContain('121.35.47.43')
    expect(wrapper.text()).toContain('CN · Guangdong · Shenzhen')
  })
})

// A DataTable stub that also renders cell-user, so the deleted badge can be asserted.
const DataTableStubWithUser = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.request_id">
        <slot name="cell-user" :row="row" />
        <slot name="cell-model" :row="row" :value="row.model" />
        <slot name="cell-billing_mode" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
      </div>
    </div>
  `,
}

const DataTableStubWithAuditCells = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.request_id">
        <slot name="cell-user" :row="row" />
        <slot name="cell-model" :row="row" />
        <slot name="cell-tokens" :row="row" />
        <slot name="cell-cost" :row="row" />
        <slot name="cell-latency" :row="row" />
        <slot name="cell-account" :row="row" />
        <slot name="cell-created_at" :row="row" :value="row.created_at" />
        <slot name="cell-ip_address" :row="row" />
      </div>
    </div>
  `,
}

describe('admin UsageTable Superdesign audit cells', () => {
  it('renders the eight-column composite hierarchy and keeps the username purple', () => {
    ipGeoMocks.getEntry.mockReturnValue({
      status: 'success',
      label: 'CN · Shanghai',
      detail: {},
    })

    const row = makeUsageRow({
      request_id: 'req-audit-2408',
      user_id: 42,
      user: {
        id: 42,
        username: 'VioletOps',
        email: 'violet@example.com',
        deleted_at: null,
      } as AdminUsageLog['user'],
      model: 'claude-3-7-sonnet',
      upstream_model: 'claude-sonnet-4',
      model_mapping_chain: 'claude-3-7-sonnet → claude-sonnet-4',
      input_tokens: 12_345,
      output_tokens: 678,
      actual_cost: 0.012345,
      total_cost: 0.015,
      long_context_billing_applied: true,
      first_token_ms: 420,
      duration_ms: 3_200,
      account: { id: 9, name: 'Anthropic · Pool A' },
      created_at: '2026-08-07T01:02:03Z',
      ip_address: '203.0.113.8',
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        auditLayout: true,
        columns: [
          { key: 'user', label: 'Audit Subject' },
          { key: 'model', label: 'Model Mapping' },
          { key: 'tokens', label: 'Token Payload' },
          { key: 'cost', label: 'Actual Charge' },
          { key: 'latency', label: 'Latency' },
          { key: 'account', label: 'Channel Account' },
          { key: 'created_at', label: 'Time / Request ID' },
          { key: 'ip_address', label: 'IP Location' },
        ],
      },
      global: {
        stubs: {
          DataTable: DataTableStubWithAuditCells,
          EmptyState: true,
          Teleport: true,
        },
      },
    })

    const subject = wrapper.get('[data-testid="audit-subject"]')
    expect(subject.get('[data-testid="audit-subject-primary"]').text()).toBe('VioletOps')
    expect(subject.get('[data-testid="audit-subject-primary"]').classes()).toContain('usage-audit-subject__primary')
    expect(subject.get('[data-testid="audit-subject-meta"]').text()).toBe('violet@example.com · #42')

    expect(wrapper.get('[data-testid="audit-model"]').text()).toContain('claude-3-7-sonnet')
    expect(wrapper.get('[data-testid="audit-model"]').text()).toContain('↳ claude-sonnet-4')
    expect(wrapper.get('.usage-audit-tokens__line--input').text()).toBe('12,345In')
    expect(wrapper.get('.usage-audit-tokens__line--output').text()).toBe('678Out')
    expect(wrapper.get('[data-testid="audit-cost"]').text()).toContain('$0.012345')
    expect(wrapper.get('[data-testid="audit-cost-base"]').text()).toBe('Cost: $0.0150')
    expect(wrapper.find('[data-testid="long-context-billing-marker"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="audit-latency-badge"]').text()).toBe('420ms')
    expect(wrapper.get('[data-testid="audit-latency-badge"]').classes()).toContain('usage-audit-latency__badge--good')
    expect(wrapper.get('[data-testid="audit-latency"]').text()).toContain('Total: 3.20s')
    expect(wrapper.get('[data-testid="audit-account"]').text()).toBe('Anthropic · Pool A')
    expect(wrapper.get('[data-testid="audit-time"]').text()).toContain('req-audit-2408')
    expect(wrapper.get('[data-testid="audit-time"]').text()).toMatch(/2026-08-07 \d{2}:\d{2}:\d{2}/)
    expect(wrapper.get('[data-testid="audit-ip"]').text()).toContain('203.0.113.8')
  })
})

describe('admin UsageTable deleted-user badge', () => {
  it('shows an existing username before the email while retaining the user id', () => {
    const row = makeUsageRow({
      request_id: 'req-named-user-1',
      user_id: 7,
      user: {
        id: 7,
        username: 'SnowOperator',
        email: 'operator@test.com',
        deleted_at: null,
      } as AdminUsageLog['user'],
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [{ key: 'user', label: 'User' }],
      },
      global: {
        stubs: {
          DataTable: DataTableStubWithUser,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    const content = wrapper.text()
    expect(content).toContain('SnowOperator')
    expect(content).toContain('operator@test.com')
    expect(content).toContain('#7')
    expect(content.indexOf('SnowOperator')).toBeLessThan(content.indexOf('operator@test.com'))
  })

  it('renders deleted badge for a soft-deleted user row', () => {
    const row = makeUsageRow({
      request_id: 'req-deleted-user-1',
      model: 'claude-3',
      user_id: 2,
      user: { id: 2, email: 'd@test.com', deleted_at: '2026-05-28T00:00:00Z' } as AdminUsageLog['user'],
      actual_cost: 0,
      total_cost: 0,
      input_cost: 0,
      output_cost: 0,
      rate_multiplier: 1,
      input_tokens: 1,
      output_tokens: 1,
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [{ key: 'user', label: 'User' }],
      },
      global: {
        stubs: {
          DataTable: DataTableStubWithUser,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Deleted')
    expect(wrapper.text()).toContain('d@test.com')
  })

  it('does NOT render deleted badge for an active user row', () => {
    const row = makeUsageRow({
      request_id: 'req-active-user-1',
      model: 'claude-3',
      user_id: 3,
      user: { id: 3, email: 'active@test.com', deleted_at: null } as AdminUsageLog['user'],
      actual_cost: 0,
      total_cost: 0,
      input_cost: 0,
      output_cost: 0,
      rate_multiplier: 1,
      input_tokens: 1,
      output_tokens: 1,
    })

    const wrapper = mount(UsageTable, {
      props: {
        data: [row],
        loading: false,
        columns: [{ key: 'user', label: 'User' }],
      },
      global: {
        stubs: {
          DataTable: DataTableStubWithUser,
          EmptyState: true,
          Icon: true,
          Teleport: true,
        },
      },
    })

    expect(wrapper.text()).not.toContain('Deleted')
    expect(wrapper.text()).toContain('active@test.com')
  })
})
