DROP TABLE IF EXISTS premade_deck_ratings;
DROP TABLE IF EXISTS premade_cards;
DROP TABLE IF EXISTS premade_info_objects;
DROP TABLE IF EXISTS premade_decks;

ALTER TABLE users
DROP COLUMN IF EXISTS display_name;
