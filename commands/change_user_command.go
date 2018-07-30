package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
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
		userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))

		fromUser := userList.User(fromNameOrEmail)
		if fromUser == nil {
			fmt.Printf("User '%s' not found\n", fromNameOrEmail)
			return nil
		}

		toUser := userList.User(toNameOrEmail)
		if toUser == nil {
			fmt.Printf("User '%s' not found\n", toNameOrEmail)
			return nil
		}

		if fromUser == toUser {
			fmt.Printf("Users '%s' and '%s' are the same\n", fromNameOrEmail, toNameOrEmail)
			return nil
		}

		if !confirmAction("Are you sure? (y or n)") {
			return nil
		}

		items, _, err := backlog.AllBacklogItems(rootDir)
		if err != nil {
			return err
		}
		for _, item := range items {
			assigned := userList.User(item.Assigned())
			if fromUser == assigned {
				item.SetAssigned(toUser.Nickname())
				err := item.Save()
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}
