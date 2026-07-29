-- +goose Up
-- +goose StatementBegin
CREATE TABLE apps_list (
    id serial primary key,
    steam_id int NOT NULL,
    name VARCHAR(500) NOT NULL,
    last_modified timestamp NOT NULL,
    last_parsed timestamp
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS apps_list;
-- +goose StatementEnd
