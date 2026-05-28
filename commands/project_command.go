package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mreider/agilemarkdown/config"
	"github.com/urfave/cli/v3"
)

// ProjectCommand prints project-level metadata as JSON: name (from
// `.am/config.yaml`'s optional `project.name`) and root (the project's
// absolute filesystem path).
//
// The VS Code extension reads this on snapshot to label the topbar.
// When `project.name` is blank, the field is the empty string and the
// extension falls back to the workspace folder basename.
var ProjectCommand = &cli.Command{
	Name:  "project",
	Usage: "Print project metadata as JSON (name from .am/config.yaml, root path)",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		out := map[string]string{"root": root, "name": ""}
		cfgPath := filepath.Join(root, ".am", "config.yaml")
		if _, statErr := os.Stat(cfgPath); statErr == nil {
			if cfg, err := config.LoadConfig(cfgPath); err == nil {
				out["name"] = cfg.Project.Name
			}
		}
		b, err := json.Marshal(out)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	},
}
