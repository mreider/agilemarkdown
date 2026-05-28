package commands

import (
	"context"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
)

var CreateUserCommand = &cli.Command{
	Name:      "create-user",
	Usage:     "Create a new user",
	ArgsUsage: "NAME EMAIL",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:   "name",
			Hidden: true,
		},
		&cli.StringFlag{
			Name:   "email",
			Hidden: true,
		},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		name := c.String("name")
		email := c.String("email")
		parts := c.Args().Slice()

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}
		action := actions.NewCreateUserAction(rootDir, name, email, parts)
		return action.Execute()
	},
}
