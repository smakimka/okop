package main

import (
	"context"
	"fmt"
	"okop/internal/config"
	"okop/internal/repo"
	"okop/pkg/steam_proxy"
	"os"
	"os/signal"

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
}
