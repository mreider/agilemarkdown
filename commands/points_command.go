package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
)

var PointsCommand = cli.Command{
	Name:      "points",
	Usage:     "Show total points by user and status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "u",
			Usage: "User Name",
		},
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
			Value: backlog.DoingStatus.Code,
		},
	},
	Action: func(c *cli.Context) error {
		user := c.String("u")
		statusCode := c.String("s")

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}

		action := actions.NewPointsAction(".", statusCode, user)
		return action.Execute()
	},
}
