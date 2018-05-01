package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"path/filepath"
	"strings"
)

var SyncCommand = cli.Command{
	Name:      "sync",
	Usage:     "Sync state",
	ArgsUsage: " ",
	Action: func(c *cli.Context) error {
		rootDirectory, _ := filepath.Abs(".")
		if err := checkIsBacklogDirectory(); err == nil {
			rootDirectory = filepath.Dir(rootDirectory)
		} else if err := checkIsRootDirectory(); err != nil {
			return err
		}

		infos, err := ioutil.ReadDir(rootDirectory)
		if err != nil {
			return err
		}
		for _, info := range infos {
			if !info.IsDir() || strings.HasPrefix(info.Name(), ".") {
				continue
			}
			backlogDir := filepath.Join(rootDirectory, info.Name())
			overview, err := backlog.LoadBacklogOverview(filepath.Join(backlogDir, backlog.OverviewFileName))
			if err != nil {
				return err
			}
			bck, err := backlog.LoadBacklog(backlogDir)
			if err != nil {
				return err
			}

			items := bck.Items()
			overview.Update(items)
		}
		return nil
		err = git.AddAll()
		if err != nil {
			return err
		}
		git.Commit("sync") // TODO commit message
		err = git.Fetch()
		if err != nil {
			return fmt.Errorf("can't fetch: %v", err)
		}
		output, err := git.Merge()
		if err != nil {
			status, _ := git.Status()
			if !strings.Contains(status, "Your branch is based on 'origin/master', but the upstream is gone.") {
				fmt.Println(output)
				git.AbortMerge()
				return fmt.Errorf("can't merge: %v", err)
			}
		}
		err = git.Push()
		if err != nil {
			return fmt.Errorf("can't push: %v", err)
		}
		return nil
	},
}
