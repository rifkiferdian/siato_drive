ALTER TABLE sharing_share_links
  ADD COLUMN allow_download boolean NOT NULL DEFAULT true AFTER expires_at,
  ADD COLUMN active boolean NOT NULL DEFAULT true AFTER allow_download,
  ADD COLUMN last_accessed_at datetime(6) NULL AFTER active,
  ADD KEY share_links_node_active_idx(node_id,active);
