package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/urfave/cli/v3"
	"strings"
)

var CreateBacklogCommand = &cli.Command{
	Name:      "create-backlog",
	Usage:     "Create a new backlog",
	ArgsUsage: "BACKLOG_NAME",
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsRootDirectory("."); err != nil {
			out, statusErr := git.Status()
			if statusErr != nil && strings.Contains(out, "fatal: not a git repository") {
				err := git.Init()
				if err != nil {
					return err
				}
				err = AddConfigAndGitIgnore(backlog.NewBacklogsStructure("."))
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}

		if c.NArg() == 0 {
			fmt.Println("A backlog name should be specified")
			return nil
		}

		backlogName := strings.Join(c.Args().Slice(), " ")

		action := actions.NewCreateBacklogAction(".", backlogName)
		return action.Execute()
	},
}
