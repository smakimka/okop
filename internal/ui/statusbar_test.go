package ui

import (
	"bytes"
	"fmt"
	"okop/internal/observer"
	"strings"
	"testing"
	"time"
)

// TestCountersReconcile pins the invariant the status bar depends on: every app
// lands in exactly one terminal bucket, so progress ends at exactly 100% and
// never overshoots.
func TestCountersReconcile(t *testing.T) {
	observer.Init()
	observer.ObserveStart()

	// seven full chunks plus a short final one, mirroring slices.Chunk(apps, 50)
	sizes := []int{50, 50, 50, 50, 50, 50, 50, 23}
	total := 0
	for _, s := range sizes {
		total += s
	}
	observer.SetTotal(total)

	for i, size := range sizes {
		bad, missing, failed := 0, 0, 0
		if i%2 == 0 {
			bad = 3
		}
		if i%3 == 0 {
			missing = 2
		}
		if i == 1 {
			failed = 1
		}

		observer.ObserveMissing(missing)
		observer.ObserveSuccessFalse(bad)
		observer.ObserveUpsertFailed(failed)
		observer.ObserveSuccess("alpha", size-bad-missing-failed)
	}

	s := observer.Snapshot()
	if sum := s.OK + s.SuccessFalse + s.Missing + s.Failed; sum != int64(total) {
		t.Fatalf("buckets sum to %d, want %d", sum, total)
	}
	if s.Processed != int64(total) {
		t.Fatalf("processed = %d, want %d", s.Processed, total)
	}
	if got := s.Percent(); got != 100 {
		t.Fatalf("percent = %v, want exactly 100", got)
	}
	if s.ETA != 0 {
		t.Errorf("ETA on a finished run = %v, want 0", s.ETA)
	}

	// a failed request is retried, so it must not advance app progress
	before := observer.Snapshot().Processed
	observer.ObserveFail("alpha")
	if after := observer.Snapshot().Processed; after != before {
		t.Errorf("ObserveFail moved progress: %d -> %d", before, after)
	}

	// proxy tallies count requests, not apps
	if p := observer.Snapshot().Proxies; len(p) != 1 || p[0].OK != 8 || p[0].Fail != 1 {
		t.Errorf("proxy tally = %+v, want one entry ok=8 fail=1", p)
	}
}

func TestStatusLinesFitWidth(t *testing.T) {
	s := observer.Stats{
		Total: 120000, OK: 74102, SuccessFalse: 778, Processed: 74880,
		Elapsed: time.Hour, Rate: 1240, ETA: 37 * time.Minute,
		Proxies: []observer.ProxyStat{{Name: "alpha", OK: 1502, Fail: 12}},
	}

	for _, width := range []int{40, 60, 80, 120, 200} {
		for _, line := range StatusLines(s, 12, width) {
			if got := len([]rune(truncate(line, width))); got > width {
				t.Errorf("width %d: line runs to %d runes", width, got)
			}
		}
	}
}

// A run that has not started yet must not divide by zero or render a bar that
// looks like progress.
func TestStatusLinesZeroState(t *testing.T) {
	lines := StatusLines(observer.Stats{Total: 100}, 0, 80)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (no proxies yet)", len(lines))
	}
	if !strings.Contains(lines[0], "0.0%") {
		t.Errorf("zero state should read 0.0%%: %q", lines[0])
	}
	if strings.ContainsRune(lines[0], gaugeFull) {
		t.Errorf("zero state should have an empty gauge: %q", lines[0])
	}
}

func TestBarEraseRepaint(t *testing.T) {
	var buf bytes.Buffer
	lines := 3
	b := New(&buf, func(int) []string {
		out := make([]string, lines)
		for i := range out {
			out[i] = fmt.Sprintf("line%d", i)
		}
		return out
	})

	b.mu.Lock()
	b.paint()
	b.mu.Unlock()
	if b.painted != 3 {
		t.Fatalf("painted = %d, want 3", b.painted)
	}

	buf.Reset()
	if _, err := b.LogWriter().Write([]byte("hello from slog\n")); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	// clearing three lines means one clear plus two up-and-clear
	if up := strings.Count(got, cursorUp); up != 2 {
		t.Errorf("cursor-up count = %d, want 2", up)
	}
	if !strings.Contains(got, "hello from slog\n") {
		t.Errorf("log text missing: %q", got)
	}
	if i := strings.Index(got, "hello"); i == -1 || !strings.Contains(got[i:], "line0") {
		t.Errorf("bar not repainted below the log line: %q", got)
	}
	if b.painted != 3 {
		t.Errorf("painted after log = %d, want 3", b.painted)
	}

	// shrinking must not strand the lines that are no longer rendered
	lines = 1
	b.mu.Lock()
	b.erase()
	b.paint()
	b.mu.Unlock()
	if b.painted != 1 {
		t.Errorf("painted after shrink = %d, want 1", b.painted)
	}

	b.Stop()
	if b.painted != 0 {
		t.Errorf("painted after Stop = %d, want 0", b.painted)
	}
	if !strings.HasSuffix(buf.String(), showCursor) {
		t.Error("Stop must restore the cursor")
	}
}

// Stop is deferred in cmd/parser, so a second call must be harmless.
func TestStopIsIdempotent(t *testing.T) {
	b := New(&bytes.Buffer{}, func(int) []string { return []string{"x"} })
	b.Run()
	b.Stop()
	b.Stop()
}

func TestComma(t *testing.T) {
	cases := map[int64]string{
		0: "0", 999: "999", 1000: "1,000", 74880: "74,880",
		1234567: "1,234,567", -4321: "-4,321",
	}
	for in, want := range cases {
		if got := comma(in); got != want {
			t.Errorf("comma(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                               "-",
		45 * time.Second:                "45s",
		37*time.Minute + 12*time.Second: "37m12s",
		62*time.Minute + 12*time.Second: "1h02m",
		3*time.Hour + 4*time.Minute:     "3h04m",
	}
	for in, want := range cases {
		if got := duration(in); got != want {
			t.Errorf("duration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateKeepsRunesIntact(t *testing.T) {
	if got := truncate("███abc", 4); got != "███a" {
		t.Errorf("truncate = %q, want %q", got, "███a")
	}
	if got := truncate("abc", 10); got != "abc" {
		t.Errorf("truncate should leave short strings alone, got %q", got)
	}
}
