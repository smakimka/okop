package provider

import (
	"context"
	"okop/internal/repo"
	"okop/internal/work"
	"okop/pkg/logger"
	"slices"
	"sync"
)

type Provider struct {
	repo *repo.Repo
	apps []int
	wg   *sync.WaitGroup
}

func New(repo *repo.Repo) *Provider {
	return &Provider{
		repo: repo,
		wg:   &sync.WaitGroup{},
	}
}

func (p *Provider) SetApps(apps []int) {
	p.apps = apps
}

// Provide feeds apps into out. For each app sent, appWG.Add(1) is called before
// the send, so the caller's WaitGroup tracks all in-flight items end-to-end.
// Apps that are filtered out (not-a-game) are not added to appWG.
func (p *Provider) Provide(ctx context.Context, out chan<- work.Item, appWG *sync.WaitGroup) {
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		l := logger.Global()
		l.Info("started producing", "apps_num", len(p.apps))

		for chunk := range slices.Chunk(p.apps, 50) {
			appWG.Add(1)
			select {
			case out <- work.Item{Apps: chunk, Attempt: 0}:
			case <-ctx.Done():
				appWG.Done() // undo Add — item was never sent
				return
			}
		}
	}()
}

func (p *Provider) Wait() {
	p.wg.Wait()
}
