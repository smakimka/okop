package repo

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
)

type cache[K comparable, V any] struct {
	m        sync.RWMutex
	instance map[K]V
}

func newCache[K comparable, V any]() *cache[K, V] {
	return &cache[K, V]{
		m:        sync.RWMutex{},
		instance: map[K]V{},
	}
}

func (c *cache[K, V]) get(k K) (V, bool) {
	c.m.RLock()
	defer c.m.RUnlock()

	v, ok := c.instance[k]
	return v, ok
}

func (c *cache[K, V]) set(k K, v V) {
	c.m.Lock()
	defer c.m.Unlock()

	c.instance[k] = v
}

func (r *Repo) PopulateCaches(ctx context.Context) error {
	developers, err := r.GetDevelopers(ctx)
	if err != nil {
		return err
	}
	for _, developer := range developers {
		r.developersCache.set(developer.Name, developer.ID)
	}

	publishers, err := r.GetPublishers(ctx)
	if err != nil {
		return err
	}
	for _, publisher := range publishers {
		r.publishersCache.set(publisher.Name, publisher.ID)
	}

	return nil
}

func (r *Repo) GetDevelopers(ctx context.Context) ([]Developer, error) {
	query := `select id, name from developers`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	developers, err := pgx.CollectRows[Developer](rows, pgx.RowToStructByName)
	if err != nil {
		return nil, err
	}

	return developers, nil
}

func (r *Repo) GetPublishers(ctx context.Context) ([]Publisher, error) {
	query := `select id, name from publishers`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	publishers, err := pgx.CollectRows[Publisher](rows, pgx.RowToStructByName)
	if err != nil {
		return nil, err
	}

	return publishers, nil
}

func (r *Repo) GetGames(ctx context.Context) ([]Game, error) {
	query := `select * from games`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	games, err := pgx.CollectRows[Game](rows, pgx.RowToStructByName)
	if err != nil {
		return nil, err
	}

	return games, nil
}

func (r *Repo) GetTags(ctx context.Context) ([]Tag, error) {
	query := `select * from tags`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	tags, err := pgx.CollectRows[Tag](rows, pgx.RowToStructByName)
	if err != nil {
		return nil, err
	}

	return tags, nil
}
