import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))

const historySource = readFileSync(resolve(directory, '../ChatHistoryPanel.vue'), 'utf8')
const historyStyles = scopedStyles(historySource)
const frameSource = readFileSync(
  resolve(directory, '../../layout/WorkspaceSidebarFrame.vue'),
  'utf8',
)
const frameStyles = scopedStyles(frameSource)
const headerSource = readFileSync(
  resolve(directory, '../../layout/WorkspaceSidebarHeader.vue'),
  'utf8',
)
const headerStyles = scopedStyles(headerSource)
const modeSwitchSource = readFileSync(resolve(directory, '../../layout/AppModeSwitch.vue'), 'utf8')
const modeSwitchStyles = scopedStyles(modeSwitchSource)
const brandSource = readFileSync(resolve(directory, '../../layout/WorkspaceSidebarBrand.vue'), 'utf8')
const brandStyles = scopedStyles(brandSource)
const workspaceTokens = readFileSync(resolve(directory, '../../../styles/luoxue-clay-tokens.css'), 'utf8')
const messageStyles = scopedStyles(readFileSync(resolve(directory, '../ChatMessageItem.vue'), 'utf8'))
const attachmentSource = readFileSync(resolve(directory, '../ChatAttachmentPicker.vue'), 'utf8')
const attachmentStyles = scopedStyles(attachmentSource)
const messageAttachmentSource = readFileSync(
  resolve(directory, '../ChatMessageAttachments.vue'),
  'utf8',
)
const messageAttachmentStyles = scopedStyles(messageAttachmentSource)
const viewSource = readFileSync(resolve(directory, '../../../views/user/ChatView.vue'), 'utf8')
const viewStyles = scopedStyles(viewSource)

function scopedStyles(source: string): string {
  const match = source.match(/<style scoped>([\s\S]*?)<\/style>/)
  if (!match?.[1]) throw new Error('Expected a scoped style block')
  return match[1]
}

function cssRule(styles: string, selector: string): string {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const match = styles.match(new RegExp(`${escapedSelector}\\s*\\{([\\s\\S]*?)\\}`))
  if (!match?.[1]) throw new Error(`Expected CSS rule for ${selector}`)
  return match[1]
}

