import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Project } from '@/types/projects'

const mocks = vi.hoisted(() => ({
  app: {} as Record<string, unknown>,
  auth: {} as Record<string, unknown>,
  chat: {} as Record<string, unknown>,
  library: {} as Record<string, unknown>,
  projects: {} as Record<string, unknown>,
  route: { params: {} as Record<string, string | undefined> },
  push: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh-CN'),
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key}:${Object.values(params).join(',')}` : key
      ),
    }),
  }
})

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return {
    ...actual,
    useRoute: () => mocks.route,
    useRouter: () => ({ push: mocks.push }),
  }
})

vi.mock('@/components/layout/workspaceResponsive', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useWorkspaceResponsiveState: () => ({
      mobileDrawer: ref(false),
      narrowSidebar: ref(false),
    }),
  }
})

vi.mock('@/stores/app', () => ({ useAppStore: () => mocks.app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => mocks.auth }))
vi.mock('@/stores/chat', () => ({ useChatStore: () => mocks.chat }))
vi.mock('@/stores/library', () => ({ useLibraryStore: () => mocks.library }))
vi.mock('@/stores/projects', () => ({ useProjectsStore: () => mocks.projects }))

import ProjectsView from '../ProjectsView.vue'

const project: Project = {
  id: 'project-1',
  name: '产品发布',
  icon: '📁',
  color: '#7c3aed',
  memoryMode: 'default',
  instructions: '保持简洁',
  conversationIds: ['conversation-1'],
  conversations: [{
    id: 'conversation-1',
    title: '发布计划',
    updatedAt: '2026-09-01T08:00:00.000Z',
  }],
  files: [],
  createdAt: 1_756_000_000_000,
  updatedAt: 1_756_100_000_000,
}

let wrapper: VueWrapper | undefined

function resetObject(target: Record<string, unknown>, values: Record<string, unknown>): void {
  for (const key of Object.keys(target)) delete target[key]
  Object.assign(target, values)
}

const projectDirectoryStub = {
  name: 'ProjectsDirectory',
  props: ['projects', 'loading', 'search', 'filter'],
  emits: ['update:search', 'update:filter', 'create', 'open', 'edit', 'delete'],
  template: `
    <section data-test="directory">
      <button data-test="create" @click="$emit('create')" />
      <button data-test="open" @click="$emit('open', projects[0]?.id)" />
      <button data-test="edit" @click="$emit('edit', projects[0])" />
      <button data-test="delete" @click="$emit('delete', projects[0]?.id)" />
    </section>
  `,
}

const projectHeaderStub = {
  name: 'ProjectDetailHeader',
  props: ['project', 'activeTab', 'prompt'],
  emits: [
    'update:activeTab',
    'update:prompt',
    'submit',
    'add-chat',
    'settings',
    'share',
    'delete',
    'back',
    'work',
  ],
  template: `
    <header data-test="detail-header">
      <button data-test="sources-tab" @click="$emit('update:activeTab', 'sources')" />
      <button data-test="submit-prompt" @click="$emit('submit', '整理发布计划')" />
      <button data-test="add-chat" @click="$emit('add-chat')" />
      <button data-test="settings" @click="$emit('settings')" />
      <button data-test="share" @click="$emit('share')" />
      <button data-test="work" @click="$emit('work')" />
    </header>
  `,
}

const projectChatsStub = {
  name: 'ProjectChatsPanel',
  props: ['conversations', 'availableConversations', 'pickerOpen'],
  emits: ['open', 'remove', 'add', 'update:pickerOpen'],
  template: `
    <section data-test="chats">
      <button data-test="open-chat" @click="$emit('open', 'conversation-1')" />
      <button data-test="remove-chat" @click="$emit('remove', 'conversation-1')" />
      <button data-test="add-existing" @click="$emit('add', 'conversation-2')" />
    </section>
  `,
}

const projectSourcesStub = {
  name: 'ProjectSourcesPanel',
  props: ['files', 'uploading'],
  emits: ['add-files', 'remove'],
  template: '<section data-test="sources" />',
}

function mountProjects(): VueWrapper {
  return mount(ProjectsView, {
    attachTo: document.body,
    global: {
      stubs: {
        Transition: false,
        AppLayout: { template: '<div><slot /></div>' },
        BaseDialog: {
          name: 'BaseDialog',
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show" data-test="dialog"><slot /><slot name="footer" /></div>',
        },
        ChatHistoryPanel: { template: '<aside data-test="history" />' },
        WorkspaceSidebarOverlayLayer: { template: '<div><slot /></div>' },
        Icon: { props: ['name'], template: '<i :data-icon="name" />' },
        ProjectsDirectory: projectDirectoryStub,
        ProjectDetailHeader: projectHeaderStub,
        ProjectChatsPanel: projectChatsStub,
        ProjectSourcesPanel: projectSourcesStub,
      },
    },
  })
}

beforeEach(() => {
  vi.resetAllMocks()
  mocks.route.params = {}
  mocks.push.mockResolvedValue(undefined)
  resetObject(mocks.app, {
    workspaceNarrowSidebarOpen: false,
    setWorkspaceNarrowSidebarOpen: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn(),
  })
  resetObject(mocks.auth, { user: { id: 'user-1' } })
  resetObject(mocks.chat, {
    activeConversationId: null,
    conversations: [{
      id: 'conversation-2',
      title: '市场物料',
      model: 'gpt-5',
      updatedAt: 1_756_200_000_000,
    }],
    searchResults: [],
    searchingHistory: false,
    conversationsHaveMore: false,
    loadingConversationPage: false,
    hydrate: vi.fn().mockResolvedValue(undefined),
    syncHistory: vi.fn().mockResolvedValue(undefined),
    loadConversationPage: vi.fn().mockResolvedValue(undefined),
    searchHistory: vi.fn().mockResolvedValue(undefined),
    selectConversation: vi.fn(),
    renameConversation: vi.fn(),
    deleteConversation: vi.fn(),
    clearConversations: vi.fn(),
  })
  resetObject(mocks.library, {
    uploadFiles: vi.fn().mockResolvedValue([]),
  })
  resetObject(mocks.projects, {
    projects: [project],
    loading: false,
    initialized: true,
    error: '',
    load: vi.fn().mockResolvedValue(undefined),
    get: vi.fn().mockResolvedValue(project),
    create: vi.fn().mockResolvedValue(project),
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(true),
    select: vi.fn(),
    addConversation: vi.fn().mockResolvedValue(true),
    removeConversation: vi.fn().mockResolvedValue(true),
    addFiles: vi.fn().mockResolvedValue([]),
    removeFile: vi.fn().mockResolvedValue(true),
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
  document.body.innerHTML = ''
})

describe('ProjectsView', () => {
  it('loads the directory and hydrates the shared chat history', async () => {
    wrapper = mountProjects()
    await flushPromises()

    expect(wrapper.findAll('[data-test="directory"]')).toHaveLength(1)
    expect(mocks.projects.load).toHaveBeenCalledOnce()
    expect(mocks.chat.hydrate).toHaveBeenCalledWith('user-1')
    expect(mocks.chat.syncHistory).toHaveBeenCalledOnce()
    expect(mocks.chat.loadConversationPage).toHaveBeenCalledWith(true)
  })

  it('opens a project and edits it through the settings dialog', async () => {
    wrapper = mountProjects()
    await flushPromises()

    await wrapper.get('[data-test="open"]').trigger('click')
    expect(mocks.push).toHaveBeenCalledWith('/projects/project-1')

    await wrapper.get('[data-test="edit"]').trigger('click')
    const dialog = wrapper.get('[data-test="dialog"]')
    await dialog.get('input[required]').setValue('发布工作台')
    await dialog.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.projects.update).toHaveBeenCalledWith('project-1', expect.objectContaining({
      name: '发布工作台',
      instructions: '保持简洁',
    }))
  })

  it('uses the compact target create dialog and keeps advanced fields out of it', async () => {
    wrapper = mountProjects()
    await flushPromises()

    await wrapper.get('[data-test="create"]').trigger('click')
    const dialog = wrapper.get('[data-test="dialog"]')
    expect(dialog.text()).toContain('projects.createProject')
    expect(dialog.text()).toContain('projects.createInfo')
    expect(dialog.find('input[placeholder="projects.projectNamePlaceholder"]').exists()).toBe(true)
    expect(dialog.find('textarea').exists()).toBe(false)
    expect(dialog.find('select').exists()).toBe(false)

    const name = dialog.get('input[required]')
    await name.setValue('新项目')
    await dialog.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.projects.create).toHaveBeenCalledWith(expect.objectContaining({
      name: '新项目',
      icon: '📁',
      color: '#0d0d0d',
      memoryMode: 'default',
    }))
    expect(mocks.push).toHaveBeenCalledWith('/projects/project-1')
  })

  it('switches tabs and carries a project prompt into a new project chat', async () => {
    mocks.route.params = { projectId: 'project-1' }
    wrapper = mountProjects()
    await flushPromises()

    expect(wrapper.findAll('[data-test="chats"]')).toHaveLength(1)
    await wrapper.get('[data-test="sources-tab"]').trigger('click')
    expect(wrapper.findAll('[data-test="sources"]')).toHaveLength(1)

    await wrapper.get('[data-test="submit-prompt"]').trigger('click')
    expect(mocks.chat.selectConversation).toHaveBeenCalledWith(null)
    expect(mocks.push).toHaveBeenCalledWith({
      path: '/chat',
      query: {
        conversation: 'new',
        project: 'project-1',
        prompt: '整理发布计划',
      },
    })
  })

  it('wires existing chat actions and the Work mode switch', async () => {
    mocks.route.params = { projectId: 'project-1' }
    wrapper = mountProjects()
    await flushPromises()

    await wrapper.get('[data-test="add-existing"]').trigger('click')
    expect(mocks.projects.addConversation).toHaveBeenCalledWith(
      'project-1',
      'conversation-2',
      expect.objectContaining({ id: 'conversation-2', title: '市场物料' }),
    )

    await wrapper.get('[data-test="remove-chat"]').trigger('click')
    expect(mocks.projects.removeConversation).toHaveBeenCalledWith('project-1', 'conversation-1')

    await wrapper.get('[data-test="open-chat"]').trigger('click')
    expect(mocks.chat.selectConversation).toHaveBeenCalledWith('conversation-1')
    expect(mocks.push).toHaveBeenCalledWith({
      path: '/chat',
      query: { conversation: 'conversation-1' },
    })

    await wrapper.get('[data-test="work"]').trigger('click')
    expect(mocks.push).toHaveBeenCalledWith('/dashboard')
  })

  it('copies the current project link when native sharing is unavailable', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'share', { configurable: true, value: undefined })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
    window.history.replaceState({}, '', '/projects/project-1')
    mocks.route.params = { projectId: 'project-1' }
    wrapper = mountProjects()
    await flushPromises()

    await wrapper.get('[data-test="share"]').trigger('click')
    await flushPromises()

    expect(writeText).toHaveBeenCalledWith(expect.stringContaining('/projects/project-1'))
    expect(mocks.app.showSuccess).toHaveBeenCalledWith('projects.shareCopied')
  })

  it('uploads a selected source and links the returned library file', async () => {
    mocks.route.params = { projectId: 'project-1' }
    const uploaded = {
      id: 'file-1',
      name: 'brief.pdf',
      size: 2048,
      mimeType: 'application/pdf',
      createdAt: '2026-09-01T08:00:00.000Z',
    }
    vi.mocked(mocks.library.uploadFiles as ReturnType<typeof vi.fn>).mockResolvedValue([uploaded])
    wrapper = mountProjects()
    await flushPromises()
    await wrapper.get('[data-test="sources-tab"]').trigger('click')

    const input = { files: [new File(['brief'], 'brief.pdf')], value: 'selected' }
    wrapper.getComponent(projectSourcesStub).vm.$emit('add-files', { target: input })
    await flushPromises()

    expect(input.value).toBe('')
    expect(mocks.library.uploadFiles).toHaveBeenCalledOnce()
    expect(mocks.projects.addFiles).toHaveBeenCalledWith('project-1', [uploaded])
  })
})
