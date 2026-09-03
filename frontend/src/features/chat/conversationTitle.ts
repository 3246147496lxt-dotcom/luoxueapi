export const CHAT_CONVERSATION_TITLE_PREVIEW_UNITS = 28

type GraphemeSegmenter = {
  segment: (value: string) => Iterable<{ segment: string }>
}

type GraphemeSegmenterConstructor = new (
  locale?: string | string[],
  options?: { granularity: 'grapheme' },
) => GraphemeSegmenter

const Segmenter = (Intl as typeof Intl & { Segmenter?: GraphemeSegmenterConstructor }).Segmenter
const graphemeSegmenter = Segmenter ? new Segmenter(undefined, { granularity: 'grapheme' }) : null

function graphemes(value: string): string[] {
  if (!graphemeSegmenter) return Array.from(value)
  return Array.from(graphemeSegmenter.segment(value), ({ segment }) => segment)
}

function displayUnits(character: string): number {
  const codePoint = character.codePointAt(0) ?? 0
  return codePoint <= 0xff ? 1 : 2
}

export function toChatConversationTitlePreview(value: string): string {
  const normalized = value.replace(/\s+/g, ' ').trim()
  let units = 0
  let preview = ''

  for (const character of graphemes(normalized)) {
    const nextUnits = displayUnits(character)
    if (units + nextUnits > CHAT_CONVERSATION_TITLE_PREVIEW_UNITS) {
      return `${preview}...`
    }
    preview += character
    units += nextUnits
  }

  return preview
}
