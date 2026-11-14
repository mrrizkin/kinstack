-- +goose Up
-- Personalization settings live outside the identity table so `users` only
-- carries account credentials and status.
CREATE TABLE user_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    theme_preference TEXT NOT NULL DEFAULT 'system'
        CHECK (theme_preference IN ('light', 'dark', 'system')),
    sound_enabled BOOLEAN NOT NULL DEFAULT true,
    page_size INTEGER NOT NULL DEFAULT 5
        CHECK (page_size IN (5, 10, 25, 50, 100)),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE user_preferences;
