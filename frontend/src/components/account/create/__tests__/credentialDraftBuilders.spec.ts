import { describe, expect, it } from 'vitest'
import {
  buildAccountCreatePayload,
  toAccountCredentialDraft,
  type AccountBaseDraft,
} from '../accountDraft'
import {
  buildAPIKeyCredentials,
  buildAPIKeyQuotaExtra,
  buildAntigravityExtra,
  buildAntigravityUpstreamCredentials,
  buildAnthropicAPIKeyExtra,
  buildAnthropicOAuthExtra,
  buildBedrockCredentials,
  buildOpenAIExtra,
  buildTempUnschedulableRules,
  buildVertexServiceAccountCredentials,
  DEEPSEEK_BASE_URL_PRESETS,
  ZHIPU_BASE_URL_PRESETS,
  defaultAPIKeyBaseURL,
  defaultZhipuBaseURL,
  inferZhipuRoutingFromBaseURL,
  isManagedZhipuBaseURL,
  zhipuBaseURLForRouting,
  normalizeZhipuAccountMode,
  normalizeZhipuApiProtocol,
  normalizePoolModeRetryCount,
  parsePoolModeRetryStatusCodes,
  parseVertexServiceAccountJSON,
} from '../credentialDraftBuilders'
import { buildZhipuAccountPayload } from '../platforms/zhipu'

const base: AccountBaseDraft = {
  name: 'Account',
  notes: '',
  proxy_id: null,
  concurrency: 10,
  load_factor: null,
  priority: 1,
  rate_multiplier: 1,
  group_ids: [3],
  expires_at: null,
  auto_pause_on_expired: true,
}

