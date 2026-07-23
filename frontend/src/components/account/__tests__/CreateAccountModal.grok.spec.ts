import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf8'
)
const accountTypePanelSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/create/panels/GrokAccountTypePanel.vue'),
  'utf8'
)
const oauthOptionsPanelSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/create/panels/GrokOAuthOptionsPanel.vue'),
  'utf8'
)
const credentialBuilderSource = readFileSync(
  resolve(process.cwd(), 'src/components/account/create/credentialDraftBuilders.ts'),
  'utf8'
)
const credentialSettingsPanelSource = readFileSync(
  resolve(
    process.cwd(),
    'src/components/account/create/panels/PlatformCredentialSettingsPanel.vue',
  ),
  'utf8',
)
const grokControllerSource = readFileSync(
  resolve(
    process.cwd(),
    'src/components/account/create/useGrokCreateAccountController.ts',
  ),
  'utf8',
)

describe('CreateAccountModal Grok account types', () => {
  it('offers API-key setup alongside OAuth with the official xAI default', () => {
    expect(accountTypePanelSource).toContain('data-testid="grok-account-type-api-key"')
    expect(accountTypePanelSource).toContain("selectCategory('apikey')")
    expect(source).toContain('apiKeyBaseUrl.value = defaultAPIKeyBaseURL(newPlatform)')
    expect(credentialBuilderSource).toContain("grok: 'https://api.x.ai/v1'")
    expect(source).toContain("form.platform === 'grok'")
    expect(credentialSettingsPanelSource).toContain("? 'xai-...'")
  })

  it('exposes custom upstream URL and header override for the OAuth create flow', () => {
    expect(oauthOptionsPanelSource).toContain('data-testid="grok-custom-base-url-toggle"')
    expect(oauthOptionsPanelSource).toContain('data-testid="grok-custom-base-url-input"')
    expect(source).toContain('form.platform === \'grok\' && isOAuthFlow')
  })

  it('validates and applies upstream config on all three Grok OAuth create paths', () => {
    // 授权码兑换 / RT 批量 / SSO 批量 3 处调用（定义为箭头函数，不计入）
    expect(grokControllerSource.match(/validateUpstreamConfig\(\)/g)?.length).toBe(3)
    expect(grokControllerSource.match(/applyUpstreamConfig\(credentials\)/g)?.length).toBe(3)
    expect(source).toContain('useGrokCreateAccountController({')
  })
})
