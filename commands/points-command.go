package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"sort"
	"strconv"
	"strings"
)

type PointsCommand struct {
	User    string `short:"u" required:"false" description:"user"`
	Status  string `short:"s" required:"false" description:"status (f, l, g or h)" default:"f"`
	RootDir string
}

func (*PointsCommand) Name() string {
	return "points"
}

func (cmd *PointsCommand) Execute(args []string) error {
	if cmd.Status != "f" && cmd.Status != "l" && cmd.Status != "g" && cmd.Status != "h" {
		return fmt.Errorf("illegal status: %s", cmd.Status)
	}
	if err := checkIsBacklogDirectory(cmd.RootDir); err != nil {
		return err
	}
	bck, err := backlog.LoadBacklog(cmd.RootDir)
	if err != nil {
		return err
	}

	items := bck.ItemsByStatusAndUser(cmd.Status, cmd.User)
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

	fmt.Printf("Status: %s\n", backlog.GetStatusByCode(cmd.Status))
	fmt.Printf("-%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)))
	fmt.Printf(" %s | %s\n", padStringRight(userHeader, maxUserLen), pointsHeader)
	fmt.Printf("-%s---%s\n", strings.Repeat("-", maxUserLen), strings.Repeat("-", len(pointsHeader)))
	for _, user := range users {
		points := int(pointsByUser[user])
		pointsStr := padIntLeft(points, len(pointsHeader))
		if points == 0 {
			pointsStr = ""
		}
		fmt.Printf(" %s | %s\n", padStringRight(user, maxUserLen), pointsStr)
	}

	return nil
}
