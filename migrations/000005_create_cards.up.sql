CREATE TABLE cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    info_object_id UUID NOT NULL REFERENCES info_objects(id) ON DELETE CASCADE,
    front TEXT NOT NULL,
    step INT NOT NULL DEFAULT 0,
    correct_answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    distractors JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cards_info_object_id ON cards(info_object_id);
CREATE INDEX idx_cards_step ON cards(info_object_id, step);
