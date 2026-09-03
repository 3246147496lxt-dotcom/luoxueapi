import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const read = (path: string): string => readFileSync(resolve(directory, path), 'utf8')

const appSidebarSource = read('../AppSidebar.vue')
const appLayoutSource = read('../AppLayout.vue')
const chatHistorySource = read('../../chat/ChatHistoryPanel.vue')
const chatViewSource = read('../../../views/user/ChatView.vue')
const accountDockSource = read('../SidebarAccountDock.vue')
const mobileHeaderSource = read('../AppMobileHeader.vue')
const frameSource = read('../WorkspaceSidebarFrame.vue')
const headerSource = read('../WorkspaceSidebarHeader.vue')
const brandSource = read('../WorkspaceSidebarBrand.vue')
const modeSwitchSource = read('../AppModeSwitch.vue')
const collapseSource = read('../useWorkspaceSidebarCollapse.ts')
const responsiveSource = read('../workspaceResponsive.ts')
const responsiveIconSource = read('../WorkspaceResponsiveSidebarIcon.vue')
const desktopHeaderIconSource = read('../WorkspaceDesktopSidebarHeaderIcon.vue')
const overlayTriggerSource = read('../WorkspaceSidebarOverlayTrigger.vue')
const overlayLayerSource = read('../WorkspaceSidebarOverlayLayer.vue')
const workspaceTokens = read('../../../styles/luoxue-clay-tokens.css')
const globalStyleSource = read('../../../style.css')

