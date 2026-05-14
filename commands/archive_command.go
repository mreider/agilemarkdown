package commands

import (
	"context"
	"fmt"
	"github.com/mreider/agilemarkdown/actions"
	"github.com/urfave/cli/v3"
	"path/filepath"
	"time"
)

var ArchiveCommand = &cli.Command{
	Name:      "archive",
	Usage:     "Archive stories before a certain date",
	ArgsUsage: "YYYY-MM-DD",
	Action: func(ctx context.Context, c *cli.Command) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}

		if c.NArg() != 1 {
			fmt.Println("A date should be specified")
			return nil
		}

		beforeDate, err := time.Parse("2006-1-2", c.Args().Get(0))
		if err != nil {
			fmt.Println("Invalid date. Should be in YYYY-MM-DD format.")
			return nil
		}

		backlogDir, _ := filepath.Abs(".")
		action := actions.NewArchiveAction(backlogDir, beforeDate)
		return action.Execute()
	},
}
