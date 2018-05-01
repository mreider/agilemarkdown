package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
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
	},
	Action: func(c *cli.Context) error {
		user := c.String("u")
		statusCode := c.String("s")

		if statusCode != "" && !backlog.IsValidStatusCode(statusCode) {
			fmt.Printf("illegal status: %s\n", statusCode)
			return nil
		}

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		bck, err := backlog.LoadBacklog(".")
		if err != nil {
			return err
		}

		var statuses []*backlog.BacklogItemStatus
		if statusCode == "" {
			statuses = []*backlog.BacklogItemStatus{backlog.StatusByCode("f"), backlog.StatusByCode("g"), backlog.StatusByCode("h")}
		} else {
			statuses = []*backlog.BacklogItemStatus{backlog.StatusByCode(statusCode)}
		}

		for _, status := range statuses {
			items := bck.ItemsByStatusAndUser(status.Code, user)
			lines := backlog.BacklogView{}.WriteBacklogItems(items, fmt.Sprintf("Status: %s", status.Name), "")
			fmt.Println(strings.Join(lines, "\n"))
			fmt.Println("")
		}

		return nil
	},
}
