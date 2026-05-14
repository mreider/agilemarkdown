package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/urfave/cli/v3"
)

// itemAndBacklogDir resolves an item path argument (relative or absolute,
// .md optional) into (backlogDir, itemBaseName.md). Both paths are
// absolute. Errors when the input does not look like an item under a
// known backlog directory.
func itemAndBacklogDir(itemArg string) (string, string, error) {
	if itemArg == "" {
		return "", "", fmt.Errorf("item path required")
	}
	if !strings.HasSuffix(itemArg, ".md") {
		itemArg += ".md"
	}
	abs, err := filepath.Abs(itemArg)
	if err != nil {
		return "", "", err
	}
	backlogDir := filepath.Dir(abs)
	return backlogDir, filepath.Base(abs), nil
}

// resolveAnchor maps an --after/--before flag value to the basename of an
// item file inside the same backlog directory. Accepts a path or a
// title-ish string with .md optional.
func resolveAnchor(backlogDir, anchor string) string {
	if anchor == "" {
		return ""
	}
	if !strings.HasSuffix(anchor, ".md") {
		anchor += ".md"
	}
	return filepath.Base(anchor)
}

// RankCommand reorders an item inside the priority file.
var RankCommand = &cli.Command{
	Name:      "rank",
	Usage:     "Reorder an item inside _priority.md",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "top", Usage: "move to the top of priority"},
		&cli.BoolFlag{Name: "bottom", Usage: "move to the bottom of priority"},
		&cli.StringFlag{Name: "after", Usage: "place immediately after this item"},
		&cli.StringFlag{Name: "before", Usage: "place immediately before this item"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am rank ITEM_PATH [--top|--bottom|--after X|--before X]")
		}
		backlogDir, itemFile, err := itemAndBacklogDir(c.Args().Get(0))
		if err != nil {
			return err
		}
		pri, err := backlog.LoadPriority(backlogDir)
		if err != nil {
			return err
		}
		ice, err := backlog.LoadIcebox(backlogDir)
		if err != nil {
			return err
		}

		// If item is in icebox, pull it into priority first.
		if pri.IndexOf(itemFile) < 0 {
			if i := ice.IndexOf(itemFile); i >= 0 {
				e := ice.Entries()[i]
				ice.Remove(itemFile)
				pri.InsertBottom(e)
				if err := ice.Save(); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("%s not found in priority or icebox; run `am sync` first", itemFile)
			}
		}

		switch {
		case c.Bool("top"):
			pri.MoveTo(itemFile, 0)
		case c.Bool("bottom"):
			pri.MoveTo(itemFile, pri.Len()-1)
		case c.String("after") != "":
			pri.MoveAfter(itemFile, resolveAnchor(backlogDir, c.String("after")))
		case c.String("before") != "":
			pri.MoveBefore(itemFile, resolveAnchor(backlogDir, c.String("before")))
		default:
			return fmt.Errorf("specify one of --top, --bottom, --after, --before")
		}
		if err := pri.Save(); err != nil {
			return err
		}
		fmt.Printf("ranked %s\n", itemFile)
		return nil
	},
}

// IceCommand moves an item from priority to icebox.
var IceCommand = &cli.Command{
	Name:      "ice",
	Usage:     "Move an item from _priority.md to _icebox.md",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "top", Usage: "place at top of icebox (default: bottom)"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am ice ITEM_PATH [--top]")
		}
		backlogDir, itemFile, err := itemAndBacklogDir(c.Args().Get(0))
		if err != nil {
			return err
		}
		pri, err := backlog.LoadPriority(backlogDir)
		if err != nil {
			return err
		}
		ice, err := backlog.LoadIcebox(backlogDir)
		if err != nil {
			return err
		}
		if pri.IndexOf(itemFile) < 0 {
			if ice.IndexOf(itemFile) >= 0 {
				return fmt.Errorf("%s already in icebox", itemFile)
			}
			return fmt.Errorf("%s not in priority; run `am sync` first", itemFile)
		}
		i := pri.IndexOf(itemFile)
		e := pri.Entries()[i]
		pri.Remove(itemFile)
		if c.Bool("top") {
			ice.InsertTop(e)
		} else {
			ice.InsertBottom(e)
		}
		if err := pri.Save(); err != nil {
			return err
		}
		if err := ice.Save(); err != nil {
			return err
		}
		fmt.Printf("iced %s\n", itemFile)
		return nil
	},
}

// UnIceCommand promotes from icebox to priority.
var UnIceCommand = &cli.Command{
	Name:      "unice",
	Usage:     "Promote item(s) from _icebox.md to _priority.md",
	ArgsUsage: "[ITEM_PATH]",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "all", Usage: "move the entire icebox to the bottom of priority, preserving order"},
		&cli.BoolFlag{Name: "top", Usage: "place at top of priority (default: bottom)"},
		&cli.StringFlag{Name: "after", Usage: "place immediately after this item"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		moveAll := c.Bool("all")
		if !moveAll && c.NArg() != 1 {
			return fmt.Errorf("usage: am unice ITEM_PATH [--top|--after X]  OR  am unice --all")
		}
		var backlogDir string
		if moveAll {
			if err := checkIsBacklogDirectory(); err != nil {
				return err
			}
			d, err := filepath.Abs(".")
			if err != nil {
				return err
			}
			backlogDir = d
		} else {
			d, _, err := itemAndBacklogDir(c.Args().Get(0))
			if err != nil {
				return err
			}
			backlogDir = d
		}
		pri, err := backlog.LoadPriority(backlogDir)
		if err != nil {
			return err
		}
		ice, err := backlog.LoadIcebox(backlogDir)
		if err != nil {
			return err
		}

		if moveAll {
			for _, e := range ice.Entries() {
				pri.InsertBottom(e)
			}
			ice = mustNewEmptyOrder(backlog.IceboxFilePath(backlogDir), "Icebox")
			if err := pri.Save(); err != nil {
				return err
			}
			if err := ice.Save(); err != nil {
				return err
			}
			fmt.Printf("moved %d items from icebox to priority\n", len(pri.Entries()))
			return nil
		}

		_, itemFile, _ := itemAndBacklogDir(c.Args().Get(0))
		if ice.IndexOf(itemFile) < 0 {
			if pri.IndexOf(itemFile) >= 0 {
				return fmt.Errorf("%s already in priority", itemFile)
			}
			return fmt.Errorf("%s not in icebox; run `am sync` first", itemFile)
		}
		i := ice.IndexOf(itemFile)
		e := ice.Entries()[i]
		ice.Remove(itemFile)
		switch {
		case c.Bool("top"):
			pri.InsertTop(e)
		case c.String("after") != "":
			anchor := resolveAnchor(backlogDir, c.String("after"))
			ai := pri.IndexOf(anchor)
			if ai < 0 {
				pri.InsertBottom(e)
			} else {
				pri.InsertAt(ai+1, e)
			}
		default:
			pri.InsertBottom(e)
		}
		if err := pri.Save(); err != nil {
			return err
		}
		if err := ice.Save(); err != nil {
			return err
		}
		fmt.Printf("uniced %s\n", itemFile)
		return nil
	},
}

func mustNewEmptyOrder(path, header string) *backlog.OrderFile {
	f, _ := backlog.LoadOrderFile(path, header)
	for _, e := range f.Entries() {
		f.Remove(e.Path)
	}
	return f
}
