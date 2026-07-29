package main

import (
	"context"
	"encoding/json"
	"okop/internal/config"
	"okop/internal/observer"
	"okop/internal/provider"
	"okop/internal/repo"
	"okop/internal/retry_queue"
	"okop/internal/work"
	"okop/internal/worker"
	"okop/pkg/logger"
	"okop/pkg/proxy_pool"
	"os"
	"os/signal"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfgData, err := os.ReadFile("proxies.json")
	if err != nil {
		panic(err)
	}
	var cfg config.PoolConfig
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		panic(err)
	}

	config, err := config.ParsePoolConfig(cfg)
	if err != nil {
		panic(err)
	}

	proxyPool, err := proxy_pool.New(config)
	if err != nil {
		panic(err)
	}

	dbConfig, err := pgxpool.ParseConfig("postgres://parser:parser_password@localhost:5432/parser_db?sslmode=disable")
	if err != nil {
		panic(err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		panic(err)
	}

	r := repo.New(pool)
	if err := r.PopulateCaches(ctx); err != nil {
		panic(err)
	}

	appIDs, err := r.GetAllGameSteamIDs(ctx)
	if err != nil {
		panic(err)
	}

	observer.Init()
	observer.SetTotal(len(appIDs))
	observer.ObserveStart()

	queue := retry_queue.New()
	workCh := make(chan work.Item, len(cfg.Proxies)*4)
	var appWG sync.WaitGroup

	go queue.Run(ctx, workCh)
	observer.SendObservations(ctx, queue.Len)

	for range len(cfg.Proxies) {
		w := worker.New(proxyPool, r, queue, &appWG)
		w.Run(ctx, workCh)
	}

	prov := provider.New(r)
	prov.SetApps(appIDs)
	prov.Provide(ctx, workCh, &appWG)
	prov.Wait()

	// Wait for all apps to reach a terminal state, or for the user to interrupt.
	done := make(chan struct{})
	go func() {
		appWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Global().Info("all apps processed")
	case <-ctx.Done():
		logger.Global().Info("interrupted")
	}
}
