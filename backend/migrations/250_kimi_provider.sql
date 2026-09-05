-- Migration: 250_kimi_provider
-- Register Kimi/Moonshot in persisted platform/provider CHECK constraints.
DO $$
DECLARE d TEXT;
BEGIN
  SELECT pg_get_constraintdef(c.oid) INTO d FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid WHERE t.relname='user_platform_quotas' AND c.conname='user_platform_quotas_platform_check';
  IF d IS NULL OR position('kimi' IN d)=0 THEN
    ALTER TABLE user_platform_quotas DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;
    ALTER TABLE user_platform_quotas ADD CONSTRAINT user_platform_quotas_platform_check CHECK (platform IN ('anthropic','openai','gemini','antigravity','grok','kimi','zhipu','deepseek'));
  END IF;
  SELECT pg_get_constraintdef(c.oid) INTO d FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid WHERE t.relname='channel_monitors' AND c.conname='channel_monitors_provider_check';
  IF d IS NULL OR position('kimi' IN d)=0 THEN
    ALTER TABLE channel_monitors DROP CONSTRAINT IF EXISTS channel_monitors_provider_check;
    ALTER TABLE channel_monitors ADD CONSTRAINT channel_monitors_provider_check CHECK (provider IN ('openai','anthropic','gemini','grok','kimi','zhipu','deepseek'));
  END IF;
  SELECT pg_get_constraintdef(c.oid) INTO d FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid WHERE t.relname='channel_monitor_request_templates' AND c.conname='channel_monitor_request_templates_provider_check';
  IF d IS NULL OR position('kimi' IN d)=0 THEN
    ALTER TABLE channel_monitor_request_templates DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_provider_check;
    ALTER TABLE channel_monitor_request_templates ADD CONSTRAINT channel_monitor_request_templates_provider_check CHECK (provider IN ('openai','anthropic','gemini','grok','kimi','zhipu','deepseek'));
  END IF;
END $$;
