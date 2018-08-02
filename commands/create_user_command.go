package commands

import (
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
)

var CreateUserCommand = cli.Command{
	Name:  "create-user",
	Usage: "Create a new user",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "name",
			Hidden: true,
		},
		cli.StringFlag{
			Name: "email",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		email := c.String("email")
		parts := c.Args()

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}
		action := actions.NewCreateUserAction(rootDir, name, email, parts)
		return action.Execute()
	},
}
