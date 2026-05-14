package backlog

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// VolatilityPercent is the standard deviation of accepted points across
// the last lookback iterations, expressed as a percentage of the mean.
// Pivotal Tracker showed this number on the dashboard. A low number
// means the team's velocity is honest week to week; a high number means
// the chart is misleading.
//
// Returns 0 when the mean is 0 (no accepted points in the window) or
// when fewer than 2 iterations have data.
func VolatilityPercent(now time.Time, accepted []*BacklogItem, c *config.Config, overrides *IterationOverrides) float64 {
	iters := CompletedIterations(now, c.Velocity.Lookback, c)
	if len(iters) < 2 {
		return 0
	}
	for i := range iters {
		num := iterationNumberFor(iters[i].Start, c)
		iters[i].Number = num
		iters[i].TeamStrength = 1.0
		iters[i].LengthWeeks = c.Iteration.LengthWeeks
		if overrides != nil {
			if rec := overrides.Find(num); rec != nil {
				if rec.TeamStrength > 0 {
					iters[i].TeamStrength = rec.TeamStrength
				} else if rec.TeamStrength == 0 {
					iters[i].TeamStrength = 0
				}
				if rec.LengthWeeks > 0 {
					iters[i].LengthWeeks = rec.LengthWeeks
				}
			}
		}
	}
	for _, item := range accepted {
		acc := item.Accepted()
		if acc.IsZero() {
			continue
		}
		for i := range iters {
			if (acc.Equal(iters[i].Start) || acc.After(iters[i].Start)) && acc.Before(iters[i].End) {
				iters[i].Points += item.estimateAsFloat()
				break
			}
		}
	}

	rates := make([]float64, 0, len(iters))
	for _, it := range iters {
		if it.TeamStrength <= 0 || it.LengthWeeks <= 0 {
			continue
		}
		rates = append(rates, (it.Points/it.TeamStrength)/float64(it.LengthWeeks))
	}
	if len(rates) < 2 {
		return 0
	}
	mean := 0.0
	for _, r := range rates {
		mean += r
	}
	mean /= float64(len(rates))
	if mean == 0 {
		return 0
	}
	variance := 0.0
	for _, r := range rates {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(rates))
	stddev := math.Sqrt(variance)
	return (stddev / mean) * 100.0
}

// CycleTime is the per-item duration from Started to Accepted. The
// canonical Pivotal definition: earliest-started timestamp to
// latest-accepted timestamp. Releases are excluded (they don't have a
// started state).
type CycleTime struct {
	Item     *BacklogItem
	Duration time.Duration
}

// CycleTimes returns one record per item that has both Started and
// Accepted timestamps. Items in the iteration window are not filtered
// here; callers can sort and slice.
func CycleTimes(items []*BacklogItem) []CycleTime {
	out := make([]CycleTime, 0, len(items))
	for _, it := range items {
		if it.Type() == "release" {
			continue
		}
		// "Started" is implied by the finished-or-later state; we don't
		// store the started timestamp explicitly. Use Created as a
		// pragmatic stand-in until status-history is modeled. (Pivotal
		// stored start_time on the story; agilemarkdown does not yet.)
		// Falls back to Modified if Created is zero.
		started := it.Created()
		if started.IsZero() {
			started = it.Modified()
		}
		acc := it.Accepted()
		if started.IsZero() || acc.IsZero() {
			continue
		}
		if acc.Before(started) {
			continue
		}
		out = append(out, CycleTime{Item: it, Duration: acc.Sub(started)})
	}
	return out
}

