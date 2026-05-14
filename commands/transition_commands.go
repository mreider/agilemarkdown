package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// transitionCmd is a thin shared implementation: open ITEM_PATH, apply the
// state transition (which auto-stamps the relevant timestamp), save.
func transitionCmd(name, usage string, target *backlog.BacklogItemStatus) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "ITEM_PATH",
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() != 1 {
				fmt.Println("path to an item file is required")
				return nil
			}
			path, err := filepath.Abs(c.Args().Get(0))
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}
			item, err := backlog.LoadBacklogItem(path)
			if err != nil {
				return err
			}
			actions.ApplyStatusTransition(item, target)
			if err := item.Save(); err != nil {
				return err
			}
			fmt.Printf("%s -> %s\n", filepath.Base(path), target.Name)
			return nil
		},
	}
}

var (
	StartCommand   = transitionCmd("start", "Mark an item as started (in progress)", backlog.StartedStatus)
	FinishCommand  = transitionCmd("finish", "Mark an item as finished (dev complete)", backlog.FinishedStatus)
	DeliverCommand = deliverCmd()
	AcceptCommand  = transitionCmd("accept", "Accept an item (counts toward velocity)", backlog.AcceptedStatus)
)

// deliverCmd transitions an item to `delivered` and, with --prompt,
// immediately renders the PM acceptance ceremony so the dev pair does
// not have to remember the next move. The default behavior matches the
// other transition verbs (silent flip).
func deliverCmd() *cli.Command {
	return &cli.Command{
		Name:      "deliver",
		Usage:     "Mark an item as delivered (deployed, awaiting acceptance)",
		ArgsUsage: "ITEM_PATH",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "prompt", Usage: "after delivering, render the PM acceptance ceremony"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() != 1 {
				fmt.Println("path to an item file is required")
				return nil
			}
			path, err := filepath.Abs(c.Args().Get(0))
			if err != nil {
				return err
			}
			if !strings.HasSuffix(path, ".md") {
				path += ".md"
			}
			item, err := backlog.LoadBacklogItem(path)
			if err != nil {
				return err
			}
			actions.ApplyStatusTransition(item, backlog.DeliveredStatus)
			if err := item.Save(); err != nil {
				return err
			}
			fmt.Printf("%s -> delivered\n", filepath.Base(path))
			if c.Bool("prompt") {
				fmt.Println()
				if err := renderAcceptancePrompt(path); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// renderAcceptancePrompt writes the canonical PM ceremony for an item
// to stdout. Used by `am deliver --prompt` and shared with the
// stand-alone `am accept-prompt` verb.
func renderAcceptancePrompt(path string) error {
	root, err := findRootDirectory()
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(root, path)
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	typ := item.Type()
	if typ == "" {
		typ = "feature"
	}
	fmt.Printf("Story: %s (%s)\n", item.Title(), rel)
	fmt.Printf("Type: %s\n", typ)
	fmt.Printf("Status staging: %s -> accepted\n", item.Status())
	if est := item.Estimate(); est != "" {
		fmt.Printf("Estimate: %s pts\n", est)
	}
	bullets := backlog.AcceptanceBulletTexts(item.Body())
	if len(bullets) > 0 {
		fmt.Println("What to verify:")
		for _, b := range bullets {
			fmt.Printf("  - %s\n", b)
		}
	} else {
		fmt.Println("What to verify: (no `## Acceptance` section in body; review the diff)")
	}
	fmt.Println()
	fmt.Println("As PM, do you accept?")
	fmt.Println("  yes:  am accept " + rel)
	fmt.Println("  no:   am reject " + rel + " --reason \"...\"")
	return nil
}

// RejectCommand transitions an item to rejected. With --reason, appends a
// dated note under "## Rejection notes" in the item body. With
// --failing-bullet N, cites the failing acceptance bullet in the note
// and reopens it. Mirrors the reject_item MCP tool.
var RejectCommand = &cli.Command{
	Name:      "reject",
	Usage:     "Reject an item; optionally cite a failing acceptance bullet",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "reason", Usage: "rejection reason; appended under '## Rejection notes' with today's date"},
		&cli.IntFlag{Name: "failing-bullet", Usage: "1-based index of the acceptance bullet that failed; cited in the rejection note"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			fmt.Println("path to an item file is required")
			return nil
		}
		path, err := filepath.Abs(c.Args().Get(0))
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			path += ".md"
		}
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		actions.ApplyStatusTransition(item, backlog.RejectedStatus)

		failingBullet := c.Int("failing-bullet")
		var failingText string
		if failingBullet > 0 {
			for _, b := range backlog.ParseAcceptance(item.Body()) {
				if b.Index == failingBullet {
					failingText = b.Text
					break
				}
			}
			if failingText == "" {
				return fmt.Errorf("failing-bullet %d not found in body", failingBullet)
			}
			if body, err := backlog.SetAcceptanceState(item.Body(), failingBullet, backlog.AcceptanceOpen, ""); err == nil {
				item.SetBody(body)
			}
		}

		reason := strings.TrimSpace(c.String("reason"))
		if reason != "" || failingText != "" {
			body := item.Body()
			if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			now := time.Now().UTC().Format("2006-01-02")
			var line string
			switch {
			case failingText != "" && reason != "":
				line = fmt.Sprintf("- %s: Acceptance bullet %d (%q) failed. Reason: %s", now, failingBullet, failingText, reason)
			case failingText != "":
				line = fmt.Sprintf("- %s: Acceptance bullet %d (%q) failed.", now, failingBullet, failingText)
			default:
				line = fmt.Sprintf("- %s: %s", now, reason)
			}
			block := "\n## Rejection notes\n\n" + line + "\n"
			item.SetBody(body + block)
		}
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s -> rejected\n", filepath.Base(path))
		return nil
	},
}
