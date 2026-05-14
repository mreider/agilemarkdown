package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/coach"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"github.com/urfave/cli/v3"
)

// InitCommand seeds an existing agilemarkdown repo with the coach-mode
// projections (CLAUDE.md, AGENTS.md, .github/copilot-instructions.md,
// .cursor/rules/coach.mdc, .claude/skills/*, .claude/hooks/coach-gate.sh,
// .claude/settings.json). Idempotent: existing files are not touched.
//
// Use this for repos that ran `am create-backlog` before v4.4 (when the
// projections did not exist) and want to inherit the coach stance.
var InitCommand = &cli.Command{
	Name:  "init",
	Usage: "Install or refresh the coach-mode projections (idempotent)",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		// Ensure .am/config.yaml exists; this matches the side effect
		// `am create-backlog` had so existing repos with no config get
		// caught up.
		structure := backlog.NewBacklogsStructure(root)
		if _, err := os.Stat(structure.ConfigFile()); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(structure.ConfigFile()), 0755); err != nil {
				return err
			}
			if err := config.Defaults().Save(structure.ConfigFile()); err != nil {
				return err
			}
			_ = git.Add(structure.ConfigFile())
			fmt.Printf("wrote .am/config.yaml\n")
		}

		written, err := coach.InstallTemplates(root)
		if err != nil {
			return err
		}
		if len(written) == 0 {
			fmt.Println("coach mode already installed; nothing to do")
			return nil
		}
		for _, p := range written {
			fmt.Printf("wrote %s\n", p)
		}
		return nil
	},
}
