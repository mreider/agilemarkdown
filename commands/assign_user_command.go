package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
)

var AssignUserCommand = cli.Command{
	Name:      "assign",
	Usage:     "Assign a story to a user",
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
		if !backlog.IsValidStatusCode(statusCode) {
			fmt.Printf("illegal status: %s\n", statusCode)
			return nil
		}
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		backlogDir, _ := filepath.Abs(".")

		action := actions.NewAssignUserAction(backlogDir, statusCode)
		return action.Execute()
	},
}