describe('shared Workspace shell contract', () => {
  it('composes both Work and Chat hosts from WorkspaceSidebarFrame and WorkspaceSidebarHeader', () => {
    expect(appSidebarSource).toContain("import WorkspaceSidebarFrame from './WorkspaceSidebarFrame.vue'")
    expect(appSidebarSource).toContain("import WorkspaceSidebarHeader from './WorkspaceSidebarHeader.vue'")
    expect(chatHistorySource).toContain(
      "import WorkspaceSidebarFrame from '@/components/layout/WorkspaceSidebarFrame.vue'",
    )
    expect(chatHistorySource).toContain(
      "import WorkspaceSidebarHeader from '@/components/layout/WorkspaceSidebarHeader.vue'",
    )

    expect(appSidebarSource).toMatch(
      /<WorkspaceSidebarFrame[\s\S]*?id="app-sidebar"[\s\S]*?:collapsed="sidebarCollapsed"[\s\S]*?:mobile="mobileViewport"[\s\S]*?:overlay="personalNarrowViewport"[\s\S]*?:placement="personalNarrowViewport \? 'flow' : 'fixed'"[\s\S]*?surface="work"/,
    )
    expect(chatHistorySource).toMatch(
      /<WorkspaceSidebarFrame[\s\S]*?:id="resolvedSidebarId"[\s\S]*?:collapsed="sidebarCollapsed"[\s\S]*?:mobile="mobile"[\s\S]*?:overlay="overlay"[\s\S]*?placement="flow"[\s\S]*?surface="chat"/,
    )
    expect(appSidebarSource).toMatch(
      /<template #header>[\s\S]*?<WorkspaceSidebarHeader[\s\S]*?:collapsed="sidebarCollapsed"[\s\S]*?:mobile="mobileViewport"/,
    )
    expect(chatHistorySource).toMatch(
      /<template #header>[\s\S]*?<WorkspaceSidebarHeader[\s\S]*?:collapsed="sidebarCollapsed"[\s\S]*?:mobile="mobile"/,
    )
  })

  it('uses one fixed brand and membership implementation in personal Work and Chat', () => {
    expect(appSidebarSource).toContain("import WorkspaceSidebarBrand from './WorkspaceSidebarBrand.vue'")
    expect(chatHistorySource).toContain(
      "import WorkspaceSidebarBrand from '@/components/layout/WorkspaceSidebarBrand.vue'",
    )
    expect(appSidebarSource).toMatch(
      /<WorkspaceSidebarBrand[\s\S]*?v-if="isPersonalWorkWorkspace"[\s\S]*?home-path="\/dashboard"/,
    )
    expect(chatHistorySource).toContain('<WorkspaceSidebarBrand home-path="/chat" />')
    expect(brandSource).toContain("const brandName = '落雪AI'")
    expect(brandSource).toContain("useUserProfileStore")
    expect(brandSource).not.toContain('useAccountSummary')
    expect(brandSource).toContain('membership.accountPlanLabel.value')
    expect(brandSource).not.toContain('appStore.siteName')
    expect(appSidebarSource).toContain(':collapsed-logo-src="appStore.siteLogo"')
    expect(chatHistorySource).toContain(':collapsed-logo-src="appStore.siteLogo"')
  })

  it('keeps one 260px expanded/overlay token and one desktop-only 68px rail token', () => {
    expect(workspaceTokens.match(/--workspace-sidebar-width:\s*260px;/g)).toHaveLength(1)
    expect(workspaceTokens.match(/--workspace-sidebar-width-collapsed:\s*68px;/g)).toHaveLength(1)
    expect(workspaceTokens).not.toContain('--workspace-chat-sidebar-width-collapsed')

    expect(frameSource).toContain('width: var(--workspace-sidebar-width);')
    expect(frameSource).toContain('min-width: var(--workspace-sidebar-width);')
    expect(frameSource).toContain('width: var(--workspace-sidebar-width-collapsed);')
    expect(frameSource).toContain('min-width: var(--workspace-sidebar-width-collapsed);')
    expect(frameSource).toContain('.workspace-sidebar-frame--overlay.workspace-sidebar-frame--collapsed')
    expect(appLayoutSource).toContain('ml-[var(--workspace-sidebar-width)]')
    expect(appLayoutSource).toContain('ml-[var(--workspace-sidebar-width-collapsed)]')
    expect(appLayoutSource).not.toContain('md:ml-[var(--workspace-sidebar-width')

    for (const [name, source] of [
      ['AppSidebar', appSidebarSource],
      ['ChatHistoryPanel', chatHistorySource],
      ['WorkspaceSidebarFrame', frameSource],
      ['WorkspaceSidebarHeader', headerSource],
    ] as const) {
      expect(source, `${name} must not consume the retired Chat-only collapse token`)
        .not.toContain('--workspace-chat-sidebar-width-collapsed')
      expect(source, `${name} must not hard-code the retired 72px collapsed width`)
        .not.toMatch(/\b72px\b/)
    }
  })

  it('fills the full Chat shell instead of inheriting the generic 1600px page cap', () => {
    expect(appLayoutSource).toMatch(
      /\.app-layout--chat-shell \.app-main-inner\s*\{[^}]*max-width:\s*none;[^}]*margin-right:\s*0 !important;/,
    )
    expect(appLayoutSource).toContain('class="app-main-inner mx-auto w-full max-w-[1600px]"')
  })

  it('copies the 768px boundary into distinct narrow-overlay and mobile-drawer classifiers', () => {
    expect(responsiveSource).toContain(
      "'(max-width: 767px) and (hover: none) and (pointer: coarse)'",
    )
    expect(responsiveSource).toContain(
      "'(max-width: 767px) and (hover: hover) and (pointer: fine)'",
    )
    expect(responsiveSource).toContain(
      'window.matchMedia(WORKSPACE_MOBILE_DRAWER_MEDIA_QUERY)',
    )
    expect(responsiveSource).toContain(
      'window.matchMedia(WORKSPACE_NARROW_SIDEBAR_MEDIA_QUERY)',
    )
    expect(appLayoutSource).toContain('useWorkspaceResponsiveState()')
    expect(appSidebarSource).toContain('useWorkspaceResponsiveState()')
    expect(chatViewSource).toContain('useWorkspaceResponsiveState()')
    expect(appSidebarSource).toContain('@media (min-width: 768px)')
    expect(appSidebarSource).toContain(
      '@media (max-width: 767px) and (hover: none) and (pointer: coarse)',
    )
    expect(appSidebarSource).not.toContain("window.matchMedia('(max-width: 1023px)')")

    expect(frameSource).toContain('@media (min-width: 768px)')
    expect(frameSource).toContain(
      '@media (max-width: 767px) and (hover: none) and (pointer: coarse)',
    )
    expect(mobileHeaderSource).toContain('class="app-mobile-header"')
    expect(mobileHeaderSource).toContain('display: none;')
    expect(globalStyleSource).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)\s*\{\s*:root\s*\{\s*--app-shell-top-offset: var\(--app-shell-mobile-bar-height\);/,
    )
    expect(appLayoutSource).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)[\s\S]*?\.app-layout--snow-shell:not\(\.app-layout--chat-shell\) \.app-main-shell\s*\{[^}]*margin-left: 0 !important;/,
    )

    const syncViewport = responsiveSource.match(
      /function syncViewportState[\s\S]*?\n {2}\}/,
    )?.[0] ?? ''
    expect(syncViewport).toContain('setWorkspaceMobileDrawer')
    expect(syncViewport).toContain('setWorkspaceNarrowSidebar')
    expect(syncViewport).not.toContain('setSidebarCollapsed')
    expect(syncViewport).not.toContain('toggleSidebar')
  })

  it('keeps the narrow responsive controls separate from the normal desktop collapse icon', () => {
    expect(appLayoutSource).toContain(
      "import WorkspaceSidebarOverlayTrigger from './WorkspaceSidebarOverlayTrigger.vue'",
    )
    expect(appLayoutSource).toContain('v-if="personalWorkspaceNarrow"')
    expect(appLayoutSource).toContain("personalWorkspaceNarrow\n            ? 'ml-0'")
    expect(overlayTriggerSource).toContain('<WorkspaceResponsiveSidebarIcon name="open" />')
    expect(overlayTriggerSource).toContain('width: var(--workspace-sidebar-action-size);')
    expect(overlayTriggerSource).toContain('height: var(--workspace-sidebar-action-size);')

    expect(appSidebarSource).toContain(
      "import WorkspaceSidebarOverlayLayer from './WorkspaceSidebarOverlayLayer.vue'",
    )
    expect(chatViewSource).toContain(
      "import WorkspaceSidebarOverlayLayer from '@/components/layout/WorkspaceSidebarOverlayLayer.vue'",
    )
    expect(appSidebarSource).toContain(':active="personalNarrowViewport"')
    expect(chatViewSource).toContain('sidebar-id="workspace-chat-sidebar-overlay"')
    expect(overlayLayerSource).toContain('position: fixed;')
    expect(overlayLayerSource).toContain('width: var(--workspace-sidebar-width);')
    expect(overlayLayerSource).toContain('background: var(--workspace-sidebar-overlay-backdrop);')
    expect(overlayLayerSource).toContain('box-shadow: var(--workspace-sidebar-overlay-shadow);')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-overlay-backdrop: var(--workspace-light-sidebar-overlay-backdrop);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-overlay-backdrop: var(--workspace-dark-sidebar-overlay-backdrop);',
    )

    expect(headerSource).toMatch(
      /<WorkspaceResponsiveSidebarIcon[\s\S]*?v-if="overlay"[\s\S]*?name="search"/,
    )
    expect(headerSource).toMatch(
      /<WorkspaceDesktopSidebarHeaderIcon[\s\S]*?v-else[\s\S]*?name="search"/,
    )
    expect(headerSource).toMatch(
      /<WorkspaceDesktopSidebarHeaderIcon[\s\S]*?name="collapse"/,
    )
    expect(headerSource).toMatch(
      /<WorkspaceResponsiveSidebarIcon name="close" aria-hidden="true"/,
    )
    expect(headerSource).toContain('<SidebarCollapseIcon')
    expect(desktopHeaderIconSource).toContain("name: 'search' | 'collapse'")
    expect(desktopHeaderIconSource).toContain('workspace-desktop-sidebar-header-icon--${name}')
    expect(responsiveIconSource).toContain("type WorkspaceResponsiveSidebarIconName = 'open' | 'close' | 'search'")
  })

  it('keeps the Chat and Work shell hosts free of the retired mode switch', () => {
    expect(appSidebarSource).not.toContain('<AppModeSwitch')
    expect(appSidebarSource).not.toContain("import AppModeSwitch")
    expect(appSidebarSource).not.toContain('handleModeChange')
    expect(chatHistorySource).not.toContain('<AppModeSwitch')
    expect(chatHistorySource).not.toContain("import AppModeSwitch")
    expect(chatHistorySource).not.toContain('handleModeChange')

    expect(modeSwitchSource).toMatch(/defineProps<\{\s*activeMode: AppShellMode\s*\}>\(\)/)
    expect(modeSwitchSource).toMatch(/defineEmits<\{\s*change: \[mode: AppShellMode\]\s*\}>\(\)/)
    expect(modeSwitchSource).not.toContain('compact')
    expect(modeSwitchSource).not.toContain('RouterLink')
    expect(modeSwitchSource).not.toContain('useRouter')
  })

  it('forbids shell hosts from reaching into AppModeSwitch with parent deep selectors', () => {
    expect(appSidebarSource).not.toMatch(/:deep\(\s*\.app-mode-switch/)
    expect(chatHistorySource).not.toMatch(/:deep\(\s*\.app-mode-switch/)
    expect(appLayoutSource).not.toMatch(/:deep\(\s*\.app-mode-switch/)
  })

  it('keeps legacy page selectors from overriding shared Header and content geometry', () => {
    expect(appSidebarSource).not.toContain('class="sidebar-header"')
    expect(appSidebarSource).not.toContain('class="sidebar-nav ')
    expect(appSidebarSource).toContain('class="workspace-sidebar-navigation scrollbar-hide"')
    expect(appLayoutSource).not.toMatch(/#app-sidebar[\s\S]*?\.sidebar-(?:header|nav)/)
    expect(appSidebarSource).not.toContain('class="sidebar"')
  })

  it('routes both hosts through one Pinia-backed state and focus handoff', () => {
    expect(appSidebarSource).toContain(
      "import { useWorkspaceSidebarCollapse } from './useWorkspaceSidebarCollapse'",
    )
    expect(chatHistorySource).toContain(
      "import { useWorkspaceSidebarCollapse } from '@/components/layout/useWorkspaceSidebarCollapse'",
    )
    for (const [name, source] of [
      ['AppSidebar', appSidebarSource],
      ['ChatHistoryPanel', chatHistorySource],
    ] as const) {
      expect(source, `${name} must consume the shared collapse adapter`)
        .toContain('useWorkspaceSidebarCollapse({')
      expect(source, `${name} must expose the shared Header ref`)
        .toMatch(/headerRef:\s*sidebarHeaderRef/)
      expect(source, `${name} must expose the shared toggle action`)
        .toMatch(/toggle:\s*toggleSidebar/)
      expect(source, `${name} must consume the adapter-rendered state`)
        .toMatch(/collapsed:\s*sidebarCollapsed/)
    }

    expect(collapseSource).toContain("import { useAppStore } from '@/stores/app'")
    expect(collapseSource).toContain('const requestedCollapsed = computed(() => appStore.sidebarCollapsed)')
    expect(collapseSource).toContain('const overlay = computed(() => toValue(options.overlay ?? false))')
    expect(collapseSource).toContain('const collapsed = computed(() => (')
    expect(collapseSource).toContain('&& !overlay.value')
    expect(collapseSource).toContain('if (!enabled.value || mobile.value || overlay.value) return')
    expect(collapseSource).not.toContain('setSidebarResponsiveExpanded')
    expect(collapseSource).not.toContain('setSidebarCompactViewport')
    expect(collapseSource).toContain('await nextTick()')
    expect(collapseSource).toContain('headerRef.value?.focusToggle()')
    expect(appSidebarSource).toContain('overlay: personalNarrowViewport')
    expect(chatHistorySource).toContain("overlay: toRef(props, 'overlay')")

    expect(chatHistorySource).not.toMatch(/const\s+collapsed\s*=\s*ref\(/)
    expect(chatHistorySource).not.toContain('collapsed.value =')
    expect(appSidebarSource).not.toContain('appStore.sidebarCollapsed')
    expect(accountDockSource).not.toContain('appStore.sidebarCollapsed')
    expect(mobileHeaderSource).not.toContain('setSidebarCollapsed')
  })

  it('passes the same rendered state through Navigation and both Account footers', () => {
    expect(appSidebarSource).toContain(
      '<UserAccountCard context="work" :collapsed="sidebarCollapsed" />',
    )
    expect(chatHistorySource).toContain(
      '<UserAccountCard context="chat" :collapsed="sidebarCollapsed" />',
    )
    expect(accountDockSource).toContain(
      'const sidebarCollapsed = computed(() => props.collapsed)',
    )
    expect(accountDockSource).toContain("useUserProfileStore")
    expect(accountDockSource).not.toContain('useAccountSummary')
    expect(appSidebarSource).toContain("'sidebar-link-collapsed': sidebarCollapsed")
    expect(chatHistorySource).toContain("'chat-history--collapsed': sidebarCollapsed")
    expect(frameSource).toContain('<slot v-if="!effectiveCollapsed" name="mode-switch" />')
  })

  it('keeps the Frame slot order and shared Header boundary explicit', () => {
    const headerSlot = frameSource.indexOf('<slot name="header" />')
    const modeSwitchSlot = frameSource.indexOf('<slot v-if="!effectiveCollapsed" name="mode-switch" />')
    const content = frameSource.indexOf('class="workspace-sidebar-frame__content"')
    const defaultSlot = frameSource.indexOf('<slot />')
    const footerSlot = frameSource.indexOf('<slot name="footer" />')

    expect(headerSlot).toBeGreaterThan(-1)
    expect(headerSlot).toBeLessThan(modeSwitchSlot)
    expect(modeSwitchSlot).toBeLessThan(content)
    expect(content).toBeLessThan(defaultSlot)
    expect(defaultSlot).toBeLessThan(footerSlot)
    expect(headerSource).toContain('height: var(--workspace-sidebar-header-height);')
    expect(headerSource).toContain(':data-testid="toggleTestId"')
    expect(headerSource).toContain("toggleTestId: 'workspace-sidebar-collapse-toggle'")
    expect(frameSource).toContain('padding: var(--workspace-sidebar-content-padding);')
    expect(frameSource).toContain('class="workspace-sidebar-frame__footer"')
    expect(chatHistorySource).not.toMatch(/:deep\(\s*\.sidebar-account-dock/)
    expect(frameSource).toContain(
      'width var(--workspace-sidebar-transition-duration) var(--workspace-sidebar-transition-easing)',
    )
    expect(appLayoutSource).toContain(
      'duration-[var(--workspace-sidebar-transition-duration)]',
    )
    expect(appLayoutSource).toContain('ease-[var(--workspace-sidebar-transition-easing)]')
  })
})
