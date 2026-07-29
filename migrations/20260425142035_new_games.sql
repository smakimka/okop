-- +goose Up
drop table if exists games cascade;
drop table if exists categories cascade;
drop table if exists genres cascade;

alter table developers add constraint developers_name_key unique (name);
alter table publishers add constraint publishers_name_key unique (name);

create table games (
    id serial primary key,
    steam_id int not null unique,
    name text not null,
    store_url_path text not null,
    is_free bool not null,
    windows bool not null,
    steamos_linux bool not null,
    mac bool not null,
    reviews_filtered_count integer not null,
    reviews_filtered_percent_positive integer not null,
    reviews_filtered_score_label varchar(100) not null,
    reviews_unfiltered_count integer not null,
    reviews_unfiltered_percent_positive integer not null,
    reviews_unfiltered_score_label varchar(100) not null,
    assets jsonb not null,
    release_steam_date timestamptz not null,
    release_original_steam_date timestamptz not null
);

create table tags (
    id integer primary key,
    name varchar(100)
);

create table game_tags (
    game_id integer not null references games(id) on delete cascade,
    tag_id  integer not null references tags(id)  on delete cascade,
    weight  integer not null,
    primary key (game_id, tag_id)
);
create index on game_tags (tag_id);

create table game_developers (
    game_id      integer not null references games(id)      on delete cascade,
    developer_id integer not null references developers(id) on delete cascade,
    primary key (game_id, developer_id)
);
create index on game_developers (developer_id);

create table game_publishers (
    game_id      integer not null references games(id)      on delete cascade,
    publisher_id integer not null references publishers(id) on delete cascade,
    primary key (game_id, publisher_id)
);
create index on game_publishers (publisher_id);

-- +goose Down
drop table if exists game_publishers;
drop table if exists game_developers;
drop table if exists game_tags;
drop table if exists publishers;
drop table if exists developers;
drop table if exists tags;
drop table if exists games;
