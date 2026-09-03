export type ChatGreetingCategoryId =
  | 'general'
  | 'productivity'
  | 'development'
  | 'creation'
  | 'learning'

export interface ChatGreetingGroup {
  id: ChatGreetingCategoryId
  label: string
  prompts: readonly string[]
}

export const CHAT_GREETING_GROUPS: readonly ChatGreetingGroup[] = [
  {
    id: 'general',
    label: '通用探索',
    prompts: [
      '今天想探索什么？',
      '有什么事情想一起解决？',
      '最近有什么想法想聊聊？',
      '今天准备完成什么？',
      '有什么问题正在困扰你？',
      '需要一个新的思路吗？',
      '让我们开始一次新的探索。',
      '有什么值得深入研究的话题？',
      '说说你的想法吧。',
      '有什么想法想和 AI 讨论？',
    ],
  },
  {
    id: 'productivity',
    label: '工作效率',
    prompts: [
      '有什么工作需要 AI 协助？',
      '准备开始一个新项目了吗？',
      '需要整理一下你的思路吗？',
      '有什么计划需要完善？',
      '需要制定一个方案吗？',
      '有什么内容需要优化？',
      '让 AI 帮你提高效率。',
      '需要分析一份资料吗？',
      '想快速解决一个问题吗？',
      '有什么任务正在等待完成？',
    ],
  },
  {
    id: 'development',
    label: '编程开发',
    prompts: [
      '遇到代码问题了吗？',
      '需要一起调试程序吗？',
      '有新的项目想法吗？',
      '需要设计技术方案吗？',
      '想优化你的代码吗？',
      '遇到了 Bug 需要排查吗？',
      '需要生成一些代码吗？',
      '想讨论系统架构吗？',
      '需要帮助理解技术文档吗？',
      '今天想解决什么开发问题？',
    ],
  },
  {
    id: 'creation',
    label: '创作内容',
    prompts: [
      '想创造一个新的想法吗？',
      '需要帮助构思内容吗？',
      '一起完成你的创意吧。',
      '有什么内容需要打磨？',
      '想写点什么吗？',
      '需要优化你的文案吗？',
      '想设计一个新的方案吗？',
    ],
  },
  {
    id: 'learning',
    label: '学习研究',
    prompts: [
      '今天想学习什么？',
      '有什么知识想深入了解？',
      '帮你理解一个复杂问题。',
      '有什么概念需要解释？',
      '想探索一个新领域吗？',
      '需要整理学习资料吗？',
    ],
  },
]

export const CHAT_GREETINGS: readonly string[] = CHAT_GREETING_GROUPS.flatMap(
  ({ prompts }) => prompts,
)

export function pickChatGreeting(
  previousGreeting?: string,
  random: () => number = Math.random,
): string {
  const candidates = previousGreeting && CHAT_GREETINGS.length > 1
    ? CHAT_GREETINGS.filter((greeting) => greeting !== previousGreeting)
    : CHAT_GREETINGS
  const sample = random()
  const normalizedSample = Number.isFinite(sample)
    ? Math.min(Math.max(sample, 0), 1)
    : 0
  const index = Math.min(
    Math.floor(normalizedSample * candidates.length),
    candidates.length - 1,
  )
  return candidates[index] ?? '今天想探索什么？'
}
