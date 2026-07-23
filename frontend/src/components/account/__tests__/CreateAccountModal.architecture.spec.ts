import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const readSource = (relativePath: string) =>
  readFileSync(resolve(process.cwd(), relativePath), 'utf8')

const modalSource = readSource(
  'src/components/account/CreateAccountModal.vue',
)
const modalImportSources = Array.from(
  modalSource.matchAll(/\bfrom\s+['"]([^'"]+)['"]/g),
  ([, source]) => source,
)
const oauthDriversSource = readSource(
  'src/components/account/create/useCreateAccountOAuthDrivers.ts',
)
const flowControllerSource = readSource(
  'src/components/account/create/useCreateAccountFlowController.ts',
)
const openAIControllerSource = readSource(
  'src/components/account/create/useOpenAICreateAccountController.ts',
)
const anthropicControllerSource = readSource(
  'src/components/account/create/useAnthropicCreateAccountController.ts',
)
const grokControllerSource = readSource(
  'src/components/account/create/useGrokCreateAccountController.ts',
)
const geminiControllerSource = readSource(
  'src/components/account/create/useGeminiCreateAccountController.ts',
)
const antigravityControllerSource = readSource(
  'src/components/account/create/useAntigravityCreateAccountController.ts',
)

describe('CreateAccountModal architecture boundaries', () => {
  it('keeps the modal as a bounded wizard orchestration shell', () => {
    const lineCount = modalSource.split('\n').length
    const importCount = modalSource.match(/^import\s/mg)?.length ?? 0

    expect(lineCount).toBeLessThanOrEqual(2200)
    expect(importCount).toBeLessThanOrEqual(45)
    expect(modalSource).toContain('<PlatformCredentialSettingsPanel')
    expect(modalSource).toContain('<TempUnschedulablePanel')
    expect(modalSource).toContain('<AccountRuntimeSettingsPanel')
    expect(modalSource).toContain('<AccountBehaviorSettingsPanel')
    expect(modalSource).toContain('id="create-account-form"')
    expect(modalSource).toContain('@submit.prevent="handleSubmit"')
  })

  it('keeps the existing public props and emitted events', () => {
    expect(modalSource).toMatch(
      /interface Props\s*\{\s*show:\s*boolean\s*proxies:\s*Proxy\[\]\s*groups:\s*AdminGroup\[\]\s*\}/,
    )
    expect(modalSource).toMatch(
      /defineEmits<\{\s*close:\s*\[\]\s*created:\s*\[\]\s*\}>\(\)/,
    )
    expect(modalSource).toContain("notifyCreated: () => emit('created')")
    expect(modalSource).toContain("emit('close')")
  })

  it('does not reach through the shell into APIs, services, or platform OAuth implementations', () => {
    expect(
      modalImportSources.some(
        (source) => source === '@/api' || source.startsWith('@/api/'),
      ),
    ).toBe(false)
    expect(
      modalImportSources.some(
        (source) =>
          source === '@/services' || source.startsWith('@/services/'),
      ),
    ).toBe(false)
    expect(
      modalImportSources.some((source) =>
        /^@\/composables\/use(?:Account|OpenAI|Gemini|Antigravity|Grok)OAuth$/.test(
          source,
        ),
      ),
    ).toBe(false)
    expect(modalSource).not.toMatch(/\badminAPI\b/)
    expect(modalSource).not.toContain('new OAuthDriverRegistry(')
    expect(modalSource).not.toContain('adminAPI.accounts.exchangeCode(')
    expect(modalSource).not.toContain('adminAPI.accounts.importCodexSession(')
    expect(modalSource).not.toContain('adminAPI.accounts.createOpenAICodexPAT(')
  })

  it('delegates every OAuth command and shared state projection to one registry', () => {
    expect(modalSource).toContain('useCreateAccountOAuthDrivers({')
    expect(modalSource).not.toContain('switch (form.platform)')
    expect(modalSource).toContain('await exchangeCurrentOAuthCode(authCode)')
    expect(modalSource).toContain('validateCurrentOAuthRefreshToken(rt)')
    expect(oauthDriversSource).toContain('new OAuthDriverRegistry({')
    expect(oauthDriversSource).toContain(
      'registry.exchangeAuthorizationCode(options.platform(), code)',
    )
    expect(oauthDriversSource).toContain(
      'registry.validateRefreshToken(options.platform(), refreshToken)',
    )
  })

  it('delegates platform credential construction to tested builders', () => {
    expect(modalSource).toContain('buildBedrockCredentials({')
    expect(modalSource).toContain('buildAntigravityUpstreamCredentials({')
    expect(modalSource).toContain('buildVertexServiceAccountCredentials({')
    expect(modalSource).toContain('buildAPIKeyCredentials({')
    expect(modalSource).toContain('buildCurrentAnthropicOAuthExtra')
    expect(modalSource).toContain('useAnthropicOAuthControlDraft()')
    expect(modalSource).toContain('anthropicOAuthControls.reset()')
    expect(modalSource).toContain('buildAPIKeyQuotaExtra(extra, {')
    expect(modalSource).toContain('resolveCreateAccountType({')
    expect(modalSource).toContain('usesCreateAccountOAuthFlow({')
    expect(modalSource).toContain('resolveGeminiSelectedTier({')
  })

  it('keeps generic create and mixed-channel effects behind the flow controller', () => {
    expect(modalSource).toContain('useCreateAccountFlowController({')
    expect(modalSource).toContain(
      'createAccount: doCreateAccount',
    )
    expect(flowControllerSource).toContain(
      'adminAPI.accounts.checkMixedChannelRisk({',
    )
    expect(flowControllerSource).toContain(
      'adminAPI.accounts.create(withConfirmFlag(payload))',
    )
    expect(flowControllerSource).toContain(
      'adminAPI.tlsFingerprintProfiles.list()',
    )
  })

  it('keeps OpenAI and Anthropic bulk/exchange effects behind platform controllers', () => {
    expect(modalSource).toContain('useOpenAICreateAccountController({')
    expect(modalSource).toContain('useAnthropicCreateAccountController({')
    expect(openAIControllerSource).toContain(
      'adminAPI.accounts.importCodexSession(',
    )
    expect(openAIControllerSource).toContain(
      'adminAPI.accounts.createOpenAICodexPAT(',
    )
    expect(openAIControllerSource).toContain(
      'handleBatchRefreshTokens(refreshToken, OPENAI_MOBILE_RT_CLIENT_ID)',
    )
    expect(anthropicControllerSource).toContain(
      'adminAPI.accounts.exchangeCode(endpoint, {',
    )
    expect(anthropicControllerSource).toContain(
      'for (let index = 0; index < keys.length; index++)',
    )
  })

  it('keeps Grok OAuth effects behind one platform controller', () => {
    expect(modalSource).toContain('useGrokCreateAccountController({')
    expect(modalSource).toContain(
      'handleExchangeAuthorizationCode: handleGrokExchange',
    )
    expect(modalSource).not.toContain('adminAPI.grok.')
    expect(modalSource).not.toContain('grokOAuth.exchangeAuthCode(')
    expect(modalSource).not.toContain('grokOAuth.validateRefreshToken(')
    expect(grokControllerSource).toContain('adminAPI.grok.createFromSSO(')
    expect(grokControllerSource).toContain('options.oauth.exchangeAuthCode({')
    expect(grokControllerSource).toContain(
      'options.oauth.validateRefreshToken(',
    )
    expect(grokControllerSource).toContain('buildGrokSSOImportRequest(')
  })

  it('keeps Gemini and Antigravity OAuth effects behind platform controllers', () => {
    expect(modalSource).toContain('useGeminiCreateAccountController({')
    expect(modalSource).toContain('useAntigravityCreateAccountController({')
    expect(modalSource).not.toContain('geminiOAuth.getCapabilities(')
    expect(modalSource).not.toContain('geminiOAuth.exchangeAuthCode(')
    expect(modalSource).not.toContain('antigravityOAuth.exchangeAuthCode(')
    expect(modalSource).not.toContain(
      'antigravityOAuth.validateRefreshToken(',
    )
    expect(geminiControllerSource).toContain(
      'const capabilities = await options.oauth.getCapabilities()',
    )
    expect(geminiControllerSource).toContain(
      'const tokenInfo = await options.oauth.exchangeAuthCode({',
    )
    expect(antigravityControllerSource).toContain(
      'const tokenInfo = await options.oauth.validateRefreshToken(',
    )
    expect(antigravityControllerSource).toContain(
      'const tokenInfo = await options.oauth.exchangeAuthCode({',
    )
  })
})
