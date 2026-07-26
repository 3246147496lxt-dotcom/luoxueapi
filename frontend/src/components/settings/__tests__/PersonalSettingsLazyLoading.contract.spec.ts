import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const directory = dirname(fileURLToPath(import.meta.url))
const layoutSource = readFileSync(
  resolve(directory, '../../layout/AppLayout.vue'),
  'utf8',
)
const dialogSource = readFileSync(
  resolve(directory, '../PersonalSettingsDialog.vue'),
  'utf8',
)

describe('personal settings lazy-loading boundary', () => {
  it('keeps the dialog out of the authenticated shell until settings are requested', () => {
    expect(layoutSource).toContain('defineAsyncComponent')
    expect(layoutSource).toContain(
      "() => import('@/components/settings/PersonalSettingsDialog.vue')",
    )
    expect(layoutSource).toContain(
      '<PersonalSettingsDialog v-if="settingsDialogRequested" />',
    )
    expect(layoutSource).not.toContain(
      "import PersonalSettingsDialog from '@/components/settings/PersonalSettingsDialog.vue'",
    )
  })

  it('loads every settings category from its own first-visit boundary', () => {
    const dynamicImports = [
      './PersonalSettingsGeneralPanel.vue',
      './PersonalSettingsAccountPanel.vue',
      './PersonalSettingsSecurityPanel.vue',
      './PersonalSettingsNotificationsPanel.vue',
      '@/components/layout/WalletSubscriptionSettingsPanel.vue',
    ]

    for (const componentPath of dynamicImports) {
      expect(dialogSource).toContain(`() => import('${componentPath}')`)
      expect(dialogSource).not.toContain(`from '${componentPath}'`)
    }
  })
})
