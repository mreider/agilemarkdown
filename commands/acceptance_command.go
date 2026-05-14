package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

// AcceptanceCommand groups the read and write operations on a story's
// acceptance bullets. Mirrors the list_acceptance, set_acceptance_state,
// and append_acceptance_bullet MCP tools.
var AcceptanceCommand = &cli.Command{
	Name:  "acceptance",
	Usage: "Manage the checkbox bullets under '## Acceptance' on a story",
	Commands: []*cli.Command{
		{
			Name:      "list",
			Usage:     "Print the acceptance bullets for an item",
			ArgsUsage: "ITEM_PATH",
			Action:    acceptanceListAction,
		},
		{
			Name:      "claim",
			Usage:     "Flip a bullet to claimed (dev says: I think this is done)",
			ArgsUsage: "ITEM_PATH INDEX",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "note", Usage: "optional claim note recorded in a trailing HTML comment"},
			},
			Action: acceptanceClaimAction,
		},
		{
			Name:      "verify",
			Usage:     "Flip a bullet to verified (PM says: this passes)",
			ArgsUsage: "ITEM_PATH INDEX",
			Action:    acceptanceVerifyAction,
		},
		{
			Name:      "reopen",
			Usage:     "Flip a bullet back to open",
			ArgsUsage: "ITEM_PATH INDEX",
			Action:    acceptanceReopenAction,
		},
		{
			Name:      "add",
			Usage:     "Append a new open bullet to '## Acceptance'",
			ArgsUsage: "ITEM_PATH TEXT",
			Action:    acceptanceAddAction,
		},
	},
}

func acceptanceListAction(ctx context.Context, c *cli.Command) error {
	if c.NArg() != 1 {
		fmt.Println("usage: am acceptance list ITEM_PATH")
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	bullets := backlog.ParseAcceptance(item.Body())
	if len(bullets) == 0 {
		fmt.Println("(no acceptance bullets)")
		return nil
	}
	for _, b := range bullets {
		fmt.Printf("%2d. %s %s", b.Index, stateMarker(b.State), b.Text)
		if b.ClaimNote != "" {
			fmt.Printf("\n    claim: %s", b.ClaimNote)
		}
		fmt.Println()
	}
	return nil
}

func acceptanceClaimAction(ctx context.Context, c *cli.Command) error {
	return writeAcceptanceState(c, backlog.AcceptanceClaimed, c.String("note"))
}

func acceptanceVerifyAction(ctx context.Context, c *cli.Command) error {
	return writeAcceptanceState(c, backlog.AcceptanceVerified, "")
}

func acceptanceReopenAction(ctx context.Context, c *cli.Command) error {
	return writeAcceptanceState(c, backlog.AcceptanceOpen, "")
}

func acceptanceAddAction(ctx context.Context, c *cli.Command) error {
	if c.NArg() < 2 {
		fmt.Println("usage: am acceptance add ITEM_PATH TEXT")
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	text := strings.TrimSpace(strings.Join(c.Args().Tail(), " "))
	if text == "" {
		fmt.Println("bullet text is required")
		return nil
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	item.SetBody(backlog.AppendAcceptanceBullet(item.Body(), text))
	item.SetModified(utils.GetCurrentTimestamp())
	if err := item.Save(); err != nil {
		return err
	}
	fmt.Printf("%s: bullet added\n", filepath.Base(path))
	return nil
}

func writeAcceptanceState(c *cli.Command, state backlog.AcceptanceState, claimNote string) error {
	verb := "acceptance " + verbForState(state)
	if c.NArg() != 2 {
		fmt.Printf("usage: am %s ITEM_PATH INDEX\n", verb)
		return nil
	}
	path := mustItemPath(c.Args().Get(0))
	idx, err := strconv.Atoi(c.Args().Get(1))
	if err != nil {
		return fmt.Errorf("index must be an integer: %w", err)
	}
	item, err := backlog.LoadBacklogItem(path)
	if err != nil {
		return err
	}
	body, err := backlog.SetAcceptanceState(item.Body(), idx, state, claimNote)
	if err != nil {
		return err
	}
	item.SetBody(body)
	item.SetModified(utils.GetCurrentTimestamp())
	if err := item.Save(); err != nil {
		return err
	}
	fmt.Printf("%s: bullet %d -> %s\n", filepath.Base(path), idx, state)
	return nil
}

func stateMarker(s backlog.AcceptanceState) string {
	switch s {
	case backlog.AcceptanceClaimed:
		return "[~]"
	case backlog.AcceptanceVerified:
		return "[x]"
	}
	return "[ ]"
}

func verbForState(s backlog.AcceptanceState) string {
	switch s {
	case backlog.AcceptanceClaimed:
		return "claim"
	case backlog.AcceptanceVerified:
		return "verify"
	}
	return "reopen"
}
