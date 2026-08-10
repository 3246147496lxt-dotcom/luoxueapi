import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import UserDashboardRecentConversations from '../UserDashboardRecentConversations.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      dashboard: {
        workspace: {
          recentConversations: 'Recent conversations',
          recentConversationsDescription: 'Continue your latest AI sessions',
          viewAll: 'View all',
          loadingConversations: 'Loading recent conversations',
          noConversations: 'No conversations yet',
          startConversation: 'Start a conversation',
          openConversation: 'Open conversation: {title}',
          messageCount: '{count} messages',
        },
      },
    },
  },
})

const conversations = [
  {
    id: 'older',
    userId: '1',
    title: 'Older session',
    model: 'gpt-old',
    messages: [],
    messageCount: 2,
    createdAt: 100,
    updatedAt: 100,
  },
  {
    id: 'latest',
    userId: '1',
    title: 'Latest session',
    model: 'gpt-latest',
    messages: [],
    messageCount: 5,
    createdAt: 200,
    updatedAt: 200,
  },
] as never

describe('UserDashboardRecentConversations', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sorts real chat conversations by recency and emits the selected id', async () => {
    vi.spyOn(Date, 'now').mockReturnValue(300)
    const wrapper = mount(UserDashboardRecentConversations, {
      props: { conversations, loading: false },
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    const buttons = wrapper.findAll('.dashboard-conversation')
    expect(buttons.map(button => button.text())).toEqual(expect.arrayContaining([
      expect.stringContaining('Latest session'),
      expect.stringContaining('Older session'),
    ]))
    expect(buttons[0].text()).toContain('Latest session')

    await buttons[0].trigger('click')
    expect(wrapper.emitted('select')).toEqual([['latest']])
  })

  it('shows an honest empty state instead of usage-log stand-ins', () => {
    const wrapper = mount(UserDashboardRecentConversations, {
      props: { conversations: [], loading: false },
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('dashboard.workspace.noConversations')
    expect(wrapper.findAll('.dashboard-conversation')).toHaveLength(0)
  })
})
