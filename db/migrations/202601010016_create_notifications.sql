-- +goose Up
CREATE TABLE notifications (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    recipient_public_id TEXT NOT NULL REFERENCES users (public_id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ
);

CREATE INDEX notifications_recipient_active_idx
    ON notifications (recipient_public_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
