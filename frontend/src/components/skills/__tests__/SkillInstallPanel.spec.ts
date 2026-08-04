import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { spawnSync } from 'node:child_process'
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
  'skills.install.title': '安装',
  'skills.install.scopeLabel': '使用范围：',
  'skills.install.globalOption': '所有项目 / 全局',
  'skills.install.projectOption': '当前项目 / 仓库级',
  'skills.install.methodLabel': '安装方式',
  'skills.install.command': '命令安装',
  'skills.install.copyForCodex': '复制给 Codex',
  'skills.install.copyCommandAria': '复制安装命令',
  'skills.install.copyPromptAria': '复制 Codex 安装指令',
  'skills.install.commandCopySuccess': '安装命令已复制',
  'skills.install.copySuccess': '安装指令已复制',
  'skills.install.commandHint': '复制命令后在终端执行',
  'skills.install.codexHint': '复制完整指令交给 Codex',
  'skills.install.verifyOr': '或',
  'skills.install.verifySuffix': '验证。',
  'skills.install.versionUnavailable': '暂无可下载版本',
  'skills.install.shaUnavailable': '未提供',
  'skills.install.codexPrompt': 'Skill={name}; slug={slug}; version={version}; url={url}; sha={sha256}; path={path}; parent={parent}',
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
    props: {
      skill: skill(currentVersion),
      version: currentVersion?.version,
      sha256: currentVersion?.sha256,
    },
    global: {
      stubs: { Icon: true },
    },
  })
}

async function selectScope(wrapper: ReturnType<typeof mountPanel>, value: InstallScope) {
  await wrapper.get('[aria-label="使用范围："]').trigger('click')
  await nextTick()
  const menus = document.body.querySelectorAll<HTMLElement>('[data-ui-portal="settings-choice-menu"]')
  const activeMenu = menus.item(menus.length - 1)
  const option = activeMenu?.querySelector<HTMLButtonElement>(`[data-choice-value="${value}"]`)
  expect(option).not.toBeNull()
  option?.click()
  await nextTick()
}

type InstallScope = 'global' | 'project'

describe('SkillInstallPanel', () => {
  beforeEach(() => clipboardState.copy.mockClear())

  it('builds copyable global and repository-scoped shell commands', async () => {
    const wrapper = mountPanel()
    const copyButton = wrapper.get('[aria-label="复制安装命令"]')
    const globalCommand = wrapper.get('.skill-install__payload code').text()

    expect(wrapper.get('[role="tab"][aria-selected="true"]').text()).toBe('命令安装')
    expect(globalCommand).toContain('SKILLS_DIR="$HOME/.agents/skills"')
    expect(globalCommand).toContain('/versions/1.4.0/download')
    expect(globalCommand).toContain('abc123')
    expect(globalCommand).toContain('STAGE_ROOT="$(mktemp -d')
    expect(globalCommand).toContain('BACKUP_DIR="$STAGE_ROOT/.previous"')
    expect(globalCommand).toContain('test -f "$STAGE_ROOT/$SKILL_SLUG/SKILL.md"')
    expect(globalCommand).toContain('mv "$STAGE_ROOT/$SKILL_SLUG" "$TARGET_DIR"')
    expect(globalCommand).toContain('trap cleanup_install EXIT')
    expect(globalCommand).not.toContain('unzip -oq')
    const syntaxCheck = spawnSync('/bin/sh', ['-n'], { input: globalCommand, encoding: 'utf8' })
    expect(syntaxCheck.stderr).toBe('')
    expect(syntaxCheck.status).toBe(0)

    await copyButton.trigger('click')
    expect(clipboardState.copy).toHaveBeenLastCalledWith(
      expect.stringContaining('$HOME/.agents/skills'),
      '安装命令已复制',
    )

    await selectScope(wrapper, 'project')
    const projectCommand = wrapper.get('.skill-install__payload code').text()
    expect(projectCommand).toContain('git rev-parse --show-toplevel')
    expect(projectCommand).toContain('|| pwd')
    expect(projectCommand).toContain('SKILLS_DIR="$REPO_ROOT/.agents/skills"')
    expect(projectCommand).not.toContain('$PWD/.agents/skills')
    await copyButton.trigger('click')
    expect(clipboardState.copy).toHaveBeenLastCalledWith(
      expect.stringContaining('git rev-parse --show-toplevel'),
      '安装命令已复制',
    )
  })

  it('switches to a Codex instruction and keeps scope-specific paths', async () => {
    const wrapper = mountPanel()
    await wrapper.findAll('[role="tab"]')[1].trigger('click')

    expect(wrapper.get('.skill-install__payload code').text()).toContain('path=$HOME/.agents/skills/api-docs')
    await wrapper.get('[aria-label="复制 Codex 安装指令"]').trigger('click')
    expect(clipboardState.copy).toHaveBeenLastCalledWith(
      expect.stringContaining('version=1.4.0'),
      '安装指令已复制',
    )

    await selectScope(wrapper, 'project')
    expect(wrapper.get('.skill-install__payload code').text()).toContain('path=<repo>/.agents/skills/api-docs')
    expect(wrapper.text()).not.toContain('已安装')
  })

  it('disables copying when no immutable version exists', () => {
    const wrapper = mountPanel(null)
    expect(wrapper.get('.skill-install__payload code').text()).toBe('暂无可下载版本')
    expect(wrapper.get('.skill-install__payload button').attributes('disabled')).toBeDefined()
    expect(wrapper.find('a[download]').exists()).toBe(false)
    expect(wrapper.find('.skill-install__steps').exists()).toBe(false)
  })
})
