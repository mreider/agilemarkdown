package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var WorkCommand = cli.Command{
	Name:      "work",
	Usage:     "Show user work by status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "u",
			Usage: "User Name",
		},
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
		cli.StringFlag{
			Name:  "t",
			Usage: "List of Tags",
		},
	},
	Action: func(c *cli.Context) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		backlogDir, _ := filepath.Abs(".")

		user := c.String("u")
		statusCode := c.String("s")
		tags := c.String("t")

		if c.NArg() > 0 {
			fmt.Printf("illegal arguments: %s\n", strings.Join(c.Args(), " "))
			return nil
		}

		action := actions.NewWorkAction(backlogDir, statusCode, user, tags)
		return action.Execute()
	},
}
