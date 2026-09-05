-- Allow DeepSeek rows in the per-user platform quota table.
--
-- The application validator and default-quota settings now expose the
-- first-class DeepSeek platform. Existing deployments still carry the older
-- CHECK constraint installed by migration 157, so update it in a new,
-- idempotent migration rather than modifying an applied migration.
ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'deepseek'));
