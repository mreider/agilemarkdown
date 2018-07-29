package commands

import (
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

var CreateUserCommand = cli.Command{
	Name:   "create-user",
	Usage:  "Create a new user",
	Hidden: true,
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
		if name == "" {
			return nil
		}

		rootDir, _ := filepath.Abs(".")
		for rootDir != "" {
			_, err := os.Stat(filepath.Join(rootDir, ".git"))
			if err == nil {
				break
			}
			rootDir = filepath.Dir(rootDir)
		}
		userList := backlog.NewUserList(filepath.Join(rootDir, backlog.UsersDirectoryName))
		if userList.AddUser(name, email) {
			return userList.Save()
		}
		return nil
	},
}
