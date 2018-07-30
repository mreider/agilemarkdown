package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var ChangeTagCommand = cli.Command{
	Name:      "change-tag",
	Usage:     "Change a tag",
	ArgsUsage: "OLD_TAG NEW_TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			fmt.Println("old and new tags should be specified")
			return nil
		}

		rootDir, _ := filepath.Abs(".")
		if err := checkIsBacklogDirectory(); err == nil {
			rootDir = filepath.Dir(rootDir)
		} else if err := checkIsRootDirectory("."); err != nil {
			return err
		}

		oldTag := strings.ToLower(c.Args()[0])
		newTag := strings.ToLower(c.Args()[1])

		action := actions.NewChangeTagAction(rootDir, oldTag, newTag)
		return action.Execute()
	},
}
