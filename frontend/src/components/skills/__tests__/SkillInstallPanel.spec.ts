import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SkillInstallPanel from '../SkillInstallPanel.vue'
import type { PublicSkill, PublicSkillVersion } from '@/api/skills'

const clipboardState = vi.hoisted(() => ({
  copy: vi.fn().mockResolvedValue(true),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copied: { value: false },
    copyToClipboard: clipboardState.copy,
  }),
}))

const translations = vi.hoisted<Record<string, string>>(() => ({
  'skills.install.title': '安装到 Codex',
  'skills.install.localOnly': '本地完成',
  'skills.install.scopeTitle': '选择使用范围',
  'skills.install.scopeDescription': '选择范围',
  'skills.install.personal': '所有项目',
  'skills.install.personalHint': '个人级',
  'skills.install.project': '当前项目',
  'skills.install.projectHint': '仓库级',
  'skills.install.actionTitle': '复制或下载',
  'skills.install.actionDescription': '选择一种方式',
  'skills.install.copyForCodex': '复制给 Codex',
  'skills.install.promptCopied': '已复制给 Codex',
  'skills.install.copySuccess': '安装指令已复制',
  'skills.install.downloadZip': '下载指定版本 ZIP',
  'skills.install.downloadStarted': '下载已开始',
  'skills.install.versionUnavailable': '暂无可下载版本',
  'skills.install.noAutoRun': '不要运行脚本',
  'skills.install.verifyTitle': '验证是否可用',
  'skills.install.verifyDescription': '显式调用',
  'skills.install.verifyOr': '或者',
  'skills.install.restartHint': '用 $skill-name 或 /skills 检查',
  'skills.install.shaUnavailable': '未提供',
  'skills.install.codexPrompt': 'Skill={name}; slug={slug}; version={version}; url={url}; sha={sha256}; path={path}',
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) => Object.entries(params ?? {}).reduce(
        (message, [name, value]) => message.replace(`{${name}}`, value),
        translations[key] ?? key,
      ),
    }),
  }
})

function version(): PublicSkillVersion {
  return {
    version: '1.4.0',
    changelog: '',
    manifest_name: 'docs',
    manifest_description: '',
    skill_md: '',
    sha256: 'abc123',
    byte_size: 128,
    unpacked_size: 256,
    file_count: 1,
    file_manifest: [],
    validation_report: { valid: true, errors: [], warnings: [] },
    created_at: '2026-08-03T10:00:00Z',
    download_count: 2,
  }
}

function skill(currentVersion: PublicSkillVersion | null = version()): PublicSkill {
  return {
    slug: 'api-docs',
    display_name: 'API Docs',
    summary: '',
    description: '',
    category: 'documentation',
    tags: [],
    icon: '',
    example_prompts: [],
    risk_notes: '',
    featured: false,
    current_version: currentVersion,
    download_count: 0,
    published_at: '',
    updated_at: '',
    versions: currentVersion ? [currentVersion] : [],
  }
}

function mountPanel(currentVersion: PublicSkillVersion | null = version()) {
  return mount(SkillInstallPanel, {
    props: { skill: skill(currentVersion) },
    global: {
      stubs: { Icon: true },
    },
  })
}

describe('SkillInstallPanel', () => {
  beforeEach(() => clipboardState.copy.mockClear())

  it('copies exact-version personal and project instructions without claiming installation', async () => {
    const wrapper = mountPanel()
    const primary = wrapper.get('.skill-install__primary')

    await primary.trigger('click')
    expect(clipboardState.copy).toHaveBeenLastCalledWith(
      expect.stringContaining('path=$HOME/.agents/skills/api-docs'),
      '安装指令已复制',
    )
    expect(clipboardState.copy.mock.calls[0][0]).toContain('version=1.4.0')
    expect(clipboardState.copy.mock.calls[0][0]).toContain('/versions/1.4.0/download')

    await wrapper.findAll('[role="radio"]')[1].trigger('click')
    await primary.trigger('click')
    expect(clipboardState.copy.mock.calls[1][0]).toContain('path=<repo>/.agents/skills/api-docs')
    expect(wrapper.text()).not.toContain('已安装')
  })

  it('disables copy and download when no immutable current version exists', () => {
    const wrapper = mountPanel(null)
    expect(wrapper.get('.skill-install__primary').attributes('disabled')).toBeDefined()
    expect(wrapper.get('.skill-install__secondary--disabled').attributes('disabled')).toBeDefined()
    expect(wrapper.find('a[download]').exists()).toBe(false)
  })
})
