-- +goose Up
-- +goose StatementBegin
-- Convert expires_at column to TIMESTAMP WITH TIME ZONE to properly store UTC timestamps
ALTER TABLE urls 
ALTER COLUMN expires_at TYPE TIMESTAMP WITH TIME ZONE USING expires_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Convert back to TIMESTAMP (without timezone)
ALTER TABLE urls 
ALTER COLUMN expires_at TYPE TIMESTAMP USING expires_at AT TIME ZONE 'UTC';
-- +goose StatementEnd

