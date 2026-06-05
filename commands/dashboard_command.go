package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

// DashboardCommand renders a one-block KPI summary across the project.
var DashboardCommand = &cli.Command{
	Name:  "dashboard",
	Usage: "One-block project dashboard: velocity, volatility, cycle time, rejection rate, accepted count",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit the dashboard as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		if c.Bool("json") {
			res, err := mcpserver.Dashboard(ctx, root, mcpserver.DashboardArgs{})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		text, err := dashboardText(root)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}

// NextCommand prints the highest-ranked unstarted, unblocked story in the
// project. Equivalent to the `next_item` MCP tool.
var NextCommand = &cli.Command{
	Name:  "next",
	Usage: "Show the next pull (top-ranked unstarted, unblocked story across the project)",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit the next-item result as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		if c.Bool("json") {
			res, err := mcpserver.NextItem(ctx, root, mcpserver.NextItemArgs{})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		text, err := nextText(root)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}

// dashboardText renders the one-block KPI summary (velocity, volatility,
// cycle time, rejection rate, accepted count) as it appears under
// `am dashboard`. Extracted so `am brief` can embed the exact same block.
func dashboardText(root string) (string, error) {
	cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
	if err != nil {
		return "", err
	}
	structure := backlog.NewBacklogsStructure(root)
	dirs, err := structure.BacklogDirs()
	if err != nil {
		return "", err
	}
	var all []*backlog.BacklogItem
	acceptedCount := 0
	for _, d := range dirs {
		bck, err := backlog.LoadBacklog(d)
		if err != nil {
			return "", err
		}
		for _, it := range bck.AllItems() {
			all = append(all, it)
			if strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
				acceptedCount++
			}
		}
	}
	now := time.Now()
	overrides, _ := backlog.LoadIterationOverrides(root)
	var feed []*backlog.BacklogItem
	for _, it := range all {
		if backlog.CountsForVelocity(it, cfg) {
			feed = append(feed, it)
		}
	}
	velocity, _, boot := backlog.ComputeVelocity(now, feed, cfg, overrides)
	volatility := backlog.VolatilityPercent(now, feed, cfg, overrides)
	median := backlog.MedianCycleTime(backlog.CycleTimes(all))
	rejRows := backlog.RejectionRates(now, all, cfg)
	latest := 0.0
	if n := len(rejRows); n > 0 {
		latest = rejRows[n-1].Percent
	}
	var b strings.Builder
	b.WriteString("Dashboard\n\n")
	if boot {
		fmt.Fprintf(&b, "  velocity:        %.0f (bootstrap)\n", velocity)
	} else {
		fmt.Fprintf(&b, "  velocity:        %.0f\n", velocity)
	}
	fmt.Fprintf(&b, "  volatility:      %.0f%%\n", volatility)
	fmt.Fprintf(&b, "  cycle time:      %s\n", formatHoursDur(median))
	fmt.Fprintf(&b, "  rejection rate:  %.0f%% (latest)\n", latest)
	fmt.Fprintf(&b, "  accepted total:  %d stories\n", acceptedCount)
	return b.String(), nil
}

// nextText renders the top-ranked unstarted, unblocked story across the
// project as it appears under `am next`. Extracted for reuse by `am brief`.
func nextText(root string) (string, error) {
	structure := backlog.NewBacklogsStructure(root)
	dirs, err := structure.BacklogDirs()
	if err != nil {
		return "", err
	}
	for _, d := range dirs {
		pri, err := backlog.LoadPriority(d)
		if err != nil {
			return "", err
		}
		bck, err := backlog.LoadBacklog(d)
		if err != nil {
			return "", err
		}
		byBase := map[string]*backlog.BacklogItem{}
		for _, it := range bck.ActiveItems() {
			byBase[filepath.Base(it.Path())] = it
		}
		for _, e := range pri.Entries() {
			it, ok := byBase[e.Path]
			if !ok || it.Blocked() {
				continue
			}
			if !strings.EqualFold(it.Status(), backlog.UnstartedStatus.Name) {
				continue
			}
			if it.Type() == "release" {
				continue
			}
			rel, _ := filepath.Rel(root, it.Path())
			var b strings.Builder
			fmt.Fprintf(&b, "%s\n", rel)
			fmt.Fprintf(&b, "  title:    %s\n", it.Title())
			if est := it.Estimate(); est != "" {
				fmt.Fprintf(&b, "  points:   %s\n", est)
			}
			if a := it.Assigned(); a != "" {
				fmt.Fprintf(&b, "  assigned: %s\n", a)
			}
			return b.String(), nil
		}
	}
	return "(no unstarted, unblocked stories in priority)\n", nil
}

func formatHoursDur(d time.Duration) string {
	h := d.Hours()
	if h <= 0 {
		return "n/a"
	}
	if h < 24 {
		return fmt.Sprintf("%.1fh", h)
	}
	return fmt.Sprintf("%.1fd", h/24.0)
}
