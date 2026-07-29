-- +goose Up
-- +goose StatementBegin
CREATE TABLE id_remap (
    appid int NOT NULL,
    steam_appid int NOT NULL,
    PRIMARY KEY (appid, steam_appid)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS id_remap;
-- +goose StatementEnd
