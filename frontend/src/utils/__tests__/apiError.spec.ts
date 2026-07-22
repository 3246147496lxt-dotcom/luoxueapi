import { describe, expect, it } from 'vitest'

import { extractApiErrorMessage, extractApiErrorStatus } from '../apiError'

describe('apiError helpers', () => {
  it('reads the current plain interceptor error shape', () => {
    const error = { status: 403, message: 'Permission denied' }

    expect(extractApiErrorStatus(error)).toBe(403)
    expect(extractApiErrorMessage(error, 'fallback')).toBe('Permission denied')
  })

  it('keeps compatibility with legacy Axios response errors', () => {
    const error = {
      response: {
        status: 404,
        data: { detail: 'Group not found' }
      }
    }

    expect(extractApiErrorStatus(error)).toBe(404)
    expect(extractApiErrorMessage(error, 'fallback')).toBe('Group not found')
  })

  it('preserves status zero for network failures', () => {
    expect(extractApiErrorStatus({ status: 0 })).toBe(0)
    expect(extractApiErrorStatus(new Error('offline'))).toBeUndefined()
  })
})
