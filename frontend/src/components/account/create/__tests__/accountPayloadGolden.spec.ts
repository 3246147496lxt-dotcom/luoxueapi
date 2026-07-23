import { describe, expect, it } from 'vitest'
import type { AccountPlatform, AccountType, CreateAccountRequest } from '@/types'
import {
  buildAccountCreatePayload,
  buildGrokSSOImportRequest,
  buildOpenAICodexPATCreateRequest,
  buildOpenAICodexSessionImportRequest,
  toAccountCredentialDraft,
  type AccountBaseDraft,
  type AccountCredentialDraft,
} from '../accountDraft'

const base: AccountBaseDraft = {
  name: 'Production account',
  notes: 'created by modal',
  proxy_id: 17,
  concurrency: 12,
  load_factor: null,
  priority: 2,
  rate_multiplier: 1.25,
  group_ids: [4, 9],
  expires_at: 1_900_000_000,
  auto_pause_on_expired: true,
}

function goldenPayload(
  platform: AccountPlatform,
  type: AccountType,
  credentials: Record<string, unknown>,
  extra?: Record<string, unknown>,
): CreateAccountRequest {
  return {
    name: 'Production account',
    notes: 'created by modal',
    platform,
    type,
    credentials,
    extra,
    proxy_id: 17,
    concurrency: 12,
    load_factor: undefined,
    priority: 2,
    rate_multiplier: 1.25,
    group_ids: [4, 9],
    expires_at: 1_900_000_000,
    auto_pause_on_expired: true,
  }
}

interface PayloadGolden {
  name: string
  draft: AccountCredentialDraft
  expected: CreateAccountRequest
}

