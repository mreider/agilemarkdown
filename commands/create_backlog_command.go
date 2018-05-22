package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"strings"
)

var CreateBacklogCommand = cli.Command{
	Name:      "create-backlog",
	Usage:     "Create a new backlog",
	ArgsUsage: "BACKLOG_NAME",
	Action: func(c *cli.Context) error {
		if err := checkIsRootDirectory(); err != nil {
			return err
		}

		if c.NArg() == 0 {
			fmt.Println("A backlog name should be specified")
			return nil
		}

		backlogName := strings.Join(c.Args(), " ")
		if backlog.IsForbiddenBacklogName(backlogName) {
			fmt.Printf("'%s' can't be used as a backlog name\n", backlogName)
			return nil
		}

		backlogFileName := strings.Replace(backlogName, " ", "-", -1)
		backlogDir := filepath.Join(".", backlogFileName)
		if info, err := os.Stat(backlogDir); err != nil && !os.IsNotExist(err) {
			return err
		} else if err == nil {
			if info.IsDir() {
				fmt.Println("the backlog directory already exists")
			} else {
				fmt.Println("a file with the same name already exists")
			}
			return nil
		}

		git.SetUpstream()

		err := os.MkdirAll(backlogDir, 0777)
		if err != nil {
			return err
		}

		err = os.MkdirAll(filepath.Join(filepath.Join(".", backlog.IdeasDirectoryName)), 0777)
		if err != nil {
			return err
		}

		overviewFileName := fmt.Sprintf("%s.md", backlogFileName)
		overviewPath := filepath.Join(".", overviewFileName)
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		overview.SetTitle(backlogName)
		overview.UpdateLinks("archive", filepath.Join(backlogDir, ArchiveFileName), ".", ".")
		overview.SetCreated()
		return overview.Save()
	},
}
