DROP INDEX IF EXISTS idx_generation_logs_deck_user;

ALTER TABLE generation_logs
DROP COLUMN IF EXISTS objects_raw,
DROP COLUMN IF EXISTS provider;
