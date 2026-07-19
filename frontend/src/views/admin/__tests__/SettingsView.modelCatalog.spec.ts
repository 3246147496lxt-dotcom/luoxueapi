import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const testDir = dirname(fileURLToPath(import.meta.url))
const viewSource = readFileSync(resolve(testDir, '../SettingsView.vue'), 'utf8')
const settingsApiSource = readFileSync(resolve(testDir, '../../../api/admin/settings.ts'), 'utf8')

describe('public model catalog setting', () => {
  it('is fail-closed in the form and included in the settings payload', () => {
    expect(viewSource).toContain('public_model_catalog_enabled: false')
    expect(viewSource).toContain('public_model_catalog_enabled: form.public_model_catalog_enabled')
    expect(viewSource).toContain('v-model="form.public_model_catalog_enabled"')
  })

  it('is represented in both read and update settings contracts', () => {
    expect(settingsApiSource.match(/public_model_catalog_enabled/g)?.length).toBeGreaterThanOrEqual(2)
  })
})
