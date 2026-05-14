package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

const (
	agreementsFileName = "team-agreements.md"
	learningsFileName  = "learnings.md"
)

// TeamAgreementsCommand reads team-agreements.md at the project root,
// or with --add appends a new bullet, or with --set overwrites the file.
//
// --add is the retro-flow path: each retro adds one or two new
// agreements. --set replaces the entire file (used to bootstrap or
// reset). Reading is the default when no flag is passed.
var TeamAgreementsCommand = &cli.Command{
	Name:      "team-agreements",
	Usage:     "Read, append to, or overwrite team-agreements.md at the project root",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "add", Usage: "append this line as a new agreement bullet (preserves existing content)"},
		&cli.StringFlag{Name: "set", Usage: "overwrite the file with this content"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		path := filepath.Join(root, agreementsFileName)
		if add := strings.TrimSpace(c.String("add")); add != "" {
			if err := appendAgreement(path, add); err != nil {
				return err
			}
		} else if set := c.String("set"); set != "" {
			if err := os.WriteFile(path, []byte(set), 0644); err != nil {
				return err
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("# Team agreements")
				fmt.Println()
				fmt.Println("No agreements recorded yet. Use `am team-agreements --add \"...\"` or edit", path)
				return nil
			}
			return err
		}
		fmt.Print(string(raw))
		return nil
	},
}

// appendAgreement adds a bullet to the agreements file, creating the
// file with a header when it does not exist. Existing content is
// preserved verbatim; the new bullet lands at the end of the list.
func appendAgreement(path, line string) error {
	var existing string
	if raw, err := os.ReadFile(path); err == nil {
		existing = string(raw)
	} else if !os.IsNotExist(err) {
		return err
	}
	if existing == "" {
		existing = "# Team agreements\n\n"
	}
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	bullet := "- " + strings.TrimPrefix(strings.TrimSpace(line), "- ") + "\n"
	return os.WriteFile(path, []byte(existing+bullet), 0644)
}

// RecordLearningCommand appends a dated one-line entry to learnings.md.
var RecordLearningCommand = &cli.Command{
	Name:      "record-learning",
	Usage:     "Append a dated one-line learning to learnings.md",
	ArgsUsage: "\"note\"",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("usage: am record-learning \"note\"")
		}
		note := strings.TrimSpace(strings.Join(c.Args().Slice(), " "))
		if note == "" {
			return fmt.Errorf("note is required")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		path := filepath.Join(root, learningsFileName)
		entry := fmt.Sprintf("- %s: %s\n", time.Now().UTC().Format("2006-01-02"), note)

		var existing string
		if raw, err := os.ReadFile(path); err == nil {
			existing = string(raw)
		} else if !os.IsNotExist(err) {
			return err
		}
		if existing == "" {
			existing = "# Learnings\n\nA running log of one-line learnings from removal experiments, scratch refactors, retros, and surprises.\n\n"
		}
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		out := existing + entry
		if err := os.WriteFile(path, []byte(out), 0644); err != nil {
			return err
		}
		fmt.Print(entry)
		return nil
	},
}
