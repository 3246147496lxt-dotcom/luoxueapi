import { describe, expect, it } from 'vitest'

import { splitBrandApiSuffix } from '../brand'

describe('splitBrandApiSuffix', () => {
  it.each([
    ['落雪API', { base: '落雪', apiSuffix: 'API' }],
    ['落雪 API', { base: '落雪', apiSuffix: 'API' }],
    ['Snow api', { base: 'Snow', apiSuffix: 'api' }],
    ['RapidAPI', { base: 'Rapid', apiSuffix: 'API' }]
  ])('splits a trailing API token from %s', (value, expected) => {
    expect(splitBrandApiSuffix(value)).toEqual(expected)
  })

  it('leaves custom names without an API suffix unchanged', () => {
    for (const value of ['雪落开发者平台', 'API', 'API 网关', 'GraphAPIary']) {
      expect(splitBrandApiSuffix(value)).toEqual({ base: value, apiSuffix: '' })
    }
  })
})
