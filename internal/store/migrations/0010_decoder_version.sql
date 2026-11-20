-- +goose Up
ALTER TABLE events ADD COLUMN IF NOT EXISTS decoder_version INT NOT NULL DEFAULT 1;
UPDATE events SET decoder_version = 1 WHERE decoder_version = 0;

-- +goose Down
ALTER TABLE events DROP COLUMN IF EXISTS decoder_version;
