package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// JournalCommand renders the audit trail as a narrative grouped by story:
// each item's arc (started -> finished -> delivered -> accepted, with the
// rationale recorded on each transition) in one place. Where `am log` is
// a flat chronological feed, `am journal` reads like a flight recorder —
// the view a human reviews after an autonomous run to reconstruct what
// the agent did and why, instead of gating each story in real time.
var JournalCommand = &cli.Command{
	Name:  "journal",
	Usage: "Narrative audit trail grouped by story (what was done and why) — the autonomous-mode review surface",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "item", Usage: "only stories whose path contains this substring"},
		&cli.BoolFlag{Name: "json", Usage: "emit the grouped journal as JSON"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		entries, err := backlog.LoadActivity(root)
		if err != nil {
			return err
		}
		itemFilter := strings.TrimSpace(c.String("item"))
		if itemFilter != "" {
			kept := entries[:0:0]
			for _, e := range entries {
				if strings.Contains(e.Item, itemFilter) {
					kept = append(kept, e)
				}
			}
			entries = kept
		}
		groups := backlog.GroupActivityByItem(entries)

		if c.Bool("json") {
			out := make([]map[string]any, 0, len(groups))
			for _, g := range groups {
				evs := make([]map[string]string, 0, len(g.Entries))
				for _, e := range g.Entries {
					evs = append(evs, map[string]string{
						"time":   e.Time.Format("2006-01-02T15:04:05Z07:00"),
						"actor":  e.Actor,
						"verb":   e.Verb,
						"detail": e.Detail,
					})
				}
				out = append(out, map[string]any{
					"item":   g.Item,
					"title":  journalTitle(root, g.Item),
					"events": evs,
				})
			}
			return emitJSON(out)
		}

		cfg, _ := config.LoadConfig(filepath.Join(root, ".am", "config.yaml"))
		mode := config.ModeCoached
		if cfg != nil {
			mode = cfg.EffectiveMode()
		}
		fmt.Printf("Journal — %s  (mode: %s)\n", filepath.Base(root), mode)
		if len(groups) == 0 {
			fmt.Println("\n(no activity recorded yet)")
			return nil
		}
		for _, g := range groups {
			fmt.Println()
			if g.Item == "" {
				fmt.Println("(project)")
			} else {
				fmt.Println(journalTitle(root, g.Item))
			}
			for _, e := range g.Entries {
				ts := e.Time.Local().Format("2006-01-02 15:04")
				line := fmt.Sprintf("  %s  %-10s (%s)", ts, e.Verb, e.Actor)
				if e.Detail != "" {
					line += "  — " + e.Detail
				}
				fmt.Println(line)
			}
		}
		return nil
	},
}

// journalTitle resolves a story's human title from its file, falling back
// to the path's base name (the story may have been archived or removed).
func journalTitle(root, item string) string {
	if item == "" {
		return "(project)"
	}
	if it, err := backlog.LoadBacklogItem(filepath.Join(root, item)); err == nil {
		if t := strings.TrimSpace(it.Title()); t != "" {
			return t
		}
	}
	return filepath.Base(item)
}
