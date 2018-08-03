package backlog

import (
	"bytes"
	"fmt"
	"github.com/buger/goterm"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/wcharczuk/go-chart"
	"strconv"
	"strings"
	"time"
)

type BacklogView struct {
}

func (bv BacklogView) WriteAsciiItems(items []*BacklogItem, status *BacklogItemStatus, withOrderNumber bool, withTotalPoints bool) []string {
	title := fmt.Sprintf("Status: %s", status.Name)
	userHeader, titleHeader, pointsHeader, tagsHeader := "User", "Title", "Points", "Tags"
	totalPointsCell := "Total Points"
	maxAssignedLen, maxTitleLen, maxTagsLen := len(userHeader), len(titleHeader), len(tagsHeader)
	for _, item := range items {
		if len(item.Assigned()) > maxAssignedLen {
			maxAssignedLen = len(item.Assigned())
		}
		if len(item.Title()) > maxTitleLen {
			maxTitleLen = len(item.Title())
		}
		tags := strings.Join(item.Tags(), " ")
		if len(tags) > maxTagsLen {
			maxTagsLen = len(tags)
		}
	}
	if maxAssignedLen < len(totalPointsCell) {
		maxAssignedLen = len(totalPointsCell)
	}

	result := make([]string, 0, 50)
	if title != "" {
		result = append(result, title)
	}
	headers := make([]string, 0, 3)
	headers = append(headers, fmt.Sprintf("-%s---%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen)))
	headers = append(headers, fmt.Sprintf(" %s | %s | %s | %s ", utils.PadStringRight(userHeader, maxAssignedLen), utils.PadStringRight(titleHeader, maxTitleLen), pointsHeader, tagsHeader))
	headers = append(headers, fmt.Sprintf("-%s---%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen)))
	if withOrderNumber {
		headers[0] = "------" + headers[0]
		headers[1] = "   # |" + headers[1]
		headers[2] = "------" + headers[2]
	}
	result = append(result, headers...)
	totalPoints := 0.0
	for i, item := range items {
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		totalPoints += estimate

		estimateStr := utils.PadIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = strings.Repeat(" ", len(pointsHeader))
		}
		tags := strings.Join(item.Tags(), " ")
		line := fmt.Sprintf(" %s | %s | %s | %s ", utils.PadStringRight(item.Assigned(), maxAssignedLen), utils.PadStringRight(item.Title(), maxTitleLen), estimateStr, utils.PadStringRight(tags, maxTagsLen))
		if withOrderNumber {
			line = fmt.Sprintf(" %s |", utils.PadIntLeft(i+1, 3)) + line
		}
		result = append(result, line)
	}
	if len(items) > 0 {
		if bv.needTotalPoints(status) {
			splitter := fmt.Sprintf("-%s---%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen))
			if withOrderNumber {
				splitter = "------" + splitter
			}
			result = append(result, splitter)

			totalPointsStr := utils.PadIntLeft(int(totalPoints), len(pointsHeader))
			if totalPoints == 0 {
				totalPointsStr = strings.Repeat(" ", len(pointsHeader))
			}
			line := fmt.Sprintf(" %s | %s | %s | %s ", utils.PadStringRight(totalPointsCell, maxAssignedLen), utils.PadStringRight("", maxTitleLen), totalPointsStr, utils.PadStringRight("", maxTagsLen))
			if withOrderNumber {
				line = " |" + line
			}
			result = append(result, line)
		}

		footer := fmt.Sprintf("-%s---%s---%s---%s-", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxTagsLen))
		if withOrderNumber {
			footer = "------" + footer
		}
		result = append(result, footer)
	}
	return result
}

func (bv BacklogView) WriteAsciiItemsWithProjectAndStatus(items []*BacklogItem, overviews map[*BacklogItem]*BacklogOverview, title string, withOrderNumber bool) []string {
	titleHeader, projectHeader, statusHeader, pointsHeader, startDateHeader, endDateHeader := "Title", "Project", "Status", "Points", "Start Date", "End Date"
	maxTitleLen, maxProjectLen, maxStatusLen, maxStartDateLen, maxEndDateLen := len(titleHeader), len(projectHeader), len(statusHeader), len(startDateHeader), len(endDateHeader)
	for _, item := range items {
		overview := overviews[item]
		if len(item.Title()) > maxTitleLen {
			maxTitleLen = len(item.Title())
		}
		if len(overview.Title()) > maxProjectLen {
			maxProjectLen = len(overview.Title())
		}
		if len(item.Status()) > maxStatusLen {
			maxStatusLen = len(item.Status())
		}
		startDate, endDate := item.TimelineStr()
		if len(startDate) > maxStartDateLen {
			maxStartDateLen = len(startDate)
		}
		if len(endDate) > maxEndDateLen {
			maxEndDateLen = len(endDate)
		}
	}

	result := make([]string, 0, 50)
	if title != "" {
		result = append(result, title)
	}
	headers := make([]string, 0, 3)
	headers = append(headers, fmt.Sprintf("-%s---%s---%s---%s---%s---%s-", strings.Repeat("-", maxTitleLen), strings.Repeat("-", maxProjectLen), strings.Repeat("-", maxStatusLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxStartDateLen), strings.Repeat("-", maxEndDateLen)))
	headers = append(headers, fmt.Sprintf(" %s | %s | %s | %s | %s | %s ", utils.PadStringRight(titleHeader, maxTitleLen), utils.PadStringRight(projectHeader, maxProjectLen), utils.PadStringRight(statusHeader, maxStatusLen), pointsHeader, startDateHeader, endDateHeader))
	headers = append(headers, fmt.Sprintf("-%s---%s---%s---%s---%s---%s-", strings.Repeat("-", maxTitleLen), strings.Repeat("-", maxProjectLen), strings.Repeat("-", maxStatusLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxStartDateLen), strings.Repeat("-", maxEndDateLen)))
	if withOrderNumber {
		headers[0] = "------" + headers[0]
		headers[1] = "   # |" + headers[1]
		headers[2] = "------" + headers[2]
	}
	result = append(result, headers...)
	for i, item := range items {
		overview := overviews[item]
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		estimateStr := utils.PadIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = strings.Repeat(" ", len(pointsHeader))
		}
		startDate, endDate := item.TimelineStr()
		line := fmt.Sprintf(" %s | %s | %s | %s | %s | %s ", utils.PadStringRight(item.Title(), maxTitleLen), utils.PadStringRight(overview.Title(), maxProjectLen), utils.PadStringRight(item.Status(), maxStatusLen), estimateStr, utils.PadStringRight(startDate, maxStartDateLen), utils.PadStringRight(endDate, maxEndDateLen))
		if withOrderNumber {
			line = fmt.Sprintf(" %s |", utils.PadIntLeft(i+1, 3)) + line
		}
		result = append(result, line)
	}
	if len(items) > 0 {
		footer := fmt.Sprintf("-%s---%s---%s---%s---%s---%s-", strings.Repeat("-", maxTitleLen), strings.Repeat("-", maxProjectLen), strings.Repeat("-", maxStatusLen), strings.Repeat("-", len(pointsHeader)), strings.Repeat("-", maxStartDateLen), strings.Repeat("-", maxEndDateLen))
		if withOrderNumber {
			footer = "------" + footer
		}
		result = append(result, footer)
	}
	return result
}

func (bv BacklogView) WriteMarkdownItems(items []*BacklogItem, status *BacklogItemStatus, baseDir, tagsDir string, userList *UserList) []string {
	result := make([]string, 0, 50)
	headers := make([]string, 0, 2)
	headers = append(headers, fmt.Sprintf("| User | Title | Points | Tags |"))
	headers = append(headers, "|---|---|:---:|---|")
	result = append(result, headers...)
	totalPoints := 0.0
	for _, item := range items {
		points, _ := strconv.ParseFloat(item.Estimate(), 64)
		totalPoints += points
		assigned := item.Assigned()
		assignedUser := userList.User(assigned)
		assignedLink := assigned
		if assignedUser != nil {
			assignedLink = MakeUserLink(assignedUser, assigned, baseDir)
		}
		line := fmt.Sprintf("| %s | %s | %s | %s |", assignedLink, MakeItemLink(item, baseDir), item.Estimate(), MakeTagLinks(item.Tags(), tagsDir, baseDir))
		result = append(result, line)
	}
	if bv.needTotalPoints(status) && len(items) > 0 {
		line := fmt.Sprintf("| Total Points | | %d | |", int(totalPoints))
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) WriteMarkdownItemsWithoutAssigned(items []*BacklogItem, status *BacklogItemStatus, baseDir, tagsDir string) []string {
	result := make([]string, 0, 50)
	headers := make([]string, 0, 2)
	headers = append(headers, fmt.Sprintf("| Title | Points | Tags |"))
	headers = append(headers, "|---|:---:|---|")
	result = append(result, headers...)
	totalPoints := 0.0
	for _, item := range items {
		points, _ := strconv.ParseFloat(item.Estimate(), 64)
		totalPoints += points
		line := fmt.Sprintf("| %s | %s | %s |", MakeItemLink(item, baseDir), item.Estimate(), MakeTagLinks(item.Tags(), tagsDir, baseDir))
		result = append(result, line)
	}
	if bv.needTotalPoints(status) && len(items) > 0 {
		line := fmt.Sprintf("| Total Points | %d | |", int(totalPoints))
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) WriteMarkdownItemsWithProject(overviews map[*BacklogItem]*BacklogOverview, items []*BacklogItem, status *BacklogItemStatus, baseDir, tagsDir string, userList *UserList) []string {
	result := make([]string, 0, 50)
	headers := make([]string, 0, 2)
	headers = append(headers, fmt.Sprintf("| User | Project | Title | Points | Tags |"))
	headers = append(headers, "|---|---|---|:---:|---|")
	result = append(result, headers...)
	totalPoints := 0.0
	for _, item := range items {
		points, _ := strconv.ParseFloat(item.Estimate(), 64)
		totalPoints += points
		assigned := item.Assigned()
		assignedUser := userList.User(assigned)
		assignedLink := assigned
		if assignedUser != nil {
			assignedLink = MakeUserLink(assignedUser, assigned, baseDir)
		}
		line := fmt.Sprintf("| %s | %s | %s | %s | %s |", assignedLink, MakeOverviewLink(overviews[item], baseDir), MakeItemLink(item, baseDir), item.Estimate(), MakeTagLinks(item.Tags(), tagsDir, baseDir))
		result = append(result, line)
	}
	if bv.needTotalPoints(status) && len(items) > 0 {
		line := fmt.Sprintf("| Total Points | | | %d | |", int(totalPoints))
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) WriteMarkdownItemsWithProjectAndStatus(overviews map[*BacklogItem]*BacklogOverview, items []*BacklogItem, baseDir, tagsDir string, userList *UserList) []string {
	result := make([]string, 0, 50)
	headers := make([]string, 0, 2)
	headers = append(headers, fmt.Sprintf("| User | Project | Title | Status | Points | Tags |"))
	headers = append(headers, "|---|---|---|---|:---:|---|")
	result = append(result, headers...)
	totalPoints := 0.0
	for _, item := range items {
		itemStatus := StatusByName(item.Status())
		if bv.needTotalPoints(itemStatus) {
			points, _ := strconv.ParseFloat(item.Estimate(), 64)
			totalPoints += points
		}
		assigned := item.Assigned()
		assignedUser := userList.User(assigned)
		assignedLink := assigned
		if assignedUser != nil {
			assignedLink = MakeUserLink(assignedUser, assigned, baseDir)
		}
		line := fmt.Sprintf("| %s | %s | %s | %s | %s | %s |", assignedLink, MakeOverviewLink(overviews[item], baseDir), MakeItemLink(item, baseDir), item.Status(), item.Estimate(), MakeTagLinks(item.Tags(), tagsDir, baseDir))
		result = append(result, line)
	}
	if len(items) > 0 {
		line := fmt.Sprintf("| Total Points (doing/planned) | | | | %d | |", int(totalPoints))
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) VelocityText(bck *Backlog, weekCount, width int) (string, error) {
	items := bck.AllItemsByStatus(FinishedStatus.Code)
	currentDate := time.Now().UTC()
	pointsByWeekDelta := make(map[int]float64)
	for _, item := range items {
		finished := item.Finished()
		if finished.IsZero() {
			finished = item.Modified()
		}
		weekDelta := utils.WeekDelta(currentDate, finished)
		if -weekCount < weekDelta && weekDelta <= 0 {
			itemPoints, _ := strconv.ParseFloat(item.Estimate(), 64)
			pointsByWeekDelta[weekDelta] += itemPoints
		}
	}

	graph := goterm.NewLineChart(width, 20)

	data := new(goterm.DataTable)
	data.AddColumn("Week")
	data.AddColumn("Points")

	for i := -weekCount + 1; i <= 0; i++ {
		data.AddRow(float64(i), pointsByWeekDelta[i])
	}

	return graph.Draw(data), nil
}

func (bv BacklogView) VelocityImage(bck *Backlog, weekCount int) ([]byte, error) {
	items := bck.AllItemsByStatus(FinishedStatus.Code)
	currentDate := time.Now().UTC()
	pointsByWeekDelta := make(map[int]float64)
	maxPoints := 0.0
	for _, item := range items {
		finished := item.Finished()
		if finished.IsZero() {
			finished = item.Modified()
		}
		weekDelta := utils.WeekDelta(currentDate, finished)
		if -weekCount < weekDelta && weekDelta <= 0 {
			itemPoints, _ := strconv.ParseFloat(item.Estimate(), 64)
			pointsByWeekDelta[weekDelta] += itemPoints
			if pointsByWeekDelta[weekDelta] > maxPoints {
				maxPoints = pointsByWeekDelta[weekDelta]
			}
		}
	}

	maxIntPoints := int(maxPoints + 0.5)
	tickSize := 5
	yTicksCount := maxIntPoints/tickSize + 1
	yTicks := make([]chart.Tick, yTicksCount+1)
	for i := 0; i < yTicksCount+1; i++ {
		label := ""
		if i%2 == 0 {
			label = strconv.Itoa(i * tickSize)
		}
		yTicks[i] = chart.Tick{Label: label, Value: float64(i * tickSize)}
	}

	xValues := make([]float64, 0, weekCount)
	yValues := make([]float64, 0, weekCount)
	xTicks := make([]chart.Tick, 0, weekCount)
	for i := -weekCount + 1; i <= 0; i++ {
		xValues = append(xValues, float64(i))
		yValues = append(yValues, pointsByWeekDelta[i])
		xTicks = append(xTicks, chart.Tick{Label: utils.WeekEnd(currentDate.AddDate(0, 0, 7*i)).Format("January 2"), Value: float64(i)})
	}

	graph := chart.Chart{
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xValues,
				YValues: yValues,
			},
		},
		XAxis: chart.XAxis{
			Style:     chart.Style{Show: true},
			Ticks:     xTicks,
			Name:      "Week",
			NameStyle: chart.Style{Show: true},
		},
		YAxis: chart.YAxis{
			Style:     chart.Style{Show: true},
			Ticks:     yTicks,
			Name:      "Points",
			NameStyle: chart.Style{Show: true},
		},
	}

	var buffer bytes.Buffer
	err := graph.Render(chart.PNG, &buffer)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (bv BacklogView) WriteMarkdownIdeas(ideas []*BacklogIdea, baseDir, tagsDir string) []string {
	result := make([]string, 0, 50)
	result = append(result, fmt.Sprintf("| Author | Idea | Tags |"))
	result = append(result, "|---|---|---|")
	for _, idea := range ideas {
		line := fmt.Sprintf("| %s | %s | %s |", idea.Author(), MakeIdeaLink(idea, baseDir), MakeTagLinks(idea.Tags(), tagsDir, baseDir))
		result = append(result, line)
	}
	return result
}

func (bv BacklogView) needTotalPoints(status *BacklogItemStatus) bool {
	return status == DoingStatus || status == PlannedStatus
}

func (bv BacklogView) ShowBacklogItems(backlogDir, statusCode string) ([]*BacklogItem, error) {
	if !IsValidStatusCode(statusCode) {
		fmt.Printf("illegal status: %s\n", statusCode)
		return nil, nil
	}
	bck, err := LoadBacklog(backlogDir)
	if err != nil {
		return nil, err
	}

	overviewPath, ok := FindOverviewFileInRootDirectory(backlogDir)
	if !ok {
		return nil, fmt.Errorf("the overview file isn't found for %s", backlogDir)
	}
	overview, err := LoadBacklogOverview(overviewPath)
	if err != nil {
		return nil, err
	}

	archivePath, _ := FindArchiveFileInDirectory(backlogDir)
	archive, err := LoadBacklogOverview(archivePath)
	if err != nil {
		return nil, err
	}

	filter := NewBacklogItemsStatusCodeFilter(statusCode)
	items := bck.FilteredActiveItems(filter)
	status := StatusByCode(statusCode)
	if len(items) == 0 {
		fmt.Printf("No items with status '%s'\n", status.Name)
		return nil, nil
	}

	sorter := NewBacklogItemsSorter(overview, archive)
	sorter.SortItemsByStatus(status, items)
	lines := bv.WriteAsciiItems(items, status, true, false)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println("")
	return items, nil
}
