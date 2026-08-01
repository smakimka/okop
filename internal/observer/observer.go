package observer

import (
	"context"
	"fmt"
	"okop/pkg/logger"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// reportInterval is how often the non-interactive reporter logs a summary line.
const reportInterval = 30 * time.Second

// proxyStats counts requests — not apps — made through a single proxy, so the
// numbers stay meaningful as a measure of proxy health.
type proxyStats struct {
	success atomic.Int64
	fail    atomic.Int64
}

var (
	// startNanos is the run start as a unix nano timestamp. Stored atomically
	// because the reporter and the status bar read it from other goroutines.
	startNanos atomic.Int64

	Total        atomic.Int64
	OK           atomic.Int64
	SuccessFalse atomic.Int64
	Missing      atomic.Int64
	Failed       atomic.Int64

	statsByProxy sync.Map // string -> *proxyStats
)

func getOrCreateStats(proxyName string) *proxyStats {
	v, _ := statsByProxy.LoadOrStore(proxyName, &proxyStats{})
	return v.(*proxyStats)
}

func Init() {
	startNanos.Store(0)
	Total.Store(0)
	OK.Store(0)
	SuccessFalse.Store(0)
	Missing.Store(0)
	Failed.Store(0)
	statsByProxy = sync.Map{}
}

func ObserveStart() {
	startNanos.Store(time.Now().UnixNano())
}

func SetTotal(val int) {
	Total.Store(int64(val))
}

// ObserveSuccess records one successful request through proxyName that yielded
// apps upserted games.
func ObserveSuccess(proxyName string, apps int) {
	getOrCreateStats(proxyName).success.Add(1)
	OK.Add(int64(apps))
}

// ObserveFail records one failed request through proxyName. The apps are not
// counted as processed — they go back on the retry queue.
func ObserveFail(proxyName string) {
	getOrCreateStats(proxyName).fail.Add(1)
}

// ObserveSuccessFalse records n apps Steam returned but marked unavailable.
func ObserveSuccessFalse(n int) {
	SuccessFalse.Add(int64(n))
}

// ObserveMissing records n apps that were requested but absent from the
// response entirely. Without this they would never reach a terminal state in
// the counters and progress could not reach 100%.
func ObserveMissing(n int) {
	Missing.Add(int64(n))
}

// ObserveUpsertFailed records n apps that were fetched but could not be stored.
// They are not retried, so they are terminal and must not be counted as OK.
func ObserveUpsertFailed(n int) {
	Failed.Add(int64(n))
}

// ProxyStat is a per-proxy request tally.
type ProxyStat struct {
	Name string
	OK   int64
	Fail int64
}

// Stats is a point-in-time view of the run. Every consumer — the status bar and
// the non-interactive reporter — renders from this, so the numbers agree.
type Stats struct {
	Total        int64
	OK           int64
	SuccessFalse int64
	Missing      int64
	Failed       int64

	// Processed is the number of apps that have reached a terminal state.
	// It is OK + SuccessFalse + Missing + Failed and never exceeds Total.
	Processed int64

	Elapsed time.Duration
	Rate    float64 // apps per second
	ETA     time.Duration

	Proxies []ProxyStat
}

// Percent returns progress in the range [0, 100].
func (s Stats) Percent() float64 {
	if s.Total <= 0 {
		return 0
	}
	return 100 * float64(s.Processed) / float64(s.Total)
}

func Snapshot() Stats {
	s := Stats{
		Total:        Total.Load(),
		OK:           OK.Load(),
		SuccessFalse: SuccessFalse.Load(),
		Missing:      Missing.Load(),
		Failed:       Failed.Load(),
	}
	s.Processed = s.OK + s.SuccessFalse + s.Missing + s.Failed

	statsByProxy.Range(func(k, v any) bool {
		p := v.(*proxyStats)
		s.Proxies = append(s.Proxies, ProxyStat{
			Name: k.(string),
			OK:   p.success.Load(),
			Fail: p.fail.Load(),
		})
		return true
	})
	slices.SortFunc(s.Proxies, func(a, b ProxyStat) int {
		return strings.Compare(a.Name, b.Name)
	})

	if start := startNanos.Load(); start != 0 {
		s.Elapsed = time.Since(time.Unix(0, start))
	}
	// Below a second of samples the rate is dominated by startup noise and
	// produces absurd figures, so leave it and the ETA unset until then.
	if s.Elapsed >= time.Second && s.Processed > 0 {
		s.Rate = float64(s.Processed) / s.Elapsed.Seconds()
		if remaining := s.Total - s.Processed; remaining > 0 {
			s.ETA = time.Duration(float64(remaining) / s.Rate * float64(time.Second))
		}
	}

	return s
}

// SendObservations starts a background goroutine that logs a status line
// periodically. It is the fallback for non-interactive output, where the status
// bar cannot be drawn. pendingRetry is called to include the retry queue depth.
func SendObservations(ctx context.Context, pendingRetry func() int) {
	go func() {
		l := logger.Global()
		l.Info("started observing")
		ticker := time.NewTicker(reportInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s := Snapshot()

				proxyLines := make([]string, 0, len(s.Proxies))
				for _, p := range s.Proxies {
					proxyLines = append(proxyLines, fmt.Sprintf("%s(ok=%d fail=%d)", p.Name, p.OK, p.Fail))
				}

				l.Info("progress",
					"processed", s.Processed,
					"total", s.Total,
					"ok", s.OK,
					"success_false", s.SuccessFalse,
					"missing", s.Missing,
					"failed", s.Failed,
					"retrying", pendingRetry(),
					"rate", fmt.Sprintf("%.0f/s", s.Rate),
					"eta", s.ETA.Round(time.Second),
					"proxies", strings.Join(proxyLines, " "),
				)
			}
		}
	}()
}
