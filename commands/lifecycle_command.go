package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

const inceptionFileName = "inception.md"

// PullCommand combines `am next` and `am start ITEM` into one verb.
// Convenience for the dev pair pulling the top-ranked unstarted story.
var PullCommand = &cli.Command{
	Name:  "pull",
	Usage: "Pull the next-ranked unstarted, unblocked story (am next followed by am start)",
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
				if err := transitionItem(it.Path(), backlog.StartedStatus); err != nil {
					return err
				}
				fmt.Printf("%s -> started\n", filepath.Base(it.Path()))
				fmt.Printf("  title:    %s\n", it.Title())
				if est := it.Estimate(); est != "" {
					fmt.Printf("  points:   %s\n", est)
				}
				if a := it.Assigned(); a != "" {
					fmt.Printf("  assigned: %s\n", a)
				}
				_ = rel
				return nil
			}
		}
		fmt.Println("(nothing to pull: no unstarted, unblocked stories at the top of priority)")
		return nil
	},
}

// transitionItem flips a single item to the given status using the
// canonical action transition (timestamps stamped automatically).
func transitionItem(path string, status *backlog.BacklogItemStatus) error {
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	actions.ApplyStatusTransition(item, status)
	return item.Save()
}

// InceptionCommand reads or seeds the project's inception.md. With no
// arguments and no existing file, it writes the canonical template.
// With `--show`, it prints the current document.
var InceptionCommand = &cli.Command{
	Name:  "inception",
	Usage: "Seed or print the project inception document at inception.md",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "show", Usage: "print the current inception.md and exit"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		path := filepath.Join(root, inceptionFileName)
		if c.Bool("show") {
			data, err := os.ReadFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("(no inception.md yet; run `am inception` without --show to seed one)")
					return nil
				}
				return err
			}
			fmt.Print(string(data))
			return nil
		}
		if _, err := os.Stat(path); err == nil {
			fmt.Println("inception.md already exists; pass --show to print it")
			return nil
		}
		if err := os.WriteFile(path, []byte(inceptionTemplate), 0644); err != nil {
			return err
		}
		fmt.Println("wrote inception.md (template)")
		fmt.Println("Edit the six sections (user, goal, reason, success, constraints, out of scope) before pulling stories.")
		return nil
	},
}

// SprintCommand groups iteration-planning verbs.
var SprintCommand = &cli.Command{
	Name:  "sprint",
	Usage: "Iteration helpers: sprint plan",
	Commands: []*cli.Command{
		sprintPlanCmd,
	},
}

