package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"sort"
	"strconv"
	"strings"
)

type PointsAction struct {
	backlogDir string
	statusCode string
	user       string
}

func NewPointsAction(backlogDir, statusCode, user string) *PointsAction {
	return &PointsAction{backlogDir: backlogDir, statusCode: statusCode, user: user}
}

func (a *PointsAction) Execute() error {
	if !backlog.IsValidStatusCode(a.statusCode) {
		fmt.Printf("illegal status: %s\n", a.statusCode)
		return nil
	}
	bck, err := backlog.LoadBacklog(a.backlogDir)
	if err != nil {
		return err
	}

	filter := &backlog.BacklogItemsAndFilter{}
	filter.And(backlog.NewBacklogItemsStatusCodeFilter(a.statusCode))
	filter.And(backlog.NewBacklogItemsAssignedFilter(a.user))
	items := bck.FilteredActiveItems(filter)

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

	fmt.Printf("Status: %s\n", backlog.StatusNameByCode(a.statusCode))
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

}