describe('create-account credential draft builders', () => {
  it('owns the default API-key endpoints for every platform', () => {
    expect(defaultAPIKeyBaseURL('anthropic')).toBe('https://api.anthropic.com')
    expect(defaultAPIKeyBaseURL('openai')).toBe('https://api.openai.com')
    expect(defaultAPIKeyBaseURL('gemini')).toBe(
      'https://generativelanguage.googleapis.com',
    )
    expect(defaultAPIKeyBaseURL('grok')).toBe('https://api.x.ai/v1')
    expect(defaultAPIKeyBaseURL('deepseek')).toBe('https://api.deepseek.com')
    expect(DEEPSEEK_BASE_URL_PRESETS).toEqual([
      { label: 'DeepSeek API', url: 'https://api.deepseek.com' },
    ])
    expect(ZHIPU_BASE_URL_PRESETS).toEqual([
      { label: 'GLM API（按量）', url: 'https://open.bigmodel.cn/api/paas/v4' },
      { label: 'GLM Coding Plan', url: 'https://open.bigmodel.cn/api/coding/paas/v4' },
      { label: 'GLM Anthropic', url: 'https://open.bigmodel.cn/api/anthropic' },
    ])
    expect(defaultZhipuBaseURL('payg', 'chat_completions')).toBe(
      'https://open.bigmodel.cn/api/paas/v4',
    )
    expect(defaultZhipuBaseURL('coding', 'chat_completions')).toBe(
      'https://open.bigmodel.cn/api/coding/paas/v4',
    )
    expect(defaultZhipuBaseURL('coding', 'anthropic')).toBe(
      'https://open.bigmodel.cn/api/anthropic',
    )
  })

  it('infers GLM mode/protocol only from exact official endpoints', () => {
    expect(inferZhipuRoutingFromBaseURL('https://open.bigmodel.cn/api/paas/v4/')).toEqual({
      mode: 'payg',
      protocol: 'chat_completions',
    })
    expect(inferZhipuRoutingFromBaseURL('https://api.z.ai/api/coding/paas/v4')).toEqual({
      mode: 'coding',
      protocol: 'chat_completions',
    })
    // Anthropic is shared by payg and Coding Plan, so mode is intentionally
    // left unspecified for callers to retain their explicit selection.
    expect(inferZhipuRoutingFromBaseURL('https://open.bigmodel.cn/api/anthropic')).toEqual({
      protocol: 'anthropic',
    })
    expect(inferZhipuRoutingFromBaseURL('https://relay.example/api/coding/paas/v4')).toBeNull()
    expect(inferZhipuRoutingFromBaseURL('https://evil.example/?next=api.z.ai')).toBeNull()
    expect(inferZhipuRoutingFromBaseURL('https://open.bigmodel.cn:8443/api/paas/v4')).toBeNull()
    expect(isManagedZhipuBaseURL('https://open.bigmodel.cn/api/paas/v4')).toBe(true)
    expect(isManagedZhipuBaseURL('https://relay.example/api/paas/v4')).toBe(false)
  })

  it('preserves the selected international GLM host when routing changes', () => {
    expect(
      zhipuBaseURLForRouting(
        'coding',
        'chat_completions',
        'https://api.z.ai/api/paas/v4',
      ),
    ).toBe('https://api.z.ai/api/coding/paas/v4')
    expect(
      zhipuBaseURLForRouting(
        'payg',
        'anthropic',
        'https://open.bigmodel.cn/api/coding/paas/v4',
      ),
    ).toBe('https://open.bigmodel.cn/api/anthropic')
    expect(
      zhipuBaseURLForRouting(
        'coding',
        'chat_completions',
        'https://relay.example/glm/v1',
      ),
    ).toBe('https://open.bigmodel.cn/api/coding/paas/v4')
  })

  it('normalizes persisted GLM routing values defensively', () => {
    expect(normalizeZhipuAccountMode(' CODING ')).toBe('coding')
    expect(normalizeZhipuAccountMode('payg')).toBe('payg')
    expect(normalizeZhipuAccountMode('responses')).toBeUndefined()
    expect(normalizeZhipuApiProtocol(' ANTHROPIC ')).toBe('anthropic')
    expect(normalizeZhipuApiProtocol('chat_completions')).toBe('chat_completions')
    expect(normalizeZhipuApiProtocol('responses')).toBeUndefined()
  })

  it('normalizes pool-mode retry input without changing the payload contract', () => {
    expect(parsePoolModeRetryStatusCodes(' 429,401 429 700 nope 503 ')).toEqual([
      401,
      429,
      503,
    ])
    expect(normalizePoolModeRetryCount(Number.NaN)).toBe(3)
    expect(normalizePoolModeRetryCount(-2)).toBe(0)
    expect(normalizePoolModeRetryCount(99)).toBe(10)
  })

  it('builds Bedrock credentials including model, pool and warmup options', () => {
    expect(
      buildBedrockCredentials({
        authMode: 'sigv4',
        accessKeyId: ' AKIA ',
        secretAccessKey: ' secret ',
        sessionToken: ' session ',
        region: '',
        forceGlobal: true,
        apiKey: '',
        modelMapping: { 'claude-*': 'anthropic.claude-3-7-sonnet' },
        enabled: true,
        retryCount: 4.9,
        retryStatusCodesInput: '429, 401',
        interceptWarmupRequests: true,
      }),
    ).toEqual({
      auth_mode: 'sigv4',
      aws_region: 'us-east-1',
      aws_access_key_id: 'AKIA',
      aws_secret_access_key: 'secret',
      aws_session_token: 'session',
      aws_force_global: 'true',
      model_mapping: { 'claude-*': 'anthropic.claude-3-7-sonnet' },
      pool_mode: true,
      pool_mode_retry_count: 4,
      pool_mode_retry_status_codes: [401, 429],
      intercept_warmup_requests: true,
    })
  })

  it('builds platform API-key credentials through one policy', () => {
    expect(
      buildAPIKeyCredentials({
        platform: 'openai',
        baseUrl: '',
        apiKey: ' sk-openai ',
        modelMapping: { 'gpt-*': 'gpt-5' },
        compactModelMapping: { 'gpt-5': 'gpt-5-mini' },
        openAIEndpointCapabilities: ['embeddings'],
        enabled: true,
        retryCount: 3,
        retryStatusCodesInput: '401 429',
        customErrorCodesEnabled: true,
        customErrorCodes: [429, 503],
        interceptWarmupRequests: true,
      }),
    ).toEqual({
      base_url: 'https://api.openai.com',
      api_key: 'sk-openai',
      model_mapping: { 'gpt-*': 'gpt-5' },
      openai_capabilities: ['embeddings'],
      compact_model_mapping: { 'gpt-5': 'gpt-5-mini' },
      pool_mode: true,
      pool_mode_retry_count: 3,
      pool_mode_retry_status_codes: [401, 429],
      custom_error_codes_enabled: true,
      custom_error_codes: [429, 503],
      intercept_warmup_requests: true,
    })

    expect(
      buildAPIKeyCredentials({
        platform: 'gemini',
        baseUrl: '',
        apiKey: ' gemini-key ',
        geminiTierId: 'aistudio_paid',
        enabled: false,
        retryCount: 3,
        retryStatusCodesInput: '',
        customErrorCodesEnabled: false,
        customErrorCodes: [],
        interceptWarmupRequests: false,
      }),
    ).toEqual({
      base_url: 'https://generativelanguage.googleapis.com',
      api_key: 'gemini-key',
      tier_id: 'aistudio_paid',
    })

    expect(
      buildAPIKeyCredentials({
        platform: 'deepseek',
        baseUrl: '',
        apiKey: ' sk-deepseek ',
        enabled: false,
        retryCount: 3,
        retryStatusCodesInput: '',
        customErrorCodesEnabled: false,
        customErrorCodes: [],
        interceptWarmupRequests: false,
      }),
    ).toEqual({
      base_url: 'https://api.deepseek.com',
      api_key: 'sk-deepseek',
    })
  })

  it('keeps legacy OpenAI capabilities implicit and persists audio opt-ins', () => {
    const commonInput = {
      platform: 'openai' as const,
      baseUrl: '',
      apiKey: 'sk-openai',
      enabled: false,
      retryCount: 3,
      retryStatusCodesInput: '',
      customErrorCodesEnabled: false,
      customErrorCodes: [],
      interceptWarmupRequests: false,
    }

    expect(
      buildAPIKeyCredentials({
        ...commonInput,
        openAIEndpointCapabilities: [],
      }),
    ).not.toHaveProperty('openai_capabilities')

    expect(
      buildAPIKeyCredentials({
        ...commonInput,
        openAIEndpointCapabilities: [
          'chat_completions',
          'embeddings',
          'audio_transcriptions',
        ],
      }).openai_capabilities,
    ).toEqual([
      'chat_completions',
      'embeddings',
      'audio_transcriptions',
    ])

    expect(
      buildAPIKeyCredentials({
        ...commonInput,
        openAIEndpointCapabilities: [
          'chat_completions',
          'audio_transcriptions',
        ],
      }).openai_capabilities,
    ).toEqual(['chat_completions', 'audio_transcriptions'])
  })

  it('feeds generated credentials through the discriminated payload adapter', () => {
    const credentials = buildAPIKeyCredentials({
      platform: 'openai',
      baseUrl: '',
      apiKey: ' sk-openai ',
      openAIEndpointCapabilities: ['chat_completions'],
      enabled: false,
      retryCount: 3,
      retryStatusCodesInput: '',
      customErrorCodesEnabled: false,
      customErrorCodes: [],
      interceptWarmupRequests: false,
    })

    expect(
      buildAccountCreatePayload(
        base,
        toAccountCredentialDraft(
          'openai',
          'apikey',
          credentials,
          { openai_long_context_billing_enabled: false },
        ),
      ),
    ).toEqual({
      name: 'Account',
      notes: '',
      platform: 'openai',
      type: 'apikey',
      credentials: {
        base_url: 'https://api.openai.com',
        api_key: 'sk-openai',
        openai_capabilities: ['chat_completions'],
      },
      extra: { openai_long_context_billing_enabled: false },
      proxy_id: null,
      concurrency: 10,
      load_factor: undefined,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [3],
      expires_at: null,
      auto_pause_on_expired: true,
    })

    const deepseekCredentials = buildAPIKeyCredentials({
      platform: 'deepseek',
      baseUrl: ' https://api.deepseek.com ',
      apiKey: ' sk-deepseek ',
      modelMapping: { 'deepseek-chat': 'deepseek-v4-flash' },
      enabled: false,
      retryCount: 3,
      retryStatusCodesInput: '',
      customErrorCodesEnabled: false,
      customErrorCodes: [],
      interceptWarmupRequests: false,
    })
    expect(
      buildAccountCreatePayload(
        base,
        toAccountCredentialDraft('deepseek', 'apikey', deepseekCredentials),
      ),
    ).toMatchObject({
      platform: 'deepseek',
      type: 'apikey',
      credentials: {
        base_url: 'https://api.deepseek.com',
        api_key: 'sk-deepseek',
        model_mapping: { 'deepseek-chat': 'deepseek-v4-flash' },
      },
    })
  })

  it('keeps typed GLM routing metadata while preserving legacy options', () => {
    const payload = buildZhipuAccountPayload(base, {
      platform: 'zhipu',
      kind: 'apikey',
      baseUrl: ' https://api.z.ai/api/coding/paas/v4 ',
      apiKey: ' sk-glm ',
      accountMode: 'coding',
      apiProtocol: 'anthropic',
      zhipuOrganization: ' org-demo ',
      zhipuProject: ' project-demo ',
      credentialOptions: {
        api_protocol: 'chat_completions',
        model_mapping: { 'glm-*': 'glm-5' },
      },
    })

    expect(payload.credentials).toEqual({
      base_url: 'https://api.z.ai/api/coding/paas/v4',
      api_key: 'sk-glm',
      api_protocol: 'anthropic',
      account_mode: 'coding',
      zhipu_organization: 'org-demo',
      zhipu_project: 'project-demo',
      model_mapping: { 'glm-*': 'glm-5' },
    })
  })

  it('builds upstream and Vertex credential records without UI state', () => {
    expect(
      buildAntigravityUpstreamCredentials({
        baseUrl: ' https://upstream.example/v1 ',
        apiKey: ' key ',
        modelMapping: { 'gemini-*': 'gemini-2.5-pro' },
        interceptWarmupRequests: true,
      }),
    ).toEqual({
      base_url: 'https://upstream.example/v1',
      api_key: 'key',
      model_mapping: { 'gemini-*': 'gemini-2.5-pro' },
      intercept_warmup_requests: true,
    })
    expect(
      buildVertexServiceAccountCredentials({
        serviceAccountJson: ' {"type":"service_account"} ',
        projectId: ' project ',
        clientEmail: ' bot@example.com ',
        location: ' global ',
      }),
    ).toEqual({
      service_account_json: '{"type":"service_account"}',
      project_id: 'project',
      client_email: 'bot@example.com',
      location: 'global',
      tier_id: 'vertex',
    })
  })

  it('parses Vertex service-account JSON into a validated narrow result', () => {
    expect(parseVertexServiceAccountJSON('')).toEqual({
      ok: false,
      reason: 'empty',
    })
    expect(parseVertexServiceAccountJSON('{bad json')).toEqual({
      ok: false,
      reason: 'invalid_json',
    })
    expect(parseVertexServiceAccountJSON('{"project_id":"project"}')).toEqual({
      ok: false,
      reason: 'missing_fields',
    })
    expect(
      parseVertexServiceAccountJSON(
        '{"project_id":"project","client_email":"bot@example.com","private_key":"secret"}',
      ),
    ).toEqual({
      ok: true,
      normalizedJson:
        '{"project_id":"project","client_email":"bot@example.com","private_key":"secret"}',
      projectId: 'project',
      clientEmail: 'bot@example.com',
    })
  })

  it('filters invalid temporary-unschedulable rules deterministically', () => {
    expect(
      buildTempUnschedulableRules([
        {
          error_code: 529,
          keywords: ' overloaded; too many ',
          duration_minutes: 60.8,
          description: ' overload ',
        },
        {
          error_code: 99,
          keywords: 'invalid',
          duration_minutes: 10,
          description: '',
        },
      ]),
    ).toEqual([
      {
        error_code: 529,
        keywords: ['overloaded', 'too many'],
        duration_minutes: 60,
        description: 'overload',
      },
    ])
  })
})