// MedianCycleTime returns the median of a CycleTime slice as a
// time.Duration. Empty slice -> 0.
func MedianCycleTime(xs []CycleTime) time.Duration {
	if len(xs) == 0 {
		return 0
	}
	durs := make([]time.Duration, len(xs))
	for i, x := range xs {
		durs[i] = x.Duration
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	mid := len(durs) / 2
	if len(durs)%2 == 1 {
		return durs[mid]
	}
	return (durs[mid-1] + durs[mid]) / 2
}

// RejectionRate computes the per-iteration rejection rate using
// Pivotal's formula: stories accepted-or-rejected during the iteration
// that have ever been rejected, divided by the total
// accepted-or-rejected during the iteration. We approximate "ever been
// rejected" with the current `status: rejected` state of the item,
// since agilemarkdown does not yet store full status history.
//
// Returns one entry per iteration in the lookback window, oldest first.
type RejectionRow struct {
	Iteration int
	Start     time.Time
	Accepted  int
	Rejected  int
	Percent   float64
}

func RejectionRates(now time.Time, items []*BacklogItem, c *config.Config) []RejectionRow {
	iters := CompletedIterations(now, c.Velocity.Lookback, c)
	rows := make([]RejectionRow, len(iters))
	for i := range iters {
		rows[i].Iteration = iterationNumberFor(iters[i].Start, c)
		rows[i].Start = iters[i].Start
	}
	bucketBy := func(t time.Time, advance func(int)) {
		if t.IsZero() {
			return
		}
		for i := range iters {
			if (t.Equal(iters[i].Start) || t.After(iters[i].Start)) && t.Before(iters[i].End) {
				advance(i)
				return
			}
		}
	}
	for _, it := range items {
		if t := it.Type(); t != "" && t != "feature" && t != "bug" {
			continue
		}
		if strings.EqualFold(it.Status(), RejectedStatus.Name) {
			// Rejected stories are bucketed by their last modification
			// time (the rejection). agilemarkdown does not store a
			// separate `rejected:` timestamp; modified is the proxy.
			bucketBy(it.Modified(), func(i int) { rows[i].Rejected++ })
			continue
		}
		bucketBy(it.Accepted(), func(i int) { rows[i].Accepted++ })
	}
	for i := range rows {
		total := rows[i].Accepted + rows[i].Rejected
		if total > 0 {
			rows[i].Percent = float64(rows[i].Rejected) / float64(total) * 100.0
		}
	}
	return rows
}

// CycleTimeASCII renders a per-iteration median cycle time chart and a
// single-line summary across all items.
func CycleTimeASCII(bck *Backlog) string {
	cts := CycleTimes(bck.AllItems())
	if len(cts) == 0 {
		return "Cycle time: (no accepted items with timestamps yet)\n"
	}
	median := MedianCycleTime(cts)
	hours := median.Hours()
	var b strings.Builder
	fmt.Fprintf(&b, "Cycle time (started -> accepted)\n\n")
	fmt.Fprintf(&b, "  median: %s   (n=%d)\n\n", formatHours(hours), len(cts))
	// Top 5 longest
	sort.Slice(cts, func(i, j int) bool { return cts[i].Duration > cts[j].Duration })
	limit := len(cts)
	if limit > 5 {
		limit = 5
	}
	b.WriteString("  longest:\n")
	for i := 0; i < limit; i++ {
		fmt.Fprintf(&b, "    %-40s  %s\n", truncate(cts[i].Item.Title(), 40), formatHours(cts[i].Duration.Hours()))
	}
	return b.String()
}

// RejectionRateASCII renders per-iteration rejection rates as a small
// table.
func RejectionRateASCII(now time.Time, items []*BacklogItem, c *config.Config) string {
	rows := RejectionRates(now, items, c)
	var b strings.Builder
	b.WriteString("Rejection rate (last iterations)\n\n")
	b.WriteString("  iteration  start         accepted  rejected  rate\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %9d  %s    %8d  %8d  %4.0f%%\n",
			r.Iteration, r.Start.Format("2006-01-02"), r.Accepted, r.Rejected, r.Percent)
	}
	b.WriteString("\n  Pivotal target band: 5-15%\n")
	return b.String()
}

func formatHours(h float64) string {
	if h < 24 {
		return fmt.Sprintf("%.1fh", h)
	}
	return fmt.Sprintf("%.1fd", h/24.0)
}
