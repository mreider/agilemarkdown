package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
	"path/filepath"
)

var ChangeStatusCommand = &cli.Command{
	Name:      "change-status",
	Usage:     "Change story status (legacy picker; prefer start/finish/deliver/accept/reject)",
	Hidden:    true,
	ArgsUsage: " ",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("List items currently in this status, then change them. %s", backlog.AllStatusesList()),
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
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
