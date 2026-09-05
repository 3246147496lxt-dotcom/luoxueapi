import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { routerKey } from 'vue-router'
import { toChatConversationTitlePreview } from '@/features/chat/conversationTitle'
import type { ChatConversation } from '@/types/chat'
import { useAppStore } from '@/stores/app'
import ChatHistoryPanel from '../ChatHistoryPanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key,
    }),
  }
})

const IconStub = {
  props: ['name', 'size'],
  template: '<span :data-icon="name" :data-size="size" />',
}

const RouterLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

const WorkspaceSidebarBrandStub = {
  props: ['homePath'],
  template: '<a data-testid="brand-stub" :href="homePath">落雪AI <span>Pro</span></a>',
}

const shellStubs = {
  Icon: IconStub,
  WorkspaceSidebarBrand: WorkspaceSidebarBrandStub,
  RouterLink: RouterLinkStub,
  SidebarCollapseIcon: { template: '<svg />' },
  UserAccountCard: {
    props: ['collapsed'],
    template: '<div data-testid="account-stub" :data-collapsed="String(collapsed)" />',
  },
}

let wrapper: VueWrapper | undefined

async function openConversationActions(target = wrapper!) {
  const trigger = target.get('.chat-history__more-action')
  await trigger.trigger('click')
  await nextTick()
  return target.get('[role="menu"]')
}

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('ChatHistoryPanel focus management', () => {
  it('不显示空的对话记录提示', () => {
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [] },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })

    expect(wrapper.find('.chat-history__empty').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('chat.history.empty')
  })

  it('uses the same compact visual title as the delete confirmation while keeping the full accessible name', () => {
    const conversation: ChatConversation = {
      id: 'long-title',
      userId: '7',
      title: '我现在的网站需要一个语音。对话聊天的功能，然后的话，我需要把这个语音聊天的音频就是它...',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [conversation] },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })

    const select = wrapper.get('.chat-history__select')
    expect(select.get('strong').text()).toBe(toChatConversationTitlePreview(conversation.title))
    expect(select.attributes('aria-label')).toBe(conversation.title)
    expect(select.get('strong').attributes('title')).toBe(conversation.title)
  })

  it('将搜索保持为图标入口，并在关闭时清除隐藏筛选与恢复焦点', async () => {
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [], searchQuery: '' },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })

    const trigger = wrapper.get('button[aria-label="chat.history.searchLabel"]')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('.chat-history__search').exists()).toBe(false)
    expect(wrapper.find('.workspace-sidebar-frame__footer').exists()).toBe(false)

    await trigger.trigger('click')
    await nextTick()

    const search = wrapper.get('.chat-history__search')
    const input = search.get('input')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-controls')).toBe(search.attributes('id'))
    expect(document.activeElement).toBe(input.element)

    await input.setValue('SMTP')
    expect(wrapper.emitted('update:searchQuery')).toContainEqual(['SMTP'])

    await input.trigger('keydown', { key: 'Escape' })
    await nextTick()

    expect(wrapper.find('.chat-history__search').exists()).toBe(false)
    expect(wrapper.emitted('update:searchQuery')).toContainEqual([''])
    expect(document.activeElement).toBe(trigger.element)
  })

  it('在 Chat shell 点击搜索后于新聊天下方展开内联搜索，而不是打开弹窗', async () => {
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })

    const trigger = wrapper.get('.workspace-sidebar-header__search')
    expect(wrapper.find('.chat-history__search').exists()).toBe(false)
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)

    await trigger.trigger('click')
    await nextTick()

    const search = wrapper.get('.chat-history__search')
    expect(search.element.parentElement?.querySelector('.chat-history__list'))
      .toBeTruthy()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(search.get('input').element)

    await trigger.trigger('click')
    await nextTick()
    expect(wrapper.find('.chat-history__search').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
  })

  it('为并存的桌面与移动搜索入口生成不同控制目标', async () => {
    const pair = mount({
      components: { ChatHistoryPanel },
      template: `
        <div>
          <ChatHistoryPanel :conversations="[]" />
          <ChatHistoryPanel :conversations="[]" mobile />
        </div>
      `,
    }, {
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })

    const targets = pair.findAll('button[aria-label="chat.history.searchLabel"]')
      .map((trigger) => trigger.attributes('aria-controls'))

    expect(targets).toHaveLength(2)
    expect(targets[0]).toBeTruthy()
    expect(targets[0]).not.toBe(targets[1])

    pair.unmount()
  })

  it('让移动抽屉在共享 Header 中保留明确的搜索与关闭入口', () => {
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true, mobile: true },
      global: {
        stubs: {
          ...shellStubs,
        },
      },
    })

    const headerButtons = wrapper.get('.workspace-sidebar-header').findAll('button')
    expect(headerButtons.map(button => button.attributes('aria-label'))).toEqual([
      'chat.actions.closeHistory',
      'chat.history.searchLabel',
    ])
    expect(headerButtons[0]?.classes()).toContain('workspace-sidebar-header__close')
  })

  it('把所有时间范围会话合并到唯一的“最近”分组并保留输入顺序', () => {
    const conversations: ChatConversation[] = [
      { id: 'today', userId: '7', title: 'Today', model: 'gpt-5', messages: [], createdAt: Date.now(), updatedAt: Date.now() },
      { id: 'week', userId: '7', title: 'Last week', model: 'gpt-5', messages: [], createdAt: 1, updatedAt: Date.now() - 8 * 86_400_000 },
      { id: 'older', userId: '7', title: 'Older', model: 'gpt-5', messages: [], createdAt: 1, updatedAt: 1 },
    ]
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations, shell: true },
      global: { stubs: shellStubs },
    })

    expect(wrapper.findAll('.chat-history__group h2')).toHaveLength(1)
    expect(wrapper.get('.chat-history__group h2').text()).toBe('chat.history.recent')
    expect(wrapper.findAll('.chat-history__select strong').map((title) => title.text()))
      .toEqual(['Today', 'Last week', 'Older'])
    expect(wrapper.text()).not.toContain('chat.history.previousDays')
    expect(wrapper.text()).not.toContain('chat.history.older')
    expect(wrapper.find('.chat-history__clear').exists()).toBe(false)
  })

  it('允许展开和收起最近分组，并同步 aria-expanded 与列表可见性', async () => {
    const conversation: ChatConversation = {
      id: 'recent-toggle',
      userId: '7',
      title: 'Recent conversation',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [conversation], shell: true },
      global: { stubs: shellStubs },
    })

    const toggle = wrapper.get('[data-testid="chat-history-recent-toggle"]')
    const listId = toggle.attributes('aria-controls')
    expect(listId).toBeTruthy()
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get(`#${listId}`).isVisible()).toBe(true)

    await toggle.trigger('click')
    await nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.get(`#${listId}`).attributes('style')).toContain('display: none')

    await toggle.trigger('click')
    await nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get(`#${listId}`).attributes('style')).not.toContain('display: none')
  })

  it('在会话悬停操作区提供编辑入口，并保留更多操作菜单', async () => {
    const conversation: ChatConversation = {
      id: 'edit-action',
      userId: '7',
      title: 'Editable conversation',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [conversation], activeId: conversation.id, shell: true },
      global: { stubs: shellStubs },
    })

    const edit = wrapper.get('.chat-history__edit-action')
    expect(edit.attributes('aria-label')).toBe('chat.actions.rename')
    expect(wrapper.get('.chat-history__more-action').attributes('aria-haspopup')).toBe('menu')

    await edit.trigger('click')
    await nextTick()
    expect(wrapper.get('input[aria-label="chat.history.renameLabel"]').element)
      .toHaveProperty('value', conversation.title)
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('按官方顺序呈现五项一级导航，并让未接入入口只显示提示而不伪造路由', async () => {
    const appStore = useAppStore()
    const push = vi.fn().mockResolvedValue(undefined)
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: {
        stubs: shellStubs,
        provide: {
          [routerKey as symbol]: { push },
        },
      },
    })

    const fixedNewChat = wrapper.get('.chat-history__header .chat-history__new')
    const scrollingNav = wrapper.get('.chat-history__list .chat-history__primary-nav--scrolling')
    const actions = [fixedNewChat, ...scrollingNav.findAll('button')]

    expect(actions.map((action) => action.attributes('aria-label'))).toEqual([
      'chat.actions.newChat',
      'chat.navigation.fileLibrary',
      'chat.navigation.projects',
      'chat.navigation.scheduled',
      'chat.navigation.plugins',
    ])
    expect(actions.map((action) => action.get('[data-icon]').attributes('data-icon'))).toEqual([
      'chatSidebarCompose',
      'chatSidebarLibrary',
      'chatSidebarProjects',
      'chatSidebarScheduled',
      'chatSidebarPlugins',
    ])
    expect(scrollingNav.findAll('a')).toHaveLength(0)
    expect(wrapper.find('.chat-history__list .chat-history__new').exists()).toBe(false)
    expect(actions[1]!.attributes('aria-disabled')).toBeUndefined()
    expect(actions[2]!.attributes('aria-disabled')).toBeUndefined()
    expect(actions.slice(3).every((action) => action.attributes('aria-disabled') === 'true'))
      .toBe(true)

    await actions[0]!.trigger('click')
    expect(wrapper.emitted('new')).toEqual([[]])

    await actions[1]!.trigger('click')
    await flushPromises()
    expect(push).toHaveBeenCalledOnce()
    expect(push).toHaveBeenCalledWith('/library')
    expect(appStore.toasts).toHaveLength(0)

    await actions[2]!.trigger('click')
    await flushPromises()
    expect(push).toHaveBeenCalledWith('/projects')
    expect(push).toHaveBeenCalledTimes(2)
    expect(appStore.toasts).toHaveLength(0)
    expect(wrapper.emitted('new')).toEqual([[]])
  })

  it('仅在显式新聊天状态选中新聊天入口，并随 activeId 同步迁移选中态', async () => {
    const conversation: ChatConversation = {
      id: 'empty-conversation',
      userId: '7',
      title: 'Empty conversation',
      model: 'gpt-5.6-sol',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [conversation], activeId: null, shell: true },
      global: { stubs: shellStubs },
    })

    const newChat = wrapper.get('.chat-history__new')
    const conversationButton = wrapper.get('.chat-history__select')
    expect(newChat.classes()).toContain('chat-history__new--active')
    expect(newChat.attributes('aria-current')).toBe('page')
    expect(conversationButton.attributes('aria-current')).toBeUndefined()

    await wrapper.setProps({ activeId: conversation.id })
    expect(newChat.classes()).not.toContain('chat-history__new--active')
    expect(newChat.attributes('aria-current')).toBeUndefined()
    expect(conversationButton.attributes('aria-current')).toBe('page')

    await wrapper.setProps({ activeId: null })
    expect(newChat.classes()).toContain('chat-history__new--active')
    expect(newChat.attributes('aria-current')).toBe('page')
    expect(conversationButton.attributes('aria-current')).toBeUndefined()
  })

  it('只滚动次级入口与历史，并在离开顶部后显示固定新聊天分割线', async () => {
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })

    const header = wrapper.get('.chat-history__header')
    const list = wrapper.get('.chat-history__list')
    const listElement = list.element as HTMLElement

    expect(header.find('.chat-history__new').exists()).toBe(true)
    expect(header.find('.chat-history__primary-action').exists()).toBe(false)
    expect(list.findAll('.chat-history__primary-action')).toHaveLength(4)
    expect(header.attributes('data-scrolled-from-top')).toBeUndefined()

    listElement.scrollTop = 24
    await list.trigger('scroll')
    expect(header.attributes('data-scrolled-from-top')).toBe('true')

    listElement.scrollTop = 0
    await list.trigger('scroll')
    expect(header.attributes('data-scrolled-from-top')).toBeUndefined()
  })

  it('在桌面 Sidebar 内切换共享 260px 与 68px 模式，并保留可访问焦点', async () => {
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: {
        conversations: [{
          id: 'collapse',
          userId: '7',
          title: 'Collapsible conversation',
          model: 'gpt-5',
          messages: [],
          createdAt: Date.now(),
          updatedAt: Date.now(),
        }],
        shell: true,
      },
      global: { stubs: shellStubs },
    })

    const toggle = wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]')
    const aside = wrapper.get('.chat-history')
    const newChatButton = wrapper.get('.chat-history__new')
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(toggle.attributes('aria-controls')).toBe(aside.attributes('id'))

    await toggle.trigger('click')
    await nextTick()

    expect(aside.classes()).toContain('chat-history--collapsed')
    const collapsedToggle = wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]')
    expect(collapsedToggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-testid="brand-stub"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="app-mode-switch"]').exists()).toBe(false)
    expect(wrapper.get('.chat-history__list').isVisible()).toBe(false)
    expect(document.activeElement).toBe(collapsedToggle.element)
    expect(wrapper.get('.chat-history__new').element).toBe(newChatButton.element)
    expect(newChatButton.classes()).toContain('chat-history__new--collapsed')
    expect(newChatButton.attributes('aria-label')).toBe('chat.actions.newChat')
    expect(newChatButton.get('span').attributes('aria-hidden')).toBe('true')
    expect(wrapper.findAll('.chat-history__primary-action')).toHaveLength(0)

    const collapsedActions = wrapper.get('[data-testid="chat-sidebar-collapsed-nav"]')
      .findAll('button, a')
    expect(collapsedActions.map((action) => action.attributes('aria-label')))
      .toEqual([
        'chat.history.searchLabel',
        'nav.chatMode',
        'chat.navigation.fileLibrary',
        'chat.navigation.projects',
      ])
    expect(collapsedActions[2]!.attributes('href')).toBe('/library')

    await wrapper.get('button[aria-label="chat.history.searchLabel"]').trigger('click')
    await nextTick()

    expect(aside.classes()).not.toContain('chat-history--collapsed')
    expect(wrapper.find('[data-testid="app-mode-switch"]').exists()).toBe(false)
    expect(wrapper.find('.chat-history__search').exists()).toBe(true)
    expect(document.activeElement).toBe(wrapper.get('.chat-history__search input').element)
  })

  it('在并存与重挂载的 Chat Sidebar 间共享同一 Pinia 折叠状态', async () => {
    const appStore = useAppStore()
    const first = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })

    await first.get('[data-testid="chat-sidebar-collapse-toggle"]').trigger('click')
    await nextTick()
    expect(appStore.sidebarCollapsed).toBe(true)

    const second = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })
    expect(second.get('.chat-history').classes()).toContain('chat-history--collapsed')
    expect(second.get('[data-testid="account-stub"]').attributes('data-collapsed')).toBe('true')

    first.unmount()
    const remounted = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })
    expect(remounted.get('.chat-history').classes()).toContain('chat-history--collapsed')

    appStore.setSidebarCollapsed(false)
    await nextTick()
    expect(second.get('.chat-history').classes()).not.toContain('chat-history--collapsed')
    expect(remounted.get('.chat-history').classes()).not.toContain('chat-history--collapsed')

    second.unmount()
    remounted.unmount()
  })

  it('窄桌面 overlay 始终渲染完整 260px Sidebar，并保留桌面 68px 偏好', async () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    wrapper = mount(ChatHistoryPanel, {
      props: {
        conversations: [],
        shell: true,
        overlay: true,
        sidebarId: 'workspace-chat-sidebar-overlay',
      },
      global: { stubs: shellStubs },
    })

    const overlaySidebar = wrapper.get('#workspace-chat-sidebar-overlay')
    expect(appStore.sidebarCollapsed).toBe(true)
    expect(overlaySidebar.classes()).toContain('workspace-sidebar-frame--overlay')
    expect(overlaySidebar.classes()).not.toContain('chat-history--collapsed')
    expect(overlaySidebar.attributes('data-sidebar-collapsed')).toBe('false')
    expect(overlaySidebar.attributes('data-sidebar-placement')).toBe('flow')
    expect(wrapper.find('[data-testid="chat-sidebar-collapse-toggle"]').exists()).toBe(false)
    expect(wrapper.get('.workspace-sidebar-header__close').attributes('aria-label'))
      .toBe('chat.actions.closeHistory')
    expect(wrapper.get('.workspace-sidebar-header__close')
      .find('.workspace-responsive-sidebar-icon--close').exists()).toBe(true)
    expect(wrapper.get('.workspace-sidebar-header').findAll('button')
      .map(button => button.attributes('aria-label'))).toEqual([
      'chat.history.searchLabel',
      'chat.actions.closeHistory',
    ])
    expect(wrapper.find('[data-testid="app-mode-switch"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-stub"]').attributes('data-collapsed')).toBe('false')

    await wrapper.get('.workspace-sidebar-header__close').trigger('click')
    expect(wrapper.emitted('close')).toEqual([[]])
    expect(appStore.sidebarCollapsed).toBe(true)

    await wrapper.setProps({ overlay: false })
    await nextTick()
    expect(wrapper.get('.chat-history').classes()).toContain('chat-history--collapsed')
    expect(wrapper.get('.chat-history').attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.get('[data-testid="account-stub"]').attributes('data-collapsed')).toBe('true')
  })

  it('折叠只隐藏搜索与重命名界面，不清空查询或未保存标题', async () => {
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: {
        conversations: [{
          id: 'draft',
          userId: '7',
          title: 'Original title',
          model: 'gpt-5',
          messages: [],
          createdAt: Date.now(),
          updatedAt: Date.now(),
        }],
        shell: true,
        searchQuery: 'SMTP',
      },
      global: { stubs: shellStubs },
    })

    const menu = await openConversationActions()
    await menu.get('[data-action="rename"]').trigger('click')
    await flushPromises()
    await wrapper.get('input[aria-label="chat.history.renameLabel"]').setValue('Unsaved title')
    const collapse = wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]')
    await collapse.trigger('click')
    await nextTick()

    expect(wrapper.find('.chat-history__search').exists()).toBe(false)
    expect(wrapper.get('.chat-history__rename').isVisible()).toBe(false)
    expect(wrapper.get('button[aria-label="chat.history.searchLabel"]').attributes('aria-expanded'))
      .toBe('false')
    expect(wrapper.emitted('update:searchQuery')).toBeUndefined()

    await wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]').trigger('click')
    await nextTick()

    expect(wrapper.get('.chat-history__search input').element).toHaveProperty('value', 'SMTP')
    expect(wrapper.get('input[aria-label="chat.history.renameLabel"]').element)
      .toHaveProperty('value', 'Unsaved title')
    expect(wrapper.get('button[aria-label="chat.history.searchLabel"]').attributes('aria-expanded'))
      .toBe('true')
  })

  it('移动抽屉保持完整结构并保留桌面折叠请求', async () => {
    const appStore = useAppStore()
    appStore.setSidebarCollapsed(true)
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true, mobile: true },
      global: { stubs: shellStubs },
    })

    expect(appStore.sidebarCollapsed).toBe(true)
    expect(wrapper.get('.chat-history').classes()).not.toContain('chat-history--collapsed')
    expect(wrapper.find('[data-testid="chat-sidebar-collapse-toggle"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="app-mode-switch"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="account-stub"]').attributes('data-collapsed')).toBe('false')

    await wrapper.setProps({ searchQuery: 'SMTP' })
    expect(appStore.sidebarCollapsed).toBe(true)

    await wrapper.setProps({ mobile: false })
    expect(wrapper.get('.chat-history').classes()).toContain('chat-history--collapsed')
    expect(wrapper.get('[data-testid="account-stub"]').attributes('data-collapsed')).toBe('true')
  })

  it('把品牌、搜索和折叠入口放在同一顶部水平容器内', () => {
    wrapper = mount(ChatHistoryPanel, {
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })

    const header = wrapper.get('.workspace-sidebar-header')
    expect(header.find('[data-testid="brand-stub"]').exists()).toBe(true)
    expect(header.find('button[aria-label="chat.history.searchLabel"]').exists()).toBe(true)
    expect(header.find('[data-testid="chat-sidebar-collapse-toggle"]').exists()).toBe(true)
    expect(header.get('.workspace-sidebar-header__actions').element.parentElement).toBe(header.element)
  })

  it('点击共享折叠入口后恢复完整 Sidebar 并把焦点留在同一按钮', async () => {
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [], shell: true },
      global: { stubs: shellStubs },
    })

    await wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]').trigger('click')
    await nextTick()
    await wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]').trigger('click')
    await nextTick()

    const collapse = wrapper.get('[data-testid="chat-sidebar-collapse-toggle"]')
    expect(wrapper.get('.chat-history').classes()).not.toContain('chat-history--collapsed')
    expect(document.activeElement).toBe(collapse.element)
  })

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
        stubs: { Icon: IconStub, RouterLink: RouterLinkStub },
      },
    })

    const menu = await openConversationActions()
    await menu.get('[data-action="rename"]').trigger('click')
    await flushPromises()
    const input = wrapper.get('input[aria-label="chat.history.renameLabel"]')
    expect(document.activeElement).toBe(input.element)

    await input.setValue('Renamed title')
    await wrapper.get('form.chat-history__rename').trigger('submit')
    await nextTick()

    expect(wrapper.emitted('rename')).toEqual([[conversation.id, 'Renamed title']])
    expect(document.activeElement).toBe(wrapper.get('.chat-history__more-action').element)
  })

  it('按官方顺序呈现会话操作菜单，并只让真实能力触发变更', async () => {
    const appStore = useAppStore()
    const conversation: ChatConversation = {
      id: 'conversation-actions',
      userId: '7',
      title: 'Design review',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [conversation], activeId: conversation.id },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })

    const pin = wrapper.get('.chat-history__pin-action')
    const trigger = wrapper.get('.chat-history__more-action')
    expect(pin.attributes('aria-disabled')).toBe('true')
    expect(pin.get('[data-icon]').attributes('data-icon')).toBe('chatHistoryPinSmall')
    expect(trigger.attributes('aria-haspopup')).toBe('menu')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(trigger.get('[data-icon]').attributes('data-icon')).toBe('chatHistoryMore')
    expect(trigger.get('[data-icon]').attributes('data-size')).toBe('sm')

    const menu = await openConversationActions()
    const items = menu.findAll('[role="menuitem"]')
    expect(items.map((item) => item.attributes('data-action'))).toEqual([
      'share',
      'rename',
      'pin',
      'archive',
      'delete',
    ])
    expect(items.map((item) => item.get('[data-icon]').attributes('data-icon'))).toEqual([
      'chatHistoryShare',
      'chatHistoryRename',
      'chatHistoryPin',
      'chatHistoryArchive',
      'chatHistoryDelete',
    ])
    expect(items.map((item) => item.attributes('aria-disabled'))).toEqual([
      'true',
      undefined,
      'true',
      'true',
      undefined,
    ])
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(trigger.attributes('aria-controls')).toBe(menu.attributes('id'))
    expect(document.activeElement).toBe(items[0]!.element)

    await items[0]!.trigger('click')
    await nextTick()
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
    expect(wrapper.emitted('rename')).toBeUndefined()
    expect(wrapper.emitted('delete')).toBeUndefined()
    expect(appStore.toasts.at(-1)).toMatchObject({
      type: 'info',
      message: 'chat.actions.unavailable',
    })

    const reopened = await openConversationActions()
    await reopened.get('[data-action="delete"]').trigger('click')
    await nextTick()
    expect(wrapper.emitted('delete')).toEqual([[conversation.id]])
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
  })

  it('鼠标删除不锁住右侧操作焦点，键盘删除仍恢复到菜单触发器', async () => {
    const conversation: ChatConversation = {
      id: 'conversation-delete-focus',
      userId: '7',
      title: 'Delete focus behavior',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [conversation], activeId: 'another-conversation' },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    const trigger = wrapper.get('.chat-history__more-action')

    const pointerMenu = await openConversationActions()
    pointerMenu.get('[data-action="delete"]').element.dispatchEvent(new MouseEvent('click', {
      bubbles: true,
      detail: 1,
    }))
    await flushPromises()
    expect(wrapper.emitted('delete')).toEqual([[conversation.id]])
    expect(document.activeElement).not.toBe(trigger.element)

    const keyboardMenu = await openConversationActions()
    keyboardMenu.get('[data-action="delete"]').element.dispatchEvent(new MouseEvent('click', {
      bubbles: true,
      detail: 0,
    }))
    await flushPromises()
    expect(wrapper.emitted('delete')).toEqual([[conversation.id], [conversation.id]])
    expect(document.activeElement).toBe(trigger.element)
  })

  it('用方向键遍历菜单，并让第一次 Escape 只关闭菜单且回到触发器', async () => {
    const conversation: ChatConversation = {
      id: 'conversation-keyboard-menu',
      userId: '7',
      title: 'Keyboard menu',
      model: 'gpt-5',
      messages: [],
      createdAt: Date.now(),
      updatedAt: Date.now(),
    }
    wrapper = mount(ChatHistoryPanel, {
      attachTo: document.body,
      props: { conversations: [conversation], activeId: conversation.id, mobile: true },
      global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
    })
    const bubbled = vi.fn()
    window.addEventListener('keydown', bubbled)
    const trigger = wrapper.get('.chat-history__more-action')

    await trigger.trigger('keydown', { key: 'ArrowUp' })
    await nextTick()
    const menu = wrapper.get('[role="menu"]')
    const items = menu.findAll<HTMLButtonElement>('[role="menuitem"]')
    expect(document.activeElement).toBe(items.at(-1)!.element)

    await items.at(-1)!.trigger('keydown', { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[0]!.element)
    await items[0]!.trigger('keydown', { key: 'End' })
    expect(document.activeElement).toBe(items.at(-1)!.element)

    await items.at(-1)!.trigger('keydown', { key: 'Escape' })
    await nextTick()
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
    expect(document.activeElement).toBe(trigger.element)
    expect(bubbled).not.toHaveBeenCalled()
    window.removeEventListener('keydown', bubbled)

    await openConversationActions()
    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await nextTick()
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)

    const scrollableMenu = await openConversationActions()
    scrollableMenu.element.dispatchEvent(new Event('scroll'))
    await nextTick()
    expect(wrapper.find('[role="menu"]').exists()).toBe(true)
    window.dispatchEvent(new Event('scroll'))
    await nextTick()
    expect(wrapper.find('[role="menu"]').exists()).toBe(false)
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
			global: { stubs: { Icon: IconStub, RouterLink: RouterLinkStub } },
		})
		const bubbled = vi.fn()
		window.addEventListener('keydown', bubbled)

		const menu = await openConversationActions()
		await menu.get('[data-action="rename"]').trigger('click')
		await flushPromises()
		await wrapper.get('input[aria-label="chat.history.renameLabel"]').trigger('keydown', { key: 'Escape' })
		await nextTick()

		expect(wrapper.find('.chat-history__rename').exists()).toBe(false)
		expect(document.activeElement).toBe(wrapper.get('.chat-history__more-action').element)
		expect(bubbled).not.toHaveBeenCalled()
		window.removeEventListener('keydown', bubbled)
	})
})
