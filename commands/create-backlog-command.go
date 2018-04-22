package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

var CreateBacklogCommand = cli.Command{
	Name:      "create-backlog",
	Usage:     "Create a new backlog",
	ArgsUsage: "BACKLOG_NAME",
	Action: func(c *cli.Context) error {
		if err := checkIsRootDirectory(); err != nil {
			return err
		}

		if c.NArg() != 1 {
			fmt.Println("A backlog name should be specified")
			return nil
		}

		backlogDir := filepath.Join(".", c.Args()[0])
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

		overviewPath := filepath.Join(backlogDir, backlog.OverviewFileName)
		overview, err := backlog.CreateBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		overview.SetTitle("")
		overview.SetCreated()
		return overview.Save()
	},
}