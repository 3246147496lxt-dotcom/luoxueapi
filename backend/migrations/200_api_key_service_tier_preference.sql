-- Per-API-key default OpenAI service tier preference.
-- Explicit client service_tier values continue to take precedence at request time.

ALTER TABLE api_keys
    ADD COLUMN IF NOT EXISTS service_tier_preference VARCHAR(20) NOT NULL DEFAULT 'standard';

-- Be defensive when upgrading databases that may have been created with a
-- nullable column during an interrupted/older rollout.
UPDATE api_keys
SET service_tier_preference = 'standard'
WHERE service_tier_preference IS NULL;

ALTER TABLE api_keys
    ALTER COLUMN service_tier_preference SET DEFAULT 'standard',
    ALTER COLUMN service_tier_preference SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'api_keys_service_tier_preference_check'
          AND conrelid = 'api_keys'::regclass
    ) THEN
        ALTER TABLE api_keys
            ADD CONSTRAINT api_keys_service_tier_preference_check
            CHECK (service_tier_preference IN ('standard', 'priority'));
    END IF;
END $$;