describe('Chat shell visual contract', () => {
  it('keeps the desktop history rail compact and neutral', () => {
    const shell = cssRule(frameStyles, '.workspace-sidebar-frame')
    const collapsedShell = cssRule(frameStyles, '.workspace-sidebar-frame--collapsed')
    const shellHeader = cssRule(headerStyles, '.workspace-sidebar-header')
    const shellHeaderActions = cssRule(headerStyles, '.workspace-sidebar-header__actions')
    const modeSwitch = cssRule(modeSwitchStyles, '.app-mode-switch')
    const primaryNav = cssRule(historyStyles, '.chat-history__primary-nav')
    const fixedHeader = cssRule(historyStyles, '.chat-history__header')
    const historyList = cssRule(historyStyles, '.chat-history__list')
    const activeNewChat = cssRule(
      historyStyles,
      '.chat-history__new--active,\n.chat-history__new--active:hover,\n.chat-history__primary-action--active,\n.chat-history__primary-action--active:hover',
    )
    const primaryAction = cssRule(
      historyStyles,
      '.chat-history__new,\n.chat-history__primary-action',
    )
    const historyItem = cssRule(historyStyles, '.chat-history__item')
    const historySelect = cssRule(historyStyles, '.chat-history__select')
    const historyTitle = cssRule(historyStyles, '.chat-history__select strong')
    const historySelectWithActions = cssRule(
      historyStyles,
      '.chat-history__item:hover .chat-history__select,\n.chat-history__item:focus-within .chat-history__select',
    )
    const historyMenu = cssRule(historyStyles, '.chat-history__conversation-menu[popover]')
    const historyMenuItem = cssRule(historyStyles, '.chat-history__conversation-menu-item')
    const historyGroup = cssRule(historyStyles, '.chat-history__group')
    const historyGroupHeading = cssRule(historyStyles, '.chat-history__group h2')

    expect(shell).toContain('width: var(--workspace-sidebar-width)')
    expect(shell).toContain('min-width: var(--workspace-sidebar-width)')
    expect(collapsedShell).toContain('width: var(--workspace-sidebar-width-collapsed)')
    expect(collapsedShell).toContain('min-width: var(--workspace-sidebar-width-collapsed)')
    expect(shellHeader).toContain('height: var(--workspace-sidebar-header-height)')
    expect(shellHeader).toContain('flex: 0 0 var(--workspace-sidebar-header-height)')
    expect(shellHeader).toContain('gap: var(--workspace-space-1)')
    expect(shellHeader).toContain(
      'padding: var(--workspace-space-2) var(--workspace-sidebar-header-padding-inline)',
    )
    expect(shellHeader).toContain(
      'padding-inline-end: var(--workspace-sidebar-header-actions-inset-end)',
    )
    expect(shellHeaderActions).toContain('gap: 0')
    expect(modeSwitch).toContain('flex: 0 0 var(--workspace-mode-switch-height)')
    expect(workspaceTokens).toContain('--workspace-sidebar-width: 260px;')
    expect(workspaceTokens).toContain('--workspace-sidebar-width-collapsed: 68px;')
    expect(workspaceTokens).not.toContain('--workspace-chat-sidebar-width-collapsed')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-width-mobile-chat: min(288px, calc(100vw - 40px));',
    )
    expect(workspaceTokens).toContain('--workspace-sidebar-header-height: 52px;')
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-header-padding-inline: var(--workspace-space-3);',
    )
    expect(workspaceTokens).toContain(
      '--workspace-sidebar-header-actions-inset-end: var(--workspace-space-1-75);',
    )
    expect(workspaceTokens).toContain('--workspace-mode-switch-height: 46px;')
    expect(workspaceTokens).toContain('--workspace-space-1: 4px;')
    expect(workspaceTokens).toContain('--workspace-space-2: 8px;')
    expect(workspaceTokens).toContain('--workspace-space-3: 12px;')
    expect(primaryNav).toContain('flex-direction: column')
    expect(primaryNav).toContain('gap: 0')
    expect(fixedHeader).toContain('flex: 0 0 auto')
    expect(fixedHeader).toContain('padding: 0')
    expect(historyList).toContain('flex: 1')
    expect(historyList).toContain('min-width: 0')
    expect(historyList).toContain('min-height: 0')
    expect(historyList).toContain('overflow-x: hidden')
    expect(historyList).toContain('overflow-y: auto')
    expect(activeNewChat).toContain('color: var(--workspace-text)')
    expect(activeNewChat).toContain('background: var(--workspace-selected)')
    expect(historyStyles).toMatch(
      /\.chat-history--shell \.chat-history__header:not\(\.chat-history__header--collapsed\)::after\s*\{[^}]*top: 100%;[^}]*height: 6px;[^}]*background: var\(--workspace-sidebar-surface\);[^}]*box-shadow: 0 1px 0 var\(--workspace-footer-divider\);[^}]*opacity: 0;/s,
    )
    expect(historyStyles).toMatch(
      /@media \(min-height: 700px\)[\s\S]*?\.chat-history__header\[data-scrolled-from-top='true'\]:not\(\.chat-history__header--collapsed\)::after\s*\{[^}]*opacity: 1;/,
    )
    expect(primaryAction).toContain('width: 100%')
    expect(primaryAction).toContain('height: 36px')
    expect(primaryAction).toContain('flex: 0 0 36px')
    expect(primaryAction).toContain('gap: 6px')
    expect(primaryAction).toContain('border-radius: 10px')
    expect(primaryAction).toContain('padding: 6px 10px')
    expect(primaryAction).toContain('border: 0')
    expect(primaryAction).toContain('background: transparent')
    expect(primaryAction).toContain('color: var(--workspace-text)')
    expect(primaryAction).toContain('font-size: 14px')
    expect(primaryAction).toContain('font-weight: 400')
    expect(primaryAction).toContain('line-height: 20px')
    expect(primaryAction).not.toContain('accent')
    expect(historyItem).toContain('min-height: 36px')
    expect(historyItem).toContain('margin: 0 6px')
    expect(historyItem).toContain('border-radius: 10px')
    expect(historySelect).toContain('padding: 6px 10px')
    expect(historySelect).toContain('font-size: 14px')
    expect(historySelect).toContain('font-weight: 400')
    expect(historySelect).toContain('line-height: 20px')
    expect(historyTitle).toContain('font-size: inherit')
    expect(historyTitle).toContain('font-weight: inherit')
    expect(historyTitle).toContain('line-height: inherit')
    expect(historyTitle).toContain('text-overflow: ellipsis')
    expect(historyTitle).toContain('white-space: nowrap')
    expect(historySelectWithActions).toContain('padding-right: 62px')
    expect(historyMenu).toContain('position: fixed')
    expect(historyMenu).toContain('width: 144px')
    expect(historyMenu).toContain('border-radius: 20px')
    expect(historyMenu).toContain('padding: 10px 0')
    expect(historyMenu).toContain('background: var(--chat-history-menu-surface)')
    expect(historyMenu).toContain('box-shadow: var(--chat-history-menu-shadow)')
    expect(historyMenu).toContain('font-size: 14px')
    expect(historyMenu).toContain('font-weight: 400')
    expect(historyMenu).toContain('line-height: 20px')
    expect(historyMenuItem).toContain('width: 124px')
    expect(historyMenuItem).toContain('min-height: 36px')
    expect(historyMenuItem).toContain('gap: 6px')
    expect(historyMenuItem).toContain('margin: 0 10px')
    expect(historyMenuItem).toContain('border-radius: 12px')
    expect(historyMenuItem).toContain('padding: 6px 32px 6px 10px')
    expect(historyGroup).toContain('margin-right: 6px')
    expect(historyGroup).toContain('margin-left: -8px')
    expect(historyStyles).toContain('--chat-history-action-color: #8f8f8f')
    expect(historyStyles).toContain('--chat-history-menu-hover: rgb(0 0 0 / 0.04)')
    expect(historyStyles).toMatch(
      /:global\(html\.dark \.chat-history\)\s*\{[^}]*--chat-history-row-text: #ffffff;[^}]*--chat-history-action-color: #afafaf;[^}]*--chat-history-menu-surface: #353535;[^}]*--chat-history-menu-text: #ffffff;/s,
    )
    expect(historyGroupHeading).toContain('margin: 16px 8px 5px')
    expect(historyGroupHeading).toContain('color: var(--workspace-sidebar-group-label)')
    expect(historyGroupHeading).toContain('font-size: var(--workspace-sidebar-group-label-size)')
    expect(historyGroupHeading).toContain('font-weight: var(--workspace-sidebar-group-label-weight)')
    expect(historyGroupHeading).toContain(
      'line-height: var(--workspace-sidebar-group-label-line-height)',
    )
    expect(historyGroupHeading).toContain('letter-spacing: normal')
    expect(historySource).not.toContain('<Icon name="chatBubble"')
    expect(historySource).not.toContain('<small>{{ formatTime')
    expect(historySource).not.toContain('class="chat-history__clear"')
    expect(historySource).toContain('<WorkspaceSidebarFrame')
    expect(historySource).toContain('<WorkspaceSidebarHeader')
    expect(historySource).toContain(':collapsed="sidebarCollapsed"')
    expect(historySource).not.toContain(':compact="sidebarCollapsed"')
    expect(historySource).toContain('toggle-test-id="chat-sidebar-collapse-toggle"')
    expect(headerSource).toContain("toggleTestId: 'workspace-sidebar-collapse-toggle'")
    expect(historySource).toContain('data-testid="chat-sidebar-collapsed-nav"')
    expect(historySource).toContain('popover="manual"')
    expect(historySource).toContain('role="menu"')
    expect(historySource).toContain('role="menuitem"')
    expect(historySource).toContain('name="chatHistoryPinSmall"')
    for (const icon of [
      'chatHistoryShare',
      'chatHistoryRename',
      'chatHistoryPin',
      'chatHistoryArchive',
      'chatHistoryDelete',
    ]) {
      expect(historySource).toContain(`icon: '${icon}'`)
    }
    expect(historySource).toMatch(
      /v-show="!sidebarCollapsed"[\s\S]*?class="chat-history__list"/,
    )
    expect(historySource).toContain('aria-disabled="true"')
    expect(historySource).toContain('to="/library"')
    expect(historySource).toMatch(
      /id: 'library',[\s\S]*?path: '\/library',[\s\S]*?available: true/,
    )
    expect(historySource).toContain("path: '/projects'")
    expect(historySource).toContain('to="/projects"')
    expect(historySource).not.toContain('to="/scheduled"')
    expect(historySource).not.toContain('to="/plugins"')
    for (const unavailableSection of ['scheduled', 'plugins']) {
      expect(historySource).toMatch(new RegExp(
        `id: '${unavailableSection}',[\\s\\S]*?path: '',[\\s\\S]*?available: false`,
      ))
    }
    const navigationCopyIndexes = [
      'chat.actions.newChat',
      'chat.navigation.fileLibrary',
      'chat.navigation.projects',
      'chat.navigation.scheduled',
      'chat.navigation.plugins',
    ].map((label) => historySource.indexOf(label))
    expect(navigationCopyIndexes.every((index) => index >= 0)).toBe(true)
    expect([...navigationCopyIndexes].sort((left, right) => left - right))
      .toEqual(navigationCopyIndexes)
    expect(historySource).toContain('v-if="conversations.length > 0" class="chat-history__group"')
    expect(historySource).not.toContain("t('chat.history.previousDays')")
    expect(historySource).not.toContain("t('chat.history.older')")
  })

  it('keeps collapsed actions neutral, reachable, and motion-safe', () => {
    const collapsedAction = cssRule(historyStyles, '.chat-history__collapsed-nav-action')
    const collapsedNav = cssRule(historyStyles, '.chat-history__collapsed-nav')
    const collapsedNewConversation = cssRule(historyStyles, '.chat-history__new--collapsed')
    const newConversationLabel = cssRule(
      historyStyles,
      '.chat-history__new > span,\n.chat-history__primary-action > span',
    )

    expect(collapsedNav).toContain('align-items: flex-start')
    expect(collapsedNav).not.toContain('align-items: center')
    expect(collapsedAction).toContain('width: 44px')
    expect(collapsedAction).toContain('height: 44px')
    expect(collapsedAction).toContain('border-radius: var(--workspace-radius-button)')
    expect(collapsedAction).not.toContain('purple')
    expect(collapsedAction).not.toContain('accent')
    expect(collapsedNewConversation).toContain('width: var(--workspace-sidebar-touch-target)')
    expect(collapsedNewConversation).toContain('height: var(--workspace-sidebar-touch-target)')
    expect(collapsedNewConversation).toContain('justify-content: center')
    expect(newConversationLabel).toContain('max-width: 12rem')
    expect(newConversationLabel).toContain('opacity: 1')
    expect(historySource).toContain("'chat-history__new--collapsed': sidebarCollapsed")
    expect(historySource).toContain(':aria-hidden="sidebarCollapsed ? \'true\' : undefined"')
    expect(historyStyles).toContain('--lx-clay-accent: var(--workspace-text-secondary)')
    expect(historyStyles).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.chat-history__collapsed-nav-action,[\s\S]*?\.chat-history__new > span\s*\{[^}]*transition: none;/,
    )
    expect(modeSwitchStyles).toContain(
      ':global(.workspace-sidebar-frame--mobile) .app-mode-switch',
    )
    expect(modeSwitchStyles).toContain(
      'flex-basis: var(--workspace-mode-switch-height-mobile)',
    )
    expect(modeSwitchStyles).toContain(
      'min-height: var(--workspace-sidebar-touch-target)',
    )
    expect(workspaceTokens).toContain('--workspace-mode-switch-height-mobile: 60px;')
    expect(workspaceTokens).toContain('--workspace-radius-button: 10px;')
  })

  it('keeps the brand and real plan on one line while the shared Header owns the collapsed swap', () => {
    const copy = cssRule(brandStyles, '.workspace-sidebar-brand__copy')
    const brandTypography = cssRule(
      brandStyles,
      '.workspace-sidebar-brand__copy strong,\n.workspace-sidebar-brand__copy span',
    )

    expect(copy).toContain('display: flex')
    expect(copy).toContain('align-items: baseline')
    expect(copy).not.toContain('flex-direction: column')
    expect(brandTypography).toContain('font-size: 18px')
    expect(brandTypography).toContain('font-weight: 600')
    expect(brandTypography).toContain('letter-spacing: -0.27px')
    expect(brandTypography).toContain('line-height: 26px')
    expect(brandStyles).toMatch(
      /\.workspace-sidebar-brand__copy strong\s*\{[^}]*color: var\(--workspace-identity-text\);/s,
    )
    expect(brandStyles).toMatch(
      /\.workspace-sidebar-brand__copy span\s*\{[^}]*color: var\(--workspace-identity-text-tertiary\);/s,
    )
    expect(brandStyles).not.toContain('.workspace-sidebar-brand:hover')
    expect(brandStyles).not.toMatch(/#(?:0d0d0d|fff|ffffff|8f8f8f|afafaf)\b/i)
    expect(headerSource).toContain('class="workspace-sidebar-header__collapsed-brand-mark"')
    expect(headerStyles).toContain('.workspace-sidebar-header__toggle--collapsed:hover')
    expect(headerStyles).toContain('.workspace-sidebar-header__toggle--collapsed:focus-visible')
    expect(headerStyles).toContain('opacity 180ms cubic-bezier(0.16, 1, 0.3, 1)')
    expect(headerStyles).toContain('transform 180ms cubic-bezier(0.16, 1, 0.3, 1)')
  })

  it('places the minimal new-chat greeting at the 42svh composer baseline', () => {
    const messageFlow = cssRule(viewStyles, '.chat-messages__inner')
    const chatCanvas = cssRule(viewStyles, '.chat-workspace__main')
    const newChatFlow = cssRule(viewStyles, '.chat-conversation-flow--new-chat')
    const newChatMessages = cssRule(viewStyles, '.chat-conversation-flow--new-chat .chat-messages-region')
    const emptyState = cssRule(viewStyles, '.chat-conversation-flow--new-chat .chat-empty-state')
    const emptyTitle = cssRule(viewStyles, '.chat-empty-state h2')
    const mobileActions = cssRule(viewStyles, '.chat-mobile-actions')
    const mobileHistoryButton = cssRule(viewStyles, '.chat-mobile-actions__history-button')
    expect(messageFlow).toContain('gap: 28px')
    expect(messageFlow).toContain('padding-bottom: 16px')
    expect(chatCanvas).toContain('background: var(--workspace-canvas)')
    expect(viewStyles).not.toMatch(
      /\.chat-workspace__main--new-chat\s*\{[^}]*background:/s,
    )
    expect(newChatFlow).toContain('overflow-y: auto')
    expect(newChatFlow).toContain('400ms cubic-bezier(0, 0, 0.2, 1)')
    expect(newChatMessages).toContain('min-height: 232px')
    expect(newChatMessages).toContain('max(232px, 42svh)')
    expect(emptyState).toContain('justify-content: flex-end')
    expect(emptyState).toContain('padding: 24px 20px 22px')
    expect(emptyTitle).toContain('display: inline-flex')
    expect(emptyTitle).toContain('min-height: 42px')
    expect(emptyTitle).toContain('align-items: baseline')
    expect(emptyTitle).toContain('font-size: 24px')
    expect(emptyTitle).toContain('font-weight: 400')
    expect(emptyTitle).toContain('letter-spacing: normal')
    expect(emptyTitle).toContain('line-height: 28px')
    expect(emptyTitle).not.toContain('transform:')
    expect(emptyTitle).toContain('color: var(--workspace-text)')
    expect(mobileActions).toContain('display: none')
    expect(mobileHistoryButton).toContain('width: 44px')
    expect(mobileHistoryButton).toContain('height: 44px')
    expect(viewSource).not.toContain('<header class="chat-toolbar">')
    expect(viewSource).not.toContain('chat-toolbar__session')
    expect(viewSource).toContain('class="chat-mobile-actions__history-button"')
    expect(viewStyles).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)[\s\S]*?\.chat-mobile-actions\s*\{[^}]*display: flex;[^}]*height: 56px;/,
    )
    expect(viewStyles).toMatch(
      /@media \(max-width: 767px\) and \(hover: none\) and \(pointer: coarse\)[\s\S]*?\.chat-conversation-flow--new-chat \.chat-messages-region\s*\{[^}]*calc\(42svh - 56px\)/,
    )
    expect(viewStyles).toMatch(
      /@media \(max-width: 700px\)[\s\S]*?\.chat-conversation-flow--new-chat \.chat-empty-state\s*\{[^}]*padding: 20px 16px 22px;/,
    )
    expect(viewStyles).not.toContain('.chat-new-chat-shortcuts')
    expect(viewStyles).not.toContain('.chat-empty-state__icon')
    expect(viewStyles).not.toMatch(/:global\(html\.dark\)\s/)
    expect(viewStyles).not.toContain(':global(html.dark .chat-workspace__main--new-chat)')
    expect(viewStyles).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.chat-conversation-flow--new-chat\s*\{[^}]*animation: none;/,
    )
  })

  it('matches the ChatGPT message measure, user bubble and rich-text rhythm', () => {
    const messageViewport = cssRule(viewStyles, '.chat-messages')
    const message = cssRule(messageStyles, '.chat-message')
    const assistant = cssRule(messageStyles, '.chat-message--assistant')
    const userBody = cssRule(messageStyles, '.chat-message--user .chat-message__body')
    const userGroup = cssRule(messageStyles, '.user-message-group')
    const userActions = cssRule(messageStyles, '.user-message-group > .chat-message__actions')
    const userBubble = cssRule(messageStyles, '.chat-message--user .chat-message__plain')
    const markdown = cssRule(messageStyles, '.chat-message__markdown')
    const heading = cssRule(messageStyles, '.chat-message__markdown :deep(h2)')
    const listSpacing = cssRule(
      messageStyles,
      '.chat-message__markdown :deep(ul),\n.chat-message__markdown :deep(ol)',
    )
    const unorderedList = cssRule(messageStyles, '.chat-message__markdown :deep(ul)')
    const listItem = cssRule(messageStyles, '.chat-message__markdown :deep(li)')
    const codeBlock = cssRule(messageStyles, '.chat-message__markdown :deep(pre)')
    const quote = cssRule(messageStyles, '.chat-message__markdown :deep(blockquote)')
    const divider = cssRule(messageStyles, '.chat-message__markdown :deep(hr)')
    const table = cssRule(messageStyles, '.chat-message__markdown :deep(table)')
    const streamingPlaceholder = cssRule(messageStyles, '.chat-message__streaming-placeholder')
    const streamingDot = cssRule(messageStyles, '.chat-message__streaming-dot')

    expect(messageViewport).toContain('padding: 52px 24px 10px')
    expect(message).toContain('width: min(100%, 768px)')
    expect(message).toContain('min-width: 0')
    expect(message).toContain('font-size: 16px')
    expect(message).toContain('font-weight: 400')
    expect(message).toContain('line-height: 26px')
    expect(message).not.toContain('font-family')
    expect(assistant).toContain('border: 0')
    expect(assistant).toContain('background: transparent')
    expect(assistant).toContain('box-shadow: none')
    expect(assistant).not.toContain('accent')
    expect(userBody).toContain('align-items: flex-end')
    expect(userGroup).toContain('display: flex')
    expect(userGroup).toContain('width: 100%')
    expect(userGroup).toContain('flex-direction: column')
    expect(userGroup).toContain('align-items: flex-end')
    expect(userGroup).toContain('margin-left: auto')
    expect(userActions).toContain('align-self: flex-end')
    expect(userActions).toContain('margin-left: auto')
    expect(userBubble).toContain('width: fit-content')
    expect(userBubble).toContain('max-width: min(70%, 640px)')
    expect(userBubble).toContain('margin-left: auto')
    expect(userBubble).toContain('border-radius: 22px')
    expect(userBubble).toContain('padding: 10px 16px')
    expect(userBubble).toContain('var(--chat-message-user-surface)')
    expect(userBubble).toContain('line-height: 24px')
    expect(markdown).toContain('font-size: 16px')
    expect(markdown).toContain('line-height: 26px')
    expect(heading).toContain('font-size: 20px')
    expect(heading).toContain('line-height: 28px')
    expect(heading).toContain('margin: 28px 0 8px')
    expect(listSpacing).toContain('margin: 16px 0 8px')
    expect(unorderedList).toContain('list-style: disc')
    expect(listItem).toContain('padding-left: 6px')
    expect(codeBlock).toContain('border-radius: 24px')
    expect(codeBlock).toContain('margin: 8px 0 16px')
    expect(codeBlock).toContain('var(--chat-message-code-surface)')
    expect(quote).toContain('padding: 8px 0 8px 24px')
    expect(quote).toContain('line-height: 24px')
    expect(quote).toContain('background: transparent')
    expect(divider).toContain('margin: 28px 0')
    expect(divider).toContain('var(--chat-message-divider)')
    expect(table).toContain('font-size: 14px')
    expect(table).toContain('line-height: 24px')
    expect(messageStyles).toMatch(
      /\.chat-message__markdown :deep\(table\)\s*\{[^}]*overflow-x: auto/,
    )
    expect(messageStyles).toContain('--chat-message-user-surface: #ececec')
    expect(messageStyles).toContain('--chat-message-user-text: #000000')
    expect(messageStyles).toContain('--chat-message-code-surface: #212121')
    expect(messageStyles).toContain(':global(html.dark .chat-message)')
    expect(messageStyles).not.toContain(':global(html.dark) .chat-message')
    expect(messageStyles).not.toContain('chat-message__avatar')
    expect(streamingPlaceholder).toContain('justify-content: flex-start')
    expect(streamingPlaceholder).toContain('height: 26px')
    expect(streamingPlaceholder).toContain('margin: 0')
    expect(streamingPlaceholder).toContain('padding: 0')
    expect(streamingDot).toContain('width: 8px')
    expect(streamingDot).toContain('height: 8px')
    expect(streamingDot).toContain('border-radius: 50%')
    expect(streamingDot).toContain('var(--chat-message-streaming-dot)')
    expect(streamingDot).toContain('chat-streaming-breathe 1.2s ease-in-out infinite')
    expect(cssRule(messageStyles, '.chat-message__markdown--streaming')).toContain(
      'chat-streaming-content-enter 180ms ease-out both',
    )
    expect(messageStyles).toContain('--chat-message-streaming-dot: #0d0d0d')
    expect(messageStyles).toContain('--chat-message-streaming-dot: #ffffff')
    expect(messageStyles).toMatch(
      /@keyframes chat-streaming-breathe\s*\{[\s\S]*?opacity: 0\.35; transform: scale\(0\.72\);[\s\S]*?opacity: 1; transform: scale\(1\);/,
    )
    expect(messageStyles).not.toContain('chat-thinking')
    expect(messageStyles).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.chat-message__markdown--streaming,[\s\S]*?\.chat-message__streaming-dot\s*\{[^}]*animation: none;/,
    )
    expect(messageStyles).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.chat-message__streaming-dot\s*\{[^}]*opacity: 1;[^}]*transform: none;/,
    )
  })

  it('renders sent images as intrinsic, right-aligned previews while retaining document cards', () => {
    const attachmentGroup = cssRule(messageAttachmentStyles, '.chat-message-attachments')
    const attachmentSpacing = cssRule(
      messageAttachmentStyles,
      '.chat-message-attachments--with-content',
    )
    const imageLists = cssRule(
      messageAttachmentStyles,
      '.chat-message-attachments__images,\n.chat-message-attachments__documents',
    )
    const imageList = cssRule(messageAttachmentStyles, '.chat-message-attachments__images')
    const imageItem = cssRule(messageAttachmentStyles, '.chat-message-attachments__image-item')
    const imageTrigger = cssRule(messageAttachmentStyles, '.chat-message-attachments__image-trigger')
    const image = cssRule(
      messageAttachmentStyles,
      '.chat-message-attachments__image-trigger img',
    )
    const documentCard = cssRule(messageAttachmentStyles, '.chat-message-attachments__item')

    expect(attachmentGroup).toContain('width: fit-content')
    expect(attachmentGroup).toContain('max-width: 100%')
    expect(attachmentGroup).toContain('align-items: flex-end')
    expect(attachmentGroup).toContain('margin: 0 0 0 auto')
    expect(attachmentSpacing).toContain('margin-bottom: 10px')
    expect(imageLists).toContain('flex-wrap: wrap')
    expect(imageLists).toContain('justify-content: flex-end')
    expect(imageList).toContain('width: fit-content')
    expect(imageList).toContain('max-width: min(760px, 72vw)')
    expect(imageItem).toContain('width: fit-content')
    expect(imageItem).toContain('margin-left: auto')
    expect(imageTrigger).toContain('width: fit-content')
    expect(imageTrigger).toContain('max-width: 100%')
    expect(imageTrigger).toContain('border: 0')
    expect(imageTrigger).toContain('border-radius: 28px')
    expect(imageTrigger).toContain('background: transparent')
    expect(imageTrigger).toContain('box-shadow: none')
    expect(image).toContain('display: block')
    expect(image).toContain('width: auto')
    expect(image).toContain('height: auto')
    expect(image).toContain('max-width: 100%')
    expect(image).toContain('max-height: 560px')
    expect(image).toContain('object-fit: contain')
    expect(image).not.toContain('aspect-ratio')
    expect(image).not.toContain('object-fit: cover')
    expect(documentCard).toContain('border: 1px solid var(--workspace-border)')
    expect(documentCard).toContain('background: var(--workspace-surface)')
    expect(messageAttachmentSource).not.toContain('chat-message-attachments__image-name')
    expect(messageAttachmentStyles).toMatch(
      /@media \(max-width: 640px\)[\s\S]*?\.chat-message-attachments__images--multiple\s*\{[^}]*width: 100%;/,
    )
    expect(messageAttachmentStyles).toMatch(
      /@media \(max-width: 640px\)[\s\S]*?\.chat-message-attachments__item\s*\{[^}]*width: 100%;[^}]*max-width: 100%;/,
    )
  })

  it('keeps the add-content menu aligned to the 768px composer contract', () => {
    const popover = cssRule(attachmentStyles, '.chat-attachment-menu-popover')
    const row = cssRule(attachmentStyles, '.chat-attachment-menu__item')
    const copy = cssRule(attachmentStyles, '.chat-attachment-menu__copy')
    const label = cssRule(attachmentStyles, '.chat-attachment-menu__label')
    const description = cssRule(attachmentStyles, '.chat-attachment-menu__description')
    const search = cssRule(attachmentStyles, '.chat-attachment-menu__search input')

    expect(attachmentSource).toContain('<Teleport to="body">')
    expect(attachmentSource).toContain('role="menu"')
    expect(attachmentSource).toContain('role="menuitem"')
    expect(attachmentSource).toContain('aria-haspopup="menu"')
    expect(attachmentSource).toContain("trigger.closest<HTMLElement>('.chat-composer')")
    expect(attachmentSource).toContain('const triggerGap = 16')
    expect(popover).toContain('grid-template-rows: minmax(0, 1fr) 36px')
    expect(popover).toContain('border-radius: 20px')
    expect(popover).toContain('padding: 10px 10px 2px')
    expect(popover).toContain('0 0 0 1px rgb(0 0 0 / 4%)')
    expect(popover).toContain('0 2px 8px rgb(0 0 0 / 4%)')
    expect(popover).toContain('0 4px 80px 8px rgb(0 0 0 / 2.4%)')
    expect(popover).not.toContain('font-family')
    expect(row).toContain('grid-template-columns: 20px minmax(0, 1fr)')
    expect(row).toContain('min-height: 36px')
    expect(row).toContain('border-radius: 12px')
    expect(row).toContain('padding: 6px 10px')
    expect(copy).toContain('gap: 12px')
    expect(label).toContain('font-size: 14px')
    expect(label).toContain('font-weight: 400')
    expect(label).toContain('letter-spacing: normal')
    expect(description).toContain('font-size: 14px')
    expect(description).toContain('font-weight: 400')
    expect(description).toContain('letter-spacing: normal')
    expect(search).toContain('height: 36px')
    expect(search).toContain('font-size: 14px')
    expect(attachmentSource).toContain('data-chat-tool-icon="paperclip"')
    expect(attachmentSource).toContain('data-chat-tool-icon="library"')
    expect(attachmentSource).toContain('data-chat-tool-icon="create-image-plugin"')
    expect(attachmentSource).toContain('data-chat-tool-icon="skill-globe-light"')
    expect(attachmentSource).toContain('data-chat-tool-icon="skill-deep-research-light"')
    expect(attachmentStyles).toContain('border-radius: 4px')
    expect(attachmentStyles).toContain('inset 0 0 0 0.5px rgb(0 0 0 / 10%)')
    expect(attachmentStyles).toContain('0 4px 16px rgb(0 0 0 / 5%)')
    expect(attachmentStyles).toContain('.chat-attachment-menu__status--persistent')
    expect(attachmentStyles).toContain('--chat-tool-menu-surface: #212121')
    expect(attachmentStyles).toContain('--chat-tool-menu-text: #ffffff')
    expect(attachmentStyles).toContain('--chat-tool-menu-description: #cdcdcd')
    expect(attachmentStyles).toContain('--chat-tool-menu-muted: #afafaf')
    expect(attachmentStyles).toContain('--chat-tool-menu-hover: rgb(255 255 255 / 10%)')
    expect(attachmentStyles).toContain('inset 0 0 1px rgb(255 255 255 / 20%)')
    expect(attachmentStyles).toMatch(
      /@media \(prefers-reduced-motion: reduce\)[\s\S]*?\.chat-attachment-menu-enter-active,[\s\S]*?transition: none;/,
    )
  })
})
