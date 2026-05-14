package backlog

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// VelocityASCII renders the per-iteration velocity chart as
// terminal-friendly text. Used by the CLI, the MCP `velocity_chart`
// tool, and embedded into `velocity.md` by sync.
//
// Output shape (heights scaled to the largest bucket):
//
//   Velocity (last 12 iterations of 1 week)
//
//     8 |                ▇
//     6 |          ▇  ▇  ▇
//     4 |    ▇  ▇  ▇  ▇  ▇  ▇
//     2 |  ▇ ▇  ▇  ▇  ▇  ▇  ▇
//     0 +---------------------
//        04/01 04/15 04/29 ...
//
// Two-character bar width keeps it readable in 80-col terminals and in
// monospace LLM contexts.
func VelocityASCII(bck *Backlog, iterationCount int, cfg *config.Config, overrides *IterationOverrides) string {
	now := time.Now().In(cfg.IterationLocation())
	weeks := cfg.Iteration.LengthWeeks
	currentStart := IterationStartFor(now, cfg)

	type bucket struct {
		start  time.Time
		points float64
	}
	buckets := make([]bucket, iterationCount)
	for i := range buckets {
		offset := iterationCount - 1 - i
		s := currentStart.AddDate(0, 0, -7*weeks*offset)
		buckets[i] = bucket{start: s}
	}

	for _, item := range bck.AllItems() {
		if !CountsForVelocity(item, cfg) {
			continue
		}
		acc := item.Accepted()
		if acc.IsZero() {
			continue
		}
		acc = acc.In(cfg.IterationLocation())
		for i := range buckets {
			end := buckets[i].start.AddDate(0, 0, 7*weeks)
			if (acc.Equal(buckets[i].start) || acc.After(buckets[i].start)) && acc.Before(end) {
				pts, _ := strconv.ParseFloat(strings.TrimSpace(item.Estimate()), 64)
				buckets[i].points += pts
				break
			}
		}
	}

	maxPoints := 0.0
	for _, b := range buckets {
		if b.points > maxPoints {
			maxPoints = b.points
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Velocity (last %d iterations of %d week%s)\n\n", iterationCount, weeks, plural(weeks))
	if maxPoints == 0 {
		b.WriteString("  (no accepted points yet in this window)\n")
		var accepted []*BacklogItem
		for _, item := range bck.AllItems() {
			if CountsForVelocity(item, cfg) {
				accepted = append(accepted, item)
			}
		}
		v, _, _ := ComputeVelocity(now, accepted, cfg, overrides)
		fmt.Fprintf(&b, "\n  velocity: %.0f (bootstrap)   volatility: n/a\n", v)
		return b.String()
	}

	const rows = 8
	step := maxPoints / float64(rows)
	if step < 1 {
		step = 1
	}
	// stepped so y-axis labels are integer multiples
	step = float64(int(step + 0.999))
	yMax := step * float64(rows)
	if yMax < maxPoints {
		yMax = step * float64(int(maxPoints/step)+1)
	}

	colWidth := 3 // bar (2) + space (1)
	leftMargin := 5

	for r := rows; r >= 1; r-- {
		threshold := step * float64(r)
		fmt.Fprintf(&b, "%4d |", int(threshold))
		for _, bk := range buckets {
			if bk.points >= threshold {
				b.WriteString(" ▇▇")
			} else if bk.points >= threshold-step/2 {
				b.WriteString(" ▃▃")
			} else {
				b.WriteString("   ")
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("     +")
	b.WriteString(strings.Repeat("-", colWidth*len(buckets)))
	b.WriteString("\n")

	// X-axis labels every other bucket to keep width modest
	b.WriteString(strings.Repeat(" ", leftMargin+1))
	for i, bk := range buckets {
		if i%2 == 0 {
			lbl := bk.start.Format("01/02")
			b.WriteString(lbl + " ")
		} else {
			b.WriteString("      ")
		}
	}
	b.WriteString("\n")

	// Footer: project velocity + volatility (computed over the rolling
	// lookback window from cfg, not the full chart range).
	var accepted []*BacklogItem
	for _, item := range bck.AllItems() {
		if CountsForVelocity(item, cfg) {
			accepted = append(accepted, item)
		}
	}
	v, _, boot := ComputeVelocity(now, accepted, cfg, overrides)
	vol := VolatilityPercent(now, accepted, cfg, overrides)
	b.WriteString("\n")
	if boot {
		fmt.Fprintf(&b, "  velocity: %.0f (bootstrap)   volatility: n/a\n", v)
	} else {
		fmt.Fprintf(&b, "  velocity: %.0f   volatility: %.0f%%\n", v, vol)
	}
	return b.String()
}

// BurnupASCII renders the per-day burnup chart for one iteration as
// terminal-friendly text. Each row is one day; "scope" is the line
// boundary, "done" is the filled bar inside it. Output stays inside
// 80 columns even at two-week iterations.
//
//	Burnup · 2026-05-04 -> 2026-05-10 (scope vs done)
//
//	  05-04   0.0 / 18.0  |░░░░░░░░░░░░░░░░░░|
//	  05-05   3.0 / 18.0  |▇▇▇░░░░░░░░░░░░░░░|
//	  05-06   3.0 / 18.0  |▇▇▇░░░░░░░░░░░░░░░|
//	  05-07   8.0 / 18.0  |▇▇▇▇▇▇▇▇░░░░░░░░░░|
//	  05-08  11.0 / 18.0  |▇▇▇▇▇▇▇▇▇▇▇░░░░░░░|
//	  ...
func BurnupASCII(rows []BurnupRow, start, end time.Time) string {
	if len(rows) == 0 {
		return fmt.Sprintf("Burnup %s -> %s\n\n  (no data)\n",
			start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	maxScope := 0.0
	for _, r := range rows {
		if r.Scope > maxScope {
			maxScope = r.Scope
		}
	}
	const barW = 18
	var b strings.Builder
	fmt.Fprintf(&b, "Burnup %s -> %s (scope vs done)\n\n",
		start.Format("2006-01-02"), end.Format("2006-01-02"))
	for _, r := range rows {
		filled := 0
		boundary := 0
		if maxScope > 0 {
			filled = int(float64(barW)*r.Done/maxScope + 0.5)
			boundary = int(float64(barW)*r.Scope/maxScope + 0.5)
		}
		if filled > barW {
			filled = barW
		}
		if boundary > barW {
			boundary = barW
		}
		if filled > boundary {
			filled = boundary
		}
		bar := strings.Repeat("▇", filled) + strings.Repeat("░", boundary-filled) + strings.Repeat(" ", barW-boundary)
		fmt.Fprintf(&b, "  %s  %4.1f / %4.1f  |%s|\n",
			r.Day.Format("01-02"), r.Done, r.Scope, bar)
	}
	return b.String()
}

// CFDASCII renders the cumulative flow as a tabular per-day report.
// The visual stack is a row of segment markers proportional to the
// counts: 'A' = accepted, 'I' = in-flight, 'B' = backlog. The
// numeric counts follow for unambiguous reading by both humans and
// LLMs.
//
//	Cumulative flow (30 days · stories per day)
//
//	  day         acc  inflt  bklg  flow
//	  2026-04-14    0     0    12   BBBBBBBBBBBB
//	  2026-04-15    1     2     9   AIIBBBBBBBBB
//	  ...
//	  2026-05-14    6     5     5   AAAAAAIIIIIBBBBB
func CFDASCII(rows []CFDRow) string {
	if len(rows) == 0 {
		return "Cumulative flow: (no data)\n"
	}
	maxTotal := 0
	for _, r := range rows {
		t := r.Accepted + r.InFlight + r.Backlog
		if t > maxTotal {
			maxTotal = t
		}
	}
	const barW = 18
	var b strings.Builder
	fmt.Fprintf(&b, "Cumulative flow (%d days · stories per day)\n\n", len(rows))
	b.WriteString("  day         acc  inflt  bklg  flow\n")
	for _, r := range rows {
		accCells, ifCells, bkCells := 0, 0, 0
		if maxTotal > 0 {
			accCells = (r.Accepted * barW) / maxTotal
			ifCells = (r.InFlight * barW) / maxTotal
			bkCells = (r.Backlog * barW) / maxTotal
			// Floor rounding can lose cells when a band is small but
			// non-zero. Ensure every non-zero band shows up.
			if r.Accepted > 0 && accCells == 0 {
				accCells = 1
			}
			if r.InFlight > 0 && ifCells == 0 {
				ifCells = 1
			}
			if r.Backlog > 0 && bkCells == 0 {
				bkCells = 1
			}
		}
		flow := strings.Repeat("A", accCells) + strings.Repeat("I", ifCells) + strings.Repeat("B", bkCells)
		fmt.Fprintf(&b, "  %s  %3d   %3d   %3d  %s\n",
			r.Day.Format("2006-01-02"), r.Accepted, r.InFlight, r.Backlog, flow)
	}
	b.WriteString("\n  legend: A=accepted  I=in-flight  B=backlog\n")
	return b.String()
}

// TypeMixASCII renders accepted-story type counts as horizontal bars.
//
//	Story type mix (lookback window · 36 accepted)
//
//	  feature  ████████████████████████████████  18  (50%)
//	  bug      ████████████                       7  (19%)
//	  chore    ██████████                         6  (17%)
//	  release  █████                              5  (14%)
func TypeMixASCII(rows []TypeMixRow, total int) string {
	if total == 0 || len(rows) == 0 {
		return "Story type mix: (no accepted stories in window)\n"
	}
	const barW = 30
	maxCount := 0
	for _, r := range rows {
		if r.Count > maxCount {
			maxCount = r.Count
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Story type mix (lookback window · %d accepted)\n\n", total)
	for _, r := range rows {
		cells := 0
		if maxCount > 0 {
			cells = (r.Count * barW) / maxCount
		}
		if r.Count > 0 && cells == 0 {
			cells = 1
		}
		bar := strings.Repeat("█", cells) + strings.Repeat(" ", barW-cells)
		fmt.Fprintf(&b, "  %-7s  %s  %3d  (%2d%%)\n", r.Type, bar, r.Count, int(r.Percent+0.5))
	}
	return b.String()
}

// TypeMixRow mirrors the shape mcpserver.TypeMixRow exposes. Defined
// here so backlog/ chart helpers don't need to import the server
// package. Keep the JSON tags identical.
type TypeMixRow struct {
	Type    string  `json:"type"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// TimelineASCII renders a horizontal text Gantt for items tagged `tag`.
// Bar width per row is normalised to a fixed BAR_W cells so 80-col output
// stays readable.
func TimelineASCII(items []*BacklogItem, tag string) string {
	type row struct {
		title string
		start time.Time
		end   time.Time
	}
	rows := make([]row, 0, len(items))
	for _, it := range items {
		s, e := it.Timeline()
		if s.IsZero() || e.IsZero() {
			continue
		}
		rows = append(rows, row{title: it.Title(), start: s, end: e})
	}
	if len(rows) == 0 {
		return fmt.Sprintf("Timeline %s: (no items with start/end dates)\n", tag)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].start.Before(rows[j].start) })

	minDate := rows[0].start
	maxDate := rows[0].end
	for _, r := range rows {
		if r.start.Before(minDate) {
			minDate = r.start
		}
		if r.end.After(maxDate) {
			maxDate = r.end
		}
	}
	totalDays := int(maxDate.Sub(minDate).Hours()/24) + 1
	if totalDays < 1 {
		totalDays = 1
	}

	const barW = 60
	titleW := 28

	var b strings.Builder
	fmt.Fprintf(&b, "Timeline %s   %s -> %s\n\n", tag, minDate.Format("2006-01-02"), maxDate.Format("2006-01-02"))
	for _, r := range rows {
		title := r.title
		if len(title) > titleW {
			title = title[:titleW-1] + "…"
		} else {
			title = title + strings.Repeat(" ", titleW-len(title))
		}
		offDays := int(r.start.Sub(minDate).Hours() / 24)
		dur := int(r.end.AddDate(0, 0, 1).Sub(r.start).Hours()/24) + 0
		offCells := offDays * barW / totalDays
		barCells := dur * barW / totalDays
		if barCells < 1 {
			barCells = 1
		}
		bar := strings.Repeat(" ", offCells) + strings.Repeat("█", barCells)
		if len(bar) > barW {
			bar = bar[:barW]
		} else {
			bar = bar + strings.Repeat(" ", barW-len(bar))
		}
		fmt.Fprintf(&b, "%s |%s| %s -> %s\n", title, bar, r.start.Format("01/02"), r.end.Format("01/02"))
	}
	return b.String()
}
