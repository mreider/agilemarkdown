package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"gopkg.in/urfave/cli.v1"
)

var ImportCommand = cli.Command{
	Name:      "import",
	Usage:     "Import an existing Pivotal Tracker story",
	ArgsUsage: "CSV_FILE",
	Action: func(c *cli.Context) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		if c.NArg() == 0 {
			fmt.Println("a csv file should be specified")
			return nil
		}

		action := actions.NewImportAction(".", c.Args())
		return action.Execute()
	},
}
