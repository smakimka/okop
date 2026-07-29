package main

import (
	"context"
	"encoding/json"
	"fmt"
	"okop/internal/repo"
	"os"
	"os/signal"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	ID           int    `json:"appid"`
	Name         string `json:"name"`
	LastModified int    `json:"last_modified"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// dbURL := "postgres://parser:parser_password@localhost:30126/parser_db?sslmode=disable"
	dbConfig, err := pgxpool.ParseConfig("postgres://parser:parser_password@localhost:5432/parser_db?sslmode=disable")
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		panic(err)
	}

	data, err := os.ReadFile("games_appid.json")
	if err != nil {
		panic(err)
	}

	apps := make([]App, 0, 200_000)
	if err := json.Unmarshal(data, &apps); err != nil {
		panic(err)
	}

	fmt.Println(len(apps))

	r := repo.New(pool)

	for ch := range slices.Chunk(apps, 1_000) {
		chunk := make([]repo.App, 0, len(ch))
		for _, app := range ch {
			chunk = append(chunk, repo.App{
				SteamID:      app.ID,
				Name:         app.Name,
				LastModified: time.Unix(int64(app.LastModified), 0),
			})
		}

		if err := r.InsertApps(ctx, chunk); err != nil {
			panic(err)
		}
	}
}
