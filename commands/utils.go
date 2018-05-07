package commands

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"path/filepath"
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
	overviewFileName := fmt.Sprintf("%s.md", filepath.Base(dir))

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
		return nil, fmt.Errorf("the index file isn't found for %s", backlogDir)
	}
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return nil, err
	}

	items := bck.ItemsByStatus(statusCode)
	status := backlog.StatusByCode(statusCode)
	if len(items) == 0 {
		fmt.Printf("No items with status '%s'\n", status.Name)
		return nil, nil
	}

	overview.SortItems(status, items)
	lines := backlog.BacklogView{}.WriteAsciiTable(items, fmt.Sprintf("Status: %s", status.Name), true)
	for _, line := range lines {
		fmt.Println(line)
	}
	fmt.Println("")
	return items, nil
}
