package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"sort"
	"strconv"
	"strings"
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
		status := c.String("s")

		if !backlog.IsValidStatusCode(status) {
			fmt.Printf("illegal status: %s\n", status)
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

		items := bck.ItemsByStatusAndUser(status, user)
		pointsByUser := make(map[string]float64)
		for _, item := range items {
			estimated, _ := strconv.ParseFloat(item.Estimate(), 64)
			pointsByUser[item.Assigned()] += estimated
		}
		users := make([]string, 0, len(pointsByUser))
		for user := range pointsByUser {
			users = append(users, user)
		}
		sort.Strings(users)

		userHeader, pointsHeader := "User", "Total Points"
		maxUserLen := len(userHeader)
		for _, user := range users {
			if len(user) > maxUserLen {
				maxUserLen = len(user)
			}
		}

		fmt.Printf("Status: %s\n", backlog.StatusNameByCode(status))
		fmt.Printf("-%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)))
		fmt.Printf(" %s | %s\n", utils.PadStringRight(userHeader, maxUserLen), pointsHeader)
		fmt.Printf("-%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)))
		for _, user := range users {
			points := int(pointsByUser[user])
			pointsStr := utils.PadIntLeft(points, len(pointsHeader))
			if points == 0 {
				pointsStr = ""
			}
			fmt.Printf(" %s | %s\n", utils.PadStringRight(user, maxUserLen), pointsStr)
		}

		return nil
	},
}
