package commands

import (
	"errors"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"path/filepath"
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
