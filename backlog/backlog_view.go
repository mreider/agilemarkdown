package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"sort"
	"strconv"
	"strings"
)

type BacklogView struct {
}

func (bv BacklogView) WriteBacklogItems(items []*BacklogItem, title string, rowDelimiter string) []string {
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

	result := make([]string, 0, 50)
	if title != "" {
		result = append(result, fmt.Sprintf("%s%s", title, rowDelimiter))
	}
	result = append(result, fmt.Sprintf("-%s---%s---%s%s", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), rowDelimiter))
	result = append(result, fmt.Sprintf(" %s | %s | %s%s", utils.PadStringRight(userHeader, maxAssignedLen), utils.PadStringRight(titleHeader, maxTitleLen), pointsHeader, rowDelimiter))
	result = append(result, fmt.Sprintf("-%s---%s---%s%s", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), rowDelimiter))
	for _, item := range items {
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		estimateStr := utils.PadIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = ""
		}
		result = append(result, fmt.Sprintf(" %s | %s | %s%s", utils.PadStringRight(item.Assigned(), maxAssignedLen), utils.PadStringRight(item.Title(), maxTitleLen), estimateStr, rowDelimiter))
	}
	if len(items) > 0 {
		result = append(result, fmt.Sprintf("-%s---%s---%s%s", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), rowDelimiter))
	}
	return result
}
