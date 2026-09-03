import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { routerKey } from 'vue-router'
import type { LibraryDownloadTicket, LibraryFile } from '@/types/library'

const directory = dirname(fileURLToPath(import.meta.url))
const libraryViewSource = readFileSync(resolve(directory, '../LibraryView.vue'), 'utf8')
const downloadHandle = 'ABCdef0123456789_-abcd'
const downloadPath = `/api/v1/library/download/${downloadHandle}` as const

const storeMocks = vi.hoisted(() => ({
  app: {} as Record<string, unknown>,
  auth: {} as Record<string, unknown>,
  chat: {} as Record<string, unknown>,
  library: {} as Record<string, unknown>,
}))

const apiMocks = vi.hoisted(() => ({
  ticket: vi.fn(),
  thumbnail: vi.fn(),
}))

const fileUiMocks = vi.hoisted(() => ({
  prepare: vi.fn(),
  start: vi.fn(),
  cancel: vi.fn(),
}))

const timeUiMocks = vi.hoisted(() => ({
  timestamp: vi.fn(),
  label: vi.fn(),
  tooltip: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh-CN'),
      t: (key: string, params?: Record<string, unknown>) => (
        params ? `${key} ${Object.values(params).join(' ')}` : key
      ),
    }),
  }
})

vi.mock('@/api/library', () => ({
  createLibraryDownloadTicket: apiMocks.ticket,
  getLibraryThumbnail: apiMocks.thumbnail,
}))

vi.mock('@/api/client', () => ({
  buildApiUrl: (path: string) => `https://api.example.test${path}`,
}))

vi.mock('@/components/library/libraryFileUi', () => ({
  formatLibraryBytes: (value: number) => `${value} B`,
  formatLibraryDate: (value: string) => value,
  libraryFileExtension: (value: LibraryFile) => value.name.split('.').pop()?.toUpperCase() ?? 'FILE',
  prepareLibraryBrowserDownload: fileUiMocks.prepare,
}))

vi.mock('@/utils/libraryFileTime', () => ({
  libraryFileModifiedTimestamp: timeUiMocks.timestamp,
  formatLibraryFileTime: timeUiMocks.label,
  formatLibraryFileTimeTooltip: timeUiMocks.tooltip,
}))

vi.mock('@/components/layout/workspaceResponsive', async () => {
  const { ref } = await vi.importActual<typeof import('vue')>('vue')
  return {
    useWorkspaceResponsiveState: () => ({
      mobileDrawer: ref(false),
      narrowSidebar: ref(false),
    }),
  }
})

vi.mock('@/stores/app', () => ({ useAppStore: () => storeMocks.app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => storeMocks.auth }))
vi.mock('@/stores/chat', () => ({ useChatStore: () => storeMocks.chat }))
vi.mock('@/stores/library', () => ({ useLibraryStore: () => storeMocks.library }))

import LibraryView from '../LibraryView.vue'

let wrapper: VueWrapper | undefined

const libraryFile: LibraryFile = {
  id: 'library-file-1',
  name: 'product-brief.docx',
  mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  size: 4096,
  source: 'uploaded',
  type: 'document',
  category: 'file',
  status: 'ready',
  createdAt: '2026-08-14T01:00:00.000Z',
  updatedAt: '2026-08-14T02:00:00.000Z',
}

function resetObject(target: Record<string, unknown>, values: Record<string, unknown>): void {
  for (const key of Object.keys(target)) delete target[key]
  Object.assign(target, values)
}