const goldens: PayloadGolden[] = [
  {
    name: 'Anthropic setup-token',
    draft: {
      platform: 'anthropic',
      kind: 'setup-token',
      credentials: {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: 1_800_000_000,
      },
      extra: { account_uuid: 'claude-account', window_cost_limit: 42 },
    },
    expected: goldenPayload(
      'anthropic',
      'setup-token',
      {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: 1_800_000_000,
      },
      { account_uuid: 'claude-account', window_cost_limit: 42 },
    ),
  },
  {
    name: 'Anthropic API key',
    draft: {
      platform: 'anthropic',
      kind: 'apikey',
      baseUrl: '  https://claude.internal/v1  ',
      apiKey: '  sk-ant-api03  ',
      credentialOptions: {
        model_mapping: { 'claude-sonnet-*': 'claude-sonnet-4-5' },
        header_override_enabled: true,
        header_overrides: { 'x-tenant': 'prod' },
        intercept_warmup_requests: true,
      },
      extra: {
        anthropic_passthrough: true,
        anthropic_apikey_auth_scheme: 'authorization_bearer',
      },
    },
    expected: goldenPayload(
      'anthropic',
      'apikey',
      {
        base_url: 'https://claude.internal/v1',
        api_key: 'sk-ant-api03',
        model_mapping: { 'claude-sonnet-*': 'claude-sonnet-4-5' },
        header_override_enabled: true,
        header_overrides: { 'x-tenant': 'prod' },
        intercept_warmup_requests: true,
      },
      {
        anthropic_passthrough: true,
        anthropic_apikey_auth_scheme: 'authorization_bearer',
      },
    ),
  },
  {
    name: 'Anthropic Bedrock SigV4',
    draft: {
      platform: 'anthropic',
      kind: 'bedrock',
      authMode: 'sigv4',
      region: '',
      accessKeyId: '  AKIAEXAMPLE  ',
      secretAccessKey: '  secret  ',
      sessionToken: '  session  ',
      forceGlobal: true,
      credentialOptions: {
        model_mapping: { 'claude-*': 'anthropic.claude-3-7-sonnet' },
        pool_mode: true,
        pool_mode_retry_count: 3,
      },
      extra: { quota_limit: 100 },
    },
    expected: goldenPayload(
      'anthropic',
      'bedrock',
      {
        auth_mode: 'sigv4',
        aws_region: 'us-east-1',
        aws_access_key_id: 'AKIAEXAMPLE',
        aws_secret_access_key: 'secret',
        aws_session_token: 'session',
        aws_force_global: 'true',
        model_mapping: { 'claude-*': 'anthropic.claude-3-7-sonnet' },
        pool_mode: true,
        pool_mode_retry_count: 3,
      },
      { quota_limit: 100 },
    ),
  },
  {
    name: 'Anthropic Bedrock bearer API key',
    draft: {
      platform: 'anthropic',
      kind: 'bedrock',
      authMode: 'apikey',
      region: ' eu-west-1 ',
      apiKey: ' bedrock-token ',
    },
    expected: goldenPayload('anthropic', 'bedrock', {
      auth_mode: 'apikey',
      aws_region: 'eu-west-1',
      api_key: 'bedrock-token',
    }),
  },
  {
    name: 'Anthropic Vertex service account',
    draft: {
      platform: 'anthropic',
      kind: 'service_account',
      serviceAccountJson: '  {"type":"service_account"}  ',
      projectId: ' project-a ',
      clientEmail: ' bot@example.iam.gserviceaccount.com ',
      location: ' us-central1 ',
      credentialOptions: { intercept_warmup_requests: true },
    },
    expected: goldenPayload('anthropic', 'service_account', {
      service_account_json: '{"type":"service_account"}',
      project_id: 'project-a',
      client_email: 'bot@example.iam.gserviceaccount.com',
      location: 'us-central1',
      tier_id: 'vertex',
      intercept_warmup_requests: true,
    }),
  },
  {
    name: 'OpenAI OAuth',
    draft: {
      platform: 'openai',
      kind: 'oauth',
      credentials: {
        access_token: 'openai-access',
        refresh_token: 'openai-refresh',
        account_id: 'chatgpt-account',
      },
      extra: {
        plan_type: 'plus',
        openai_oauth_responses_websockets_v2_mode: 'ctx_pool',
        openai_long_context_billing_enabled: false,
      },
    },
    expected: goldenPayload(
      'openai',
      'oauth',
      {
        access_token: 'openai-access',
        refresh_token: 'openai-refresh',
        account_id: 'chatgpt-account',
      },
      {
        plan_type: 'plus',
        openai_oauth_responses_websockets_v2_mode: 'ctx_pool',
        openai_long_context_billing_enabled: false,
      },
    ),
  },
  {
    name: 'OpenAI API key',
    draft: {
      platform: 'openai',
      kind: 'apikey',
      apiKey: ' sk-openai ',
      credentialOptions: {
        openai_capabilities: ['chat_completions'],
        compact_model_mapping: { 'gpt-5': 'gpt-5-mini' },
        pool_mode: true,
        pool_mode_retry_status_codes: [401, 429],
      },
      extra: {
        openai_passthrough: true,
        openai_responses_mode: 'force_responses',
        openai_apikey_responses_websockets_v2_mode: 'http_bridge',
      },
    },
    expected: goldenPayload(
      'openai',
      'apikey',
      {
        base_url: 'https://api.openai.com',
        api_key: 'sk-openai',
        openai_capabilities: ['chat_completions'],
        compact_model_mapping: { 'gpt-5': 'gpt-5-mini' },
        pool_mode: true,
        pool_mode_retry_status_codes: [401, 429],
      },
      {
        openai_passthrough: true,
        openai_responses_mode: 'force_responses',
        openai_apikey_responses_websockets_v2_mode: 'http_bridge',
      },
    ),
  },
  {
    name: 'Gemini OAuth',
    draft: {
      platform: 'gemini',
      kind: 'oauth',
      credentials: {
        access_token: 'gemini-access',
        refresh_token: 'gemini-refresh',
        tier_id: 'google_ai_pro',
      },
      extra: { email: 'gemini@example.com' },
    },
    expected: goldenPayload(
      'gemini',
      'oauth',
      {
        access_token: 'gemini-access',
        refresh_token: 'gemini-refresh',
        tier_id: 'google_ai_pro',
      },
      { email: 'gemini@example.com' },
    ),
  },
  {
    name: 'Gemini AI Studio API key',
    draft: {
      platform: 'gemini',
      kind: 'apikey',
      apiKey: ' gemini-key ',
      tierId: 'aistudio_paid',
      credentialOptions: {
        model_mapping: { 'gemini-2.5-pro': 'gemini-2.5-flash' },
      },
    },
    expected: goldenPayload('gemini', 'apikey', {
      base_url: 'https://generativelanguage.googleapis.com',
      api_key: 'gemini-key',
      tier_id: 'aistudio_paid',
      model_mapping: { 'gemini-2.5-pro': 'gemini-2.5-flash' },
    }),
  },
  {
    name: 'Gemini Vertex service account',
    draft: {
      platform: 'gemini',
      kind: 'service_account',
      serviceAccountJson: '{"private_key":"secret"}',
      projectId: 'project-gemini',
      clientEmail: 'gemini@example.iam.gserviceaccount.com',
      location: 'global',
    },
    expected: goldenPayload('gemini', 'service_account', {
      service_account_json: '{"private_key":"secret"}',
      project_id: 'project-gemini',
      client_email: 'gemini@example.iam.gserviceaccount.com',
      location: 'global',
      tier_id: 'vertex',
    }),
  },
  {
    name: 'Antigravity upstream',
    draft: {
      platform: 'antigravity',
      kind: 'upstream',
      baseUrl: ' https://antigravity.internal/v1 ',
      apiKey: ' upstream-key ',
      credentialOptions: {
        model_mapping: { 'gemini-*': 'gemini-2.5-pro' },
        intercept_warmup_requests: true,
      },
      extra: { mixed_scheduling: true, allow_overages: true },
    },
    expected: goldenPayload(
      'antigravity',
      'apikey',
      {
        base_url: 'https://antigravity.internal/v1',
        api_key: 'upstream-key',
        model_mapping: { 'gemini-*': 'gemini-2.5-pro' },
        intercept_warmup_requests: true,
      },
      { mixed_scheduling: true, allow_overages: true },
    ),
  },
  {
    name: 'Grok OAuth with custom upstream',
    draft: {
      platform: 'grok',
      kind: 'oauth',
      credentials: {
        access_token: 'grok-access',
        refresh_token: 'grok-refresh',
        header_override_enabled: true,
        header_overrides: { 'x-tenant': 'grok' },
      },
      baseUrl: ' https://grok.internal/v1 ',
      extra: { email: 'grok@example.com' },
    },
    expected: goldenPayload(
      'grok',
      'oauth',
      {
        access_token: 'grok-access',
        refresh_token: 'grok-refresh',
        header_override_enabled: true,
        header_overrides: { 'x-tenant': 'grok' },
        base_url: 'https://grok.internal/v1',
      },
      { email: 'grok@example.com' },
    ),
  },
  {
    name: 'Grok OAuth without a custom upstream',
    draft: {
      platform: 'grok',
      kind: 'oauth',
      credentials: {
        access_token: 'grok-official-access',
        refresh_token: 'grok-official-refresh',
      },
    },
    expected: goldenPayload('grok', 'oauth', {
      access_token: 'grok-official-access',
      refresh_token: 'grok-official-refresh',
    }),
  },
  {
    name: 'Grok API key with official default',
    draft: {
      platform: 'grok',
      kind: 'apikey',
      apiKey: ' xai-key ',
      credentialOptions: {
        model_mapping: { 'grok-*': 'grok-4' },
        custom_error_codes_enabled: true,
        custom_error_codes: [429, 503],
      },
    },
    expected: goldenPayload('grok', 'apikey', {
      base_url: 'https://api.x.ai/v1',
      api_key: 'xai-key',
      model_mapping: { 'grok-*': 'grok-4' },
      custom_error_codes_enabled: true,
      custom_error_codes: [429, 503],
    }),
  },
]

