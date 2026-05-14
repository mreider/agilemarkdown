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
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		simulate := c.Bool("simulate")
		user := c.String("user")

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		if c.NArg() == 0 {
			if !simulate {
				fmt.Println("an item name should be specified")
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
		// Stage the new item into _icebox.md so it shows up in views
		// without requiring a separate `am sync` pass. Matches the
		// behavior of the MCP create_item tool.
		dir, err := filepath.Abs(".")
		if err != nil {
			return nil
		}
		fileName := utils.GetValidFileName(itemTitle) + ".md"
		ice, err := backlog.LoadIcebox(dir)
		if err != nil {
			return nil
		}
		if ice.IndexOf(fileName) >= 0 {
			return nil
		}
		pri, perr := backlog.LoadPriority(dir)
		if perr == nil && pri.IndexOf(fileName) >= 0 {
			return nil
		}
		ice.InsertBottom(backlog.OrderEntry{Title: itemTitle, Path: fileName})
		_ = ice.Save()
		return nil
	},
}
