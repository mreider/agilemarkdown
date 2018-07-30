package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	configName    = ".config.json"
	defaultConfig = `
{
  "SmtpServer": "",
  "SmtpUser": "",
  "SmtpPassword": "",
  "EmailFrom": "",
  "RemoteGitUrlFormat": "%s/blob/master/%s",
  "RemoteWebUrlFormat": ""
}`
)

func checkIsBacklogDirectory() error {
	_, ok := backlog.FindOverviewFileInRootDirectory(".")
	if !ok {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

func checkIsRootDirectory(dir string) error {
	if !git.IsRootGitDirectory(dir) {
		return errors.New("Error, please change directory to a root git folder")
	}
	return nil
}

func findRootDirectory() (string, error) {
	dir, _ := filepath.Abs(".")
	for dir != "" {
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			break
		}
		newDir := filepath.Dir(dir)
		if newDir == dir {
			dir = ""
		}
	}
	if dir == "" {
		return "", fmt.Errorf("can't find root directory from '%s'", dir)
	}
	return dir, nil
}

func AddConfigAndGitIgnore(rootDir string) {
	hasChanges := false

	configPath := filepath.Join(rootDir, configName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		ioutil.WriteFile(configPath, []byte(strings.TrimLeftFunc(defaultConfig, unicode.IsSpace)), 0644)
		git.Add(configPath)
		hasChanges = true
	}
	gitIgnorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		ioutil.WriteFile(gitIgnorePath, []byte(configName), 0644)
		git.Add(gitIgnorePath)
		hasChanges = true
	}

	if hasChanges {
		git.Commit("configuration", "")
	}
}
