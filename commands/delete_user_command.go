package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
	"strings"
)

var DeleteUserCommand = &cli.Command{
	Name:      "delete-user",
	Usage:     "Delete the user",
	ArgsUsage: "USER",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.Args().Len() == 0 {
			fmt.Println("User name, email or prefix should be specified")
			return nil
		}
		nameOrEmail := strings.Join(c.Args().Slice(), " ")

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		action := actions.NewDeleteUserAction(rootDir, nameOrEmail)
		return action.Execute()
	},
}
