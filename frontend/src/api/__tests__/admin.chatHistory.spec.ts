import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get
  }
}))

import {
  getUserConversation,
  listUserConversations
} from '@/api/admin/chatHistory'

describe('admin chat history API', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('loads only conversation metadata with cursor pagination', async () => {
    get.mockResolvedValue({
      data: {
        items: [
          {
            id: 'conv-2',
            model: 'gpt-5.5',
            message_count: 8,
            status: 'active',
            created_at: '2026-07-24T01:00:00Z',
            updated_at: '2026-07-25T01:00:00Z'
          }
        ],
        next_cursor: 'cursor-older',
        has_more: true
      }
    })
    const controller = new AbortController()

    const result = await listUserConversations(
      42,
      { cursor: 'cursor-current', limit: 30 },
      { signal: controller.signal }
    )

    expect(get).toHaveBeenCalledWith(
      '/admin/users/42/chat/conversations',
      {
        params: { cursor: 'cursor-current', limit: 30 },
        signal: controller.signal
      }
    )
    expect(result).toEqual({
      items: [
        expect.objectContaining({
          id: 'conv-2',
          model: 'gpt-5.5',
          message_count: 8
        })
      ],
      next_cursor: 'cursor-older',
      has_more: true
    })
    expect(result.items[0]).not.toHaveProperty('title')
    expect(result.items[0]).not.toHaveProperty('messages')
  })

  it('loads one audited content page and preserves message paging fields', async () => {
    get.mockResolvedValue({
      data: {
        data: {
          conversation: {
            id: 'conv/private id',
            title: 'Support case',
            model: 'gpt-5.5',
            status: 'completed',
            created_at: '2026-07-24T01:00:00Z',
            updated_at: '2026-07-25T01:00:00Z'
          },
          messages: [
            {
              id: 'msg-101',
              position: 101,
              role: 'assistant',
              content: 'Resolved',
              status: 'completed',
              created_at: '2026-07-25T01:00:00Z'
            }
          ],
          next_before_position: 101,
          has_more: true
        }
      }
    })
    const controller = new AbortController()

    const result = await getUserConversation(
      '42',
      'conv/private id',
      { before_position: 201, limit: 100 },
      { signal: controller.signal }
    )

    expect(get).toHaveBeenCalledWith(
      '/admin/users/42/chat/conversations/conv%2Fprivate%20id',
      {
        params: { before_position: 201, limit: 100 },
        signal: controller.signal
      }
    )
    expect(result).toEqual({
      conversation: expect.objectContaining({
        id: 'conv/private id',
        title: 'Support case'
      }),
      messages: [
        expect.objectContaining({
          id: 'msg-101',
          position: 101,
          content: 'Resolved'
        })
      ],
      next_before_position: 101,
      has_more: true
    })
  })
})
