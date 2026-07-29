-- +goose Up
-- +goose StatementBegin
ALTER TABLE games
ADD COLUMN coming_soon bool NOT NULL DEFAULT false,
ADD COLUMN release_date date;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE games
DROP COLUMN coming_soon,
DROP COLUMN release_date;
-- +goose StatementEnd