describe('account platform payload golden contracts', () => {
  it.each(goldens)('$name', ({ draft, expected }) => {
    expect(buildAccountCreatePayload(base, draft)).toEqual(expected)
  })

  it.each(goldens)('$name round-trips the modal compatibility boundary', ({ expected }) => {
    const draft = toAccountCredentialDraft(
      expected.platform,
      expected.type,
      expected.credentials,
      expected.extra,
    )
    expect(buildAccountCreatePayload(base, draft)).toEqual(expected)
  })

  it('defensively copies mutable base and payload records', () => {
    const draft = {
      platform: 'openai',
      kind: 'oauth',
      credentials: { access_token: 'token' },
      extra: { plan_type: 'plus' },
    } satisfies AccountCredentialDraft
    const payload = buildAccountCreatePayload(base, draft)

    payload.group_ids?.push(99)
    payload.credentials.access_token = 'changed'
    if (payload.extra) payload.extra.plan_type = 'free'

    expect(base.group_ids).toEqual([4, 9])
    expect(draft.credentials).toEqual({ access_token: 'token' })
    expect(draft.extra).toEqual({ plan_type: 'plus' })
  })
})

describe('platform-specific bulk creation payloads', () => {
  it('builds the OpenAI Codex session import contract', () => {
    expect(
      buildOpenAICodexSessionImportRequest(
        base,
        '  {"access_token":"token"}  ',
        { model_mapping: { 'gpt-5': 'gpt-5-mini' } },
        { openai_long_context_billing_enabled: true },
      ),
    ).toEqual({
      content: '{"access_token":"token"}',
      name: 'Production account',
      notes: 'created by modal',
      proxy_id: 17,
      concurrency: 12,
      load_factor: undefined,
      priority: 2,
      rate_multiplier: 1.25,
      group_ids: [4, 9],
      expires_at: 1_900_000_000,
      auto_pause_on_expired: true,
      credential_extras: { model_mapping: { 'gpt-5': 'gpt-5-mini' } },
      extra: { openai_long_context_billing_enabled: true },
      update_existing: true,
    })
  })

  it('builds the OpenAI Codex PAT creation contract', () => {
    expect(
      buildOpenAICodexPATCreateRequest(
        base,
        '  codex-pat  ',
        { temp_unschedulable_enabled: true },
      ),
    ).toEqual({
      access_token: 'codex-pat',
      name: 'Production account',
      notes: 'created by modal',
      proxy_id: 17,
      concurrency: 12,
      load_factor: undefined,
      priority: 2,
      rate_multiplier: 1.25,
      group_ids: [4, 9],
      expires_at: 1_900_000_000,
      auto_pause_on_expired: true,
      credential_extras: { temp_unschedulable_enabled: true },
      extra: undefined,
    })
  })

  it('builds the Grok SSO batch creation contract', () => {
    expect(
      buildGrokSSOImportRequest(base, ['sso-a', 'sso-b'], {
        base_url: 'https://grok.internal/v1',
        header_override_enabled: true,
      }),
    ).toEqual({
      sso_tokens: ['sso-a', 'sso-b'],
      name: 'Production account',
      notes: 'created by modal',
      proxy_id: 17,
      group_ids: [4, 9],
      credentials: {
        base_url: 'https://grok.internal/v1',
        header_override_enabled: true,
      },
      concurrency: 12,
      load_factor: undefined,
      priority: 2,
      rate_multiplier: 1.25,
      expires_at: 1_900_000_000,
      auto_pause_on_expired: true,
    })
  })
})
