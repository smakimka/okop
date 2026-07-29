-- +goose Up
-- +goose StatementBegin
ALTER TABLE status_fail ADD COLUMN count int DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE status_fail DROP COLUMN count;
-- +goose StatementEnd