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

function mountMessage(message: ChatMessage) {
  return mount(ChatMessageItem, {
    props: { message },
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
    expect(markdown.get('pre code').text()).toContain('const answer = 42')
  })

  it('streaming 阶段仅输出转义纯文本，不挂载富 HTML', () => {
    const content = '<form action="https://evil.example"><input autofocus><strong>继续</strong></form>'
    const wrapper = mountMessage(assistantMessage({
      content,
      status: 'streaming',
    }))

    expect(wrapper.find('.chat-message__markdown').exists()).toBe(false)
    expect(wrapper.get('.chat-message__plain').text()).toBe(content)
    expect(wrapper.find('form').exists()).toBe(false)
    expect(wrapper.find('input').exists()).toBe(false)
    expect(wrapper.find('strong').exists()).toBe(false)
    expect(wrapper.get('.chat-message__plain').html()).toContain('&lt;form')
  })

  it('显示服务端结算回执、实际模型、Token、费用与余额', () => {
    const wrapper = mountMessage(assistantMessage({
      content: 'Done',
      receiptId: 'receipt-1',
      settlementStatus: 'charged',
      actualModel: 'gpt-5.5-2026-07-01',
      inputTokens: 1200,
      outputTokens: 80,
      cacheCreationTokens: 20,
      cacheReadTokens: 900,
      grossCost: 0.003,
      chargedAmount: 0.0024,
      balanceBefore: 0.5024,
      balanceAfter: 0.5,
    }))
    const receipt = wrapper.get('[data-test="chat-receipt"]')

    expect(receipt.text()).toContain('chat.receipt.status.charged')
    expect(receipt.text()).toContain('gpt-5.5-2026-07-01')
    expect(receipt.text()).toContain('1,200')
    expect(receipt.text()).toContain('900')
    expect(receipt.text()).toContain('$0.003000')
    expect(receipt.text()).toContain('$0.002400')
    expect(receipt.text()).toContain('$0.50')
    expect(wrapper.get('[data-test="chat-receipt-recharge"]').attributes('href')).toBe('/purchase')
  })

  it('pending 回执仅显示核对状态，不提前展示或估算费用', () => {
    const wrapper = mountMessage(assistantMessage({
      content: 'Partial',
      receiptId: 'receipt-pending',
      settlementStatus: 'pending',
      requestedModel: 'gpt-5.5',
    }))
    const receipt = wrapper.get('[data-test="chat-receipt"]')

    expect(receipt.text()).toContain('chat.receipt.status.pending')
    expect(receipt.find('.chat-message__receipt-details').exists()).toBe(false)
    expect(receipt.find('[data-test="chat-receipt-recharge"]').exists()).toBe(false)
    expect(receipt.text()).not.toContain('gpt-5.5')
  })

  it('订阅额度和未扣费状态使用独立结算文案', async () => {
    const wrapper = mountMessage(assistantMessage({
      receiptId: 'receipt-subscription',
      settlementStatus: 'subscription',
      chargedAmount: 0,
    }))
    expect(wrapper.get('[data-test="chat-receipt"]').text())
      .toContain('chat.receipt.status.subscription')

    await wrapper.setProps({
      message: assistantMessage({
        receiptId: 'receipt-free',
        settlementStatus: 'not_charged',
        chargedAmount: 0,
      }),
    })
    expect(wrapper.get('[data-test="chat-receipt"]').text())
      .toContain('chat.receipt.status.notCharged')
  })

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
