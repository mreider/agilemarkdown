package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

var CreateItemCommand = &cli.Command{
	Name:      "create-item",
	Usage:     "Create a new item for the backlog",
	ArgsUsage: "ITEM_NAME",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:   "simulate",
			Hidden: true,
		},
		&cli.StringFlag{
			Name:   "user",
			Hidden: true,
		},
		// Target column for the freshly-created item. Defaults to
		// icebox (the historical behavior). When set to "priority",
		// the item is inserted directly into _priority.md instead of
		// _icebox.md so the + button on the Current column doesn't
		// have to chain through `am unice`.
		&cli.StringFlag{
			Name:  "target",
			Usage: "target column for the new item: icebox (default) or priority",
		},
		// Position within the target column. Defaults to bottom (the
		// historical behavior, and the safe choice). The VS Code
		// Settings tab exposes this per-column so users can flip the
		// default to top if they prefer new ideas at the top.
		&cli.StringFlag{
			Name:  "position",
			Usage: "position within the target column: bottom (default) or top",
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		simulate := c.Bool("simulate")
		user := c.String("user")
		target := strings.ToLower(strings.TrimSpace(c.String("target")))
		position := strings.ToLower(strings.TrimSpace(c.String("position")))
		if target != "priority" {
			target = "icebox"
		}
		if position != "top" {
			position = "bottom"
		}

		// Running outside a backlog folder used to print the error and
		// exit 0 (silent failure for any wrapper that got cwd wrong).
		// Return the error so the caller learns about it.
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		if c.NArg() == 0 {
			if !simulate {
				return fmt.Errorf("an item name should be specified")
			}
			return nil
		}
		itemTitle := strings.Join(c.Args().Slice(), " ")
		action := actions.NewCreateItemAction(".", itemTitle, user, simulate)
		if err := action.Execute(); err != nil {
			return err
		}
		if simulate {
			return nil
		}
		// Insert into the target column at the requested position. The
		// item file already exists on disk thanks to CreateItemAction;
		// here we just place its order-file entry.
		dir, err := filepath.Abs(".")
		if err != nil {
			return nil
		}
		fileName := utils.GetValidFileName(itemTitle) + ".md"
		entry := backlog.OrderEntry{Title: itemTitle, Path: fileName}
		recordActivity("created", filepath.Join(dir, fileName), fmt.Sprintf("target=%s position=%s", target, position))

		ice, _ := backlog.LoadIcebox(dir)
		pri, _ := backlog.LoadPriority(dir)

		// Guard against double-inserting if the user re-runs the verb on
		// an existing file.
		alreadyInIcebox := ice != nil && ice.IndexOf(fileName) >= 0
		alreadyInPriority := pri != nil && pri.IndexOf(fileName) >= 0

		if target == "priority" {
			if alreadyInPriority {
				return nil
			}
			if alreadyInIcebox {
				ice.Remove(fileName)
				_ = ice.Save()
			}
			if pri == nil {
				return nil
			}
			if position == "top" {
				pri.InsertAt(0, entry)
			} else {
				pri.InsertBottom(entry)
			}
			_ = pri.Save()
		} else {
			if alreadyInIcebox {
				return nil
			}
			if alreadyInPriority {
				return nil
			}
			if ice == nil {
				return nil
			}
			if position == "top" {
				ice.InsertTop(entry)
			} else {
				ice.InsertBottom(entry)
			}
			_ = ice.Save()
		}
		return nil
	},
}
