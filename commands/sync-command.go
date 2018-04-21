package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var SyncCommand = cli.Command{
	Name:      "sync",
	Usage:     "Sync state",
	ArgsUsage: " ",
	Action: func(c *cli.Context) error {
		err := git.AddAll()
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
