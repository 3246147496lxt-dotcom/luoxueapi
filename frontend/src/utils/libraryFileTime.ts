const INVALID_DATE_LABEL = '—'
const MILLISECONDS_PER_DAY = 24 * 60 * 60 * 1000

export interface LibraryFileTimeFormatOptions {
  now?: Date | string | number
  locale?: string
  timeZone?: string
}

export interface LibraryFileModifiedTimeRecord {
  updatedAt: string
  createdAt: string
}

interface CalendarDate {
  year: number
  month: number
  day: number
}

function parseDate(value: Date | string | number | null | undefined): Date | null {
  if (value === null || value === undefined || value === '') return null

  const date = value instanceof Date ? new Date(value.getTime()) : new Date(value)
  return Number.isNaN(date.getTime()) ? null : date
}

function calendarDate(date: Date, timeZone?: string): CalendarDate {
  const parts = new Intl.DateTimeFormat('en-US-u-ca-gregory-nu-latn', {
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    timeZone
  }).formatToParts(date)

  const partValue = (type: Intl.DateTimeFormatPartTypes): number =>
    Number(parts.find(part => part.type === type)?.value)

  return {
    year: partValue('year'),
    month: partValue('month'),
    day: partValue('day')
  }
}

function calendarDayNumber(date: CalendarDate): number {
  return Math.floor(Date.UTC(date.year, date.month - 1, date.day) / MILLISECONDS_PER_DAY)
}

function calendarWeekStart(date: CalendarDate): number {
  const dayNumber = calendarDayNumber(date)
  const dayOfWeek = new Date(Date.UTC(date.year, date.month - 1, date.day)).getUTCDay()
  return dayNumber - dayOfWeek
}

function dateTimeFormatter(
  locale: string | undefined,
  timeZone: string | undefined,
  options: Intl.DateTimeFormatOptions
): Intl.DateTimeFormat {
  return new Intl.DateTimeFormat(locale, {
    ...options,
    calendar: 'gregory',
    timeZone
  })
}

/** Selects the record timestamp without deriving any date from the filename. */
export function libraryFileModifiedTimestamp<T extends LibraryFileModifiedTimeRecord>(file: T): string {
  return file.updatedAt.trim() || file.createdAt.trim()
}

/**
 * Formats a library timestamp using the calendar date in the user's time zone.
 * The caller remains responsible for selecting updated_at before created_at.
 */
export function formatLibraryFileTime(
  timestamp: string | null | undefined,
  options: LibraryFileTimeFormatOptions = {}
): string {
  const date = parseDate(timestamp)
  const now = parseDate(options.now ?? new Date())
  if (!date || !now) return INVALID_DATE_LABEL

  const fileDate = calendarDate(date, options.timeZone)
  const currentDate = calendarDate(now, options.timeZone)
  const fileDay = calendarDayNumber(fileDate)
  const currentDay = calendarDayNumber(currentDate)
  const dayDifference = currentDay - fileDay

  if (dayDifference === 0) {
    return dateTimeFormatter(options.locale, options.timeZone, {
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23'
    }).format(date)
  }

  if (dayDifference === 1) {
    return new Intl.RelativeTimeFormat(options.locale, { numeric: 'auto' }).format(-1, 'day')
  }

  if (calendarWeekStart(fileDate) === calendarWeekStart(currentDate)) {
    return dateTimeFormatter(options.locale, options.timeZone, { weekday: 'long' }).format(date)
  }

  if (fileDate.year === currentDate.year) {
    return dateTimeFormatter(options.locale, options.timeZone, {
      month: 'long',
      day: 'numeric'
    }).format(date)
  }

  return dateTimeFormatter(options.locale, options.timeZone, {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  }).format(date)
}

/** Formats the complete local date and time used by the modified-time tooltip. */
export function formatLibraryFileTimeTooltip(
  timestamp: string | null | undefined,
  options: LibraryFileTimeFormatOptions = {}
): string {
  const date = parseDate(timestamp)
  if (!date) return INVALID_DATE_LABEL

  return dateTimeFormatter(options.locale, options.timeZone, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23'
  }).format(date)
}
