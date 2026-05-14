package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
)

var ChangeUserCommand = &cli.Command{
	Name:      "change-user",
	Usage:     "Change the user",
	ArgsUsage: "OLD-USER NEW-USER",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.Args().Len() < 2 {
			fmt.Println("Name, email or prefix of both users should be specified")
			return nil
		}
		if c.Args().Len() > 2 {
			fmt.Println("Only two (2) arguments are allowed")
			return nil
		}
		fromNameOrEmail, toNameOrEmail := c.Args().Get(0), c.Args().Get(1)

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		action := actions.NewChangeUserAction(rootDir, fromNameOrEmail, toNameOrEmail)
		return action.Execute()
	},
}
