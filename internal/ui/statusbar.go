// Package ui renders a status bar pinned to the bottom of the terminal while
// log lines scroll above it.
package ui

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	repaintInterval = 250 * time.Millisecond

	hideCursor = "\x1b[?25l"
	showCursor = "\x1b[?25h"
	clearLine  = "\r\x1b[K"
	cursorUp   = "\x1b[A"

	defaultWidth = 80
	maxWidth     = 200
)

// Bar paints a block of lines at the bottom of the terminal and keeps them
// there. Anything written through LogWriter is inserted above the block.
type Bar struct {
	w      io.Writer
	render func(width int) []string

	mu      sync.Mutex
	painted int // lines currently on screen

	stop     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// New returns a Bar that writes to w. render is called on each repaint and
// returns the lines to display, given the terminal width.
func New(w io.Writer, render func(width int) []string) *Bar {
	return &Bar{
		w:      w,
		render: render,
		stop:   make(chan struct{}),
	}
}

// IsTerminal reports whether f is a character device, i.e. whether drawing
// escape sequences to it makes sense. Piped or redirected output is not.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// Run starts repainting until ctx is done or Stop is called.
func (b *Bar) Run() {
	b.mu.Lock()
	fmt.Fprint(b.w, hideCursor)
	b.paint()
	b.mu.Unlock()

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()

		ticker := time.NewTicker(repaintInterval)
		defer ticker.Stop()

		for {
			select {
			case <-b.stop:
				return
			case <-ticker.C:
				b.mu.Lock()
				b.erase()
				b.paint()
				b.mu.Unlock()
			}
		}
	}()
}

// Stop halts repainting, clears the bar and restores the cursor. It is safe to
// call more than once, so it can be deferred and also called explicitly.
func (b *Bar) Stop() {
	b.stopOnce.Do(func() { close(b.stop) })
	b.wg.Wait()

	b.mu.Lock()
	defer b.mu.Unlock()
	b.erase()
	fmt.Fprint(b.w, showCursor)
}

// LogWriter returns a writer that prints above the bar. Give it to the logger
// so log output and the bar do not overwrite each other.
func (b *Bar) LogWriter() io.Writer {
	return logWriter{b}
}

type logWriter struct{ b *Bar }

func (lw logWriter) Write(p []byte) (int, error) {
	lw.b.mu.Lock()
	defer lw.b.mu.Unlock()

	// erase leaves the cursor at the start of the bar's first line, so the log
	// line lands exactly there and the bar is repainted below it.
	lw.b.erase()
	n, err := lw.b.w.Write(p)
	lw.b.paint()

	return n, err
}

// paint writes the bar and leaves the cursor at the end of its last line.
// Callers must hold b.mu.
func (b *Bar) paint() {
	width := terminalWidth()
	lines := b.render(width)

	for i, line := range lines {
		if i > 0 {
			fmt.Fprint(b.w, "\n")
		}
		fmt.Fprint(b.w, clearLine+truncate(line, width))
	}

	b.painted = len(lines)
}

// erase removes the painted lines and leaves the cursor at the start of where
// the first one was. Callers must hold b.mu.
func (b *Bar) erase() {
	if b.painted == 0 {
		return
	}

	fmt.Fprint(b.w, clearLine)
	for i := 1; i < b.painted; i++ {
		fmt.Fprint(b.w, cursorUp+clearLine)
	}

	b.painted = 0
}

// terminalWidth reports the usable width. COLUMNS is consulted so the bar can
// adapt without pulling in a terminal-size dependency; 80 is a safe default.
func terminalWidth() int {
	n, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || n < 20 {
		return defaultWidth
	}
	return min(n, maxWidth)
}

// truncate shortens s to at most width columns, counting runes rather than
// bytes so multi-byte block characters are not cut in half.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= width {
		return s
	}

	return string(runes[:width])
}

// comma formats n with thousands separators.
func comma(n int64) string {
	s := strconv.FormatInt(n, 10)
	if n < 0 {
		return "-" + comma(-n)
	}
	if len(s) <= 3 {
		return s
	}

	var out strings.Builder
	for i, d := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(d)
	}

	return out.String()
}

// duration renders d compactly: 1h02m, 37m12s, 45s.
func duration(d time.Duration) string {
	if d <= 0 {
		return "-"
	}

	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