describe('create-account extra builders', () => {
  it('builds OpenAI and Anthropic API-key extras without modal branches', () => {
    const openai = buildOpenAIExtra({
      platform: 'openai',
      accountCategory: 'apikey',
      apiKeyWebSocketMode: 'http_bridge',
      passthroughEnabled: true,
      longContextBillingEnabled: true,
      codexCLIOnlyEnabled: false,
      codexCLIOnlyAppServerEnabled: false,
      compactMode: 'force_on',
      responsesMode: 'force_responses',
      textGenerationCapabilityEnabled: true,
    })
    expect(openai).toEqual({
      openai_apikey_responses_websockets_v2_mode: 'http_bridge',
      openai_apikey_responses_websockets_v2_enabled: true,
      openai_passthrough: true,
      openai_long_context_billing_enabled: true,
      openai_compact_mode: 'force_on',
      openai_responses_mode: 'force_responses',
    })
    expect(
      buildAnthropicAPIKeyExtra(
        {
          platform: 'anthropic',
          accountCategory: 'apikey',
          passthroughEnabled: true,
          authScheme: 'authorization_bearer',
          webSearchEmulationMode: 'force',
        },
        openai,
      ),
    ).toEqual({
      ...(openai || {}),
      anthropic_passthrough: true,
      anthropic_apikey_auth_scheme: 'authorization_bearer',
      web_search_emulation: 'force',
    })
  })

  it('shares Anthropic OAuth controls across code and cookie flows', () => {
    expect(
      buildAnthropicOAuthExtra(
        { account_uuid: 'account' },
        {
          windowCostEnabled: true,
          windowCostLimit: 42,
          windowCostStickyReserve: null,
          sessionLimitEnabled: true,
          maxSessions: 3,
          sessionIdleTimeout: null,
          rpmLimitEnabled: true,
          baseRpm: null,
          rpmStrategy: 'sticky_exempt',
          rpmStickyBuffer: 2,
          userMsgQueueMode: 'serialize',
          tlsFingerprintEnabled: true,
          tlsFingerprintProfileId: 9,
          sessionIdMaskingEnabled: true,
          cacheTTLOverrideEnabled: true,
          cacheTTLOverrideTarget: '10m',
          customBaseUrlEnabled: true,
          customBaseUrl: ' https://claude.example ',
        },
      ),
    ).toEqual({
      account_uuid: 'account',
      window_cost_limit: 42,
      window_cost_sticky_reserve: 10,
      max_sessions: 3,
      session_idle_timeout_minutes: 5,
      base_rpm: 15,
      rpm_strategy: 'sticky_exempt',
      rpm_sticky_buffer: 2,
      user_msg_queue_mode: 'serialize',
      enable_tls_fingerprint: true,
      tls_fingerprint_profile_id: 9,
      session_id_masking_enabled: true,
      cache_ttl_override_enabled: true,
      cache_ttl_override_target: '10m',
      custom_base_url_enabled: true,
      custom_base_url: 'https://claude.example',
    })
  })

  it('builds quota and Antigravity extras as defensive records', () => {
    expect(
      buildAPIKeyQuotaExtra(
        { existing: true },
        {
          quotaLimit: 100,
          quotaDailyLimit: 10,
          quotaWeeklyLimit: 50,
          dailyResetMode: 'fixed',
          dailyResetHour: null,
          weeklyResetMode: 'fixed',
          weeklyResetDay: null,
          weeklyResetHour: 8,
          resetTimezone: null,
        },
      ),
    ).toEqual({
      existing: true,
      quota_limit: 100,
      quota_daily_limit: 10,
      quota_weekly_limit: 50,
      quota_daily_reset_mode: 'fixed',
      quota_daily_reset_hour: 0,
      quota_weekly_reset_mode: 'fixed',
      quota_weekly_reset_day: 1,
      quota_weekly_reset_hour: 8,
      quota_reset_timezone: 'UTC',
    })
    expect(
      buildAntigravityExtra({
        mixedScheduling: true,
        allowOverages: true,
      }),
    ).toEqual({ mixed_scheduling: true, allow_overages: true })
  })
})
