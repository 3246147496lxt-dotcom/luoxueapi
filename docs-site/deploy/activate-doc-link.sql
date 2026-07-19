BEGIN;

SET LOCAL lock_timeout = '5s';

INSERT INTO settings (key, value)
VALUES ('doc_url', 'https://luoxueapi.cc/tutorial-docs/')
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = NOW();

UPDATE settings AS s
SET value = COALESCE(
      (
        SELECT jsonb_agg(entry.item ORDER BY entry.ordinality)
        FROM jsonb_array_elements(s.value::jsonb)
             WITH ORDINALITY AS entry(item, ordinality)
        WHERE entry.item ->> 'url' IS DISTINCT FROM 'md:guide'
      ),
      '[]'::jsonb
    )::text,
    updated_at = NOW()
WHERE key = 'custom_menu_items';

DO $$
BEGIN
  IF (SELECT value FROM settings WHERE key = 'doc_url')
       IS DISTINCT FROM 'https://luoxueapi.cc/tutorial-docs/' THEN
    RAISE EXCEPTION 'doc_url verification failed';
  END IF;

  IF EXISTS (
    SELECT 1
    FROM settings AS s,
         jsonb_array_elements(s.value::jsonb) AS item
    WHERE s.key = 'custom_menu_items'
      AND item ->> 'url' = 'md:guide'
  ) THEN
    RAISE EXCEPTION 'legacy md:guide menu removal failed';
  END IF;
END
$$;

COMMIT;
