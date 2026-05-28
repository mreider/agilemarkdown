package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// DeleteItemCommand permanently removes a story file from disk and
// drops its entry from `_priority.md` and `_icebox.md` in the same
// backlog. Pivotal-style hard delete. Recoverable from git history;
// not recoverable inside the working tree.
//
// Guards mirror the transition verbs: requireItemFile resolves the
// argument against cwd and refuses missing paths, so an `am delete-item
// Foo.md` run from the wrong dir cannot wipe the wrong file.
var DeleteItemCommand = &cli.Command{
	Name:      "delete-item",
	Usage:     "Permanently delete a story file and remove it from the priority/icebox order",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("path to an item file is required")
		}
		path, err := requireItemFile(c.Args().Get(0))
		if err != nil {
			return err
		}
		// The backlog dir is the parent of the item file. Open the
		// order files there so we can drop the entry alongside the
		// disk delete. Doing both writes here keeps the views coherent
		// without forcing the caller to `am sync` afterwards.
		dir := filepath.Dir(path)
		base := filepath.Base(path)
		if pri, err := backlog.LoadPriority(dir); err == nil {
			if pri.IndexOf(base) >= 0 {
				pri.Remove(base)
				if err := pri.Save(); err != nil {
					return err
				}
			}
		}
		if ice, err := backlog.LoadIcebox(dir); err == nil {
			if ice.IndexOf(base) >= 0 {
				ice.Remove(base)
				if err := ice.Save(); err != nil {
					return err
				}
			}
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		fmt.Printf("%s deleted\n", base)
		return nil
	},
}
