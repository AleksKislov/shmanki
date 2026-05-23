CREATE TABLE generation_drafts (
    generation_id UUID PRIMARY KEY REFERENCES generation_logs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id UUID NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    objects_raw JSONB NOT NULL,
    model VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL DEFAULT (NOW() + INTERVAL '24 hours')
);

CREATE INDEX idx_generation_drafts_user_deck ON generation_drafts(user_id, deck_id);
CREATE INDEX idx_generation_drafts_expires_at ON generation_drafts(expires_at);
