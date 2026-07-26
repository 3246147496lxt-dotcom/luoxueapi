import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import type { ChatConversation } from '@/types/chat'
import ChatHistoryPanel from '../ChatHistoryPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string) => key,
  }),
}))

const IconStub = {
  props: ['name'],
  template: '<span :data-icon="name" />',
}

let wrapper: VueWrapper | undefined

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('ChatHistoryPanel focus management', () => {
  it('重命名保存后把焦点恢复到对应会话按钮', async () => {
    const conversation: ChatConversation = {
      id: 'conversation-focus',
      userId: '7',
      title: 'Original title',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: {
        conversations: [conversation],
        activeId: conversation.id,
        mobile: true,
      },
      global: {
        stubs: { Icon: IconStub },
      },
    })

    await wrapper.get('button[aria-label="chat.actions.rename"]').trigger('click')
    const input = wrapper.get('input[aria-label="chat.history.renameLabel"]')
    expect(document.activeElement).toBe(input.element)

    await input.setValue('Renamed title')
    await wrapper.get('form.chat-history__rename').trigger('submit')
    await nextTick()

    expect(wrapper.emitted('rename')).toEqual([[conversation.id, 'Renamed title']])
    expect(document.activeElement).toBe(wrapper.get('.chat-history__select').element)
  })

  it('Escape 只取消重命名并保持焦点在历史面板内', async () => {
		const conversation: ChatConversation = {
			id: 'conversation-escape',
			userId: '7',
			title: 'Original title',
			model: 'gpt-5',
			messages: [],
			createdAt: Date.now(),
			updatedAt: Date.now(),
		}
		wrapper = mount(ChatHistoryPanel, {
			attachTo: document.body,
			props: { conversations: [conversation], activeId: conversation.id, mobile: true },
			global: { stubs: { Icon: IconStub } },
		})
		const bubbled = vi.fn()
		window.addEventListener('keydown', bubbled)

		await wrapper.get('button[aria-label="chat.actions.rename"]').trigger('click')
		await wrapper.get('input[aria-label="chat.history.renameLabel"]').trigger('keydown', { key: 'Escape' })
		await nextTick()

		expect(wrapper.find('.chat-history__rename').exists()).toBe(false)
		expect(document.activeElement).toBe(wrapper.get('.chat-history__select').element)
		expect(bubbled).not.toHaveBeenCalled()
		window.removeEventListener('keydown', bubbled)
	})
})
