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

func (bv BacklogView) WriteBacklogItems(items []*BacklogItem, title string, withOrderNumber bool) []string {
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
		result = append(result, fmt.Sprintf("%s", title))
	}
	headers := make([]string, 0, 3)
	headers = append(headers, fmt.Sprintf("-%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader))))
	headers = append(headers, fmt.Sprintf(" %s | %s | %s ", utils.PadStringRight(userHeader, maxAssignedLen), utils.PadStringRight(titleHeader, maxTitleLen), pointsHeader))
	headers = append(headers, fmt.Sprintf("-%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader))))
	if withOrderNumber {
		headers[0] = "------" + headers[0]
		headers[1] = "   # |" + headers[1]
		headers[2] = "------" + headers[2]
	}
	result = append(result, headers...)
	for i, item := range items {
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		estimateStr := utils.PadIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = strings.Repeat("", len(pointsHeader))
		}
		line := fmt.Sprintf(" %s | %s | %s ", utils.PadStringRight(item.Assigned(), maxAssignedLen), utils.PadStringRight(item.Title(), maxTitleLen), estimateStr)
		if withOrderNumber {
			line = fmt.Sprintf(" %s |", utils.PadIntLeft(i+1, 3)) + line
		}
		result = append(result, line)
	}
	if len(items) > 0 {
		footer := fmt.Sprintf("-%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
		if withOrderNumber {
			footer = "------" + footer
		}
		result = append(result, footer)
	}
	return result
}
