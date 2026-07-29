package proxy_pool

import (
	"context"
	"errors"
	"okop/pkg/steam_proxy"
	"time"
)

type Timeouts struct {
	AppInfoMS time.Duration
}

type TrottleStrategy struct {
	Multi     int
	MinWaitMS time.Duration
	MaxWaitMS time.Duration
}

type Config struct {
	Proxies         []steam_proxy.Config
	Timeouts        Timeouts
	TrottleStrategy TrottleStrategy
}

type Handle struct {
	p          *Pool
	proxy      *steam_proxy.Client
	lastWaitMS time.Duration
}

type Pool struct {
	cfg       *Config
	available chan *Handle
}

func New(cfg *Config) (*Pool, error) {
	p := &Pool{
		cfg:       cfg,
		available: make(chan *Handle, len(cfg.Proxies)),
	}

	for _, proxyCfg := range cfg.Proxies {
		proxy, err := steam_proxy.New(proxyCfg)
		if err != nil {
			return nil, err
		}

		p.available <- &Handle{
			p:          p,
			proxy:      proxy,
			lastWaitMS: cfg.TrottleStrategy.MinWaitMS,
		}
	}

	return p, nil
}

func (p *Pool) GetHandle(ctx context.Context) (*Handle, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case h := <-p.available:
		return h, nil
	}
}

func (h *Handle) Name() string {
	return h.proxy.Name()
}

func (h *Handle) GetItems(ctx context.Context, appIDs []int) (*steam_proxy.GetItemsResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, h.p.cfg.Timeouts.AppInfoMS)
	defer cancel()

	nextWait := h.p.cfg.TrottleStrategy.MinWaitMS

	info, err := h.proxy.GameInfos(timeoutCtx, appIDs)
	// 3 для того чтобы после максимума снова стал минимум
	if err != nil && errors.Is(err, steam_proxy.ErrRateLimited) && h.lastWaitMS != h.p.cfg.TrottleStrategy.MaxWaitMS {
		nextWait = min(
			h.p.cfg.TrottleStrategy.MaxWaitMS,
			time.Duration(h.p.cfg.TrottleStrategy.Multi)*h.lastWaitMS,
		)
	}

	go func() {
		time.Sleep(nextWait)
		h.p.available <- h
	}()

	return info, err
}
