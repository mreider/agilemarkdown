package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var CreateItemCommand = cli.Command{
	Name:      "create-item",
	Usage:     "Create a new item for the backlog",
	ArgsUsage: "ITEM_NAME",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "simulate",
			Hidden: true,
		},
		cli.StringFlag{
			Name:   "user",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
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
		itemTitle := strings.Join(c.Args(), " ")
		action := actions.NewCreateItemAction(".", itemTitle, user, simulate)
		return action.Execute()
	},
}
