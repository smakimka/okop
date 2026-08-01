package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"okop/internal/config"
	"okop/internal/observer"
	"okop/internal/provider"
	"okop/internal/repo"
	"okop/internal/retry_queue"
	"okop/internal/ui"
	"okop/internal/work"
	"okop/internal/worker"
	"okop/pkg/logger"
	"okop/pkg/proxy_pool"
	"os"
	"os/signal"
	"sync"
	"time"

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

	poolCfg, err := config.ParsePoolConfig(cfg)
	if err != nil {
		panic(err)
	}

	proxyPool, err := proxy_pool.New(poolCfg)
	if err != nil {
		panic(err)
	}

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

	// On a terminal, pin a status bar to the bottom and route log output through
	// it. Otherwise fall back to periodic summary lines, which is what piped
	// output and `docker compose up` get.
	if ui.IsTerminal(os.Stderr) {
		bar := ui.New(os.Stderr, func(width int) []string {
			return ui.StatusLines(observer.Snapshot(), queue.Len(), width)
		})
		logger.Configure(bar.LogWriter(), slog.LevelInfo)

		bar.Run()
		defer bar.Stop()
	} else {
		observer.SendObservations(ctx, queue.Len)
	}

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
		summarize("all apps processed")
	case <-ctx.Done():
		summarize("interrupted")
	}
}

// summarize logs the final tally. It runs before the deferred bar.Stop, so the
// line is written through the bar's writer and lands above it.
func summarize(msg string) {
	s := observer.Snapshot()
	logger.Global().Info(msg,
		"processed", s.Processed,
		"total", s.Total,
		"ok", s.OK,
		"success_false", s.SuccessFalse,
		"missing", s.Missing,
		"failed", s.Failed,
		"elapsed", s.Elapsed.Round(time.Second),
	)
}
