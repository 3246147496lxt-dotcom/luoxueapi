import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ChatMessageItem from '../ChatMessageItem.vue'
import type { ChatMessage } from '@/types/chat'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key} ${Object.values(params).join(' ')}` : key
      ),
    }),
  }
})

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

function assistantMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id: 'assistant-1',
    role: 'assistant',
    content: '',
    createdAt: 1,
    status: 'complete',
    ...overrides,
  }
}

interface MessageStateProps {
  retryable?: boolean
  retrying?: boolean
  announceFailure?: boolean
}

function mountMessage(message: ChatMessage, state: MessageStateProps = {}) {
  return mount(ChatMessageItem, {
    props: { message, ...state },
    global: {
      stubs: {
        Icon: IconStub,
        RouterLink: RouterLinkStub,
      },
    },
  })
}

describe('ChatMessageItem safe Markdown rendering', () => {
  it('剥离样式、表单、媒体和事件属性', () => {
    const wrapper = mountMessage(assistantMessage({
      content: [
        '普通内容',
        '<style>body { display: none }</style>',
        '<p style="position:fixed" onclick="alert(1)">段落</p>',
        '<form action="https://evil.example/collect">',
        '<input name="password" onfocus="alert(1)">',
        '<button type="submit">提交</button>',
        '</form>',
        '<img src="https://evil.example/pixel" onerror="alert(1)">',
      ].join('\n\n'),
    }))

    const markdown = wrapper.get('.chat-message__markdown')
    expect(markdown.find('style').exists()).toBe(false)
    expect(markdown.find('form').exists()).toBe(false)
    expect(markdown.find('input').exists()).toBe(false)
    expect(markdown.find('button').exists()).toBe(false)
    expect(markdown.find('img').exists()).toBe(false)
    expect(markdown.find('[style]').exists()).toBe(false)
    expect(markdown.find('[onclick]').exists()).toBe(false)
    expect(markdown.find('[onfocus]').exists()).toBe(false)
    expect(markdown.find('[onerror]').exists()).toBe(false)
    expect(markdown.html()).not.toContain('evil.example')
  })

  it('保留安全链接、行内代码和代码块', () => {
    const wrapper = mountMessage(assistantMessage({
      content: [
        '[查看文档](https://example.com/docs "文档")',
        '',
        '使用 `const value = 1`。',
        '',
        '```ts',
        'const answer = 42',
        '```',
      ].join('\n'),
    }))

    const markdown = wrapper.get('.chat-message__markdown')
    const link = markdown.get('a')
    expect(link.attributes('href')).toBe('https://example.com/docs')
    expect(link.attributes('title')).toBe('文档')
    expect(markdown.get('p code').text()).toBe('const value = 1')
    expect(markdown.get('pre').attributes('data-language')).toBe('TypeScript')
    expect(markdown.get('pre code').text()).toContain('const answer = 42')
  })

  it('保留标题、列表、引用、表格与分隔线的语义结构', () => {
    const wrapper = mountMessage(assistantMessage({
      content: [
        '## 小标题',
        '',
        '- 第一项',
        '  - 子项',
        '- 第二项',
        '',
        '> 引用内容',
        '',
        '| 名称 | 状态 |',
        '| --- | --- |',
        '| Chat | 可用 |',
        '',
        '---',
      ].join('\n'),
    }))

    const markdown = wrapper.get('.chat-message__markdown')
    expect(markdown.get('h2').text()).toBe('小标题')
    expect(markdown.findAll('ul')).toHaveLength(2)
    expect(markdown.get('blockquote').text()).toBe('引用内容')
    expect(markdown.get('table th').text()).toBe('名称')
    expect(markdown.get('table td').text()).toBe('Chat')
    expect(markdown.find('hr').exists()).toBe(true)
  })

  it('支持三种标准 Markdown 分隔线并保留 Setext 标题语义', () => {
    const wrapper = mountMessage(assistantMessage({
      content: [
        '第一层',
        '',
        '---',
        '',
        '第二层',
        '',
        '***',
        '',
        '第三层',
        '',
        '___',
        '',
        'Setext 标题',
        '---',
      ].join('\n'),
    }))

    const markdown = wrapper.get('.chat-message__markdown')
    expect(markdown.findAll('hr')).toHaveLength(3)
    expect(markdown.get('h2').text()).toBe('Setext 标题')
  })

  it('空的完整回复不会占用一个不可见操作栏', () => {
    const wrapper = mountMessage(assistantMessage())
    expect(wrapper.find('.chat-message__actions').exists()).toBe(false)
  })

  it('移除不安全协议并把未知代码语言限制为安全标签', () => {
    const wrapper = mountMessage(assistantMessage({
      content: [
        '[危险链接](javascript:alert(1))',
        '',
        '```custom-language<script>',
        '<img src=x onerror=alert(1)>',
        '```',
      ].join('\n'),
    }))

    const markdown = wrapper.get('.chat-message__markdown')
    expect(markdown.get('a').attributes('href')).toBeUndefined()
    expect(markdown.get('pre').attributes('data-language')).toBe('custom-language')
    expect(markdown.get('pre code').text()).toContain('<img src=x onerror=alert(1)>')
    expect(markdown.find('img').exists()).toBe(false)
  })

  it('空内容 streaming 只显示正文行首的单个呼吸圆点，并在首个可见 token 后切换为 Markdown', async () => {
    const message = assistantMessage({
      content: ' \n\t',
      status: 'streaming',
    })
    const wrapper = mountMessage(message)
    const article = wrapper.get('article').element

    expect(wrapper.findAll('.chat-message__streaming-dot')).toHaveLength(1)
    expect(wrapper.get('.chat-message__streaming-placeholder').attributes('aria-hidden')).toBe('true')
    expect(wrapper.find('.chat-message__plain').exists()).toBe(false)
    expect(wrapper.find('.chat-message__markdown').exists()).toBe(false)
    expect(wrapper.find('.chat-message__meta').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('chat.message.generating')
    expect(wrapper.get('article').attributes('aria-busy')).toBe('true')

    await wrapper.setProps({
      message: { ...message, content: ' \n**首个回答**' },
    })

    expect(wrapper.get('article').element).toBe(article)
    expect(wrapper.find('.chat-message__streaming-dot').exists()).toBe(false)
    expect(wrapper.find('.chat-message__plain').exists()).toBe(false)
    expect(wrapper.get('.chat-message__markdown').classes()).toContain('chat-message__markdown--streaming')
    expect(wrapper.get('.chat-message__markdown strong').text()).toBe('首个回答')
    expect(wrapper.get('article').attributes('aria-busy')).toBe('true')

    await wrapper.setProps({
      message: { ...message, content: ' \n**首个回答**', status: 'complete' },
    })

    expect(wrapper.get('article').element).toBe(article)
    expect(wrapper.get('.chat-message__markdown').classes()).not.toContain('chat-message__markdown--streaming')
    expect(wrapper.get('.chat-message__markdown strong').text()).toBe('首个回答')
    expect(wrapper.get('article').attributes('aria-busy')).toBeUndefined()
    expect(wrapper.find('.chat-message__actions').exists()).toBe(true)
  })

  it('streaming Markdown 复用完成态的安全清洗链路', () => {
    const content = '<form action="https://evil.example"><input autofocus><strong>继续</strong></form>'
    const wrapper = mountMessage(assistantMessage({
      content,
      status: 'streaming',
    }))

    expect(wrapper.find('.chat-message__plain').exists()).toBe(false)
    expect(wrapper.find('form').exists()).toBe(false)
    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.get('.chat-message__markdown strong').text()).toBe('继续')
    expect(wrapper.find('[autofocus]').exists()).toBe(false)
    expect(wrapper.get('article').attributes('aria-busy')).toBe('true')
  })

  it.each(['complete', 'error', 'stopped'] as const)(
    '%s 状态的空白内容不会继续显示 streaming 圆点',
    (status) => {
      const wrapper = mountMessage(assistantMessage({ content: '\n\t', status }))

      expect(wrapper.find('.chat-message__streaming-dot').exists()).toBe(false)
      expect(wrapper.find('.chat-message__plain').exists()).toBe(false)
      expect(wrapper.find('.chat-message__markdown').exists()).toBe(false)
      expect(wrapper.find('.chat-message__actions').exists()).toBe(false)
      if (status === 'error') {
        expect(wrapper.get('.chat-message__failure').attributes('role')).toBe('group')
        expect(wrapper.find('[role="alert"]').exists()).toBe(false)
      }
      if (status === 'stopped') expect(wrapper.get('.chat-message__status').text()).toBe('chat.message.stopped')
    },
  )

  it.each([
    ['Unauthorized', 'chat.errors.sessionExpired'],
    ['Forbidden', 'chat.errors.permissionDenied'],
    ['Bad Gateway', 'chat.errors.serviceUnavailable'],
  ] as const)('不会展示历史消息中的原始 HTTP 错误 %s', (raw, expectedKey) => {
    const wrapper = mountMessage(assistantMessage({
      status: 'error',
      errorMessage: raw,
    }), { retryable: true })

    expect(wrapper.get('.chat-message__error').text()).toContain(expectedKey)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain(raw)
  })

  it('只有本次实时失败才通过 alert 公告，历史失败保持安静', () => {
    const message = assistantMessage({
      status: 'error',
      errorCode: 'HTTP_502',
      errorMessage: 'Bad Gateway',
    })
    const historical = mountMessage(message, { retryable: true })
    const live = mountMessage(message, { retryable: true, announceFailure: true })

    expect(historical.find('[role="alert"]').exists()).toBe(false)
    expect(live.get('.chat-message__error').attributes('role')).toBe('alert')
    expect(live.get('[role="alert"]').text()).toContain('chat.errors.serviceUnavailable')
  })

  it('可恢复失败在消息内显示文字重试按钮并触发 retry', async () => {
    const wrapper = mountMessage(assistantMessage({
      content: '已经生成的部分内容',
      status: 'error',
      errorCode: 'HTTP_502',
      errorMessage: 'Bad Gateway',
    }), { retryable: true })

    expect(wrapper.get('.chat-message__markdown').text()).toBe('已经生成的部分内容')
    expect(wrapper.get('.chat-message__failure').element.closest('article')).toBe(
      wrapper.get('article').element,
    )
    expect(wrapper.get('.chat-message__retry-button').text()).toBe('chat.actions.retryFailed')
    expect(wrapper.findAll('.chat-message__icon-button')).toHaveLength(1)

    await wrapper.get('.chat-message__retry-button').trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })

  it('重试中保留失败按钮并呈现不可重复提交的忙碌状态', async () => {
    const wrapper = mountMessage(assistantMessage({
      status: 'error',
      errorCode: 'NETWORK_ERROR',
      errorMessage: 'Failed to fetch',
    }), { retrying: true })

    const button = wrapper.get<HTMLButtonElement>('.chat-message__retry-button')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-busy')).toBe('true')
    expect(button.text()).toBe('chat.actions.retrying')

    await button.trigger('click')
    expect(wrapper.emitted('retry')).toBeUndefined()
  })

  it('不可恢复和已被替代的失败不显示重试或旧错误', () => {
    const forbidden = mountMessage(assistantMessage({
      status: 'error',
      errorCode: 'HTTP_403',
      errorMessage: 'Forbidden',
    }), { retryable: true })
    expect(forbidden.find('.chat-message__retry-button').exists()).toBe(false)
    expect(forbidden.find('.chat-message__actions').exists()).toBe(false)

    const superseded = mountMessage(assistantMessage({
      status: 'error',
      errorCode: 'HTTP_502',
      errorMessage: 'Bad Gateway',
      excludedFromContext: true,
      supersededByMessageId: 'assistant-2',
    }), { retryable: true })
    expect(superseded.find('[role="alert"]').exists()).toBe(false)
    expect(superseded.find('.chat-message__retry-button').exists()).toBe(false)
    expect(superseded.text()).not.toContain('Bad Gateway')
  })

  it('不会为 user streaming 状态渲染 assistant 呼吸圆点', () => {
    const wrapper = mountMessage({
      id: 'user-streaming',
      role: 'user',
      content: '',
      createdAt: 1,
      status: 'streaming',
    })

    expect(wrapper.find('.chat-message__streaming-dot').exists()).toBe(false)
  })

  it.each(['pending', 'charged', 'not_charged', 'subscription', 'failed'] as const)(
    '结算状态为 %s 时仍只展示回答内容，不把计费明细带进 Chat 消息流',
    (settlementStatus) => {
      const wrapper = mountMessage(assistantMessage({
        content: 'Visible assistant answer',
        receiptId: `receipt-${settlementStatus}`,
        settlementStatus,
        actualModel: 'internal-billing-model-id',
        inputTokens: 123456,
        outputTokens: 654321,
        cacheCreationTokens: 222222,
        cacheReadTokens: 333333,
        grossCost: 987.654321,
        chargedAmount: 876.54321,
        balanceBefore: 765.4321,
        balanceAfter: 654.321,
      }))

      expect(wrapper.get('.chat-message__markdown').text()).toBe('Visible assistant answer')
      expect(wrapper.find('[data-test="chat-receipt"]').exists()).toBe(false)
      expect(wrapper.find('[data-test="chat-receipt-recharge"]').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('internal-billing-model-id')
      expect(wrapper.text()).not.toContain('123456')
      expect(wrapper.text()).not.toContain('987.654321')
      expect(wrapper.text()).not.toContain('876.54321')
      expect(wrapper.text()).not.toContain('765.4321')
    },
  )

  it('被后续 retry 替代的旧 attempt 保留内容并标记为 superseded', () => {
    const wrapper = mountMessage(assistantMessage({
      content: 'Old partial answer',
      status: 'stopped',
      excludedFromContext: true,
      supersededByMessageId: 'assistant-2',
    }))

    expect(wrapper.text()).toContain('Old partial answer')
    expect(wrapper.get('.chat-message__status').text()).toBe('chat.message.superseded')
  })
})
