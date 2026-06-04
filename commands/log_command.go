package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// LogCommand renders the semantic audit trail recorded in
// `.am/activity.log`: a chronological feed of every state-changing
// backlog operation (created, started, finished, delivered, accepted,
// rejected, set-type, estimate, bug, chore). It answers "what has been
// happening on this backlog?" in one place, which is exactly the view
// you want when an autonomous agent has been working unattended.
//
// Distinct from `am history ITEM`, which shells out to git log for a
// single file: `am log` is pre-commit, multi-item, and intent-level.
var LogCommand = &cli.Command{
	Name:  "log",
	Usage: "Show the backlog activity log (audit trail of state changes)",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "limit", Aliases: []string{"n"}, Value: 50, Usage: "show at most N most-recent entries (0 = all)"},
		&cli.StringFlag{Name: "item", Usage: "only entries whose item path contains this substring"},
		&cli.StringFlag{Name: "verb", Usage: "only entries with this verb (created, started, delivered, accepted, ...)"},
		&cli.BoolFlag{Name: "json", Usage: "emit entries as JSON"},
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
		verbFilter := strings.ToLower(strings.TrimSpace(c.String("verb")))
		filtered := entries[:0:0]
		for _, e := range entries {
			if itemFilter != "" && !strings.Contains(e.Item, itemFilter) {
				continue
			}
			if verbFilter != "" && !strings.EqualFold(e.Verb, verbFilter) {
				continue
			}
			filtered = append(filtered, e)
		}

		// Most-recent-last in the file; apply the limit to the tail so
		// `am log` shows the latest activity by default.
		limit := c.Int("limit")
		if limit > 0 && len(filtered) > limit {
			filtered = filtered[len(filtered)-limit:]
		}

		if c.Bool("json") {
			out := make([]map[string]string, 0, len(filtered))
			for _, e := range filtered {
				out = append(out, map[string]string{
					"time":   e.Time.Format("2006-01-02T15:04:05Z07:00"),
					"actor":  e.Actor,
					"verb":   e.Verb,
					"item":   e.Item,
					"detail": e.Detail,
				})
			}
			return emitJSON(out)
		}

		if len(filtered) == 0 {
			fmt.Println("(no activity recorded yet)")
			return nil
		}
		for _, e := range filtered {
			ts := e.Time.Local().Format("2006-01-02 15:04")
			line := fmt.Sprintf("%s  %-9s  %-12s  %s", ts, truncate(e.Actor, 9), e.Verb, e.Item)
			if e.Detail != "" {
				line += "  (" + e.Detail + ")"
			}
			fmt.Println(line)
		}
		return nil
	},
}
