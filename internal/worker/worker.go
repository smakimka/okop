package worker

import (
	"context"
	"okop/internal/observer"
	"okop/internal/repo"
	"okop/internal/retry_queue"
	"okop/internal/work"
	"okop/pkg/logger"
	"okop/pkg/proxy_pool"
	"strconv"
	"sync"
)

type Worker struct {
	pool  *proxy_pool.Pool
	repo  *repo.Repo
	queue *retry_queue.Queue
	wg    *sync.WaitGroup
}

func New(pool *proxy_pool.Pool, repo *repo.Repo, queue *retry_queue.Queue, wg *sync.WaitGroup) *Worker {
	return &Worker{
		pool:  pool,
		repo:  repo,
		queue: queue,
		wg:    wg,
	}
}

func (w *Worker) Run(ctx context.Context, in <-chan work.Item) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-in:
				if !ok {
					return
				}
				w.process(ctx, item)
			}
		}
	}()
}

func (w *Worker) process(ctx context.Context, item work.Item) {
	l := logger.Global()

	handle, err := w.pool.GetHandle(ctx)
	if err != nil {
		// ctx cancelled
		return
	}

	info, err := handle.GetItems(ctx, item.Apps)
	if err != nil {
		observer.ObserveFail(handle.Name())
		w.queue.Add(item)
		l.Debug("queued for retry", "app_ids", item.Apps, "proxy", handle.Name(), "error", err)
		return
	}

	// Every app in the chunk is terminal after this point — nothing here is
	// requeued — so each one lands in exactly one bucket and the buckets sum
	// back to len(item.Apps).
	var ok, bad, failed int
	for _, info := range info.Response.StoreItems {
		if info.Success == 0 || !info.Visible {
			id := strconv.Itoa(int(info.AppID))
			l.Error("success false", "id", id)
			bad++
			continue
		}

		if err := w.repo.UpsertGame(ctx, &info); err != nil {
			l.Error("upsert failed", "app_id", info.AppID, "error", err)
			failed++
			continue
		}
		ok++
	}

	// Steam may omit apps from the response entirely.
	if missing := len(item.Apps) - len(info.Response.StoreItems); missing > 0 {
		observer.ObserveMissing(missing)
	}
	observer.ObserveSuccessFalse(bad)
	observer.ObserveUpsertFailed(failed)
	observer.ObserveSuccess(handle.Name(), ok)

	w.wg.Done()
}
