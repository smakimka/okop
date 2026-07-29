package observer

import (
	"context"
	"fmt"
	"okop/pkg/logger"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type proxyStats struct {
	success atomic.Int64
	fail    atomic.Int64
}

var (
	startTime time.Time

	Total        atomic.Int64
	SuccessFalse atomic.Int64
	Skipped      atomic.Int64

	statsByProxy sync.Map // string -> *proxyStats
)

func getOrCreateStats(proxyName string) *proxyStats {
	v, _ := statsByProxy.LoadOrStore(proxyName, &proxyStats{})
	return v.(*proxyStats)
}

func Init() {
	startTime = time.Time{}
	Total = atomic.Int64{}
	SuccessFalse = atomic.Int64{}
	Skipped = atomic.Int64{}
	statsByProxy = sync.Map{}
}

func ObserveStart() {
	startTime = time.Now()
}

func SetTotal(val int) {
	Total.Store(int64(val))
}

func ObserveSuccess(proxyName string) {
	s := getOrCreateStats(proxyName)
	s.success.Add(50)
}

func ObserveFail(proxyName string) {
	getOrCreateStats(proxyName).fail.Add(50)
}

func ObserveSuccessFalse() {
	SuccessFalse.Add(1)
}

func ObserveSkip() {
	Skipped.Add(1)
}

// SendObservations starts a background goroutine that logs a status update
// every 5 minutes. pendingRetry is called to include the retry queue depth in
// the status message.
func SendObservations(ctx context.Context, pendingRetry func() int) {
	go func() {
		l := logger.Global()
		l.Info("started observing")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var totalSuccess, totalFail int64
				var proxyLines []string

				statsByProxy.Range(func(k, v any) bool {
					name := k.(string)
					s := v.(*proxyStats)
					ok := s.success.Load()
					fail := s.fail.Load()
					totalSuccess += ok
					totalFail += fail
					proxyLines = append(proxyLines, fmt.Sprintf("%s(ok=%d fail=%d)", name, ok, fail))
					return true
				})

				processed := totalSuccess + Skipped.Load() + SuccessFalse.Load()
				total := Total.Load()

				var eta time.Duration
				if processed > 0 && time.Since(startTime) > 0 {
					eta = time.Duration(float64(total-totalSuccess) / (float64(processed) / float64(time.Since(startTime))))
				}

				l.Info("progress",
					"processed", processed,
					"total", total,
					"skipped", Skipped.Load(),
					"success_false", SuccessFalse.Load(),
					"retrying", pendingRetry(),
					"ok", totalSuccess,
					"fail", totalFail,
					"eta", eta,
					"proxies", strings.Join(proxyLines, " "),
				)
			}
		}
	}()
}
