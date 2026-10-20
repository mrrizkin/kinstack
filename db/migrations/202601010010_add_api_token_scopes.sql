-- +goose Up
-- Token scopes are an allowlist of permission codes. The effective permission
-- of a bearer request is the intersection of the owner's permissions and these
-- scopes; an empty array grants nothing.
ALTER TABLE api_tokens ADD COLUMN scopes TEXT[] NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE api_tokens DROP COLUMN scopes;
