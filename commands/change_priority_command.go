package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
)

var ChangePriorityCommand = cli.Command{
	Name:      "change-priority",
	Usage:     "Change story priority",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
	},
	Action: func(c *cli.Context) error {
		var items []*backlog.BacklogItem
		var err error
		if items, err = showBacklogItems(c); items == nil {
			return err
		}
		//reader := bufio.NewReader(os.Stdin)
		return nil
	},
}
