package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
)

var ChangeStatusCommand = cli.Command{
	Name:      "change-status",
	Usage:     "Change story status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
	},
	Action: func(c *cli.Context) error {
		statusCode := c.String("s")
		if statusCode == "" {
			fmt.Println("-s option is required")
			return nil
		}

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		backlogDir, _ := filepath.Abs(".")

		action := actions.NewChangeStatusAction(backlogDir, statusCode)
		return action.Execute()
	},
}
