package backlog

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/config"
)

// Iteration is a Pivotal-style fixed time window. Items are assigned to
// an iteration by their `accepted` timestamp falling inside [Start, End).
type Iteration struct {
	Number       int       // canonical 1-based iteration number (stable across reloads)
	Start        time.Time // inclusive
	End          time.Time // exclusive
	Items        []*BacklogItem
	Points       float64
	TeamStrength float64 // 1.0 = full strength; 0 = excluded from velocity
	LengthWeeks  int     // weeks for this specific iteration (after any override)
}

// IterationStartFor returns the start of the iteration window that
// contains `t`, given the project iteration config.
//
// The window aligns to the configured start day of week, week-of-year math
// in the configured timezone, and length_weeks (1/2/3/4). For length>1, an
// epoch (the start day of ISO-week 1, year 2000 in the project tz) is used
// to anchor the multi-week pattern so it remains stable across config
// reloads.
func IterationStartFor(t time.Time, c *config.Config) time.Time {
	loc := c.IterationLocation()
	t = t.In(loc)
	startWeekday := c.IterationStartWeekday()

	// Walk back to most recent start-of-week day at midnight.
	weekStart := startOfWeekDay(t, startWeekday, loc)

	if c.Iteration.LengthWeeks <= 1 {
		return weekStart
	}

	// Anchor multi-week iterations to a stable epoch so the same calendar
	// week always belongs to the same iteration regardless of when the
	// project was created.
	epoch := startOfWeekDay(time.Date(2000, time.January, 3, 0, 0, 0, 0, loc), startWeekday, loc)
	weeksSinceEpoch := int(weekStart.Sub(epoch).Hours()/24/7 + 0.5)
	offset := weeksSinceEpoch % c.Iteration.LengthWeeks
	if offset < 0 {
		offset += c.Iteration.LengthWeeks
	}
	return weekStart.AddDate(0, 0, -7*offset)
}

// IterationEndFor returns the end (exclusive) of the iteration window
// containing `t`.
func IterationEndFor(t time.Time, c *config.Config) time.Time {
	return IterationStartFor(t, c).AddDate(0, 0, 7*c.Iteration.LengthWeeks)
}

// startOfWeekDay returns midnight of the most recent `weekday` at or before t.
func startOfWeekDay(t time.Time, weekday time.Weekday, loc *time.Location) time.Time {
	t = t.In(loc)
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	delta := (int(d.Weekday()) - int(weekday) + 7) % 7
	return d.AddDate(0, 0, -delta)
}

// CompletedIterations returns iterations strictly before the one that
// contains `now`, in chronological order, going back `count` iterations.
// Used by velocity strategies.
func CompletedIterations(now time.Time, count int, c *config.Config) []Iteration {
	if count <= 0 {
		return nil
	}
	currentStart := IterationStartFor(now, c)
	weeks := c.Iteration.LengthWeeks
	out := make([]Iteration, 0, count)
	for i := count; i >= 1; i-- {
		start := currentStart.AddDate(0, 0, -7*weeks*i)
		end := start.AddDate(0, 0, 7*weeks)
		out = append(out, Iteration{
			Number: -i,
			Start:  start,
			End:    end,
		})
	}
	return out
}

// CountsForVelocity reports whether item should contribute to velocity:
// type=feature, status=accepted, with a positive numeric estimate. Bugs
// and chores only count when the project config opts in.
func CountsForVelocity(item *BacklogItem, c *config.Config) bool {
	if !strings.EqualFold(item.Status(), AcceptedStatus.Name) {
		return false
	}
	switch item.Type() {
	case "", "feature":
		// default
	case "bug":
		if !c.StoryTypes.BugEstimable {
			return false
		}
	case "chore":
		if !c.StoryTypes.ChoreEstimable {
			return false
		}
	case "release":
		return false
	}
	pts, _ := strconv.ParseFloat(strings.TrimSpace(item.Estimate()), 64)
	return pts > 0
}

