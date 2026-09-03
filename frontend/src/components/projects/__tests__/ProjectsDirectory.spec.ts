import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createI18n, type MessageContext } from 'vue-i18n'

import ProjectsDirectory from '../ProjectsDirectory.vue'
import type { Project } from '@/types/projects'

type TestLocale = 'en' | 'zh'
type ProjectFilter = 'all' | 'mine' | 'shared'

interface DirectoryProps {
  projects: Project[]
  loading?: boolean
  search: string
  filter: ProjectFilter
}

const rawMessages = {
  en: {
    projects: {
      title: 'Projects',
      searchPlaceholder: 'Search projects',
      new: 'New',
      filtersLabel: 'Project filters',
      filterAll: 'All',
      filterMine: 'Mine',
      filterShared: 'Shared',
      listLabel: 'Project list',
      nameColumn: 'Name',
      modifiedColumn: 'Updated',
      loading: 'Loading projects',
      openProject: 'Open {name}',
      moreActions: 'More actions for {name}',
      projectActions: 'Actions for {name}',
      edit: 'Edit',
      delete: 'Delete',
      sharedEmptyTitle: 'No projects shared with you',
      sharedEmptyDescription: 'Projects shared with you will appear here.',
      searchEmptyTitle: 'No matching projects',
      searchEmptyDescription: 'Try another search.',
      emptyTitle: 'Create your first project',
      emptyDescription: 'Projects keep related work together.',
      noProjects: 'No projects',
      newProject: 'New project',
      recentlyUpdated: 'Recently updated',
      today: 'Today',
      yesterday: 'Yesterday',
    },
  },
  zh: {
    projects: {
      title: '项目',
      searchPlaceholder: '搜索项目',
      new: '新建',
      filtersLabel: '项目筛选',
      filterAll: '全部',
      filterMine: '我的',
      filterShared: '共享',
      listLabel: '项目列表',
      nameColumn: '名称',
      modifiedColumn: '更新时间',
      loading: '正在加载项目',
      openProject: '打开 {name}',
      moreActions: '{name} 的更多操作',
      projectActions: '{name} 的项目操作',
      edit: '编辑',
      delete: '删除',
      sharedEmptyTitle: '还没有与你共享的项目',
      sharedEmptyDescription: '共享给你的项目会显示在这里。',
      searchEmptyTitle: '没有匹配的项目',
      searchEmptyDescription: '请尝试其他搜索词。',
      emptyTitle: '创建你的第一个项目',
      emptyDescription: '项目可集中整理相关工作。',
      noProjects: '暂无项目',
      newProject: '新建项目',
      recentlyUpdated: '最近更新',
      today: '今天',
      yesterday: '昨天',
    },
  },
}

type RawMessages = { [key: string]: string | RawMessages }
type CompiledMessages = {
  [key: string]: ((context: MessageContext) => string) | CompiledMessages
}

function compileMessages(source: RawMessages): CompiledMessages {
  return Object.fromEntries(Object.entries(source).map(([key, value]) => [
    key,
    typeof value === 'string'
      ? (context: MessageContext) => value.replace(
          /\{([^}]+)\}/g,
          (_match, name: string) => String(context.named(name)),
        )
      : compileMessages(value),
  ]))
}

const messages = {
  en: compileMessages(rawMessages.en),
  zh: compileMessages(rawMessages.zh),
}

const projects: Project[] = [
  {
    id: 'project-apollo',
    name: 'Apollo research',
    icon: '🚀',
    color: '#0d0d0d',
    memoryMode: 'default',
    instructions: '',
    conversationIds: [],
    conversations: [],
    files: [],
    createdAt: Date.now() - 86_400_000,
    updatedAt: Date.now(),
  },
  {
    id: 'project-roadmap',
    name: 'Product roadmap',
    icon: '📍',
    color: '#0d0d0d',
    memoryMode: 'project-only',
    instructions: '',
    conversationIds: [],
    conversations: [],
    files: [],
    createdAt: Date.now() - 172_800_000,
    updatedAt: Date.now() - 86_400_000,
  },
]

const wrappers: VueWrapper[] = []

