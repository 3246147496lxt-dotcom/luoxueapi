COPY (
  SELECT key, value, updated_at
  FROM settings
  WHERE key IN ('doc_url', 'custom_menu_items')
  ORDER BY key
) TO '/tmp/luoxue-doc-settings-before-same-origin.csv'
WITH (FORMAT csv, HEADER true);
