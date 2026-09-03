import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { LibraryUploadItem, LibraryUploadState } from '@/types/library'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (!params) return key
        const values = Object.entries(params).map(([name, value]) => `${name}=${value}`).join(' ')
        return `${key} ${values}`
      },
    }),
  }
})

import LibraryUploadTray from '../LibraryUploadTray.vue'

const directory = dirname(fileURLToPath(import.meta.url))
const traySource = readFileSync(resolve(directory, '../LibraryUploadTray.vue'), 'utf8')
let wrapper: VueWrapper | undefined

function upload(
  key: string,
  state: LibraryUploadState,
  overrides: Partial<LibraryUploadItem> = {},
): LibraryUploadItem {
  return {
    key,
    file: new File([key], `${key}.pdf`, { type: 'application/pdf' }),
    progress: state === 'ready' ? 100 : 0,
    state,
    ...overrides,
  }
}

function mountTray(uploads: LibraryUploadItem[]) {
  return mount(LibraryUploadTray, {
    attachTo: document.body,
    props: { uploads },
    global: {
      stubs: {
        Icon: {
          props: ['name'],
          template: '<span :data-icon="name" />',
        },
      },
    },
  })
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('LibraryUploadTray', () => {
  it('stays out of the accessibility tree when there are no upload tasks', () => {
    wrapper = mountTray([])

    expect(wrapper.find('.library-upload-tray').exists()).toBe(false)
  })

  it('renders queued and active progress without announcing every percentage update', () => {
    wrapper = mountTray([
      upload('active', 'uploading', { progress: 42 }),
      upload('waiting', 'queued'),
    ])

    const tray = wrapper.get('.library-upload-tray')
    expect(tray.attributes('role')).toBe('region')
    expect(tray.attributes('aria-busy')).toBe('true')
    expect(tray.attributes('aria-label')).toBe('library.uploadTray.title count=2')
    expect(wrapper.get('.library-upload-tray__summary strong').text()).toBe('0/2')
    expect(wrapper.findAll('[role="listitem"]')).toHaveLength(2)

    const progressbar = wrapper.get('[role="progressbar"]')
    expect(progressbar.attributes('aria-valuenow')).toBe('42')
    expect(progressbar.attributes('aria-valuemin')).toBe('0')
    expect(progressbar.attributes('aria-valuemax')).toBe('100')
    expect(progressbar.attributes('aria-valuetext')).toBe(
      'library.uploadTray.status.uploading progress=42',
    )
    expect(wrapper.get('[aria-live="polite"]').text()).toBe(
      'library.uploadTray.summary completed=0 total=2',
    )
    expect(wrapper.find('[aria-label="library.uploadTray.dismiss"]').exists()).toBe(false)
  })

  it('collapses accessibly and reopens when new work is registered', async () => {
    const first = upload('first', 'uploading', { progress: 20 })
    wrapper = mountTray([first])
    const toggle = wrapper.get('[aria-controls="library-upload-tray-body"]')

    expect(toggle.attributes('aria-expanded')).toBe('true')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-expanded')).toBe('false')
    expect(wrapper.get('#library-upload-tray-body').isVisible()).toBe(false)

    await wrapper.setProps({ uploads: [first, upload('second', 'queued')] })
    await wrapper.vm.$nextTick()
    expect(toggle.attributes('aria-expanded')).toBe('true')
    expect(wrapper.get('#library-upload-tray-body').isVisible()).toBe(true)
  })

  it('offers retry for errors and only allows dismissing an entirely settled tray', async () => {
    wrapper = mountTray([
      upload('complete', 'ready'),
      upload('failed', 'error', {
        errorCode: 'LIBRARY_FILE_TOO_LARGE',
        errorMessage: 'server fallback',
      }),
    ])

    expect(wrapper.get('.library-upload-tray').attributes('aria-busy')).toBe('false')
    expect(wrapper.get('.library-upload-tray__summary strong').text()).toBe('2/2')
    const readyCheck = wrapper.get('.library-upload-tray__ready-check')
    expect(readyCheck.attributes('viewBox')).toBe('0 0 12 12')
    expect(readyCheck.get('path').attributes('d')).toBe('M2.25 6.375l3 3 4.5-6.75')
    expect(wrapper.get('.library-upload-tray__row--error .library-upload-tray__copy span').text())
      .toBe('library.errors.tooLarge')

    const retry = wrapper.get('.library-upload-tray__retry')
    expect(retry.attributes('aria-label')).toBe('library.uploadTray.retryNamed name=failed.pdf')
    await retry.trigger('click')
    expect(wrapper.emitted('retry')).toEqual([['failed']])

    await wrapper.get('[aria-label="library.uploadTray.dismiss"]').trigger('click')
    expect(wrapper.emitted('dismiss')).toEqual([[]])
  })

  it('uses a viewport-fixed, theme-aware tray with a safe mobile inset', () => {
    expect(traySource).toMatch(
      /\.library-upload-tray\s*\{[^}]*position:\s*fixed;[^}]*inset-inline-end:\s*16px;[^}]*bottom:\s*16px;/s,
    )
    expect(traySource).toContain('width: min(420px, calc(100vw - 32px));')
    expect(traySource).toContain('border: 0;')
    expect(traySource).toContain('background: var(--workspace-popover-surface);')
    expect(traySource).toContain('inset 0 0 0 1px var(--workspace-popover-border)')
    expect(traySource).toContain('var(--workspace-popover-shadow);')
    expect(traySource).toMatch(
      /\.library-upload-tray__header\s*\{[^}]*box-sizing:\s*border-box;[^}]*min-height:\s*52px;/s,
    )
    expect(traySource).toMatch(
      /\.library-upload-tray__row\s*\{[^}]*min-height:\s*62px;/s,
    )
    expect(traySource).toMatch(
      /\.library-upload-tray__ready-check\s*\{[^}]*width:\s*12px;[^}]*height:\s*12px;[^}]*stroke-width:\s*1\.25;/s,
    )
    expect(traySource).toMatch(
      /@media \(max-width: 640px\) \{[\s\S]*?inset-inline:\s*12px;[\s\S]*?env\(safe-area-inset-bottom\)/,
    )
    expect(traySource).toContain('@media (prefers-reduced-motion: reduce)')
  })
})
