-- Fields required to later fit FSRS weights against real review history.
-- These cannot be backfilled, so they are added before the data accumulates.

ALTER TABLE review_logs
    -- Which model + parameter set produced this schedule. Rows written before
    -- this migration came from the pre-canonical difficulty/interval formulas
    -- and must not be pooled with later rows when fitting.
    ADD COLUMN params_version VARCHAR(64) NOT NULL DEFAULT 'legacy-pre-canonical',
    -- Actual days since the previous review of this card. Distinct from
    -- interval_before (days *scheduled*); the gap between them is the signal
    -- the optimizer learns from.
    ADD COLUMN elapsed_days FLOAT NOT NULL DEFAULT 0,
    -- Time from card shown to answer submitted. Null when the client did not
    -- report it.
    ADD COLUMN review_duration_ms INT,
    -- IANA zone of the reviewer at review time, snapshotted rather than joined
    -- so day bucketing stays correct after a user relocates. Optimizers
    -- deduplicate to one review per card per local day.
    ADD COLUMN user_timezone VARCHAR(64);

ALTER TABLE review_logs
    ALTER COLUMN params_version DROP DEFAULT;
