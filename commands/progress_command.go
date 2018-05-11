package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"strconv"
)

var ProgressCommand = cli.Command{
	Name:      "progress",
	Usage:     "Show the progress of a backlog over time",
	ArgsUsage: "NUMBER_OF_WEEKS",
	Action: func(c *cli.Context) error {
		var weekCount int
		if c.NArg() > 0 {
			weekCount, _ = strconv.Atoi(c.Args()[0])
		}
		if weekCount <= 0 {
			weekCount = 12
		}

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}

		bck, err := backlog.LoadBacklog(".")
		if err != nil {
			return err
		}
		chart, err := backlog.BacklogView{}.Progress(bck, weekCount, 84)
		if err != nil {
			return err
		}
		fmt.Println(chart)

		return nil
	},
}
