package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var CreateIdeaCommand = cli.Command{
	Name:      "create-idea",
	Usage:     "Create a new idea",
	ArgsUsage: "IDEA_TITLE",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "simulate",
			Hidden: true,
		},
		cli.StringFlag{
			Name:   "user",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
		simulate := c.Bool("simulate")
		user := c.String("user")

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		if c.NArg() == 0 {
			if !simulate {
				fmt.Println("an idea name should be specified")
			}
			return nil
		}

		ideaTitle := strings.Join(c.Args(), " ")
		action := actions.NewCreateIdeaAction(rootDir, ideaTitle, user, simulate)
		return action.Execute()
	},
}
