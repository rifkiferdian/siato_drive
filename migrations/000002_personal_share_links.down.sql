ALTER TABLE sharing_share_links
  DROP KEY share_links_node_active_idx,
  DROP COLUMN last_accessed_at,
  DROP COLUMN active,
  DROP COLUMN allow_download;
