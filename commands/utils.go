package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"sort"
)

func checkIsBacklogDirectory() error {
	overviewPath := filepath.Join(".", backlog.OverviewFileName)
	_, err := os.Stat(overviewPath)
	if err != nil {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

func checkIsRootDirectory() error {
	overviewPath := filepath.Join(".", ".git")
	_, err := os.Stat(overviewPath)
	if err != nil {
		return errors.New("Error, please change directory to a root git folder")
	}
	return nil
}

func existsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func printBacklogItems(items []*backlog.BacklogItem, title string) {
	maxUserLen, maxTitleLen := 0, 0
	for _, item := range items {
		if len(item.Assigned()) > maxUserLen {
			maxUserLen = len(item.Assigned())
		}
		if len(item.Title()) > maxTitleLen {
			maxTitleLen = len(item.Title())
		}
	}

	maxLen := len(title)
	if maxUserLen+3+maxTitleLen > maxLen {
		maxLen = maxUserLen + 3 + maxTitleLen
	}

	fmt.Printf("  # | %s\n", title)
	fmt.Printf("------%s-\n", strings.Repeat("-", maxLen))
	for i, item := range items {
		fmt.Printf("%s | %s | %s\n", PadIntLeft(i+1, 3), PadStringRight(item.Title(), maxTitleLen), item.Assigned())
	}
}

func printBacklogItemsForStatus(items []*backlog.BacklogItem, status *backlog.BacklogItemStatus) {
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

	fmt.Printf("Status: %s\n", status.Name)
	fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
	fmt.Printf(" %s | %s | %s\n", PadStringRight(userHeader, maxAssignedLen), PadStringRight(titleHeader, maxTitleLen), pointsHeader)
	fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
	for _, item := range items {
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		estimateStr := PadIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = ""
		}
		fmt.Printf(" %s | %s | %s\n", PadStringRight(item.Assigned(), maxAssignedLen), PadStringRight(item.Title(), maxTitleLen), estimateStr)
	}
}

func PadIntLeft(value, width int) string {
	result := strconv.Itoa(value)
	if len(result) < width {
		result = strings.Repeat(" ", width-len(result)) + result
	}
	return result
}

func PadStringRight(value string, width int) string {
	result := value
	if len(result) < width {
		result += strings.Repeat(" ", width-len(result))
	}
	return result
}

func WeekStart(value time.Time) time.Time {
	weekday := value.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	weekStart := value.Add(time.Duration(-(weekday-1)*24) * time.Hour)
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	return weekStart
}

func WeekDelta(baseValue, value time.Time) int {
	return int(WeekStart(value).Sub(WeekStart(baseValue)).Hours()) / 24 / 7
}
