DROP INDEX IF EXISTS idx_cards_card_type;

ALTER TABLE cards
DROP CONSTRAINT IF EXISTS chk_cards_card_type;

ALTER TABLE cards
DROP COLUMN IF EXISTS card_type;
