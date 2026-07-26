import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const {
  listUserConversations,
  getUserConversation,
  routerPush,
  routeParams
} = vi.hoisted(() => ({
  listUserConversations: vi.fn(),
  getUserConversation: vi.fn(),
  routerPush: vi.fn(),
  routeParams: { userId: '42' }
}))

vi.mock('@/api/admin/chatHistory', () => ({
  listUserConversations,
  getUserConversation,
  adminChatHistoryAPI: {
    listUserConversations,
    getUserConversation
  },
  default: {
    listUserConversations,
    getUserConversation
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: routeParams
  }),
  useRouter: () => ({
    push: routerPush
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import AdminUserChatHistoryView from '../AdminUserChatHistoryView.vue'

const conversation = {
  id: 'conv-1',
  model: 'gpt-5.5',
  message_count: 2,
  status: 'completed',
  created_at: '2026-07-24T01:00:00Z',
  updated_at: '2026-07-25T01:00:00Z'
}

const detailPage = {
  conversation: {
    id: 'conv-1',
    title: 'Private support conversation',
    model: 'gpt-5.5',
    status: 'completed',
    created_at: '2026-07-24T01:00:00Z',
    updated_at: '2026-07-25T01:00:00Z'
  },
  messages: [
    {
      id: 'msg-101',
      position: 101,
      role: 'user',
      content: '<script>window.compromised = true</script>',
      status: 'completed',
      created_at: '2026-07-25T01:00:00Z'
    },
    {
      id: 'msg-102',
      position: 102,
      role: 'assistant',
      content: 'Plain-text reply',
      status: 'completed',
      created_at: '2026-07-25T01:01:00Z'
    }
  ],
  next_before_position: 101,
  has_more: true
}

const mountView = () => mount(AdminUserChatHistoryView, {
  global: {
    stubs: {
      AppLayout: {
        template: '<div><slot /></div>'
      },
      AdminPageHeader: {
        props: ['title', 'description'],
        template: `
          <header>
            <h1>{{ title }}</h1>
            <p>{{ description }}</p>
            <slot name="meta" />
            <slot name="secondary-actions" />
          </header>
        `
      },
      Icon: true
    }
  }
})

describe('AdminUserChatHistoryView', () => {
  beforeEach(() => {
    listUserConversations.mockReset()
    getUserConversation.mockReset()
    routerPush.mockReset()
    routeParams.userId = '42'

    listUserConversations.mockResolvedValue({
      items: [conversation],
      has_more: false
    })
    getUserConversation.mockResolvedValue(detailPage)
  })

  it('loads metadata first and accesses escaped content only after an explicit selection', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listUserConversations).toHaveBeenCalledWith(
      '42',
      { limit: 30 },
      { signal: expect.anything() }
    )
    expect(getUserConversation).not.toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('Private support conversation')
    expect(wrapper.find('[data-test="conversation-unselected"]').exists()).toBe(true)

    await wrapper.get('[data-test="conversation-conv-1"]').trigger('click')
    await flushPromises()

    expect(getUserConversation).toHaveBeenCalledWith(
      '42',
      'conv-1',
      { limit: 100 },
      { signal: expect.anything() }
    )
    expect(wrapper.find('[data-test="privacy-notice"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Private support conversation')
    expect(wrapper.text()).toContain('<script>window.compromised = true</script>')
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.get('[data-test="conversation-list-pane"]').classes()).toContain('hidden')
    expect(wrapper.get('[data-test="conversation-detail-pane"]').classes()).toContain('flex')

    await wrapper.get('[data-test="back-to-conversation-list"]').trigger('click')

    expect(wrapper.get('[data-test="conversation-list-pane"]').classes()).toContain('flex')
    expect(wrapper.get('[data-test="conversation-detail-pane"]').classes()).toContain('hidden')
  })

  it('loads earlier messages with the audit-triggering position cursor and keeps ascending order', async () => {
    getUserConversation
      .mockResolvedValueOnce(detailPage)
      .mockResolvedValueOnce({
        conversation: detailPage.conversation,
        messages: [
          {
            id: 'msg-2',
            position: 2,
            role: 'assistant',
            content: 'Older reply',
            status: 'completed',
            created_at: '2026-07-24T01:01:00Z'
          },
          {
            id: 'msg-1',
            position: 1,
            role: 'user',
            content: 'Older question',
            status: 'completed',
            created_at: '2026-07-24T01:00:00Z'
          }
        ],
        has_more: false
      })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-test="conversation-conv-1"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="load-older-messages"]').trigger('click')
    await flushPromises()

    expect(getUserConversation).toHaveBeenLastCalledWith(
      '42',
      'conv-1',
      {
        before_position: 101,
        limit: 100
      },
      { signal: expect.anything() }
    )
    expect(
      wrapper.findAll('[data-message-position]').map(node => node.attributes('data-message-position'))
    ).toEqual(['1', '2', '101', '102'])
    expect(wrapper.text()).toContain('admin.chatHistory.reachedBeginning')
  })

  it('uses reversible cursor pagination for the metadata list', async () => {
    listUserConversations
      .mockResolvedValueOnce({
        items: [conversation],
        next_cursor: 'cursor-page-2',
        has_more: true
      })
      .mockResolvedValueOnce({
        items: [{ ...conversation, id: 'conv-older' }],
        has_more: false
      })
      .mockResolvedValueOnce({
        items: [conversation],
        next_cursor: 'cursor-page-2',
        has_more: true
      })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="next-conversation-page"]').trigger('click')
    await flushPromises()

    expect(listUserConversations).toHaveBeenLastCalledWith(
      '42',
      { cursor: 'cursor-page-2', limit: 30 },
      { signal: expect.anything() }
    )
    expect(wrapper.find('[data-test="conversation-conv-older"]').exists()).toBe(true)

    await wrapper.get('[data-test="previous-conversation-page"]').trigger('click')
    await flushPromises()

    expect(listUserConversations).toHaveBeenLastCalledWith(
      '42',
      { limit: 30 },
      { signal: expect.anything() }
    )
    expect(wrapper.find('[data-test="conversation-conv-1"]').exists()).toBe(true)
  })

  it('shows dedicated list and missing-conversation error states', async () => {
    listUserConversations.mockRejectedValueOnce({ status: 503 })

    const listErrorWrapper = mountView()
    await flushPromises()

    expect(listErrorWrapper.find('[data-test="conversation-list-error"]').exists()).toBe(true)
    listErrorWrapper.unmount()

    listUserConversations.mockResolvedValueOnce({
      items: [conversation],
      has_more: false
    })
    getUserConversation.mockRejectedValueOnce({ status: 404 })

    const detailErrorWrapper = mountView()
    await flushPromises()
    await detailErrorWrapper.get('[data-test="conversation-conv-1"]').trigger('click')
    await flushPromises()

    expect(detailErrorWrapper.get('[data-test="conversation-detail-error"]').text()).toContain(
      'admin.chatHistory.notFoundTitle'
    )
  })
})
