package main

import (
	"context"
	"fmt"
	"okop/internal/config"
	"okop/internal/repo"
	"okop/pkg/steam_proxy"
	"os"
	"os/signal"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	pgCfg, err := config.ParsePostgres()
	if err != nil {
		panic(err)
	}
	dbConfig, err := pgxpool.ParseConfig(pgCfg.DSN())
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		panic(err)
	}

	r := repo.New(pool)

	client, err := steam_proxy.New(steam_proxy.Config{
		Name: "test",
		URL:  "",
	})
	if err != nil {
		panic(err)
	}

	res, err := client.Tags(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println("got tags ", len(res.Response.Tags))

	for _, tag := range res.Response.Tags {
		if err := r.UpdateTag(ctx, repo.Tag{
			ID:   tag.TagID,
			Name: tag.Name,
		}); err != nil {
			fmt.Println("failed to update tag", tag.TagID, err)
		}
	}

	if err := backfillUnnamed(ctx, r, client); err != nil {
		panic(err)
	}
}

// resolveBatch caps how many ids go into one GetLocalizedNameForTags call, since
// the request travels as a query parameter.
const resolveBatch = 200

// backfillUnnamed names tags the parser created from a game reference but which
// GetTagList does not cover. Steam drops tags from the store vocabulary while
// still returning them on games, so those ids are never named by the loop above
// no matter how often it runs. GetLocalizedNameForTags resolves them by id.
func backfillUnnamed(ctx context.Context, r *repo.Repo, client *steam_proxy.Client) error {
	ids, err := r.UnnamedTagIDs(ctx)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	fmt.Println("resolving unnamed tags", len(ids))

	resolved := make(map[int]bool, len(ids))
	for chunk := range slices.Chunk(ids, resolveBatch) {
		res, err := client.TagNames(ctx, chunk)
		if err != nil {
			return err
		}

		for _, tag := range res.Response.Tags {
			if tag.Name == "" {
				continue
			}

			if err := r.UpdateTag(ctx, repo.Tag{
				ID:   tag.TagID,
				Name: tag.Name,
			}); err != nil {
				fmt.Println("failed to update tag", tag.TagID, err)
				continue
			}

			resolved[tag.TagID] = true
			fmt.Println("resolved tag", tag.TagID, tag.Name)
		}
	}

	// Anything Steam will not name stays null; report it rather than leaving it
	// to be rediscovered by hand.
	for _, id := range ids {
		if !resolved[id] {
			fmt.Println("still unnamed, steam returned no name for tag", id)
		}
	}

	return nil
}
