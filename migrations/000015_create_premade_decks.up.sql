ALTER TABLE users
ADD COLUMN display_name TEXT;

CREATE TABLE premade_decks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    source TEXT NOT NULL CHECK (source IN ('official', 'community')),
    source_deck_id UUID REFERENCES decks(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    language_code VARCHAR(20) NOT NULL DEFAULT 'en',
    category TEXT NOT NULL,
    is_published BOOLEAN NOT NULL DEFAULT TRUE,
    rating_avg NUMERIC(3,2) NOT NULL DEFAULT 0,
    rating_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_premade_decks_source_deck_id
    ON premade_decks(source_deck_id)
    WHERE source_deck_id IS NOT NULL;

CREATE INDEX idx_premade_decks_source ON premade_decks(source);
CREATE INDEX idx_premade_decks_category ON premade_decks(category);
CREATE INDEX idx_premade_decks_language_code ON premade_decks(language_code);
CREATE INDEX idx_premade_decks_user_id ON premade_decks(user_id);
CREATE INDEX idx_premade_decks_is_published ON premade_decks(is_published);
CREATE INDEX idx_premade_decks_rating_avg ON premade_decks(rating_avg DESC);

CREATE TABLE premade_info_objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    premade_deck_id UUID NOT NULL REFERENCES premade_decks(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    discipline VARCHAR(50) NOT NULL DEFAULT 'programming',
    content_type VARCHAR(50) NOT NULL DEFAULT 'text',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_premade_info_objects_deck_id ON premade_info_objects(premade_deck_id);

CREATE TABLE premade_cards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    premade_info_object_id UUID NOT NULL REFERENCES premade_info_objects(id) ON DELETE CASCADE,
    front TEXT NOT NULL,
    step INT NOT NULL DEFAULT 0,
    correct_answers JSONB NOT NULL DEFAULT '[]'::jsonb,
    distractors JSONB NOT NULL DEFAULT '[]'::jsonb,
    card_type VARCHAR(32) NOT NULL DEFAULT 'concept',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_premade_cards_card_type
        CHECK (card_type IN ('concept', 'signature', 'trace', 'line_order', 'choose_snippet', 'fix_bug'))
);

CREATE INDEX idx_premade_cards_info_object_id ON premade_cards(premade_info_object_id);
CREATE INDEX idx_premade_cards_step ON premade_cards(premade_info_object_id, step);
CREATE INDEX idx_premade_cards_card_type ON premade_cards(card_type);

CREATE TABLE premade_deck_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    premade_deck_id UUID NOT NULL REFERENCES premade_decks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score SMALLINT NOT NULL CHECK (score BETWEEN 1 AND 5),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (premade_deck_id, user_id)
);

CREATE INDEX idx_premade_deck_ratings_deck_id ON premade_deck_ratings(premade_deck_id);
