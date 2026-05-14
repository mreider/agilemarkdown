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
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		structure := backlog.NewBacklogsStructure(root)
		dirs, err := structure.BacklogDirs()
		if err != nil {
			return err
		}
		var all []*backlog.BacklogItem
		acceptedCount := 0
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return err
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
		fmt.Println("Dashboard")
		fmt.Println()
		if boot {
			fmt.Printf("  velocity:        %.0f (bootstrap)\n", velocity)
		} else {
			fmt.Printf("  velocity:        %.0f\n", velocity)
		}
		fmt.Printf("  volatility:      %.0f%%\n", volatility)
		fmt.Printf("  cycle time:      %s\n", formatHoursDur(median))
		fmt.Printf("  rejection rate:  %.0f%% (latest)\n", latest)
		fmt.Printf("  accepted total:  %d stories\n", acceptedCount)
		return nil
	},
}

// NextCommand prints the highest-ranked unstarted, unblocked story in the
// project. Equivalent to the `next_item` MCP tool.
var NextCommand = &cli.Command{
	Name:  "next",
	Usage: "Show the next pull (top-ranked unstarted, unblocked story across the project)",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		structure := backlog.NewBacklogsStructure(root)
		dirs, err := structure.BacklogDirs()
		if err != nil {
			return err
		}
		for _, d := range dirs {
			pri, err := backlog.LoadPriority(d)
			if err != nil {
				return err
			}
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return err
			}
			byBase := map[string]*backlog.BacklogItem{}
			for _, it := range bck.ActiveItems() {
				byBase[filepath.Base(it.Path())] = it
			}
			for _, e := range pri.Entries() {
				it, ok := byBase[e.Path]
				if !ok {
					continue
				}
				if it.Blocked() {
					continue
				}
				if !strings.EqualFold(it.Status(), backlog.UnstartedStatus.Name) {
					continue
				}
				if it.Type() == "release" {
					continue
				}
				rel, _ := filepath.Rel(root, it.Path())
				fmt.Printf("%s\n", rel)
				fmt.Printf("  title:    %s\n", it.Title())
				if e := it.Estimate(); e != "" {
					fmt.Printf("  points:   %s\n", e)
				}
				if a := it.Assigned(); a != "" {
					fmt.Printf("  assigned: %s\n", a)
				}
				return nil
			}
		}
		fmt.Println("(no unstarted, unblocked stories in priority)")
		return nil
	},
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
