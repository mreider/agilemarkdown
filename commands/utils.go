package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
	"strings"
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
		fmt.Printf("%s | %s | %s\n", utils.PadIntLeft(i+1, 3), utils.PadStringRight(item.Title(), maxTitleLen), item.Assigned())
	}
}
