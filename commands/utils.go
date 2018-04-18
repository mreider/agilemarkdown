package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func checkIsBacklogDirectory(dirPath string) error {
	overviewPath := filepath.Join(dirPath, backlog.OverviewFileName)
	_, err := os.Stat(overviewPath)
	if err != nil {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

func checkIsRootDirectory(dirPath string) error {
	overviewPath := filepath.Join(dirPath, ".git")
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
	fmt.Printf("  # | %s\n", title)
	fmt.Printf("-----%s\n", strings.Repeat("-", len(title)+2))
	for i, item := range items {
		fmt.Printf("%s | %s\n", padIntLeft(i+1, 3), item.Title())
	}
}

func padIntLeft(value, width int) string {
	result := strconv.Itoa(value)
	if len(result) < width {
		result = strings.Repeat(" ", width-len(result)) + result
	}
	return result
}

func padStringRight(value string, width int) string {
	result := value
	if len(result) < width {
		result += strings.Repeat(" ", width-len(result))
	}
	return result
}
