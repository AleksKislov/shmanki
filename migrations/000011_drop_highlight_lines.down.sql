ALTER TABLE cards ADD COLUMN highlight_lines JSONB NOT NULL DEFAULT '[]'::jsonb;
