package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"sort"
	"strconv"
	"strings"
)

var WorkCommand = cli.Command{
	Name:      "work",
	Usage:     "Show user work by status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "u",
			Usage: "User Namne",
		},
		cli.StringFlag{
			Name:  "s",
			Usage: "Status - either l (landed), f (flying), g (gate), or h (hangar).",
			Value: "f",
		},
	},
	Action: func(c *cli.Context) error {
		user := c.String("u")
		status := c.String("s")

		if status != "f" && status != "l" && status != "g" && status != "h" {
			fmt.Printf("illegal status: %s\n", status)
			return nil
		}
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(".")
		if err != nil {
			return err
		}

		items := bck.ItemsByStatusAndUser(status, user)
		sort.Slice(items, func(i, j int) bool {
			if items[i].Assigned() < items[j].Assigned() {
				return true
			}
			if items[i].Assigned() > items[j].Assigned() {
				return false
			}
			return items[i].Title() < items[j].Title()
		})

		userHeader, titleHeader, pointsHeader := "User", "Title", "Points"
		maxAssignedLen, maxTitleLen := len(userHeader), len(titleHeader)
		for _, item := range items {
			if len(item.Assigned()) > maxAssignedLen {
				maxAssignedLen = len(item.Assigned())
			}
			if len(item.Title()) > maxTitleLen {
				maxTitleLen = len(item.Title())
			}
		}

		fmt.Printf("Status: %s\n", backlog.GetStatusByCode(status))
		fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
		fmt.Printf(" %s | %s | %s\n", padStringRight(userHeader, maxAssignedLen), padStringRight(titleHeader, maxTitleLen), pointsHeader)
		fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
		for _, item := range items {
			estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
			estimateStr := padIntLeft(int(estimate), len(pointsHeader))
			if estimate == 0 {
				estimateStr = ""
			}
			fmt.Printf(" %s | %s | %s\n", padStringRight(item.Assigned(), maxAssignedLen), padStringRight(item.Title(), maxTitleLen), estimateStr)
		}

		return nil
	},
}
