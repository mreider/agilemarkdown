package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var DeleteTagCommand = cli.Command{
	Name:      "delete-tag",
	Usage:     "Delete a tag",
	ArgsUsage: "TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			fmt.Println("a tag should be specified")
			return nil
		}

		rootDir, err := findRootDirectory()
		if err != nil {
			return err
		}

		tag := strings.ToLower(c.Args()[0])
		action := actions.NewDeleteTagAction(rootDir, tag)
		return action.Execute()
	},
}
