package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"okop/pkg/steam_proxy"
)

// A spread of well-known Steam app IDs to test against.
var testAppIDs = []int{
	730,     // CS2
	570,     // Dota 2
	440,     // TF2
	271590,  // GTA V
	1091500, // Cyberpunk 2077
	1172470, // Apex Legends
	578080,  // PUBG
	292030,  // The Witcher 3
	230410,  // Warframe
	252950,  // Rocket League
}

type reqResult struct {
	appID    int
	latency  time.Duration
	attempts int
	err      error
}

type proxyStats struct {
	name     string
	proxyURL string
	reqs     []reqResult
}

func runProxy(cfg steam_proxy.Config) proxyStats {
	timeout := 10 * time.Second

	client, err := steam_proxy.New(steam_proxy.Config{
		Name: cfg.Name,
		URL:  cfg.URL,
	})
	if err != nil {
		return proxyStats{name: cfg.Name, reqs: []reqResult{{err: err}}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	reqs := make([]reqResult, len(testAppIDs))
	for i, appID := range testAppIDs {
		start := time.Now()
		_, reqErr := client.GameInfos(ctx, []int{appID})
		reqs[i] = reqResult{
			appID:   appID,
			latency: time.Since(start),
			err:     reqErr,
		}
	}

	return proxyStats{name: cfg.Name, reqs: reqs}
}

func printStats(s proxyStats) {
	var (
		successes     int
		rateLimited   int
		totalLatency  time.Duration
		minLatency    = time.Duration(math.MaxInt64)
		maxLatency    time.Duration
		totalAttempts int
	)

	for _, r := range s.reqs {
		totalAttempts += r.attempts
		if r.err == nil {
			successes++
			totalLatency += r.latency
			if r.latency < minLatency {
				minLatency = r.latency
			}
			if r.latency > maxLatency {
				maxLatency = r.latency
			}
		}
	}

	n := len(s.reqs)
	successPct := float64(successes) / float64(n) * 100

	fmt.Printf("=== %s (%s) ===\n", s.name, s.proxyURL)
	fmt.Printf("  Success:   %d/%d  (%.0f%%)\n", successes, n, successPct)
	if rateLimited > 0 {
		fmt.Printf("  429s:      %d\n", rateLimited)
	}
	if successes > 0 {
		avg := (totalLatency / time.Duration(successes)).Round(time.Millisecond)
		fmt.Printf("  Latency:   avg %-8s  min %-8s  max %s\n",
			avg, minLatency.Round(time.Millisecond), maxLatency.Round(time.Millisecond))
	}
	fmt.Printf("  Attempts:  %d total  (avg %.2f/req)\n", totalAttempts, float64(totalAttempts)/float64(n))

	fmt.Printf("\n  %-10s  %-10s  %-8s  %s\n", "APP_ID", "LATENCY", "ATTEMPTS", "RESULT")
	fmt.Printf("  %s\n", strings.Repeat("-", 46))
	for _, r := range s.reqs {
		lat := r.latency.Round(time.Millisecond)
		if r.err != nil {
			fmt.Printf("  %-10d  %-10s  %-8d  FAIL — %v\n", r.appID, lat, r.attempts, r.err)
		} else {
			fmt.Printf("  %-10d  %-10s  %-8d  OK\n", r.appID, lat, r.attempts)
		}
	}

	fmt.Println()
}

func main() {
	cfgFile := flag.String("config", "proxies.json", "path to proxy config file")
	flag.Parse()

	data, err := os.ReadFile(*cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading config: %v\n", err)
		os.Exit(1)
	}

	type config struct {
		Proxies []steam_proxy.Config `json:"proxies"`
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing config: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.Proxies) == 0 {
		fmt.Fprintln(os.Stderr, "no proxies found in config")
		os.Exit(1)
	}

	fmt.Printf("Testing %d proxy(s), %d requests /req\n\n",
		len(cfg.Proxies), len(testAppIDs))

	stats := make([]proxyStats, len(cfg.Proxies))
	var wg sync.WaitGroup
	for i, p := range cfg.Proxies {
		wg.Add(1)
		go func(i int, p steam_proxy.Config) {
			defer wg.Done()
			stats[i] = runProxy(p)
			stats[i].proxyURL = p.URL // stored for display; caller can redact if needed
		}(i, p)
	}
	wg.Wait()

	for _, s := range stats {
		printStats(s)
	}
}
