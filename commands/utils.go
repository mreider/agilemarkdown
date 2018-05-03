package commands

import (
	"errors"
	"github.com/mreider/agilemarkdown/backlog"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func checkIsBacklogDirectory() error {
	_, ok := findOverviewFileInDirectory(".")
	if !ok {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

func findOverviewFileInDirectory(dir string) (string, bool) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, info := range infos {
		if !info.IsDir() && strings.HasPrefix(info.Name(), backlog.OverviewFileNamePrefix) {
			return filepath.Join(dir, info.Name()), true
		}
	}
	return "", false
}

func existsOverviewFileName(rootDir string, overviewFileName string) bool {
	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return false
	}
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		overviewPath := filepath.Join(rootDir, info.Name(), overviewFileName)
		if _, err := os.Stat(overviewPath); err == nil {
			return true
		}
	}
	return false
}

func checkIsRootDirectory() error {
	gitFolder := filepath.Join(".", ".git")
	_, err := os.Stat(gitFolder)
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
