CREATE TABLE card_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    stability FLOAT NOT NULL DEFAULT 0,
    difficulty FLOAT NOT NULL DEFAULT 5,
    retrievability FLOAT NOT NULL DEFAULT 0,
    due_date TIMESTAMP,
    last_review TIMESTAMP,
    interval_days FLOAT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    reps INT NOT NULL DEFAULT 0,
    lapses INT NOT NULL DEFAULT 0,
    UNIQUE (card_id, user_id)
);

CREATE INDEX idx_card_states_due ON card_states(user_id, due_date)
    WHERE status IN ('learning', 'review', 'relearning');

CREATE INDEX idx_card_states_user_status ON card_states(user_id, status);
