ALTER TABLE generation_logs
ADD COLUMN provider VARCHAR(100) NOT NULL DEFAULT '',
ADD COLUMN objects_raw JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX idx_generation_logs_deck_user ON generation_logs(deck_id, user_id, created_at DESC);
