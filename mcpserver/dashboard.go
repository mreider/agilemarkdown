package mcpserver

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DashboardArgs struct {
	Backlog string `json:"backlog,omitempty" jsonschema:"optional backlog filter; default aggregates across the project"`
}

type DashboardResult struct {
	Velocity        float64 `json:"velocity"`
	VelocityBoot    bool    `json:"velocity_bootstrap"`
	Volatility      float64 `json:"volatility_percent"`
	CycleTimeHours  float64 `json:"cycle_time_median_hours"`
	RejectionPct    float64 `json:"rejection_rate_latest_percent"`
	StoriesAccepted int     `json:"stories_accepted_total"`
}

func dashboardTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, DashboardArgs) (*mcp.CallToolResult, DashboardResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args DashboardArgs) (*mcp.CallToolResult, DashboardResult, error) {
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, DashboardResult{}, err
		}
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, DashboardResult{}, err
		}
		all := make([]*backlog.BacklogItem, 0, 64)
		acceptedCount := 0
		for _, d := range dirs {
			if args.Backlog != "" && filepath.Base(d) != args.Backlog {
				continue
			}
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, DashboardResult{}, err
			}
			for _, it := range bck.AllItems() {
				all = append(all, it)
				if strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
					acceptedCount++
				}
			}
		}
		now := time.Now()
		overrides, _ := backlog.LoadIterationOverrides(root.Root())
		var feedAccepted []*backlog.BacklogItem
		for _, it := range all {
			if backlog.CountsForVelocity(it, cfg) {
				feedAccepted = append(feedAccepted, it)
			}
		}
		velocity, _, boot := backlog.ComputeVelocity(now, feedAccepted, cfg, overrides)
		volatility := backlog.VolatilityPercent(now, feedAccepted, cfg, overrides)
		median := backlog.MedianCycleTime(backlog.CycleTimes(all))
		rejRows := backlog.RejectionRates(now, all, cfg)
		latest := 0.0
		if n := len(rejRows); n > 0 {
			latest = rejRows[n-1].Percent
		}
		return nil, DashboardResult{
			Velocity:        velocity,
			VelocityBoot:    boot,
			Volatility:      volatility,
			CycleTimeHours:  median.Hours(),
			RejectionPct:    latest,
			StoriesAccepted: acceptedCount,
		}, nil
	}
}

type TypeMixArgs struct {
	Backlog string `json:"backlog,omitempty"`
}

