package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// SetReleaseDateCommand stamps `release_date:` on a release-type
// story. The date drives the marker's position in the priority view
// and, once today is past, surfaces a Ship affordance on the bar.
// Accepts ISO YYYY-MM-DD; passing an empty string clears the field.
var SetReleaseDateCommand = &cli.Command{
	Name:      "set-release-date",
	Usage:     "Set the release_date on a release-type item (YYYY-MM-DD; empty clears)",
	ArgsUsage: "ITEM_PATH DATE",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 2 {
			return fmt.Errorf("usage: am set-release-date ITEM_PATH DATE")
		}
		path, err := requireItemFile(c.Args().Get(0))
		if err != nil {
			return err
		}
		date := strings.TrimSpace(c.Args().Get(1))
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		item.SetReleaseDate(date)
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s release_date=%s\n", filepath.Base(path), date)
		return nil
	},
}
