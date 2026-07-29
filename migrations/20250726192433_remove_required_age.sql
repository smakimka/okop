-- +goose Up
-- +goose StatementBegin
ALTER TABLE games
DROP COLUMN required_age;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE games
ADD COLUMN required_age INT NOT NULL DEFAULT 0;
-- +goose StatementEnd