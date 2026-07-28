import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('../client', () => ({ apiClient: client }))

import { download, get, list } from '../admin/desktopDiagnostics'

describe('admin Desktop diagnostics API', () => {
  beforeEach(() => client.get.mockReset())

  it('uses the admin-only paginated list endpoint', async () => {
    const page = { items: [], total: 0, page: 2, page_size: 25, pages: 0 }
    client.get.mockResolvedValue({ data: page })

    await expect(list({ page: 2, page_size: 25 })).resolves.toBe(page)
    expect(client.get).toHaveBeenCalledWith('/admin/desktop/diagnostics', {
      params: { page: 2, page_size: 25 },
    })
  })

  it('encodes diagnostic ids for detail and audited download requests', async () => {
    const detail = { id: 'diagnostic/id', diagnostic: {} }
    const blob = new Blob(['{}'], { type: 'application/json' })
    client.get.mockResolvedValueOnce({ data: detail }).mockResolvedValueOnce({ data: blob })

    await expect(get('diagnostic/id')).resolves.toBe(detail)
    await expect(download('diagnostic/id')).resolves.toBe(blob)

    expect(client.get).toHaveBeenNthCalledWith(
      1,
      '/admin/desktop/diagnostics/diagnostic%2Fid',
    )
    expect(client.get).toHaveBeenNthCalledWith(
      2,
      '/admin/desktop/diagnostics/diagnostic%2Fid/download',
      { responseType: 'blob' },
    )
  })
})
