-- Manual rollback companion for 199_skill_import.sql.
--
-- This project embeds only top-level, forward-only migrations. Keep this file
-- outside the embedded migration glob and execute it only from a reviewed
-- maintenance runbook after confirming no imported catalog data is needed.

-- Dropping a table also drops its triggers. Avoid explicit DROP TRIGGER ... ON
-- statements: PostgreSQL errors when the relation itself is absent, which
-- would make this recovery script unusable after a partial migration or on a
-- second reviewed execution.
DROP TABLE IF EXISTS skill_import_events;
DROP TABLE IF EXISTS skill_version_origins;
DROP TABLE IF EXISTS skill_origins;
DROP TABLE IF EXISTS skill_import_run_items;

ALTER TABLE IF EXISTS skill_import_schedules
    DROP CONSTRAINT IF EXISTS skill_import_schedules_last_run_fk;

DROP TABLE IF EXISTS skill_import_runs;
DROP TABLE IF EXISTS skill_import_schedules;
DROP TABLE IF EXISTS skill_import_sources;

DROP FUNCTION IF EXISTS enforce_skill_import_append_only_row();
DROP FUNCTION IF EXISTS prevent_skill_import_history_delete();

DROP INDEX IF EXISTS idx_skills_catalog_public_order;

ALTER TABLE IF EXISTS skills
    DROP CONSTRAINT IF EXISTS skills_catalog_source_rank_check,
    DROP CONSTRAINT IF EXISTS skills_catalog_source_priority_check,
    DROP CONSTRAINT IF EXISTS skills_origin_url_check,
    DROP COLUMN IF EXISTS catalog_source_rank,
    DROP COLUMN IF EXISTS catalog_source_priority,
    DROP COLUMN IF EXISTS origin_url;
