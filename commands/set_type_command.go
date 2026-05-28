package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// validTypes are the four story types agilemarkdown recognizes.
// `release` is allowed but skips the state machine; users typically
// won't toggle to/from release from the card, but accept it here so
// the CLI matches the schema.
var validTypes = map[string]bool{
	"feature": true,
	"bug":     true,
	"chore":   true,
	"release": true,
}

// SetTypeCommand changes the `type:` frontmatter on a story file.
// Used by the VS Code extension's TYPE chooser (detail panel) so the
// human can flip a story between feature/bug/chore without editing
// YAML by hand. The coach rule "bugs and chores are not pointed" is
// enforced at the UI layer (the extension strips the estimate when
// switching to those types); keeping that out of the CLI keeps this
// verb composable.
var SetTypeCommand = &cli.Command{
	Name:      "set-type",
	Usage:     "Set the type frontmatter of an item (feature, bug, chore, release)",
	ArgsUsage: "ITEM_PATH TYPE",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 2 {
			return fmt.Errorf("usage: am set-type ITEM_PATH TYPE")
		}
		path, err := requireItemFile(c.Args().Get(0))
		if err != nil {
			return err
		}
		t := strings.ToLower(strings.TrimSpace(c.Args().Get(1)))
		if !validTypes[t] {
			return fmt.Errorf("invalid type %q; valid: feature, bug, chore, release", t)
		}
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		item.SetType(t)
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s type=%s\n", filepath.Base(path), t)
		return nil
	},
}
