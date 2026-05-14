package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

var BlockCommand = &cli.Command{
	Name:      "block",
	Usage:     "Mark a story as blocked, optionally with a reason",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "reason", Usage: "blocker reason; stored in `blocked_reason:` frontmatter"},
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
		item.SetBlocked(true, c.String("reason"))
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s -> blocked\n", filepath.Base(path))
		return nil
	},
}

var UnblockCommand = &cli.Command{
	Name:      "unblock",
	Usage:     "Clear a story's blocked flag",
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
		item.SetBlocked(false, "")
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s -> unblocked\n", filepath.Base(path))
		return nil
	},
}
