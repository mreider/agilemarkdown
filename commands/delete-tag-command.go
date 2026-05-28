package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
	"strings"
)

var DeleteTagCommand = &cli.Command{
	Name:      "delete-tag",
	Usage:     "Delete a tag",
	ArgsUsage: "TAG",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			fmt.Println("a tag should be specified")
			return nil
		}

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		tag := strings.ToLower(c.Args().Get(0))
		action := actions.NewDeleteTagAction(rootDir, tag)
		return action.Execute()
	},
}
