-- +goose Up
ALTER TABLE user_preferences
    ADD COLUMN page_size INTEGER NOT NULL DEFAULT 5
        CHECK (page_size IN (5, 10, 25, 50, 100));

-- +goose Down
ALTER TABLE user_preferences DROP COLUMN page_size;
