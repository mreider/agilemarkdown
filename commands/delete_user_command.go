package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
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
		userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))

		user := userList.User(nameOrEmail)
		if user == nil {
			fmt.Printf("User '%s' not found\n", nameOrEmail)
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
			if user == assigned {
				item.SetAssigned("")
				err := item.Save()
				if err != nil {
					return err
				}
			}
		}

		if !userList.DeleteUser(nameOrEmail) {
			fmt.Printf("Can't delete the user '%s'\n", nameOrEmail)
			return nil
		}
		return nil
	},
}
