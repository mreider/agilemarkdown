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
		} else {
			dir = newDir
		}
	}
	if dir == "" {
		return "", fmt.Errorf("can't find root directory from '%s'", dir)
	}
	return dir, nil
}

func AddConfigAndGitIgnore(root *backlog.BacklogsStructure) error {
	hasChanges := false

	configPath := root.ConfigFile()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err := ioutil.WriteFile(configPath, []byte(strings.TrimLeftFunc(defaultConfig, unicode.IsSpace)), 0644)
		if err != nil {
			return err
		}
		err = git.Add(configPath)
		if err != nil {
			return err
		}
		hasChanges = true
	}
	gitIgnorePath := filepath.Join(root.Root(), ".gitignore")
	if _, err := os.Stat(gitIgnorePath); os.IsNotExist(err) {
		err := ioutil.WriteFile(gitIgnorePath, []byte(filepath.Base(configPath)), 0644)
		if err != nil {
			return err
		}
		err = git.Add(gitIgnorePath)
		if err != nil {
			return err
		}
		hasChanges = true
	}

	if hasChanges {
		err := git.Commit("configuration", "")
		if err != nil {
			return err
		}
	}
	return nil
}
