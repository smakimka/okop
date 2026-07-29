package config

import (
	"okop/pkg/proxy_pool"
	"okop/pkg/steam_proxy"
	"time"
)

type Timeouts struct {
	AppInfo string `json:"app_info"`
}

type TrottleStrategy struct {
	Multi   int    `json:"multi"`
	MinWait string `json:"min_wait"`
	MaxWait string `json:"max_wait"`
}

type PoolConfig struct {
	Proxies         []steam_proxy.Config `json:"proxies"`
	Timeouts        Timeouts             `json:"timeouts"`
	TrottleStrategy TrottleStrategy      `json:"throttle_strategy"`
}

func ParsePoolConfig(cfg PoolConfig) (*proxy_pool.Config, error) {
	appInfo, err := time.ParseDuration(cfg.Timeouts.AppInfo)
	if err != nil {
		return nil, err
	}

	minWait, err := time.ParseDuration(cfg.TrottleStrategy.MinWait)
	if err != nil {
		return nil, err
	}

	maxWait, err := time.ParseDuration(cfg.TrottleStrategy.MaxWait)
	if err != nil {
		return nil, err
	}
	return &proxy_pool.Config{
		Proxies: cfg.Proxies,
		Timeouts: proxy_pool.Timeouts{
			AppInfoMS: appInfo,
		},
		TrottleStrategy: proxy_pool.TrottleStrategy{
			Multi:     cfg.TrottleStrategy.Multi,
			MinWaitMS: minWait,
			MaxWaitMS: maxWait,
		},
	}, nil
}
