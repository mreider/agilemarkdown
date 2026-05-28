package commands

import (
	"context"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
)

func NewSyncCommand() *cli.Command {
	return &cli.Command{
		Name:      "sync",
		Usage:     "Sync state",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:   "test",
				Hidden: true,
			},
			&cli.StringFlag{
				Name:   "author",
				Hidden: true,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			rootDir, err := findRootDirectory()
			if err != nil {
				return err
			}

			action := actions.NewSyncAction(rootDir, c.String("author"), c.Bool("test"))
			return action.Execute()
		},
	}
}
