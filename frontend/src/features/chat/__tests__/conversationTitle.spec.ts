import { describe, expect, it } from 'vitest'
import {
  CHAT_CONVERSATION_TITLE_PREVIEW_UNITS,
  toChatConversationTitlePreview,
} from '../conversationTitle'

describe('conversation title presentation', () => {
  it('keeps short titles unchanged and normalizes whitespace', () => {
    expect(toChatConversationTitlePreview('  Short   title  ')).toBe('Short title')
  })

  it('matches the Sidebar preview budget for long Chinese and Latin titles', () => {
    expect(toChatConversationTitlePreview(
      '我现在的网站需要一个语音。对话聊天的功能，然后的话，我需要把这个语音聊天的音频就是它...',
    )).toBe('我现在的网站需要一个语音。对...')
    expect(toChatConversationTitlePreview('abcdefghijklmnopqrstuvwxyz0123456789'))
      .toBe('abcdefghijklmnopqrstuvwxyz01...')
    expect(CHAT_CONVERSATION_TITLE_PREVIEW_UNITS).toBe(28)
  })

  it('never cuts a joined emoji grapheme in half', () => {
    expect(toChatConversationTitlePreview(`一二三四五六七八九十甲乙丙${'👨‍👩‍👧‍👦'}后续`))
      .toBe(`一二三四五六七八九十甲乙丙${'👨‍👩‍👧‍👦'}...`)
  })
})
