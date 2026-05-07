ALTER TABLE cards DROP CONSTRAINT chk_cards_card_type;

ALTER TABLE cards
ADD CONSTRAINT chk_cards_card_type
CHECK (card_type IN ('concept', 'signature', 'trace', 'line_order', 'choose_snippet', 'fix_bug'));
