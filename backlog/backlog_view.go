package backlog

import (
	"fmt"
	"github.com/buger/goterm"
	"github.com/mreider/agilemarkdown/utils"
	"strconv"
	"strings"
	"time"
)

type BacklogView struct {
}

func (bv BacklogView) WriteAsciiTable(items []*BacklogItem, title string, withOrderNumber bool) []string {
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

func (bv BacklogView) WriteMarkdownTable(items []*BacklogItem) []string {
	result := make([]string, 0, 50)
	headers := make([]string, 0, 2)
	headers = append(headers, fmt.Sprintf(" User | Title | Points "))
	headers = append(headers, "---|---|:---:")
	result = append(result, headers...)
	for _, item := range items {
		line := fmt.Sprintf(" %s | [%s](%s) | %s ", item.Assigned(), item.Title(), item.Name(), item.Estimate())
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) Progress(bck *Backlog, weekCount, width int) (string, error) {
	items := bck.ItemsByStatus(FinishedStatus.Code)
	currentDate := time.Now().UTC()
	pointsByWeekDelta := make(map[int]float64)
	for _, item := range items {
		modified := item.Modified()
		weekDelta := utils.WeekDelta(currentDate, modified)
		if -weekCount < weekDelta && weekDelta <= 0 {
			itemPoints, _ := strconv.ParseFloat(item.Estimate(), 64)
			pointsByWeekDelta[weekDelta] += itemPoints
		}
	}

	chart := goterm.NewLineChart(width, 20)

	data := new(goterm.DataTable)
	data.AddColumn("Week")
	data.AddColumn("Points")

	for i := -weekCount + 1; i <= 0; i++ {
		data.AddRow(float64(i), pointsByWeekDelta[i])
	}

	return chart.Draw(data), nil
}
