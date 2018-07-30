package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var DeleteUserCommand = cli.Command{
	Name:  "delete-user",
	Usage: "Delete the user",
	Action: func(c *cli.Context) error {
		if len(c.Args()) == 0 {
			fmt.Println("User name, email or prefix should be specified")
			return nil
		}
		nameOrEmail := strings.Join(c.Args(), " ")

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		action := actions.NewDeleteUserAction(rootDir, nameOrEmail)
		return action.Execute()
	},
}
