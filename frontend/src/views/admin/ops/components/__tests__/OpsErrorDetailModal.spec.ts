import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'
import type { OpsErrorDetail } from '@/api/admin/ops'

const mocks = vi.hoisted(() => ({
  getRequestErrorDetail: vi.fn(),
  getUpstreamErrorDetail: vi.fn(),
  listRequestErrorUpstreamErrors: vi.fn(),
  copyToClipboard: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail: mocks.getRequestErrorDetail,
    getUpstreamErrorDetail: mocks.getUpstreamErrorDetail,
    listRequestErrorUpstreamErrors: mocks.listRequestErrorUpstreamErrors,
  },
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: mocks.copyToClipboard }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError: mocks.showError }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${String(params.id ?? '')}` : key,
    }),
  }
})

function errorDetail(overrides: Partial<OpsErrorDetail> = {}): OpsErrorDetail {
  return {
    id: 41,
    created_at: '2026-06-05T23:59:50Z',
    phase: 'upstream',
    type: 'provider_error',
    error_owner: 'provider',
    error_source: 'upstream_http',
    severity: 'P1',
    status_code: 529,
    platform: 'anthropic',
    model: 'claude-opus-4-8',
    resolved: false,
    client_request_id: 'client-41',
    request_id: 'req-41',
    message: 'provider overloaded',
    user_id: 7,
    user_email: 'alice@example.com',
    api_key_id: 9,
    api_key_name: 'production',
    account_id: 13,
    account_name: 'upstream-a',
    group_id: 5,
    group_name: 'default',
    request_path: '/v1/messages',
    inbound_endpoint: '/v1/messages',
    upstream_endpoint: '/v1/messages',
    requested_model: 'claude-opus-4-8',
    upstream_model: 'claude-opus-4-8-20260601',
    request_type: 2,
    error_body: '{"error":{"message":"overloaded"}}',
    is_business_limited: false,
    ...overrides,
  }
}

const wrappers: VueWrapper[] = []

function mountDrawer() {
  const wrapper = mount(OpsErrorDetailModal, {
    attachTo: document.body,
    props: {
      show: true,
      errorId: 41,
      errorType: 'request',
    },
    global: {
      stubs: {
        Teleport: true,
        Transition: { props: ['name'], template: '<slot />' },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(() => {
  mocks.getRequestErrorDetail.mockReset().mockResolvedValue(errorDetail())
  mocks.getUpstreamErrorDetail.mockReset().mockResolvedValue(errorDetail())
  mocks.listRequestErrorUpstreamErrors.mockReset().mockResolvedValue({
    items: [],
    total: 0,
    page: 1,
    page_size: 100,
  })
  mocks.copyToClipboard.mockReset().mockResolvedValue(true)
  mocks.showError.mockReset()
})

afterEach(() => {
  while (wrappers.length) wrappers.pop()?.unmount()
  document.body.replaceChildren()
})

describe('OpsErrorDetailModal read-only drawer', () => {
  it('loads a non-modal drawer with classification and email plus user ID truth', async () => {
    const wrapper = mountDrawer()
    await flushPromises()

    const drawer = wrapper.get('[data-testid="ops-error-detail-drawer"]')
    expect(drawer.attributes('role')).toBe('dialog')
    expect(drawer.attributes('aria-modal')).toBeUndefined()
    expect(wrapper.text()).toContain('admin.ops.errorLog.readOnly')
    expect(wrapper.text()).toContain('529')
    expect(wrapper.text()).toContain('P1')
    expect(wrapper.text()).toContain('usage.errors.categories.upstream')
    expect(wrapper.text()).toContain('alice@example.com · #7')
    expect(wrapper.text()).toContain('req-41')
    expect(wrapper.text()).toContain('provider overloaded')
    expect(wrapper.text()).not.toContain('admin.ops.errorDetail.markResolved')
  })

  it('copies the raw message and request ID without exposing a write action', async () => {
    const wrapper = mountDrawer()
    await flushPromises()

    await wrapper
      .get('button[aria-label="common.copy admin.ops.errorDetail.message"]')
      .trigger('click')
    await wrapper
      .get('button[aria-label="common.copy admin.ops.errorDetail.requestId"]')
      .trigger('click')

    expect(mocks.copyToClipboard).toHaveBeenNthCalledWith(1, 'provider overloaded')
    expect(mocks.copyToClipboard).toHaveBeenNthCalledWith(2, 'req-41')
  })

  it('moves focus into the announced drawer and restores the invoking control on close', async () => {
    const trigger = document.createElement('button')
    trigger.textContent = 'Open error detail'
    document.body.appendChild(trigger)
    trigger.focus()

    const wrapper = mountDrawer()
    await flushPromises()

    const drawer = wrapper.get<HTMLElement>('[data-testid="ops-error-detail-drawer"]')
    expect(document.activeElement).toBe(drawer.element)

    await wrapper.setProps({ show: false })
    await flushPromises()
    expect(document.activeElement).toBe(trigger)
  })

  it('closes on Escape while leaving the surrounding page unlocked', async () => {
    const wrapper = mountDrawer()
    await flushPromises()

    expect(document.body.style.overflow).toBe('')
    document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    }))
    await flushPromises()

    expect(wrapper.emitted('update:show')).toEqual([[false]])
    expect(document.body.style.overflow).toBe('')
  })
})
