package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// AlignCommand prints a structured view of one story so the human (or
// the calling agent) can sanity-check the story before pulling it. The
// LLM-driven restatement that the am-align skill performs is layered
// on top; this CLI verb is read-only and printable.
var AlignCommand = &cli.Command{
	Name:      "align",
	Usage:     "Print a structured view of a story before pulling it",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am align ITEM_PATH")
		}
		path := mustItemPath(c.Args().Get(0))
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		fmt.Printf("Story:    %s (%s)\n", item.Title(), filepath.Base(path))
		typ := item.Type()
		if typ == "" {
			typ = "feature"
		}
		fmt.Printf("Type:     %s\n", typ)
		fmt.Printf("Status:   %s\n", item.Status())
		if est := strings.TrimSpace(item.Estimate()); est != "" {
			fmt.Printf("Estimate: %s pts\n", est)
		} else {
			fmt.Println("Estimate: (not pointed)")
		}
		if assignees := item.Assignees(); len(assignees) > 0 {
			fmt.Printf("Owners:   %s\n", strings.Join(assignees, ", "))
		}
		if item.Blocked() {
			fmt.Printf("Blocked:  yes")
			if reason := item.BlockedReason(); reason != "" {
				fmt.Printf(" (%s)", reason)
			}
			fmt.Println()
		}

		// Acceptance bullets removed from the model. The body is the
		// only "what to verify" surface left, and skills like
		// /am-align use the LLM to restate it; we don't render it
		// here. Keep the warning for an unpointed feature since
		// estimation is still a real gate.
		var warnings []string
		if typ == "feature" && strings.TrimSpace(item.Estimate()) == "" {
			warnings = append(warnings, "not estimated; point the story before starting")
		}
		if len(warnings) > 0 {
			fmt.Println()
			fmt.Println("Warnings:")
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
		}
		return nil
	},
}
