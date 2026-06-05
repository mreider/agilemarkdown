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
	"github.com/urfave/cli/v3"
)

// requireItemFile resolves an ITEM_PATH argument to an absolute path and
// verifies the file exists on disk. Without this guard the underlying
// markdown loader returns a fresh in-memory file for any missing path,
// so a typo like `am start Login-Flow.md` (run from the repo root instead
// of the backlog dir) silently writes a phantom file at the wrong
// location with no title or type.
func requireItemFile(arg string) (string, error) {
	path, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no such item: %s", path)
		}
		return "", err
	}
	return path, nil
}

// transitionCmd is a thin shared implementation: open ITEM_PATH, apply the
// state transition (which auto-stamps the relevant timestamp), save.
func transitionCmd(name, usage string, target *backlog.BacklogItemStatus) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		ArgsUsage: "ITEM_PATH",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "note", Usage: "rationale for this transition, recorded to the audit trail (am log / am journal)"},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			if c.NArg() != 1 {
				fmt.Println("path to an item file is required")
				return nil
			}
			path, err := requireItemFile(c.Args().Get(0))
			if err != nil {
				return err
			}
			item, err := backlog.LoadBacklogItem(path)
			if err != nil {
				return err
			}
			actions.ApplyStatusTransition(item, target)
			if err := item.Save(); err != nil {
				return err
			}
			// Capture the rationale on the audit trail. In autonomous mode
			// a self-accept with no note is still logged honestly so the
			// journal shows the seam was crossed deliberately.
			detail := strings.TrimSpace(c.String("note"))
			if detail == "" && target == backlog.AcceptedStatus && autonomousModeActive() {
				detail = "self-accepted (autonomous mode)"
			}
			recordActivity(item.Status(), path, detail)
			fmt.Printf("%s -> %s\n", filepath.Base(path), target.Name)
			if detail != "" {
				fmt.Printf("  note: %s\n", detail)
			}
			return nil
		},
	}
}

var (
	StartCommand   = transitionCmd("start", "Mark an item as started (in progress)", backlog.StartedStatus)
	FinishCommand  = transitionCmd("finish", "Mark an item as finished (dev complete)", backlog.FinishedStatus)
	DeliverCommand = deliverCmd()
	AcceptCommand  = transitionCmd("accept", "Accept an item (counts toward velocity)", backlog.AcceptedStatus)
	// UnstartCommand resets a story all the way back to unstarted,
	// clearing started/finished/delivered/accepted timestamps. Pivotal
	// allows this (e.g., "I claimed too soon, putting it back in the
	// queue"). The CLI exposes it as a verb because the VS Code
	// extension's state-machine row in the detail panel lets the user
	// click any state, and without unstart that click failed with
	// "unsupported status: unstarted".
	UnstartCommand = transitionCmd("unstart", "Reset an item back to unstarted (clears all transition timestamps)", backlog.UnstartedStatus)
)

func deliverCmd() *cli.Command {
	return transitionCmd("deliver", "Mark an item as delivered (deployed, awaiting acceptance)", backlog.DeliveredStatus)
}

// RejectCommand transitions an item to rejected. With --reason, appends
// a dated note under "## Rejection notes" in the item body.
var RejectCommand = &cli.Command{
	Name:      "reject",
	Usage:     "Reject an item; optionally cite a reason",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "reason", Usage: "rejection reason; appended under '## Rejection notes' with today's date"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			fmt.Println("path to an item file is required")
			return nil
		}
		path, err := requireItemFile(c.Args().Get(0))
		if err != nil {
			return err
		}
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		actions.ApplyStatusTransition(item, backlog.RejectedStatus)

		reason := strings.TrimSpace(c.String("reason"))
		if reason != "" {
			body := item.Body()
			if !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			now := time.Now().UTC().Format("2006-01-02")
			line := fmt.Sprintf("- %s: %s", now, reason)
			block := "\n## Rejection notes\n\n" + line + "\n"
			item.SetBody(body + block)
		}
		if err := item.Save(); err != nil {
			return err
		}
		recordActivity("rejected", path, reason)
		fmt.Printf("%s -> rejected\n", filepath.Base(path))
		return nil
	},
}
