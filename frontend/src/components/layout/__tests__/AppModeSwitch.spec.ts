import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AppModeSwitch from '../AppModeSwitch.vue'

const directory = dirname(fileURLToPath(import.meta.url))
const componentSource = readFileSync(resolve(directory, '../AppModeSwitch.vue'), 'utf8')
const workspaceTokens = readFileSync(
  resolve(directory, '../../../styles/luoxue-clay-tokens.css'),
  'utf8',
)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const labels: Record<string, string> = {
    'nav.chatMode': 'Chat',
    'nav.workMode': 'Work',
  }
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => labels[key] ?? key }),
  }
})

describe('AppModeSwitch', () => {
  it('marks Work active and emits only an explicit change to Chat', async () => {
    const wrapper = mount(AppModeSwitch, {
      props: { activeMode: 'work' },
    })

    expect(wrapper.get('[data-testid="app-mode-switch"]').attributes('data-active-mode')).toBe('work')
    expect(wrapper.get('[aria-pressed="true"]').text()).toBe('Work')
    expect(wrapper.findAll('button')).toHaveLength(2)

    await wrapper.get('[aria-pressed="true"]').trigger('click')
    expect(wrapper.emitted('change')).toBeUndefined()

    await wrapper.findAll('button')[0]!.trigger('click')
    expect(wrapper.emitted('change')).toEqual([['chat']])
  })

  it('marks Chat active and emits only an explicit change to Work', async () => {
    const wrapper = mount(AppModeSwitch, {
      props: { activeMode: 'chat' },
    })

    expect(wrapper.get('[data-testid="app-mode-switch"]').attributes('data-active-mode')).toBe('chat')
    expect(wrapper.get('[aria-pressed="true"]').text()).toBe('Chat')
    expect(wrapper.findAll('button')).toHaveLength(2)

    await wrapper.get('[aria-pressed="true"]').trigger('click')
    expect(wrapper.emitted('change')).toBeUndefined()

    await wrapper.findAll('button')[1]!.trigger('click')
    expect(wrapper.emitted('change')).toEqual([['work']])
  })

  it('owns its complete geometry, typography, and theme contract', () => {
    expect(componentSource).toMatch(/defineProps<\{\s*activeMode: AppShellMode\s*\}>\(\)/)
    expect(componentSource).toMatch(/defineEmits<\{\s*change: \[mode: AppShellMode\]\s*\}>\(\)/)
    expect(componentSource).not.toContain('compact')
    expect(componentSource).not.toContain('RouterLink')
    expect(componentSource).not.toContain('useRouter')
    expect(componentSource).not.toContain(':deep(')

    expect(componentSource).toMatch(
      /\.app-mode-switch\s*\{[^}]*height: var\(--workspace-mode-switch-height\);/s,
    )
    expect(componentSource).not.toContain('font-family:')
    expect(componentSource).toMatch(
      /\.app-mode-switch__control\s*\{[^}]*border-radius: var\(--workspace-mode-switch-radius\);[^}]*background: var\(--workspace-mode-switch-track\);/s,
    )
    expect(componentSource).toMatch(
      /\.app-mode-switch__option\s*\{[^}]*border-radius: var\(--workspace-mode-switch-option-radius\);[^}]*font-size: var\(--workspace-type-navigation-size\);[^}]*font-weight: var\(--workspace-type-navigation-weight\);/s,
    )
    expect(componentSource).toContain('color: var(--workspace-mode-switch-text-muted);')
    expect(componentSource).toContain('background: var(--workspace-mode-switch-hover);')
    expect(componentSource).toContain('background: var(--workspace-mode-switch-active);')
    expect(componentSource).toContain('box-shadow: var(--workspace-mode-switch-active-shadow);')

    expect(workspaceTokens).toContain('--workspace-mode-switch-height: 46px;')
    expect(workspaceTokens).toContain('--workspace-mode-switch-height-mobile: 60px;')
    expect(workspaceTokens).toContain(
      "html:not(.dark) .app-mode-switch[data-active-mode='chat']",
    )
    expect(workspaceTokens).toContain("html.dark .app-mode-switch[data-active-mode='chat']")
    expect(componentSource).toContain(
      ':global(.workspace-sidebar-frame--mobile) .app-mode-switch__option',
    )
    expect(workspaceTokens).toContain('--workspace-type-navigation-size: 14px;')
    expect(workspaceTokens).toContain('--workspace-type-navigation-weight: 500;')
  })
})
