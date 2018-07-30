package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"strings"
)

var TimelineCommand = cli.Command{
	Name:      "timeline",
	Usage:     "Build a new timeline",
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
		action := actions.NewTimelineAction(rootDir, tag)
		return action.Execute()
	},
}
