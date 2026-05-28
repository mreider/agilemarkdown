package backlog

import (
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// VelocityHistoryEntry is one row of structured velocity history. `Planned`
// is filled with accepted-or-rejected points in the iteration window since
// agilemarkdown does not yet store a planning snapshot per iteration.
type VelocityHistoryEntry struct {
	Iteration    int
	Start        time.Time
	Planned      float64
	Accepted     float64
	LengthWeeks  int
	TeamStrength float64
}

// VelocityHistory returns one entry per completed iteration in the lookback
// window, oldest first.
func VelocityHistory(now time.Time, items []*BacklogItem, c *config.Config, overrides *IterationOverrides, count int) []VelocityHistoryEntry {
	if count <= 0 {
		count = c.Velocity.Lookback
	}
	iters := CompletedIterations(now, count, c)
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
	rows := make([]VelocityHistoryEntry, len(iters))
	for i := range iters {
		rows[i] = VelocityHistoryEntry{
			Iteration:    iters[i].Number,
			Start:        iters[i].Start,
			LengthWeeks:  iters[i].LengthWeeks,
			TeamStrength: iters[i].TeamStrength,
		}
	}
	for _, it := range items {
		acc := it.Accepted()
		mod := it.Modified()
		isAccepted := strings.EqualFold(it.Status(), AcceptedStatus.Name)
		isRejected := strings.EqualFold(it.Status(), RejectedStatus.Name)
		pts := it.estimateAsFloat()
		if pts <= 0 {
			continue
		}
		typ := it.Type()
		if typ != "" && typ != "feature" && typ != "bug" {
			continue
		}
		for i := range iters {
			if isAccepted && !acc.IsZero() && (acc.Equal(iters[i].Start) || acc.After(iters[i].Start)) && acc.Before(iters[i].End) {
				rows[i].Accepted += pts
				rows[i].Planned += pts
				break
			}
			if isRejected && !mod.IsZero() && (mod.Equal(iters[i].Start) || mod.After(iters[i].Start)) && mod.Before(iters[i].End) {
				rows[i].Planned += pts
				break
			}
		}
	}
	return rows
}

// BurnupRow is a single day in a burnup window.
type BurnupRow struct {
	Day   time.Time
	Scope float64
	Done  float64
}

// CFDRow is one day of a cumulative flow diagram. Counts are story
// counts (not points). Three bands now that `started:` is tracked:
// accepted, in-flight (started but not yet accepted), and backlog
// (existed but not started). `release` items are excluded as date
// markers.
type CFDRow struct {
	Day      time.Time
	Accepted int
	InFlight int
	Backlog  int
}

// CFDRows builds a per-day cumulative flow over [start, end).
func CFDRows(items []*BacklogItem, start, end time.Time) []CFDRow {
	if !end.After(start) {
		return nil
	}
	days := int(end.Sub(start).Hours()/24 + 0.5)
	if days <= 0 {
		days = 1
	}
	rows := make([]CFDRow, days)
	for d := 0; d < days; d++ {
		rows[d].Day = start.AddDate(0, 0, d)
	}
	for _, it := range items {
		if it.Type() == "release" {
			continue
		}
		created := it.Created()
		if created.IsZero() {
			created = it.Modified()
		}
		started := it.Started()
		acc := it.Accepted()
		for d := 0; d < days; d++ {
			day := rows[d].Day
			cutoff := day.AddDate(0, 0, 1)
			existedByDay := !created.IsZero() && !created.After(cutoff)
			startedByDay := !started.IsZero() && !started.After(cutoff)
			acceptedByDay := !acc.IsZero() && !acc.After(cutoff)
			if !existedByDay {
				continue
			}
			switch {
			case acceptedByDay:
				rows[d].Accepted++
			case startedByDay:
				rows[d].InFlight++
			default:
				rows[d].Backlog++
			}
		}
	}
	return rows
}

// BurnupRows returns one row per day in [start, end). Scope is the total
// points of all stories accepted or in flight whose Created (or Modified)
// is at or before that day. Done is the cumulative accepted points.
func BurnupRows(items []*BacklogItem, start, end time.Time) []BurnupRow {
	if !end.After(start) {
		return nil
	}
	days := int(end.Sub(start).Hours()/24 + 0.5)
	if days <= 0 {
		days = 1
	}
	rows := make([]BurnupRow, days)
	for d := 0; d < days; d++ {
		rows[d].Day = start.AddDate(0, 0, d)
	}
	for _, it := range items {
		if it.Type() == "release" {
			continue
		}
		pts := it.estimateAsFloat()
		if pts <= 0 {
			continue
		}
		created := it.Created()
		if created.IsZero() {
			created = it.Modified()
		}
		acc := it.Accepted()
		for d := 0; d < days; d++ {
			day := rows[d].Day
			if !created.IsZero() && !created.After(day.AddDate(0, 0, 1)) && created.Before(end) {
				rows[d].Scope += pts
			}
			if !acc.IsZero() && !acc.After(day.AddDate(0, 0, 1)) && acc.Before(end) {
				rows[d].Done += pts
			}
		}
	}
	return rows
}
