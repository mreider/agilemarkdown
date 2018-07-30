package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var CreateUserCommand = cli.Command{
	Name:  "create-user",
	Usage: "Create a new user",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name: "name",
		},
		cli.StringFlag{
			Name: "email",
		},
	},
	Action: func(c *cli.Context) error {
		name := c.String("name")
		email := c.String("email")
		args := c.Args()

		if len(args) == 0 && name == "" {
			return nil
		}

		if email == "" && len(args) > 0 && strings.Contains(args[len(args)-1], "@") {
			email = args[len(args)-1]
			args = args[:len(args)-1]
		}

		if name == "" && len(args) > 0 {
			name = strings.Join(args, " ")
		}

		if name == "" || email == "" {
			fmt.Println("Both name and email should be specified")
			return nil
		}

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}
		userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))

		existingUser := userList.User(email)
		if existingUser != nil {
			if existingUser.Name() != name {
				fmt.Printf("User '%s' with email %s already exists\n", existingUser.Name(), email)
			} else {
				fmt.Printf("User '%s' already exists\n", name)
			}
			return nil
		}

		existingUser = userList.User(name)
		if existingUser != nil {
			fmt.Printf("User '%s' already exists\n", name)
			return nil
		}

		if userList.AddUser(name, email) {
			return userList.Save()
		} else {
			fmt.Printf("Can't add the user '%s'\n", name)
			return nil
		}
		return nil
	},
}
