CREATE TABLE review_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stability_before FLOAT NOT NULL,
    difficulty_before FLOAT NOT NULL,
    retrievability_before FLOAT NOT NULL,
    interval_before FLOAT NOT NULL,
    status_before VARCHAR(20) NOT NULL,
    stability_after FLOAT NOT NULL,
    difficulty_after FLOAT NOT NULL,
    interval_after FLOAT NOT NULL,
    status_after VARCHAR(20) NOT NULL,
    rating SMALLINT NOT NULL,
    answered_tokens JSONB NOT NULL DEFAULT '[]'::jsonb,
    was_correct BOOLEAN NOT NULL,
    wrong_attempts_count INT NOT NULL DEFAULT 0,
    distractor_clicks_count INT NOT NULL DEFAULT 0,
    incorrect_tokens_clicked JSONB NOT NULL DEFAULT '[]'::jsonb,
    attempts JSONB NOT NULL DEFAULT '[]'::jsonb,
    reviewed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_logs_card_user ON review_logs(card_id, user_id, reviewed_at DESC);
CREATE INDEX idx_review_logs_user ON review_logs(user_id, reviewed_at DESC);
