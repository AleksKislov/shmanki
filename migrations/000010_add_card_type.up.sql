ALTER TABLE cards
ADD COLUMN card_type VARCHAR(32) NOT NULL DEFAULT 'concept';

ALTER TABLE cards
ADD CONSTRAINT chk_cards_card_type
CHECK (card_type IN ('concept', 'signature', 'trace', 'line_order', 'block_order', 'choose_snippet', 'fix_bug'));

CREATE INDEX idx_cards_card_type ON cards(card_type);