function mountDirectory(
  props: Partial<DirectoryProps> = {},
  locale: TestLocale = 'en',
): VueWrapper {
  const i18n = createI18n({
    legacy: false,
    locale,
    fallbackLocale: 'en',
    messages,
  })
  const wrapper = mount(ProjectsDirectory, {
    props: {
      projects,
      loading: false,
      search: '',
      filter: 'all',
      ...props,
    },
    global: {
      plugins: [i18n],
      stubs: {
        Icon: true,
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
})

describe('ProjectsDirectory', () => {
  it('renders the supplied projects in the default list', () => {
    const wrapper = mountDirectory()

    expect(wrapper.findAll('.projects-directory__row')).toHaveLength(2)
    expect(wrapper.text()).toContain('Apollo research')
    expect(wrapper.text()).toContain('Product roadmap')
    expect(wrapper.find('.projects-directory__empty').exists()).toBe(false)
    expect(wrapper.find('.projects-directory__table').classes()).not.toContain('projects-directory__table--empty')
    expect(wrapper.find('.projects-directory__table-head').exists()).toBe(true)
  })

  it('emits search input changes', async () => {
    const wrapper = mountDirectory()

    await wrapper.get('input[type="search"]').setValue('roadmap')

    expect(wrapper.emitted('update:search')).toEqual([['roadmap']])
  })

  it('emits the selected filter', async () => {
    const wrapper = mountDirectory()

    await wrapper.findAll('.projects-directory__filters button')[2].trigger('click')

    expect(wrapper.emitted('update:filter')).toEqual([['shared']])
  })

  it('emits the selected project id when a row is opened', async () => {
    const wrapper = mountDirectory()

    await wrapper.findAll('.projects-directory__row-open')[0].trigger('click')

    expect(wrapper.emitted('open')).toEqual([['project-apollo']])
  })

  it('emits create from the primary new-project button', async () => {
    const wrapper = mountDirectory()

    await wrapper.get('.projects-directory__create').trigger('click')

    expect(wrapper.emitted('create')).toEqual([[]])
  })

  it('emits edit and delete from the project actions menu', async () => {
    const wrapper = mountDirectory()

    await wrapper.findAll('.projects-directory__more')[0].trigger('click')
    expect(wrapper.get('[role="menu"]').attributes('aria-label')).toBe('Actions for Apollo research')
    await wrapper.findAll('[role="menuitem"]')[0].trigger('click')
    expect(wrapper.emitted('edit')).toEqual([[projects[0]]])

    await wrapper.findAll('.projects-directory__more')[0].trigger('click')
    await wrapper.findAll('[role="menuitem"]')[1].trigger('click')
    expect(wrapper.emitted('delete')).toEqual([['project-apollo']])
  })

  it('shows an honest empty state for shared projects', () => {
    const wrapper = mountDirectory({ filter: 'shared' })

    expect(wrapper.findAll('.projects-directory__row')).toHaveLength(0)
    expect(wrapper.text()).toContain('No projects shared with you')
    expect(wrapper.text()).toContain('Projects shared with you will appear here.')
    expect(wrapper.find('.projects-directory__empty button').exists()).toBe(false)
  })

  it('matches the target minimal empty state when no projects exist', () => {
    const wrapper = mountDirectory({ projects: [] })

    expect(wrapper.get('.projects-directory__table').classes()).toContain('projects-directory__table--empty')
    expect(wrapper.get('.projects-directory__empty').classes()).toContain('projects-directory__empty--initial')
    expect(wrapper.get('.projects-directory__empty-icon').exists()).toBe(true)
    expect(wrapper.get('.projects-directory__empty strong').text()).toBe('No projects')
    expect(wrapper.find('.projects-directory__empty p').exists()).toBe(false)
    expect(wrapper.find('.projects-directory__empty button').exists()).toBe(false)
  })

  it('renders localized copy in English and Chinese', () => {
    const english = mountDirectory({}, 'en')
    const chinese = mountDirectory({}, 'zh')

    expect(english.get('h1').text()).toBe('Projects')
    expect(english.get('.projects-directory__create').text()).toBe('New')
    expect(chinese.get('h1').text()).toBe('项目')
    expect(chinese.get('.projects-directory__create').text()).toBe('新建')
  })
})
