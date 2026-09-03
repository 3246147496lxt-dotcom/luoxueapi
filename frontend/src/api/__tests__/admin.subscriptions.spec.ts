import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({ get: vi.fn() }))

vi.mock('../client', () => ({ apiClient: client }))

import { listByUser } from '../admin/subscriptions'

describe('admin subscriptions API', () => {
  beforeEach(() => client.get.mockReset())

  it('uses the dedicated user endpoint and returns its array payload', async () => {
    const subscriptions = [{ id: 91, user_id: 17, group_id: 8, status: 'active' }]
    client.get.mockResolvedValue({ data: subscriptions })

    await expect(listByUser(17)).resolves.toBe(subscriptions)
    expect(client.get).toHaveBeenCalledWith('/admin/users/17/subscriptions')
  })
})
