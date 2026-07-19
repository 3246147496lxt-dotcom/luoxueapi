import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post, put } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, post, put },
}))

import {
  candidates,
  create,
  list,
  publish,
  unpublish,
  update,
  type CreateModelCatalogRequest,
} from '@/api/admin/modelCatalog'

describe('admin model catalog API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    put.mockReset()
  })

  it('lists catalog entries and omits the all-status sentinel', async () => {
    get.mockResolvedValue({ data: { items: [{ id: 1 }], total: 1 } })

    const result = await list({ status: 'all', search: 'claude' })

    expect(get).toHaveBeenCalledWith('/admin/model-catalog', {
      params: { status: undefined, search: 'claude' },
    })
    expect(result).toEqual({ items: [{ id: 1 }], total: 1 })
  })

  it('normalizes legacy array and candidates envelope responses', async () => {
    get.mockResolvedValueOnce({ data: [{ id: 2 }] })
    get.mockResolvedValueOnce({ data: { candidates: [{ model: 'gpt-5' }], total: 1 } })

    await expect(list()).resolves.toEqual({ items: [{ id: 2 }], total: 1 })
    await expect(candidates()).resolves.toEqual({ items: [{ model: 'gpt-5' }], total: 1 })
  })

  it('uses the catalog lifecycle endpoints and preserves nullable public group drafts', async () => {
    const draft: CreateModelCatalogRequest = {
      model: 'claude-sonnet-4-5',
      platform: 'anthropic',
      display_name_zh: 'Claude Sonnet 4.5',
      public_group_id: null,
    }
    post.mockResolvedValue({ data: { id: 9 } })
    put.mockResolvedValue({ data: { id: 9 } })

    await create(draft)
    const replacement = { ...draft, public_group_id: 3, featured: true }
    await update(9, replacement)
    await publish(9)
    await unpublish(9)

    expect(post).toHaveBeenNthCalledWith(1, '/admin/model-catalog', draft)
    expect(put).toHaveBeenCalledWith('/admin/model-catalog/9', replacement)
    expect(post).toHaveBeenNthCalledWith(2, '/admin/model-catalog/9/publish')
    expect(post).toHaveBeenNthCalledWith(3, '/admin/model-catalog/9/unpublish')
  })
})
