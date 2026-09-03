import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkspaceDesktopSidebarHeaderIcon from '../WorkspaceDesktopSidebarHeaderIcon.vue'

const directory = dirname(fileURLToPath(import.meta.url))
const componentSource = readFileSync(
  resolve(directory, '../WorkspaceDesktopSidebarHeaderIcon.vue'),
  'utf8',
)

describe('WorkspaceDesktopSidebarHeaderIcon', () => {
  it('renders the dedicated ChatGPT desktop search glyph at 20px', () => {
    const wrapper = mount(WorkspaceDesktopSidebarHeaderIcon, {
      props: { name: 'search' },
    })

    expect(wrapper.attributes('viewBox')).toBe('0 0 24 24')
    expect(wrapper.attributes('fill')).toBe('currentColor')
    expect(wrapper.classes()).toContain('workspace-desktop-sidebar-header-icon--search')
    expect(wrapper.get('path').attributes('d')).toBe(
      'M10.993 2.904a8 8 0 0 1 6.122 13.146l3.923 3.923a.75.75 0 0 1-1.06 1.06l-3.931-3.93a8 8 0 1 1-5.054-14.2m0 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13',
    )
  })

  it('renders the dedicated ChatGPT desktop sidebar glyph at 20px', () => {
    const wrapper = mount(WorkspaceDesktopSidebarHeaderIcon, {
      props: { name: 'collapse' },
    })

    expect(wrapper.attributes('viewBox')).toBe('0 0 20 20')
    expect(wrapper.attributes('fill')).toBe('currentColor')
    expect(wrapper.classes()).toContain('workspace-desktop-sidebar-header-icon--collapse')
    expect(wrapper.get('path').attributes('d')).toContain('M6.835 4c-.451.004')
    expect(wrapper.get('path').attributes('d')).toContain('H8.164L8.165 4z')
  })

  it('keeps both desktop header glyphs on the exact shared 20px box', () => {
    expect(componentSource).toMatch(
      /\.workspace-desktop-sidebar-header-icon\s*\{[^}]*width: var\(--workspace-space-5\);[^}]*height: var\(--workspace-space-5\);/s,
    )
  })
})
