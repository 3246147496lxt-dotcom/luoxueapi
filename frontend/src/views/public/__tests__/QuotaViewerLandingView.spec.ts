import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import QuotaViewerLandingView from '@/views/public/QuotaViewerLandingView.vue'

const state = vi.hoisted(() => ({
  authenticated: false,
  query: {} as Record<string, string>,
  fullPath: '/quota-viewer',
  push: vi.fn(),
  replace: vi.fn(),
  issue: vi.fn(),
  resolve: vi.fn(),
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({ get isAuthenticated() { return state.authenticated } }),
  useAppStore: () => ({ siteName: '落雪API', cachedPublicSettings: null }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({
      path: '/quota-viewer',
      get query() { return state.query },
      get fullPath() { return state.fullPath },
      hash: '',
    }),
    useRouter: () => ({ push: state.push, replace: state.replace }),
  }
})

vi.mock('@/api/quotaViewer', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/quotaViewer')>()
  return {
    ...actual,
    issueQuotaViewerInstallerDownload: state.issue,
    resolveQuotaViewerInstallerDownloadURL: state.resolve,
  }
})

const text: Record<string, string> = {
  'quotaViewerLanding.preview.week': '周剩余 32%',
  'quotaViewerLanding.preview.reset': '2 天 14 小时后重置',
  'quotaViewerLanding.preview.month': '月剩余 32% · 8/25 到期',
  'quotaViewerLanding.runway.items.account.title': '账户',
  'quotaViewerLanding.runway.items.install.title': '安装',
  'quotaViewerLanding.runway.items.pair.title': '配对',
  'quotaViewerLanding.runway.items.desktop.title': '桌面',
  'quotaViewerLanding.runway.items.renew.title': '续费',
}

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN', __v_isRef: true },
      t: (key: string) => text[key] || key,
    }),
  }
})

function release(platform: 'macos' | 'windows') {
  const macos = platform === 'macos'
  return {
    platform,
    version: macos ? '2.0.0-rc.6' : '2.0.0-rc.5',
    filename: macos
      ? 'Luoxue-Quota-Viewer_2.0.0-rc.6_macOS-universal-UNNOTARIZED.dmg'
      : 'Luoxue-Quota-Viewer_2.0.0-rc.5_windows-x64_NSIS-UNSIGNED.exe',
    sha256: macos
      ? '63c7e3890ef45e882a99e609c73f3fa5e8452cc708bc4e1ed6f0278bdc519f92'
      : '2f1a0b37f3720d02564d306bc5393e71edfbc2ea5d788b89f87feb9a2b4a27d1',
    size: macos ? 10_655_545 : 5_110_179,
    architecture: macos ? 'universal' : 'x64',
    signing_status: macos ? 'unsigned-unnotarized' : 'unsigned',
    expires_in: 60,
    download_path: `/api/v1/quota/releases/${platform}/latest/download?code=${'a'.repeat(43)}`,
  }
}

function mountView() {
  return mount(QuotaViewerLandingView, {
    global: {
      stubs: {
        PublicSiteLayout: { template: '<div><slot /></div>' },
        Icon: true,
        RouterLink: { props: ['to'], template: '<a :href="typeof to === \'string\' ? to : to.path"><slot /></a>' },
      },
    },
  })
}

describe('QuotaViewerLandingView', () => {
  beforeEach(() => {
    state.authenticated = false
    state.query = {}
    state.fullPath = '/quota-viewer'
    state.push.mockReset()
    state.replace.mockReset().mockImplementation(async (location: {
      path: string
      query?: Record<string, string>
      hash?: string
    }) => {
      state.query = { ...(location.query || {}) }
      const search = new URLSearchParams(state.query).toString()
      state.fullPath = `${location.path}${search ? `?${search}` : ''}${location.hash || ''}`
    })
    state.issue.mockReset()
    state.resolve.mockReset().mockReturnValue(
      `http://localhost/api/v1/quota/releases/macos/latest/download?code=${'a'.repeat(43)}`,
    )
  })

  it('keeps the approved copy compact and orders the five-step runway', () => {
    const wrapper = mountView()
    expect(wrapper.findAll('h1')).toHaveLength(1)
    expect(wrapper.text()).toContain('周剩余 32%')
    expect(wrapper.text()).toContain('2 天 14 小时后重置')
    expect(wrapper.text()).toContain('月剩余 32% · 8/25 到期')
    expect(wrapper.text()).not.toContain('2026-')
    expect(wrapper.findAll('.quota-runway strong').map((node) => node.text())).toEqual([
      '账户', '安装', '配对', '桌面', '续费',
    ])
  })

  it('sends an anonymous macOS download through login with the exact return intent', async () => {
    const wrapper = mountView()
    await wrapper.get('[data-testid="download-macos"]').trigger('click')

    expect(state.push).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/quota-viewer?download=macos' },
    })
    expect(state.issue).not.toHaveBeenCalled()
  })

  it('allows a signed-in non-member to issue and trigger a one-time download', async () => {
    state.authenticated = true
    state.issue.mockResolvedValue(release('windows'))
    state.resolve.mockReturnValue(
      `http://localhost/api/v1/quota/releases/windows/latest/download?code=${'a'.repeat(43)}`,
    )
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const wrapper = mountView()

    await wrapper.get('[data-testid="download-windows"]').trigger('click')
    await flushPromises()

    expect(state.issue).toHaveBeenCalledOnce()
    expect(state.issue).toHaveBeenCalledWith('windows')
    expect(state.replace).toHaveBeenNthCalledWith(1, {
      path: '/quota-viewer',
      query: { download: 'windows' },
      hash: '',
    })
    expect(state.replace).toHaveBeenLastCalledWith({ path: '/quota-viewer', query: {}, hash: '' })
    expect(click).toHaveBeenCalledOnce()
    click.mockRestore()
  })

  it('persists a manual download intent before an expired session can redirect', async () => {
    state.authenticated = true
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    state.issue.mockImplementation(async () => {
      expect(state.fullPath).toBe('/quota-viewer?download=macos')
      throw { status: 401, code: 'TOKEN_REFRESH_FAILED' }
    })
    const wrapper = mountView()

    await wrapper.get('[data-testid="download-macos"]').trigger('click')
    await flushPromises()

    expect(state.replace).toHaveBeenCalledOnce()
    expect(state.replace).toHaveBeenCalledWith({
      path: '/quota-viewer',
      query: { download: 'macos' },
      hash: '',
    })
    expect(state.query).toEqual({ download: 'macos' })
    expect(consoleError).toHaveBeenCalledOnce()
    consoleError.mockRestore()
  })

  it('continues a login return once and clears the download query before navigation', async () => {
    state.authenticated = true
    state.query = { download: 'macos' }
    state.fullPath = '/quota-viewer?download=macos'
    state.issue.mockResolvedValue(release('macos'))
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    mountView()
    await flushPromises()

    expect(state.issue).toHaveBeenCalledOnce()
    expect(state.replace).toHaveBeenCalledWith({ path: '/quota-viewer', query: {}, hash: '' })
    expect(click).toHaveBeenCalledOnce()
    click.mockRestore()
  })
})
