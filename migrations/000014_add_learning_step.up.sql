ALTER TABLE card_states
    ADD COLUMN learning_step INT NOT NULL DEFAULT 0;

UPDATE card_states
SET status = 'review'
WHERE status = 'learning' AND stability > 0;
