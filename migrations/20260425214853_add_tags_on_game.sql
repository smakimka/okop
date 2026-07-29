-- +goose Up
alter table games add column tag_ids int[] not null default '{}';
update games g set tag_ids = coalesce(
    (select array_agg(tag_id order by tag_id) from game_tags where game_id = g.id),
    '{}'
);
create index games_tag_ids_gin on games using gin (tag_ids);

-- +goose Down
drop index games_tag_ids_gin;
alter table games drop column tag_ids;
