import { describe, expect, it } from 'vitest'

import {
  formatLibraryFileTime,
  formatLibraryFileTimeTooltip,
  libraryFileModifiedTimestamp,
  type LibraryFileTimeFormatOptions
} from '@/utils/libraryFileTime'

const fixedOptions: LibraryFileTimeFormatOptions = {
  now: '2026-08-14T13:42:00+08:00',
  locale: 'zh-CN',
  timeZone: 'Asia/Shanghai'
}

describe('libraryFileTime', () => {
  it.each([
    ['today as 24-hour time', '2026-08-14T13:38:00+08:00', '13:38'],
    ['yesterday without a time', '2026-08-13T20:00:00+08:00', '昨天'],
    ['Sunday in the current week', '2026-08-09T04:46:00+08:00', '星期日'],
    ['an earlier date in the current year', '2026-08-08T23:57:00+08:00', '8月8日'],
    ['a date from another year', '2025-12-31T23:59:00+08:00', '2025年12月31日']
  ])('formats %s', (_case, timestamp, expected) => {
    expect(formatLibraryFileTime(timestamp, fixedOptions)).toBe(expected)
  })

  it('uses Sunday as the explicit start of the natural week', () => {
    expect(formatLibraryFileTime('2026-08-09T00:00:00+08:00', fixedOptions)).toBe('星期日')
    expect(formatLibraryFileTime('2026-08-08T23:59:59+08:00', fixedOptions)).toBe('8月8日')
  })

  it('classifies dates after converting them into the requested time zone', () => {
    expect(formatLibraryFileTime('2026-08-13T17:38:00Z', fixedOptions)).toBe('01:38')
  })

  it('returns an em dash for invalid or missing timestamps', () => {
    expect(formatLibraryFileTime('not-a-date', fixedOptions)).toBe('—')
    expect(formatLibraryFileTime(undefined, fixedOptions)).toBe('—')
    expect(formatLibraryFileTimeTooltip('not-a-date', fixedOptions)).toBe('—')
  })

  it('formats the complete local timestamp for the tooltip', () => {
    expect(formatLibraryFileTimeTooltip('2026-08-14T13:38:00+08:00', fixedOptions)).toBe(
      '2026年8月14日 13:38'
    )
  })

  it('depends only on the timestamp supplied by the file record', () => {
    const file = {
      name: 'CleanShot 2025-01-01.png',
      updated_at: '2026-08-14T13:38:00+08:00'
    }

    expect(formatLibraryFileTime(file.updated_at, fixedOptions)).toBe('13:38')
  })

  it('selects updatedAt before createdAt and ignores the filename', () => {
    expect(libraryFileModifiedTimestamp({
      name: 'CleanShot 2025-01-01.png',
      updatedAt: ' 2026-08-14T13:38:00+08:00 ',
      createdAt: '2025-01-01T00:00:00+08:00'
    })).toBe('2026-08-14T13:38:00+08:00')

    expect(libraryFileModifiedTimestamp({
      updatedAt: '   ',
      createdAt: ' 2026-08-13T20:00:00+08:00 '
    })).toBe('2026-08-13T20:00:00+08:00')
  })
})