type TypeMixRow struct {
	Type    string  `json:"type"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type TypeMixResult struct {
	Rows  []TypeMixRow `json:"rows"`
	Total int          `json:"total"`
}

func typeMixTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, TypeMixArgs) (*mcp.CallToolResult, TypeMixResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args TypeMixArgs) (*mcp.CallToolResult, TypeMixResult, error) {
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, TypeMixResult{}, err
		}
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, TypeMixResult{}, err
		}
		now := time.Now().In(cfg.IterationLocation())
		iters := backlog.CompletedIterations(now, cfg.Velocity.Lookback, cfg)
		var lo time.Time
		if len(iters) > 0 {
			lo = iters[0].Start
		}
		counts := map[string]int{"feature": 0, "bug": 0, "chore": 0, "release": 0}
		total := 0
		for _, d := range dirs {
			if args.Backlog != "" && filepath.Base(d) != args.Backlog {
				continue
			}
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, TypeMixResult{}, err
			}
			for _, it := range bck.AllItems() {
				if !strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
					continue
				}
				if !lo.IsZero() && it.Accepted().Before(lo) {
					continue
				}
				typ := it.Type()
				if typ == "" {
					typ = "feature"
				}
				counts[typ]++
				total++
			}
		}
		out := make([]TypeMixRow, 0, len(counts))
		for k, v := range counts {
			pct := 0.0
			if total > 0 {
				pct = float64(v) / float64(total) * 100.0
			}
			out = append(out, TypeMixRow{Type: k, Count: v, Percent: pct})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
		ascii := backlog.TypeMixASCII(toBacklogTypeMix(out), total)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: ascii}},
		}, TypeMixResult{Rows: out, Total: total}, nil
	}
}

// toBacklogTypeMix converts the MCP-facing row type to the
// backlog-package one for ASCII rendering. Identical layouts, two
// packages — the conversion is trivial and lets the chart helper
// stay free of mcpserver imports.
func toBacklogTypeMix(rows []TypeMixRow) []backlog.TypeMixRow {
	out := make([]backlog.TypeMixRow, len(rows))
	for i, r := range rows {
		out[i] = backlog.TypeMixRow{Type: r.Type, Count: r.Count, Percent: r.Percent}
	}
	return out
}

type VelocityHistoryArgs struct {
	Backlog        string `json:"backlog"`
	IterationCount int    `json:"iteration_count,omitempty" jsonschema:"defaults to lookback"`
}

type VelocityHistoryRow struct {
	Iteration    int     `json:"iteration"`
	Start        string  `json:"start"`
	Planned      float64 `json:"planned"`
	Accepted     float64 `json:"accepted"`
	LengthWeeks  int     `json:"length_weeks"`
	TeamStrength float64 `json:"team_strength"`
}

type VelocityHistoryResult struct {
	Rows []VelocityHistoryRow `json:"rows"`
}

func velocityHistoryTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, VelocityHistoryArgs) (*mcp.CallToolResult, VelocityHistoryResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args VelocityHistoryArgs) (*mcp.CallToolResult, VelocityHistoryResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, VelocityHistoryResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, VelocityHistoryResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, VelocityHistoryResult{}, err
		}
		count := args.IterationCount
		if count <= 0 {
			count = cfg.Velocity.Lookback
		}
		now := time.Now()
		overrides, _ := backlog.LoadIterationOverrides(root.Root())
		rows := backlog.VelocityHistory(now, bck.AllItems(), cfg, overrides, count)
		out := make([]VelocityHistoryRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, VelocityHistoryRow{
				Iteration:    r.Iteration,
				Start:        r.Start.Format("2006-01-02"),
				Planned:      r.Planned,
				Accepted:     r.Accepted,
				LengthWeeks:  r.LengthWeeks,
				TeamStrength: r.TeamStrength,
			})
		}
		return nil, VelocityHistoryResult{Rows: out}, nil
	}
}

type CFDArgs struct {
	Days int `json:"days,omitempty" jsonschema:"lookback window in days; default 30"`
}

type CFDRow struct {
	Day      string `json:"day"`
	Accepted int    `json:"accepted"`
	InFlight int    `json:"in_flight"`
	Backlog  int    `json:"backlog"`
}

type CFDResult struct {
	Rows []CFDRow `json:"rows"`
}

func cfdTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, CFDArgs) (*mcp.CallToolResult, CFDResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args CFDArgs) (*mcp.CallToolResult, CFDResult, error) {
		days := args.Days
		if days <= 0 {
			days = 30
		}
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, CFDResult{}, err
		}
		all := make([]*backlog.BacklogItem, 0, 64)
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, CFDResult{}, err
			}
			all = append(all, bck.AllItems()...)
		}
		end := time.Now().UTC()
		start := end.AddDate(0, 0, -days)
		rows := backlog.CFDRows(all, start, end)
		out := make([]CFDRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, CFDRow{Day: r.Day.Format("2006-01-02"), Accepted: r.Accepted, InFlight: r.InFlight, Backlog: r.Backlog})
		}
		ascii := backlog.CFDASCII(rows)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: ascii}},
		}, CFDResult{Rows: out}, nil
	}
}

type BurnupArgs struct {
	Backlog string `json:"backlog"`
	Offset  int    `json:"offset,omitempty" jsonschema:"window offset; zero is the current iteration"`
}

type BurnupRow struct {
	Day   string  `json:"day"`
	Scope float64 `json:"scope"`
	Done  float64 `json:"done"`
}

type BurnupResult struct {
	Iteration int         `json:"iteration"`
	Start     string      `json:"start"`
	End       string      `json:"end"`
	Rows      []BurnupRow `json:"rows"`
}

func burnupChartTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, BurnupArgs) (*mcp.CallToolResult, BurnupResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args BurnupArgs) (*mcp.CallToolResult, BurnupResult, error) {
		dir, err := resolveBacklogDir(root, args.Backlog)
		if err != nil {
			return nil, BurnupResult{}, err
		}
		cfg, err := config.LoadConfig(filepath.Join(root.Root(), ".am", "config.yaml"))
		if err != nil {
			return nil, BurnupResult{}, err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return nil, BurnupResult{}, err
		}
		now := time.Now().In(cfg.IterationLocation())
		start := backlog.IterationStartFor(now, cfg).AddDate(0, 0, 7*cfg.Iteration.LengthWeeks*args.Offset)
		end := start.AddDate(0, 0, 7*cfg.Iteration.LengthWeeks)
		rows := backlog.BurnupRows(bck.AllItems(), start, end)
		out := make([]BurnupRow, 0, len(rows))
		for _, r := range rows {
			out = append(out, BurnupRow{Day: r.Day.Format("2006-01-02"), Scope: r.Scope, Done: r.Done})
		}
		ascii := backlog.BurnupASCII(rows, start, end)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: ascii}},
		}, BurnupResult{
			Iteration: 0, // canonical number not always meaningful for offset windows
			Start:     start.Format("2006-01-02"),
			End:       end.Format("2006-01-02"),
			Rows:      out,
		}, nil
	}
}
