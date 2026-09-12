import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountAssistantWorkspace from '../AccountAssistantWorkspace.vue'

const { chatHealthMock, cleanupHealthMock, addHealthMock, grokSSOMock, proxiesMock, groupsMock, watcherSettingsMock, watcherUpdateMock, watcherScanMock, watcherImportMock } = vi.hoisted(() => ({
  chatHealthMock: vi.fn(),
  cleanupHealthMock: vi.fn(),
  addHealthMock: vi.fn(),
  grokSSOMock: vi.fn(),
  proxiesMock: vi.fn(),
  groupsMock: vi.fn(),
  watcherSettingsMock: vi.fn(),
  watcherUpdateMock: vi.fn(),
  watcherScanMock: vi.fn(),
  watcherImportMock: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

vi.mock('@/api/admin/accounts', () => ({
  chatHealth: chatHealthMock,
  cleanupHealth: cleanupHealthMock,
  addHealth: addHealthMock,
  getImportWatcherSettings: watcherSettingsMock,
  updateImportWatcherSettings: watcherUpdateMock,
  scanImportWatcher: watcherScanMock,
  importWatcherFile: watcherImportMock
}))
vi.mock('@/api/admin/grok', () => ({
  default: { createFromSSO: grokSSOMock }
}))
vi.mock('@/api/admin/proxies', () => ({
  default: { getAll: proxiesMock }
}))
vi.mock('@/api/admin/groups', () => ({
  default: { getAll: groupsMock }
}))

const cleanupItem = {
  account_id: 41,
  name: 'expired-oauth',
  platform: 'openai',
  type: 'oauth',
  status: 'error',
  reason: 'auth_failed',
  summary: 'Authentication failed or token is unusable',
  cleanup: true,
  tested: false
}

function mountWorkspace() {
  return mount(AccountAssistantWorkspace, {
    attachTo: document.body,
    global: {
      stubs: {
        Icon: true,
        ConfirmDialog: {
          name: 'ConfirmDialog',
          inheritAttrs: false,
          props: ['show', 'title', 'message', 'danger', 'confirmText', 'cancelText'],
          emits: ['confirm', 'cancel'],
          template: `
            <div v-if="show" :data-test="$attrs['data-test'] || 'account-assistant-confirm'">
              <button type="button" data-test="account-assistant-confirm-ok" @click="$emit('confirm')">ok</button>
            </div>
          `
        }
      }
    }
  })
}

describe('AccountAssistantWorkspace', () => {
  beforeEach(() => {
    chatHealthMock.mockReset()
    cleanupHealthMock.mockReset()
    addHealthMock.mockReset()
    grokSSOMock.mockReset()
    proxiesMock.mockReset()
    groupsMock.mockReset()
    watcherSettingsMock.mockReset()
    watcherUpdateMock.mockReset()
    watcherScanMock.mockReset()
    watcherImportMock.mockReset()
    chatHealthMock.mockResolvedValue({
      intent: 'scan',
      reply: 'found one',
      scan: {
        items: [cleanupItem],
        total: 1,
        tested: 0,
        cleanup_count: 1,
        truncated: false
      },
      proposal: { account_ids: [41], items: [cleanupItem] },
      tool_traces: [{ name: 'scan_account_pool', status: 'ok', summary: 'scanned account pool' }]
    })
    cleanupHealthMock.mockResolvedValue({ deleted: [41], failed: [] })
    addHealthMock.mockResolvedValue({
      created: [{
        index: 0,
        name: 'imported-openai-1',
        platform: 'openai',
        type: 'apikey',
        key_hint: 'sk-••••aaaa',
        summary: 'API Key · openai'
      }],
      failed: []
    })
    grokSSOMock.mockResolvedValue({ created: [{ index: 0, name: 'grok-sso-1' }], failed: [] })
    proxiesMock.mockResolvedValue([{ id: 3, name: 'proxy-a', protocol: 'http', status: 'active' }])
    groupsMock.mockResolvedValue([{ id: 8, name: 'Grok group', platform: 'grok', status: 'active' }])
    watcherSettingsMock.mockResolvedValue({ enabled: false, output_dir: '', auto_discover: true })
    watcherUpdateMock.mockImplementation(async (value: unknown) => value)
    watcherScanMock.mockResolvedValue({ files: [] })
    watcherImportMock.mockResolvedValue({ imported: 2, created: [], failed: [] })
  })

  it('sends natural language from chips and never deletes from chat', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()

    await wrapper.get('[data-test="account-assistant-chip-scan"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(chatHealthMock).toHaveBeenCalledTimes(1)
    // The primary scan shortcut performs a live, pool-wide check explicitly.
    expect(chatHealthMock.mock.calls[0]?.[0].intent).toBe('test')
    expect(chatHealthMock.mock.calls[0]?.[0].messages.at(-1)).toMatchObject({
      role: 'user',
      content: 'admin.accounts.copilot.suggestions.scan'
    })
    expect(cleanupHealthMock).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="account-assistant-tool-scan_account_pool"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('keeps delete disabled until a review item is selected and confirmed in the dialog', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()

    const deleteSelector = '[data-test="account-assistant-delete"]'
    expect(wrapper.get(deleteSelector).attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-test="account-assistant-confirm"]').exists()).toBe(false)

    await wrapper.get('[data-test="account-assistant-chip-scan"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(wrapper.get(deleteSelector).attributes('disabled')).toBeDefined()
    const item = wrapper.get('[data-test="account-assistant-item-41"]')
    expect((item.element as HTMLInputElement).checked).toBe(false)

    await item.setValue(true)
    await nextTick()
    expect(wrapper.get(deleteSelector).attributes('disabled')).toBeUndefined()

    await wrapper.get(deleteSelector).trigger('click')
    await nextTick()
    expect(wrapper.find('[data-test="account-assistant-confirm"]').exists()).toBe(true)
    expect(cleanupHealthMock).not.toHaveBeenCalled()

    await wrapper.get('[data-test="account-assistant-item-41"]').setValue(false)
    await nextTick()
    expect(wrapper.get(deleteSelector).attributes('disabled')).toBeDefined()

    await wrapper.get('[data-test="account-assistant-item-41"]').setValue(true)
    await nextTick()
    await wrapper.get(deleteSelector).trigger('click')
    await nextTick()
    await wrapper.get('[data-test="account-assistant-confirm-ok"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(cleanupHealthMock).toHaveBeenCalledTimes(1)
    expect(cleanupHealthMock).toHaveBeenCalledWith({
      account_ids: [41],
      confirm: 'DELETE'
    })
    wrapper.unmount()
  })

  it('does not parse delete IDs from free text in the composer', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()
    chatHealthMock.mockResolvedValueOnce({
      intent: 'help',
      reply: 'scan first'
    })

    await wrapper.get('[data-test="account-assistant-input"]').setValue('删除 41')
    await wrapper.get('[data-test="account-assistant-form"]').trigger('submit')
    await flushPromises()
    await nextTick()

    expect(chatHealthMock).toHaveBeenCalledTimes(1)
    expect(cleanupHealthMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('keeps pasted keys out of chat messages and only adds after confirmation', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()
    chatHealthMock.mockResolvedValueOnce({
      intent: 'add',
      reply: 'parsed',
      add_proposal: {
        items: [{
          index: 0,
          name: 'imported-openai-1',
          platform: 'openai',
          type: 'apikey',
          key_hint: 'sk-••••aaaa',
          summary: 'API Key · openai'
        }]
      }
    })

    const secret = 'sk-secret-token-aaaa'
    await wrapper.get('[data-test="account-assistant-input"]').setValue(secret)
    await wrapper.get('[data-test="account-assistant-form"]').trigger('submit')
    await flushPromises()
    await nextTick()

    expect(chatHealthMock).toHaveBeenCalledTimes(1)
    const payload = chatHealthMock.mock.calls[0]?.[0]
    expect(payload.handoff).toBe(secret)
    expect(payload.messages.at(-1)).toMatchObject({
      role: 'user',
      content: 'admin.accounts.copilot.handoffSubmitted'
    })
    expect(JSON.stringify(payload.messages)).not.toContain(secret)
    expect(addHealthMock).not.toHaveBeenCalled()

    const addButton = wrapper.get('[data-test="account-assistant-add"]')
    expect(addButton.attributes('disabled')).toBeDefined()

    const item = wrapper.get('[data-test="account-assistant-add-item-0"]')
    expect((item.element as HTMLInputElement).checked).toBe(false)
    await item.setValue(true)
    await nextTick()
    expect(addButton.attributes('disabled')).toBeUndefined()

    await addButton.trigger('click')
    await nextTick()
    expect(wrapper.find('[data-test="account-assistant-add-confirm"]').exists()).toBe(true)
    expect(addHealthMock).not.toHaveBeenCalled()

    await wrapper.get('[data-test="account-assistant-confirm-ok"]').trigger('click')
    await flushPromises()
    await nextTick()

    expect(addHealthMock).toHaveBeenCalledTimes(1)
    expect(addHealthMock).toHaveBeenCalledWith({
      handoff: secret,
      indexes: [0],
      confirm: 'ADD'
    })
    wrapper.unmount()
  })

  it('detects Grok SSO tokens and imports through the Grok SSO endpoint with selected routing', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()
    const token = 'eyJhbGciOiJub25lIn0.eyJzdWIiOiIxIn0.signature'
    await wrapper.get('[data-test="account-assistant-input"]').setValue(token)
    await wrapper.get('[data-test="account-assistant-form"]').trigger('submit')
    await flushPromises()
    await nextTick()

    expect(chatHealthMock).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="account-assistant-sso-options"]').exists()).toBe(true)
    await wrapper.get('[data-test="account-assistant-sso-proxy"]').setValue('3')
    await wrapper.get('[data-test="account-assistant-add-item-0"]').setValue(true)
    await wrapper.get('[data-test="account-assistant-add"]').trigger('click')
    await wrapper.get('[data-test="account-assistant-confirm-ok"]').trigger('click')
    await flushPromises()

    expect(grokSSOMock).toHaveBeenCalledWith(expect.objectContaining({
      sso_tokens: [token],
      proxy_id: 3,
      group_ids: []
    }))
    expect(addHealthMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('discovers registration tool SSO files without exposing token contents and imports after one confirmation', async () => {
    const wrapper = mountWorkspace()
    await flushPromises()
    watcherScanMock.mockResolvedValueOnce({
      files: [{ name: 'accounts_20260912_001.txt', platform: 'grok', token_count: 3, new: true }]
    })
    await wrapper.get('[data-test="account-assistant-settings"] summary').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="account-assistant-watcher-dir"]').setValue('/tokens')
    await wrapper.get('[data-test="account-assistant-watcher-scan"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="account-assistant-add-item-0"]').element.parentElement?.parentElement?.textContent).toContain('account')
    expect(wrapper.text()).toContain('Grok SSO')
    expect(wrapper.text()).not.toContain('eyJ')
    expect(wrapper.get('[data-test="account-assistant-add"]').attributes('disabled')).toBeUndefined()
    await wrapper.get('[data-test="account-assistant-add"]').trigger('click')
    await wrapper.get('[data-test="account-assistant-confirm-ok"]').trigger('click')
    await flushPromises()
    expect(watcherImportMock).toHaveBeenCalledWith(expect.objectContaining({
      file: 'accounts_20260912_001.txt',
      confirm: 'ADD',
      indexes: []
    }))
    wrapper.unmount()
  })

  it.each([
    ['refresh_token', '{"refresh_token":"refresh-secret"}'],
    ['client_secret', '{"client_secret":"client-secret"}'],
    ['authorization', 'authorization: Bearer bearer-secret']
  ])('keeps %s credentials out of chat messages', async (_label, secret) => {
    const wrapper = mountWorkspace()
    await flushPromises()
    chatHealthMock.mockResolvedValueOnce({
      intent: 'add',
      reply: 'parsed',
      add_proposal: { items: [] }
    })

    await wrapper.get('[data-test="account-assistant-input"]').setValue(secret)
    await wrapper.get('[data-test="account-assistant-form"]').trigger('submit')
    await flushPromises()

    const payload = chatHealthMock.mock.calls[0]?.[0]
    expect(payload.handoff).toBe(secret)
    expect(payload.messages.at(-1)?.content).toBe('admin.accounts.copilot.handoffSubmitted')
    expect(JSON.stringify(payload.messages)).not.toContain(secret)
    wrapper.unmount()
  })
})
