package ui

import (
	"fmt"
	"okop/internal/observer"
	"strings"
)

const (
	gaugeFull  = '█'
	gaugeEmpty = '░'

	minGauge = 10
	maxGauge = 40
)

// StatusLines renders a run snapshot as the lines of the status bar. It is the
// render function to hand to New.
func StatusLines(s observer.Stats, retrying, width int) []string {
	tail := fmt.Sprintf(
		" %5.1f%%  %s / %s   %s   eta %s",
		s.Percent(), comma(s.Processed), comma(s.Total), rate(s.Rate), duration(s.ETA),
	)

	const head = " apps  "
	gauge := min(max(width-len([]rune(head))-len([]rune(tail)), minGauge), maxGauge)

	lines := []string{
		head + bar(s.Percent(), gauge) + tail,
		fmt.Sprintf(
			" ok %s   invisible %s   missing %s   failed %s   retry %s   elapsed %s",
			comma(s.OK), comma(s.SuccessFalse), comma(s.Missing), comma(s.Failed),
			comma(int64(retrying)), duration(s.Elapsed),
		),
	}

	if len(s.Proxies) > 0 {
		parts := make([]string, 0, len(s.Proxies))
		for _, p := range s.Proxies {
			parts = append(parts, fmt.Sprintf("%s ok=%s fail=%s", p.Name, comma(p.OK), comma(p.Fail)))
		}
		lines = append(lines, " proxies  "+strings.Join(parts, "   "))
	}

	return lines
}

// bar renders a gauge of the given width filled to percent.
func bar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	filled = min(max(filled, 0), width)

	return strings.Repeat(string(gaugeFull), filled) +
		strings.Repeat(string(gaugeEmpty), width-filled)
}

// rate renders apps per second, held to a sensible width.
func rate(perSecond float64) string {
	if perSecond <= 0 {
		return "   -/s"
	}
	return fmt.Sprintf("%s/s", comma(int64(perSecond)))
}
