import { describe, expect, it } from 'vitest'
import {
  CHAT_GREETING_GROUPS,
  CHAT_GREETINGS,
  pickChatGreeting,
} from '../chatGreetings'

describe('chatGreetings', () => {
  it('keeps all five product categories and the complete Chinese prompt set', () => {
    expect(CHAT_GREETING_GROUPS.map(({ label }) => label)).toEqual([
      '通用探索',
      '工作效率',
      '编程开发',
      '创作内容',
      '学习研究',
    ])
    expect(CHAT_GREETING_GROUPS.map(({ prompts }) => prompts.length)).toEqual([10, 10, 10, 7, 6])
    expect(CHAT_GREETINGS).toHaveLength(43)
    expect(new Set(CHAT_GREETINGS).size).toBe(CHAT_GREETINGS.length)
    expect(CHAT_GREETINGS).toContain('今天想探索什么？')
    expect(CHAT_GREETINGS).toContain('需要整理学习资料吗？')
  })

  it('selects a prompt deterministically across the random range', () => {
    expect(pickChatGreeting(undefined, () => 0)).toBe(CHAT_GREETINGS[0])
    expect(pickChatGreeting(undefined, () => 0.999999)).toBe(CHAT_GREETINGS.at(-1))
    expect(pickChatGreeting(undefined, () => 1)).toBe(CHAT_GREETINGS.at(-1))
    expect(pickChatGreeting(undefined, () => Number.NaN)).toBe(CHAT_GREETINGS[0])
  })

  it('does not repeat the visible prompt when a new chat is created', () => {
    expect(pickChatGreeting(CHAT_GREETINGS[0], () => 0)).toBe(CHAT_GREETINGS[1])
  })
})