var sprintPlanCmd = &cli.Command{
	Name:  "plan",
	Usage: "Render the iteration plan for a backlog (top of priority up to rolling velocity, with warnings)",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit the iteration plan as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		if c.Bool("json") {
			res, err := mcpserver.SprintPlan(ctx, root, mcpserver.SprintPlanArgs{Backlog: filepath.Base(dir)})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		pri, err := backlog.LoadPriority(dir)
		if err != nil {
			return err
		}
		var feed []*backlog.BacklogItem
		for _, it := range bck.AllItems() {
			if backlog.CountsForVelocity(it, cfg) {
				feed = append(feed, it)
			}
		}
		overrides, _ := backlog.LoadIterationOverrides(root)
		velocity, _, _ := backlog.ComputeVelocity(time.Now(), feed, cfg, overrides)
		byBase := map[string]*backlog.BacklogItem{}
		for _, it := range bck.ActiveItems() {
			byBase[filepath.Base(it.Path())] = it
		}

		fmt.Printf("Iteration plan (%s)   velocity %.0f / iteration\n\n", filepath.Base(dir), velocity)
		var pts float64
		var warnings []string
		printed := 0
		below := 0
		for i, e := range pri.Entries() {
			it, ok := byBase[e.Path]
			if !ok {
				continue
			}
			est := it.Estimate()
			var p float64
			fmt.Sscanf(est, "%f", &p)
			belowLine := velocity > 0 && pts+p > velocity
			if belowLine {
				if below == 0 {
					fmt.Printf("\nBacklog (below the line, not committed):\n")
				}
				below++
			} else {
				pts += p
				printed++
			}
			marker := "★"
			if it.Type() == "bug" {
				marker = "●"
			} else if it.Type() == "chore" {
				marker = "○"
			} else if it.Type() == "release" {
				marker = "▲"
			}
			estStr := est
			if estStr == "" {
				estStr = "—"
			}
			bullets := acceptanceBulletCount(it.Body())
			ac := fmt.Sprintf("acceptance: %d", bullets)
			if bullets == 0 && (it.Type() == "" || it.Type() == "feature") {
				ac = "acceptance: missing"
				warnings = append(warnings, fmt.Sprintf("%s: no `## Acceptance` section", e.Path))
			}
			fmt.Printf("  %2d. %s  %-32s %-7s %sp   %s\n", i+1, marker, truncate(it.Title(), 32), it.Type(), estStr, ac)
			if it.Type() == "feature" && p > 8 {
				warnings = append(warnings, fmt.Sprintf("%s: feature %s pts exceeds 8-point cap", e.Path, est))
			}
			if it.Type() == "feature" && (est == "" || p == 0) {
				warnings = append(warnings, fmt.Sprintf("%s: feature with no estimate", e.Path))
			}
		}
		fmt.Printf("\nCommitted: %.0f pts across %d stories. Velocity: %.0f.\n", pts, printed, velocity)
		if velocity > 0 && pts > velocity {
			fmt.Printf("Overcommit: %.0f pts above rolling velocity. Trim from the bottom.\n", pts-velocity)
		}
		if len(warnings) > 0 {
			fmt.Println("\nWarnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
		}
		return nil
	},
}

// RetroCommand renders an end-of-iteration summary the human reads
// before answering the three retro questions.
var RetroCommand = &cli.Command{
	Name:  "retro",
	Usage: "Print an end-of-iteration retro summary (velocity, mix, accepted, rejected, recent learnings)",
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
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		var all []*backlog.BacklogItem
		for _, d := range dirs {
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return err
			}
			all = append(all, bck.AllItems()...)
		}
		var feed []*backlog.BacklogItem
		acceptedCount, rejectedCount := 0, 0
		for _, it := range all {
			if backlog.CountsForVelocity(it, cfg) {
				feed = append(feed, it)
			}
			if strings.EqualFold(it.Status(), backlog.AcceptedStatus.Name) {
				acceptedCount++
			}
			if strings.EqualFold(it.Status(), backlog.RejectedStatus.Name) {
				rejectedCount++
			}
		}
		overrides, _ := backlog.LoadIterationOverrides(root)
		velocity, _, _ := backlog.ComputeVelocity(time.Now(), feed, cfg, overrides)
		volatility := backlog.VolatilityPercent(time.Now(), feed, cfg, overrides)
		median := backlog.MedianCycleTime(backlog.CycleTimes(all))
		rejRows := backlog.RejectionRates(time.Now(), all, cfg)
		latest := 0.0
		if n := len(rejRows); n > 0 {
			latest = rejRows[n-1].Percent
		}
		fmt.Println("Retro summary")
		fmt.Println()
		fmt.Printf("  velocity:        %.0f pts\n", velocity)
		fmt.Printf("  volatility:      %.0f%%\n", volatility)
		fmt.Printf("  cycle time:      %s (median)\n", formatHoursDur(median))
		fmt.Printf("  rejection rate:  %.0f%% (latest, target band 5-15%%)\n", latest)
		fmt.Printf("  accepted total:  %d\n", acceptedCount)
		fmt.Printf("  rejected total:  %d\n", rejectedCount)
		fmt.Println()
		fmt.Println("Three questions:")
		fmt.Println("  1. What worked?")
		fmt.Println("  2. What did not work?")
		fmt.Println("  3. What changes for next iteration?")
		fmt.Println()
		fmt.Println("Record outputs:")
		fmt.Println("  am team-agreements --set \"...\"      (new agreement)")
		fmt.Println("  am record-learning \"...\"            (one-line learning)")
		// Show recent learnings if any.
		learningsPath := filepath.Join(root, "learnings.md")
		if data, err := os.ReadFile(learningsPath); err == nil {
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			n := len(lines)
			if n > 5 {
				lines = lines[n-5:]
			}
			fmt.Println()
			fmt.Println("Recent learnings:")
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				fmt.Printf("  %s\n", line)
			}
		}
		return nil
	},
}

func acceptanceBulletCount(body string) int {
	return len(backlog.AcceptanceBulletTexts(body))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "…"
}

const inceptionTemplate = `# Inception

## The user

Who, specifically, are we building this for? Real users with real circumstances, not a demographic.

## The goal

What changes for the user when we ship? Concrete, observable.

## The reason

Why this, why now? What changes if we do not?

## Success

How will we know it worked? The smallest signal that would tell the team the project did its job.

## Constraints

What can't move? Budget, deadline, dependencies, regulatory.

## Out of scope

What are we explicitly not doing?
`
