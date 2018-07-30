package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
)

var ChangeUserCommand = cli.Command{
	Name:  "change-user",
	Usage: "Change the user",
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 2 {
			fmt.Println("Name, email or prefix of both users should be specified")
			return nil
		}
		if len(c.Args()) > 2 {
			fmt.Println("Only two (2) arguments are allowed")
			return nil
		}
		fromNameOrEmail, toNameOrEmail := c.Args()[0], c.Args()[1]

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		action := actions.NewChangeUserAction(rootDir, fromNameOrEmail, toNameOrEmail)
		return action.Execute()
	},
}
