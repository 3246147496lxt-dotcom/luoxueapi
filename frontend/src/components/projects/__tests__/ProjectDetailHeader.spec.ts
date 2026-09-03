import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'

import ProjectDetailHeader from '../ProjectDetailHeader.vue'
import type { Project } from '@/types/projects'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const labels: Record<string, string> = {
    'projects.projectMenu': 'Project menu',
    'projects.shareAction': 'Share',
    'projects.projectSettings': 'Project settings',
    'projects.delete': 'Delete project',
    'projects.addChat': 'Add chat',
    'projects.startChatPlaceholder': 'Start a chat in this project',
    'projects.newChatInProject': 'New chat in {name}',
    'projects.sendMessage': 'Send message',
    'projects.tabsLabel': 'Project content',
    'projects.chatsTab': 'Chats',
    'projects.sourcesTab': 'Sources',
    'projects.chatMode': 'Chats',
    'projects.workMode': 'Work',
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, value),
        labels[key] ?? key,
      ),
    }),
  }
})

const IconStub = {
  props: ['name', 'size'],
  template: '<span :data-icon="name" :data-size="size" />',
}

const AppModeSwitchStub = {
  props: ['activeMode', 'chatLabel', 'workLabel'],
  emits: ['change'],
  template: `
    <div data-testid="mode-switch" :data-active-mode="activeMode">
      <span>{{ chatLabel }}</span>
      <button type="button" data-testid="work-mode" @click="$emit('change', 'work')">{{ workLabel }}</button>
    </div>
  `,
}

const project: Project = {
  id: 'project-apollo',
  name: 'Apollo research',
  icon: '🚀',
  color: '#0d0d0d',
  memoryMode: 'default',
  instructions: '',
  conversationIds: [],
  conversations: [],
  files: [],
  createdAt: 1_788_192_000_000,
  updatedAt: 1_788_278_400_000,
}

type ProjectTab = 'chats' | 'sources'

interface HeaderProps {
  project: Project
  activeTab: ProjectTab
  prompt: string
}

let wrapper: VueWrapper | undefined

function mountHeader(props: Partial<HeaderProps> = {}): VueWrapper {
  wrapper = mount(ProjectDetailHeader, {
    attachTo: document.body,
    props: {
      project,
      activeTab: 'chats',
      prompt: '',
      ...props,
    },
    global: {
      stubs: {
        Icon: IconStub,
        AppModeSwitch: AppModeSwitchStub,
      },
    },
  })
  return wrapper
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('ProjectDetailHeader', () => {
  it('exposes the project composer, tabs, and actions with accessible names', () => {
    const target = mountHeader()

    expect(target.get('h1').text()).toBe('Apollo research')
    expect(target.get('.project-detail-header__mobile-title').text()).toBe('Apollo research')
    expect(target.get('.project-detail-header__prompt').attributes()).toMatchObject({
      'aria-label': 'Start a chat in this project',
      placeholder: 'New chat in Apollo research',
    })
    expect(target.get('.project-detail-header__add-chat').attributes('aria-label')).toBe('Add chat')
    expect(target.get('.project-detail-header__submit').attributes('aria-label')).toBe('Send message')
    expect(target.get('.project-detail-header__share').text()).toBe('Share')
    expect(target.get('[data-testid="mode-switch"]').text()).toContain('Chats')

    const tablist = target.get('[role="tablist"]')
    const tabs = target.findAll('[role="tab"]')
    expect(tablist.attributes('aria-label')).toBe('Project content')
    expect(tabs.map(tab => tab.attributes('aria-controls'))).toEqual([
      'project-chats-panel',
      'project-sources-panel',
    ])

    const menuTriggers = target.findAll('button[aria-label="Project menu"]')
    expect(menuTriggers).toHaveLength(2)
    expect(menuTriggers.every(trigger => trigger.attributes('aria-haspopup') === 'menu')).toBe(true)
    expect(menuTriggers[0].attributes('aria-controls')).toBe(menuTriggers[1].attributes('aria-controls'))
  })

  it('implements the active tab v-model contract', async () => {
    const target = mountHeader()
    const tabs = target.findAll('[role="tab"]')

    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(tabs[0].attributes('tabindex')).toBe('0')
    expect(tabs[1].attributes('aria-selected')).toBe('false')
    expect(tabs[1].attributes('tabindex')).toBe('-1')

    await tabs[1].trigger('click')
    expect(target.emitted('update:activeTab')).toEqual([['sources']])

    await target.setProps({ activeTab: 'sources' })
    expect(tabs[0].attributes('aria-selected')).toBe('false')
    expect(tabs[1].attributes('aria-selected')).toBe('true')
    expect(tabs[1].classes()).toContain('is-active')
  })

  it('emits prompt input and submits Enter without submitting Shift+Enter', async () => {
    const target = mountHeader()
    const prompt = target.get('.project-detail-header__prompt')

    await prompt.setValue('Summarize the launch plan')
    expect(target.emitted('update:prompt')).toEqual([['Summarize the launch plan']])

    await target.setProps({ prompt: 'Summarize the launch plan' })
    await prompt.trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(target.emitted('submit')).toBeUndefined()

    await prompt.trigger('keydown', { key: 'Enter' })
    expect(target.emitted('submit')).toEqual([['Summarize the launch plan']])
  })

  it('emits add-chat, share, and work actions', async () => {
    const target = mountHeader()

    await target.get('.project-detail-header__add-chat').trigger('click')
    await target.get('.project-detail-header__share').trigger('click')
    await target.get('[data-testid="work-mode"]').trigger('click')

    expect(target.emitted('add-chat')).toEqual([[]])
    expect(target.emitted('share')).toEqual([[]])
    expect(target.emitted('work')).toEqual([[]])
  })

  it('manages the accessible project menu and emits settings and delete', async () => {
    const target = mountHeader()
    const trigger = target.get('.project-detail-header__actions .project-detail-header__icon-button')

    expect(trigger.attributes('aria-expanded')).toBe('false')
    await trigger.trigger('click')
    await nextTick()

    const menu = target.get('[role="menu"]')
    let items = target.findAll('[role="menuitem"]')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(menu.attributes('id')).toBe(trigger.attributes('aria-controls'))
    expect(menu.attributes('aria-label')).toBe('Project menu')
    expect(items.map(item => item.text())).toEqual(['Project settings', 'Delete project'])
    expect(document.activeElement).toBe(items[0].element)

    await items[0].trigger('click')
    expect(target.emitted('settings')).toEqual([[]])
    expect(target.find('[role="menu"]').exists()).toBe(false)

    await trigger.trigger('click')
    await nextTick()
    items = target.findAll('[role="menuitem"]')
    await items[1].trigger('click')
    expect(target.emitted('delete')).toEqual([[]])

    await trigger.trigger('click')
    await nextTick()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await nextTick()
    expect(target.find('[role="menu"]').exists()).toBe(false)
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(trigger.element)
  })
})
