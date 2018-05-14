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
		statusCode := c.String("s")

		if !backlog.IsValidStatusCode(statusCode) {
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

		filter := &backlog.BacklogItemsAndFilter{}
		filter.And(backlog.NewBacklogItemsStatusCodeFilter(statusCode))
		filter.And(backlog.NewBacklogItemsAssignedFilter(user))
		items := bck.FilteredItems(filter)

		pointsByUser := make(map[string]float64)
		tagsByUser := make(map[string][]string)
		for _, item := range items {
			estimated, _ := strconv.ParseFloat(item.Estimate(), 64)
			pointsByUser[item.Assigned()] += estimated

			tags := item.Tags()
			for _, tag := range tags {
				if !utils.ContainsStringIgnoreCase(tagsByUser[item.Assigned()], tag) {
					tagsByUser[item.Assigned()] = append(tagsByUser[item.Assigned()], tag)
				}
			}
		}
		users := make([]string, 0, len(pointsByUser))
		for user := range pointsByUser {
			users = append(users, user)
		}
		sort.Strings(users)

		userHeader, pointsHeader, tagsHeader := "User", "Total Points", "Tags"
		maxUserLen, maxTagsLen := len(userHeader), len(tagsHeader)
		for _, user := range users {
			if len(user) > maxUserLen {
				maxUserLen = len(user)
			}

			tags := strings.Join(tagsByUser[user], " ")
			if len(tags) > maxTagsLen {
				maxTagsLen = len(tags)
			}
		}

		fmt.Printf("Status: %s\n", backlog.StatusNameByCode(statusCode))
		fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen))
		fmt.Printf(" %s | %s | %s\n", utils.PadStringRight(userHeader, maxUserLen), pointsHeader, tagsHeader)
		fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen))
		for _, user := range users {
			points := int(pointsByUser[user])
			pointsStr := utils.PadIntLeft(points, len(pointsHeader))
			if points == 0 {
				pointsStr = strings.Repeat(" ", len(pointsHeader))
			}
			tags := strings.Join(tagsByUser[user], " ")
			fmt.Printf(" %s | %s | %s\n", utils.PadStringRight(user, maxUserLen), pointsStr, tags)
		}

		return nil
	},
}
