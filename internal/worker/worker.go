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

	for _, info := range info.Response.StoreItems {
		if info.Success == 0 || !info.Visible {
			id := strconv.Itoa(int(info.AppID))
			l.Error("success false", "id", id)
			observer.ObserveSuccessFalse()
			continue
		}

		if err := w.repo.UpsertGame(ctx, &info); err != nil {
			l.Error("upsert failed", "app_id", info.AppID, "error", err)
		}
	}

	observer.ObserveSuccess(handle.Name())
	l.Debug("success", "proxy", handle.Name())
	w.wg.Done()
}
