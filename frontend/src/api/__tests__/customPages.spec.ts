import { beforeEach, describe, expect, it, vi } from 'vitest'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))

vi.mock('@/api/client', () => ({
  apiClient: { post },
}))

import { requestCustomPageLaunch } from '../customPages'

describe('custom page launch API', () => {
  beforeEach(() => {
    post.mockReset()
  })

  it('uses the menu id path and forwards only approved context fields', async () => {
    post.mockResolvedValue({
      data: {
        launch_url: 'https://external.example?s2a_client_id=client&s2a_launch_code=once',
        auth_mode: 'exchange_code',
        expires_in: 60,
      },
    })

    const response = await requestCustomPageLaunch('menu/item', {
      theme: 'dark',
      lang: 'zh-CN',
      ui_mode: 'new_tab',
      src_host: 'https://app.example.com',
      // Runtime callers cannot smuggle arbitrary fields into the external protocol.
      token: 'must-not-leak',
    } as any)

    expect(post).toHaveBeenCalledWith(
      '/user/custom-pages/menu%2Fitem/launch',
      {
        theme: 'dark',
        lang: 'zh-CN',
        ui_mode: 'new_tab',
        src_host: 'https://app.example.com',
      },
    )
    expect(response.expires_in).toBe(60)
  })
})
