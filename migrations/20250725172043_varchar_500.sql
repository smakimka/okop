-- +goose Up
-- +goose StatementBegin
ALTER TABLE developers 
    ALTER COLUMN name TYPE varchar(500);

ALTER TABLE publishers 
    ALTER COLUMN name TYPE varchar(500);

ALTER TABLE categories 
    ALTER COLUMN name TYPE varchar(500);

ALTER TABLE genres 
    ALTER COLUMN name TYPE varchar(500);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE developers 
    ALTER COLUMN name TYPE varchar(100);

ALTER TABLE publishers 
    ALTER COLUMN name TYPE varchar(100);

ALTER TABLE categories 
    ALTER COLUMN name TYPE varchar(100);

ALTER TABLE genres 
    ALTER COLUMN name TYPE varchar(100);
-- +goose StatementEnd
