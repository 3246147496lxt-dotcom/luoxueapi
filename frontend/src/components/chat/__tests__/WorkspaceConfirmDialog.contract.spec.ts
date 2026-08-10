import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const dialogSource = readFileSync(resolve(directory, '../../common/BaseDialog.vue'), 'utf8')
const historySource = readFileSync(resolve(directory, '../ChatHistoryPanel.vue'), 'utf8')
const chatViewSource = readFileSync(resolve(directory, '../../../views/user/ChatView.vue'), 'utf8')
const tokenSource = readFileSync(resolve(directory, '../../../styles/luoxue-clay-tokens.css'), 'utf8')

describe('Workspace delete confirmation contract', () => {
  it('locks the measured ChatGPT dialog geometry without changing the default modal', () => {
    expect(dialogSource).toContain("type DialogVariant = 'default' | 'workspace-confirm'")
    expect(dialogSource).toContain("variant: 'default'")
    expect(dialogSource).toContain("'modal-content--workspace-confirm'")
    expect(dialogSource).toMatch(/\.modal-content--workspace-confirm\s*\{[\s\S]*?border-radius: var\(--workspace-radius-card\);[\s\S]*?box-shadow: var\(--workspace-confirm-shadow\);/)
    expect(dialogSource).toMatch(/\.modal-content--workspace-confirm \.modal-header\s*\{[^}]*height: 52px;[^}]*padding: 10px 10px 10px var\(--workspace-space-4\);[^}]*border: 0;/s)
    expect(dialogSource).toMatch(/\.modal-content--workspace-confirm \.modal-title\s*\{[^}]*font-size: 18px;[^}]*font-weight: var\(--workspace-type-body-weight\);[^}]*line-height: 28px;/s)
    expect(dialogSource).toMatch(/\.modal-content--workspace-confirm \.modal-footer\s*\{[^}]*min-height: 68px;[^}]*padding: var\(--workspace-space-4\);[^}]*gap: var\(--workspace-space-3\);[^}]*border: 0;/s)
  })

  it('uses the measured light and dark palette through shared Workspace tokens', () => {
    for (const declaration of [
      '--workspace-light-confirm-surface: #ffffff;',
      '--workspace-light-confirm-text: #0d0d0d;',
      '--workspace-light-confirm-text-secondary: #8f8f8f;',
      '--workspace-light-confirm-cancel-border: rgb(0 0 0 / 0.15);',
      '--workspace-light-confirm-danger: #ff002a;',
      '--workspace-light-confirm-backdrop: rgb(227 227 227 / 0.5);',
      '--workspace-dark-confirm-surface: #212121;',
      '--workspace-dark-confirm-text: #ffffff;',
      '--workspace-dark-confirm-text-secondary: #afafaf;',
      '--workspace-dark-confirm-cancel-border: rgb(255 255 255 / 0.15);',
      '--workspace-dark-confirm-cancel-hover: #424242;',
      '--workspace-dark-confirm-backdrop: rgb(0 0 0 / 0.5);',
    ]) {
      expect(tokenSource).toContain(declaration)
    }
    expect(tokenSource).toContain('--workspace-confirm-surface: var(--workspace-light-confirm-surface);')
    expect(tokenSource).toContain('--workspace-confirm-surface: var(--workspace-dark-confirm-surface);')
  })

  it('keeps memory as display copy while preserving the existing delete action', () => {
    expect(chatViewSource).toContain(":variant=\"confirmation?.kind === 'delete' ? 'workspace-confirm' : 'default'\"")
    expect(chatViewSource).toContain(':show-close-button="confirmation?.kind !== \'delete\'"')
    expect(chatViewSource).toContain('class="chat-delete-confirm__settings-word"')
    expect(chatViewSource).not.toMatch(/chat-delete-confirm__settings-word[^>]*(?:@click|href|to=)/)
    expect(chatViewSource).toContain('chatStore.deleteConversation(action.conversationId)')
    expect(chatViewSource).toMatch(/\.chat-delete-confirm__button\s*\{[^}]*height: 36px;[^}]*padding: 0 var\(--workspace-space-3\);[^}]*border-radius: var\(--workspace-radius-pill\);[^}]*font-weight: var\(--workspace-type-navigation-weight\);/s)
  })

  it('uses one compact title presentation in the Sidebar and delete confirmation', () => {
    expect(historySource).toContain('toChatConversationTitlePreview(conversation.title)')
    expect(historySource).toContain(':aria-label="conversation.title"')
    expect(chatViewSource).toContain('displayTitle: toChatConversationTitlePreview(title)')
    expect(chatViewSource).toContain(':aria-label="confirmation.title"')
    expect(chatViewSource).toContain('chatStore.deleteConversation(action.conversationId)')
  })
})
