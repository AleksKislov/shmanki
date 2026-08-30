ALTER TABLE review_logs
    DROP COLUMN params_version,
    DROP COLUMN elapsed_days,
    DROP COLUMN review_duration_ms,
    DROP COLUMN user_timezone;