// ComputeVelocity computes project velocity using Pivotal Tracker's
// canonical formula:
//
//	velocity_per_week = SUM(iter_points_i / iter_strength_i) / SUM(iter_length_weeks_i)
//	displayed         = floor(velocity_per_week * default_iteration_length_weeks)
//
// Iterations with team_strength == 0 are excluded from both sums.
// Iteration length and strength can be set per iteration via
// `.am/iterations.yaml`; absent records default to project-level
// length_weeks and strength=1.0.
//
// `accepted` is the set of items already filtered by AcceptedStatus +
// CountsForVelocity. `overrides` may be nil.
//
// Returns the floored displayed velocity, the iterations used (most
// recent last, with TeamStrength + LengthWeeks populated), and a flag
// indicating whether the velocity fell back to InitialVelocity (no
// accepted points in the lookback window, or all overridden out).
func ComputeVelocity(now time.Time, accepted []*BacklogItem, c *config.Config, overrides *IterationOverrides) (float64, []Iteration, bool) {
	if c.Velocity.Strategy == "manual" {
		return float64(c.Velocity.InitialVelocity), nil, false
	}

	iters := CompletedIterations(now, c.Velocity.Lookback, c)
	if len(iters) == 0 {
		return float64(c.Velocity.InitialVelocity), nil, true
	}

	// Resolve per-iteration overrides (canonical number, strength, length).
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
					// 0 = explicit exclusion, but only if user set the
					// record. Distinguish from absent by checking field
					// presence: if the record exists at all, honor 0.
					iters[i].TeamStrength = 0
				}
				if rec.LengthWeeks > 0 {
					iters[i].LengthWeeks = rec.LengthWeeks
				}
			}
		}
	}

	// Bucket accepted items into their iteration.
	for _, item := range accepted {
		acc := item.Accepted()
		if acc.IsZero() {
			continue
		}
		for i := range iters {
			if (acc.Equal(iters[i].Start) || acc.After(iters[i].Start)) && acc.Before(iters[i].End) {
				pts, _ := strconv.ParseFloat(strings.TrimSpace(item.Estimate()), 64)
				iters[i].Items = append(iters[i].Items, item)
				iters[i].Points += pts
				break
			}
		}
	}

	// Default: rolling (Pivotal canonical). Strict is a non-Pivotal
	// extension that trims the iterations with the highest and lowest
	// normalized weekly rate before applying the same SUM/SUM formula.
	use := iters
	if c.Velocity.Strategy == "strict" && len(iters) >= 3 {
		use = trimHighLowByRate(iters)
	}

	numerator := 0.0
	denominator := 0.0
	for _, it := range use {
		if it.TeamStrength <= 0 {
			continue // excluded from both sums
		}
		numerator += it.Points / it.TeamStrength
		denominator += float64(it.LengthWeeks)
	}
	if denominator == 0 || numerator == 0 {
		return float64(c.Velocity.InitialVelocity), iters, true
	}
	perWeek := numerator / denominator
	displayed := perWeek * float64(c.Iteration.LengthWeeks)
	return floorFloat(displayed), iters, false
}

// trimHighLowByRate drops the iteration with the highest normalized
// weekly rate and the one with the lowest, used by the non-Pivotal
// "strict" strategy. Iterations with strength==0 are kept as-is in the
// caller's slice but skipped at sum time.
func trimHighLowByRate(iters []Iteration) []Iteration {
	type ranked struct {
		idx  int
		rate float64
	}
	rs := make([]ranked, 0, len(iters))
	for i, it := range iters {
		if it.TeamStrength <= 0 || it.LengthWeeks <= 0 {
			continue
		}
		rs = append(rs, ranked{i, (it.Points / it.TeamStrength) / float64(it.LengthWeeks)})
	}
	if len(rs) < 3 {
		return iters
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].rate < rs[j].rate })
	drop := map[int]bool{rs[0].idx: true, rs[len(rs)-1].idx: true}
	out := make([]Iteration, 0, len(iters)-2)
	for i, it := range iters {
		if drop[i] {
			continue
		}
		out = append(out, it)
	}
	return out
}

func floorFloat(x float64) float64 {
	if x < 0 {
		return -float64(int(-x + 0.0000001) + 1)
	}
	return float64(int(x + 0.0000001))
}
