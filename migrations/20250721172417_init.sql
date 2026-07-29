-- +goose Up
-- +goose StatementBegin
create table developers (
    id serial primary key,
    name varchar(100)
);

create table publishers (
    id serial primary key,
    name varchar(100)
);

create table categories (
    id serial primary key,
    steam_id int unique,
    name varchar(100)
);

create table genres (
    id serial primary key,
    steam_id varchar(32) unique,
    name varchar(100)
);

create table games (
    id serial primary key,
    steam_id int not null unique,
    name varchar(500) not null,
    is_free bool not null,

    developers int[] not null,
    publishers int[] not null,

    windows bool not null,
    linux bool not null,
    mac bool not null,

    categories int[] not null,
    genres int[] not null,

    recommendations int not null,
    required_age int not null
);

create table not_games (
    id serial primary key,
    steam_id int not null unique,
    name varchar(500) not null,
    type varchar(100) not null
);

create table status_fail(
    id serial primary key,
    steam_id int not null unique,
    name varchar(500) not null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS games;
DROP TABLE IF EXISTS developers;
DROP TABLE IF EXISTS publishers;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS genres;
DROP TABLE IF EXISTS not_games;
DROP TABLE IF EXISTS status_fail;
-- +goose StatementEnd
