package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const ArchiveFileName = "archive.md"

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
	_, ok := findOverviewFileInRootDirectory(".")
	if !ok {
		return errors.New("Error, please change directory to a backlog folder")
	}
	return nil
}

func findOverviewFileInRootDirectory(dir string) (string, bool) {
	dir, _ = filepath.Abs(dir)
	rootDir := filepath.Dir(dir)
	overviewName := filepath.Base(dir)
	if backlog.IsForbiddenBacklogName(overviewName) {
		return "", false
	}
	overviewFileName := fmt.Sprintf("%s.md", overviewName)

	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return "", false
	}
	for _, info := range infos {
		if info.Name() == overviewFileName {
			return filepath.Join(rootDir, info.Name()), true
		}
	}
	return "", false
}

func findArchiveFileInDirectory(dir string) (string, bool) {
	dir, _ = filepath.Abs(dir)
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, info := range infos {
		if info.Name() == ArchiveFileName {
			return filepath.Join(dir, info.Name()), true
		}
	}
	return filepath.Join(dir, ArchiveFileName), false
}

func checkIsRootDirectory(dir string) error {
	if !git.IsRootGitDirectory(dir) {
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

func showBacklogItems(c *cli.Context) ([]*backlog.BacklogItem, error) {
	statusCode := c.String("s")

	if statusCode == "" {
		fmt.Println("-s option is required")
		return nil, nil
	}
	if !backlog.IsValidStatusCode(statusCode) {
		fmt.Printf("illegal status: %s\n", statusCode)
		return nil, nil
	}
	if err := checkIsBacklogDirectory(); err != nil {
		fmt.Println(err)
		return nil, nil
	}
	backlogDir, _ := filepath.Abs(".")
	bck, err := backlog.LoadBacklog(backlogDir)
	if err != nil {
		return nil, err
	}

	overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
	if !ok {
		return nil, fmt.Errorf("the overview file isn't found for %s", backlogDir)
	}
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return nil, err
	}

	archivePath, _ := findArchiveFileInDirectory(backlogDir)
	archive, err := backlog.LoadBacklogOverview(archivePath)
	if err != nil {
		return nil, err
	}

	filter := backlog.NewBacklogItemsStatusCodeFilter(statusCode)
	items := bck.FilteredActiveItems(filter)
	status := backlog.StatusByCode(statusCode)
	if len(items) == 0 {
		fmt.Printf("No items with status '%s'\n", status.Name)
		return nil, nil
	}

	sorter := backlog.NewBacklogItemsSorter(overview, archive)
	sorter.SortItemsByStatus(status, items)
	lines := backlog.BacklogView{}.WriteAsciiItems(items, fmt.Sprintf("Status: %s", status.Name), true)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println("")
	return items, nil
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