function mountLibrary() {
  return mount(LibraryView, {
    attachTo: document.body,
    global: {
      provide: {
        [routerKey as symbol]: { push: vi.fn().mockResolvedValue(undefined) },
      },
      stubs: {
        Transition: false,
        AppLayout: {
          props: ['variant', 'shellMode'],
          template: '<div data-test="app-layout"><slot /></div>',
        },
        ChatHistoryPanel: {
          props: ['activeSection'],
          template: '<aside data-test="history" :data-active-section="activeSection" />',
        },
        LibraryFilePreviewDialog: {
          props: ['show', 'file'],
          emits: ['close', 'download', 'remove'],
          template: '<div data-test="preview" :data-show="String(show)">{{ file?.id }}<button v-if="file" data-test="preview-close" @click="$emit(\'close\')" /><button v-if="file" data-test="preview-download" @click="$emit(\'download\', file)" /><button v-if="file" data-test="preview-remove" @click="$emit(\'remove\', file)" /></div>',
        },
        Icon: { props: ['name'], template: '<span :data-icon="name" />' },
      },
    },
  })
}

beforeEach(() => {
  vi.resetAllMocks()
  timeUiMocks.timestamp.mockImplementation((file: LibraryFile) => (
    file.updatedAt.trim() || file.createdAt.trim()
  ))
  timeUiMocks.label.mockImplementation((timestamp: string) => `label:${timestamp}`)
  timeUiMocks.tooltip.mockImplementation((timestamp: string) => `tooltip:${timestamp}`)
  fileUiMocks.prepare.mockReturnValue({
    start: fileUiMocks.start,
    cancel: fileUiMocks.cancel,
  })
  apiMocks.thumbnail.mockResolvedValue(new Blob(['thumbnail'], { type: 'image/jpeg' }))
  resetObject(storeMocks.app, {
    sidebarCollapsed: false,
    workspaceNarrowSidebarOpen: false,
    setSidebarCollapsed: vi.fn(),
    setWorkspaceNarrowSidebarOpen: vi.fn(),
    showSuccess: vi.fn(),
    showError: vi.fn(),
  })
  resetObject(storeMocks.auth, { user: null })
  resetObject(storeMocks.chat, {
    activeConversationId: null,
    conversations: [],
    searchResults: [],
    searchingHistory: false,
    conversationsHaveMore: false,
    loadingConversationPage: false,
    hydrate: vi.fn().mockResolvedValue(undefined),
    syncHistory: vi.fn().mockResolvedValue(undefined),
    loadConversationPage: vi.fn().mockResolvedValue(undefined),
    searchHistory: vi.fn().mockResolvedValue(undefined),
    invalidateHistorySearch: vi.fn(),
    selectConversation: vi.fn(),
    renameConversation: vi.fn(),
    deleteConversation: vi.fn(),
    clearConversations: vi.fn(),
  })
  resetObject(storeMocks.library, {
    files: [],
    total: 0,
    loading: false,
    initialized: true,
    error: '',
    searchKeyword: '',
    category: 'all',
    source: 'all',
    fileType: 'all',
    sort: 'updated_desc',
    currentView: 'list',
    selectedFileIds: new Set<string>(),
    selectedFiles: [],
    hasMore: false,
    uploads: [],
    storageUsage: null,
    load: vi.fn().mockResolvedValue(undefined),
    loadMore: vi.fn().mockResolvedValue(undefined),
    fetchStorage: vi.fn().mockResolvedValue(undefined),
    setSearchKeyword: vi.fn(),
    setCategory: vi.fn(),
    setFilters: vi.fn(),
    setSort: vi.fn(),
    setView: vi.fn(),
    toggleSelection: vi.fn(),
    clearSelection: vi.fn(),
    uploadFiles: vi.fn().mockResolvedValue([]),
    retryUpload: vi.fn().mockResolvedValue(null),
    dismissUpload: vi.fn(),
    dismissSettledUploads: vi.fn(),
    removeFiles: vi.fn().mockResolvedValue({ deleted: [], failed: [] }),
  })
  apiMocks.ticket.mockResolvedValue({
    downloadPath,
    filename: 'download.bin',
    fileCount: 1,
    totalBytes: 4096,
    expiresIn: 60,
  } satisfies LibraryDownloadTicket)
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('LibraryView', () => {
  it('renders the library workspace and loads files plus storage on mount', async () => {
    wrapper = mountLibrary()
    await flushPromises()

    expect(wrapper.get('[data-test="history"]').attributes('data-active-section')).toBe('library')
    expect(wrapper.get('.library-content').attributes('aria-label')).toBe('library.title')
    expect(wrapper.get('h1').text()).toBe('library.title')
    expect(wrapper.findAll('[role="tab"]').map((tab) => tab.text())).toEqual([
      'library.categories.all',
      'library.categories.image',
      'library.categories.file',
    ])
    expect(wrapper.findAll('[role="tab"]')[0]?.attributes('aria-selected')).toBe('true')
    expect(wrapper.find('.library-search [data-icon="librarySearch"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="library.toolbar.filter"] [data-icon="libraryFilter"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="library.views.grid"] [data-icon="libraryGrid"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="library.views.list"] [data-icon="libraryList"]').exists()).toBe(true)
    const filterButton = wrapper.get('button[aria-label="library.toolbar.filter"]')
    const gridButton = wrapper.get('button[aria-label="library.views.grid"]')
    const listButton = wrapper.get('button[aria-label="library.views.list"]')
    expect(filterButton.attributes('title')).toBeUndefined()
    expect(gridButton.attributes('title')).toBeUndefined()
    expect(listButton.attributes('title')).toBeUndefined()
    expect(gridButton.attributes('aria-describedby')).toBe('library-grid-view-tooltip')
    expect(listButton.attributes('aria-describedby')).toBe('library-list-view-tooltip')
    expect(wrapper.get('#library-grid-view-tooltip').text()).toBe('library.views.gridTooltip')
    expect(wrapper.get('#library-list-view-tooltip').text()).toBe('library.views.listTooltip')
    expect(listButton.attributes('aria-pressed')).toBe('true')
    expect(listButton.classes()).toContain('library-icon-button--active')
    expect(storeMocks.library.load).toHaveBeenCalledOnce()
    expect(storeMocks.library.fetchStorage).toHaveBeenCalledOnce()
  })

  it('uses the official filter open state instead of a native title tooltip', async () => {
    wrapper = mountLibrary()
    await flushPromises()

    const filterButton = wrapper.get('button[aria-label="library.toolbar.filter"]')
    expect(filterButton.attributes('aria-expanded')).toBe('false')
    expect(filterButton.classes()).not.toContain('library-icon-button--active')

    await filterButton.trigger('click')
    expect(filterButton.attributes('aria-expanded')).toBe('true')
    expect(filterButton.classes()).toContain('library-icon-button--active')

    await filterButton.trigger('click')
    expect(filterButton.attributes('aria-expanded')).toBe('false')
    expect(filterButton.classes()).not.toContain('library-icon-button--active')
  })

  it('keeps the official toolbar hover, selected, and tooltip timing contract', () => {
    expect(libraryViewSource).toContain('--library-control-hover: rgb(0 0 0 / 0.05);')
    expect(libraryViewSource).toMatch(
      /\.library-icon-button--active\s*\{[^}]*color:\s*var\(--workspace-text\);[^}]*background:\s*var\(--library-control-active\);/s,
    )
    expect(libraryViewSource).toMatch(
      /@media \(hover: hover\) and \(pointer: fine\) \{[\s\S]*?\.library-icon-button:hover \{ background: var\(--library-control-hover\); \}/,
    )
    expect(libraryViewSource).toContain('opacity 150ms cubic-bezier(0.4, 0, 0.2, 1),')
    expect(libraryViewSource).toContain('visibility 0s linear 150ms;')
    expect(libraryViewSource).toContain('transition-delay: 250ms, 250ms;')
    expect(libraryViewSource).toContain('.library-icon-button:focus-visible .library-icon-tooltip')
  })

  it('changes category and view, opens upload selection, and previews a file', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    wrapper = mountLibrary()
    await flushPromises()

    await wrapper.findAll('[role="tab"]')[1]!.trigger('click')
    expect(storeMocks.library.setCategory).toHaveBeenCalledWith('image')
    expect(storeMocks.library.load).toHaveBeenCalledTimes(2)

    await wrapper.get('button[aria-label="library.views.grid"]').trigger('click')
    expect(storeMocks.library.setView).toHaveBeenCalledWith('grid')

    const input = wrapper.get('input[type="file"]')
    const inputClick = vi.spyOn(input.element as HTMLInputElement, 'click')
    await wrapper.get('.library-new__trigger').trigger('click')
    await wrapper.get('.library-new__menu [role="menuitem"]').trigger('click')
    expect(inputClick).toHaveBeenCalledOnce()

    await wrapper.get('.library-list__open').trigger('click')
    expect(wrapper.get('[data-test="preview"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="preview"]').text()).toBe('library-file-1')
    expect(wrapper.get('.library-main').attributes('style')).toContain('display: none')
    expect(storeMocks.app.setSidebarCollapsed).toHaveBeenCalledWith(true)

    await wrapper.get('[data-test="preview-close"]').trigger('click')
    expect(wrapper.get('[data-test="preview"]').attributes('data-show')).toBe('false')
    expect(wrapper.get('.library-main').attributes('style') ?? '').not.toContain('display: none')
    expect(storeMocks.app.setSidebarCollapsed).toHaveBeenLastCalledWith(false)
  })

  it('uses a valid ARIA table hierarchy and native controls for row actions', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    wrapper = mountLibrary()
    await flushPromises()

    const table = wrapper.get('[role="table"]')
    expect(table.findAll('[role="rowgroup"]')).toHaveLength(2)
    const row = table.get('.library-list__row')
    expect(row.attributes('role')).toBe('row')
    expect(row.attributes('tabindex')).toBeUndefined()
    expect(row.findAll('[role="cell"]')).toHaveLength(4)

    const preview = row.get<HTMLButtonElement>('.library-list__open')
    expect(preview.element.tagName).toBe('BUTTON')
    expect(preview.attributes('aria-label')).toContain('product-brief.docx')
    const checkbox = row.get<HTMLInputElement>('input[type="checkbox"]')
    checkbox.element.focus()
    expect(document.activeElement).toBe(checkbox.element)
    await checkbox.setValue(true)
    expect(storeMocks.library.toggleSelection).toHaveBeenCalledWith('library-file-1', true)
    await preview.trigger('click')
    expect(wrapper.get('[data-test="preview"]').attributes('data-show')).toBe('true')
  })

  it('renders modified time from the record timestamp with a complete tooltip', async () => {
    const timestamp = '2026-08-14T13:38:00+08:00'
    storeMocks.library.files = [{
      ...libraryFile,
      name: 'CleanShot 2025-01-01.png',
      createdAt: '2025-01-01T00:00:00+08:00',
      updatedAt: timestamp,
    }]
    storeMocks.library.total = 1
    wrapper = mountLibrary()
    await flushPromises()

    const modifiedTime = wrapper.get('.library-list__row time')
    expect(modifiedTime.attributes('datetime')).toBe(timestamp)
    expect(modifiedTime.attributes('title')).toBe(`tooltip:${timestamp}`)
    expect(modifiedTime.text()).toBe(`label:${timestamp}`)
    expect(timeUiMocks.timestamp).toHaveBeenCalledWith(expect.objectContaining({
      name: 'CleanShot 2025-01-01.png',
      createdAt: '2025-01-01T00:00:00+08:00',
      updatedAt: timestamp,
    }))
  })

  it('leaves the viewer before opening the existing delete confirmation', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    wrapper = mountLibrary()
    await flushPromises()

    await wrapper.get('.library-list__open').trigger('click')
    await wrapper.get('[data-test="preview-remove"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="preview"]').attributes('data-show')).toBe('false')
    expect(document.body.textContent).toContain('library.deleteConfirm.singleTitle')
    expect(storeMocks.app.setSidebarCollapsed).toHaveBeenLastCalledWith(false)
  })

  it('keeps selection visible on keyboard focus and aligns touch navigation to 767px', () => {
    expect(libraryViewSource).toMatch(
      /\.library-select-control\s*\{[^}]*width:\s*44px;[^}]*height:\s*44px;/s,
    )
    expect(libraryViewSource).toContain('.library-select-control:focus-within,')
    expect(libraryViewSource).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\) \{[\s\S]*?\.library-mobile-actions \{ display: flex;/,
    )
    expect(libraryViewSource).not.toMatch(
      /@media \(max-width: 700px\) \{\s*\.library-mobile-actions \{ display: flex;/,
    )
  })

  it('shows unlimited storage explicitly and omits a misleading progress bar', async () => {
    storeMocks.library.storageUsage = {
      usedBytes: 8192,
      limitBytes: 0,
      remainingBytes: 0,
      overLimit: false,
      unlimited: true,
    }
    wrapper = mountLibrary()
    await flushPromises()

    expect(wrapper.get('.library-storage').text()).toContain('8192 B · common.unlimited')
    expect(wrapper.find('.library-storage__track').exists()).toBe(false)
  })

  it('mounts the fixed upload tray and safely dismisses settled activity', async () => {
    storeMocks.library.uploads = [{
      key: 'finished-upload',
      file: new File(['image'], 'finished.png', { type: 'image/png' }),
      progress: 100,
      state: 'ready',
    }]
    wrapper = mountLibrary()
    await flushPromises()

    expect(wrapper.get('.library-upload-tray').attributes('aria-busy')).toBe('false')
    await wrapper.get('[aria-label="library.uploadTray.dismiss"]').trigger('click')
    expect(storeMocks.library.dismissSettledUploads).toHaveBeenCalledOnce()
  })

  it('starts a browser-native stream without overriding the server-provided filename', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    const response: LibraryDownloadTicket = {
      downloadPath,
      filename: '服务器文件名.docx',
      fileCount: 1,
      totalBytes: 4096,
      expiresIn: 60,
    }
    apiMocks.ticket.mockResolvedValueOnce(response)
    wrapper = mountLibrary()
    await flushPromises()

    await wrapper.get('.library-list__open').trigger('click')
    await wrapper.get('[data-test="preview-download"]').trigger('click')
    await flushPromises()

    expect(apiMocks.ticket).toHaveBeenCalledWith(['library-file-1'])
    expect(fileUiMocks.prepare).toHaveBeenCalledWith(
      'https://api.example.test/api/v1/library/download/AAAAAAAAAAAAAAAAAAAAAA',
    )
    expect(fileUiMocks.prepare.mock.invocationCallOrder[0]).toBeLessThan(
      apiMocks.ticket.mock.invocationCallOrder[0]!,
    )
    expect(fileUiMocks.start).toHaveBeenCalledWith(
      `https://api.example.test${downloadPath}`,
    )
  })

  it('does not issue a ticket when the browser blocks synchronous download preparation', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    fileUiMocks.prepare.mockImplementationOnce(() => {
      throw new Error('popup blocked')
    })
    wrapper = mountLibrary()
    await flushPromises()

    await wrapper.get('.library-list__open').trigger('click')
    await wrapper.get('[data-test="preview-download"]').trigger('click')
    await flushPromises()

    expect(apiMocks.ticket).not.toHaveBeenCalled()
    expect(fileUiMocks.start).not.toHaveBeenCalled()
    expect(storeMocks.app.showError).toHaveBeenCalledWith('library.errors.downloadFailed')
  })

  it('closes the reserved browser target when ticket creation fails', async () => {
    storeMocks.library.files = [libraryFile]
    storeMocks.library.total = 1
    apiMocks.ticket.mockRejectedValueOnce(new Error('unavailable'))
    wrapper = mountLibrary()
    await flushPromises()

    await wrapper.get('.library-list__open').trigger('click')
    await wrapper.get('[data-test="preview-download"]').trigger('click')
    await flushPromises()

    expect(fileUiMocks.prepare).toHaveBeenCalledOnce()
    expect(fileUiMocks.cancel).toHaveBeenCalledOnce()
    expect(fileUiMocks.start).not.toHaveBeenCalled()
    expect(storeMocks.app.showError).toHaveBeenCalledWith('library.errors.downloadFailed')
  })
})
