package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

// emitJSON marshals v to the standard JSON shape the VS Code extension
// (and any future per-verb consumer) expects, and writes it to stdout.
func emitJSON(v any) error {
	out, err := json.Marshal(v)
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// ShowCommand wraps view-only renderers (priority, icebox, epic,
// iteration). All output is ASCII so it works inline in a chat or a
// terminal.
var ShowCommand = &cli.Command{
	Name:      "show",
	Usage:     "Render a view: priority, icebox, epic, or iteration",
	ArgsUsage: "VIEW [ARGS]",
	Commands: []*cli.Command{
		showPriorityCmd,
		showIceboxCmd,
		showEpicCmd,
		showIterationCmd,
		showBurnupCmd,
		showCFDCmd,
	},
}

var showCFDCmd = &cli.Command{
	Name:  "cfd",
	Usage: "Render the project's cumulative-flow diagram (accepted / in-flight / backlog per day) as ASCII",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "days", Value: 30, Usage: "lookback window in days"},
		&cli.BoolFlag{Name: "json", Usage: "emit structured rows instead of ASCII"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		if c.Bool("json") {
			res, err := mcpserver.CumulativeFlow(ctx, root, mcpserver.CFDArgs{Days: c.Int("days")})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		structure := backlog.NewBacklogsStructure(root)
		dirs, err := structure.BacklogDirs()
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
		end := time.Now().UTC()
		start := end.AddDate(0, 0, -c.Int("days"))
		rows := backlog.CFDRows(all, start, end)
		fmt.Print(backlog.CFDASCII(rows))
		return nil
	},
}

var showBurnupCmd = &cli.Command{
	Name:      "burnup",
	Usage:     "Render an ASCII per-day burnup for the current iteration",
	ArgsUsage: "[OFFSET]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit the burnup as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		offset := 0
		if c.NArg() == 1 {
			fmt.Sscanf(c.Args().Get(0), "%d", &offset)
		}
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
			res, err := mcpserver.BurnupChart(ctx, root, mcpserver.BurnupArgs{Backlog: filepath.Base(dir), Offset: offset})
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
		now := time.Now().In(cfg.IterationLocation())
		start := backlog.IterationStartFor(now, cfg).AddDate(0, 0, 7*cfg.Iteration.LengthWeeks*offset)
		end := start.AddDate(0, 0, 7*cfg.Iteration.LengthWeeks)
		rows := backlog.BurnupRows(bck.AllItems(), start, end)
		fmt.Print(backlog.BurnupASCII(rows, start, end))
		return nil
	},
}

var showPriorityCmd = &cli.Command{
	Name:  "priority",
	Usage: "Render _priority.md split into iteration bands using rolling velocity",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "iterations", Usage: "max iteration bands to draw before dumping the rest as Backlog", Value: 2},
		&cli.BoolFlag{Name: "hide-accepted", Usage: "drop already-accepted items from the rendering"},
		&cli.BoolFlag{Name: "json", Usage: "emit the priority list as JSON (machine-readable)"},
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
			res, err := mcpserver.PriorityList(ctx, root, mcpserver.PriorityListArgs{Backlog: filepath.Base(dir)})
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
		text, err := backlog.PriorityASCII(bck, dir, cfg, time.Now(), c.Int("iterations"), c.Bool("hide-accepted"))
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}

var showIceboxCmd = &cli.Command{
	Name:  "icebox",
	Usage: "Render _icebox.md as a plain stack-rank list",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit the icebox list as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		dir, err := filepath.Abs(".")
		if err != nil {
			return err
		}
		if c.Bool("json") {
			root, err := findRootDirectory()
			if err != nil {
				return err
			}
			res, err := mcpserver.IceboxList(ctx, root, mcpserver.IceboxListArgs{Backlog: filepath.Base(dir)})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		text, err := backlog.IceboxASCII(bck, dir)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}

var showEpicCmd = &cli.Command{
	Name:      "epic",
	Usage:     "Render an ASCII burnup for a single epic by slug",
	ArgsUsage: "EPIC_SLUG",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "emit epic progress as JSON (machine-readable)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am show epic EPIC_SLUG")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		if c.Bool("json") {
			res, err := mcpserver.EpicProgress(ctx, root, mcpserver.EpicProgressArgs{Slug: c.Args().Get(0)})
			if err != nil {
				return err
			}
			return emitJSON(res)
		}
		text, err := backlog.EpicASCII(root, c.Args().Get(0))
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}

var showIterationCmd = &cli.Command{
	Name:      "iteration",
	Usage:     "Render a single iteration window: 0=current, 1=next, ...",
	ArgsUsage: "[OFFSET]",
	Action: func(ctx context.Context, c *cli.Command) error {
		offset := 0
		if c.NArg() == 1 {
			fmt.Sscanf(c.Args().Get(0), "%d", &offset)
		}
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
		cfg, err := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(dir)
		if err != nil {
			return err
		}
		text, err := backlog.IterationASCII(bck, dir, cfg, time.Now(), offset)
		if err != nil {
			return err
		}
		fmt.Print(text)
		return nil
	},
}
