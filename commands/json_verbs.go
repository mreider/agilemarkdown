package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/mcpserver"
	"github.com/urfave/cli/v3"
)

// Data-export verbs. Each emits JSON to stdout matching the
// corresponding MCP tool's structured-content shape. The VS Code
// extension calls these per-action instead of hosting a long-lived
// `am mcp` child.

var ListBacklogsCommand = &cli.Command{
	Name:  "list-backlogs",
	Usage: "Emit the project's backlog names as JSON",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		res, err := mcpserver.ListBacklogs(ctx, root)
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var ListItemsCommand = &cli.Command{
	Name:      "list-items",
	Usage:     "Emit every active item in a backlog as JSON (optional status / tag filter)",
	ArgsUsage: "BACKLOG_NAME",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "status", Usage: "filter to a single status"},
		&cli.StringFlag{Name: "tag", Usage: "filter to a single tag"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am list-items BACKLOG_NAME")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		res, err := mcpserver.ListItems(ctx, root, mcpserver.ListItemsArgs{
			Backlog: c.Args().Get(0),
			Status:  c.String("status"),
			Tag:     c.String("tag"),
		})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var GetItemCommand = &cli.Command{
	Name:      "get-item",
	Usage:     "Emit a single item's frontmatter + body as JSON",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am get-item ITEM_PATH")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		rel, err := relToRoot(root, c.Args().Get(0))
		if err != nil {
			return err
		}
		res, err := mcpserver.GetItem(ctx, root, mcpserver.GetItemArgs{Path: rel})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var GetCommentsCommand = &cli.Command{
	Name:      "get-comments",
	Usage:     "Emit an item's '## Comments' section as JSON",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am get-comments ITEM_PATH")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		rel, err := relToRoot(root, c.Args().Get(0))
		if err != nil {
			return err
		}
		res, err := mcpserver.GetComments(ctx, root, mcpserver.GetCommentsArgs{Path: rel})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var TypeMixCommand = &cli.Command{
	Name:  "type-mix",
	Usage: "Emit the accepted-story type mix (feature/bug/chore/release) as JSON",
	Action: func(ctx context.Context, c *cli.Command) error {
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		res, err := mcpserver.TypeMix(ctx, root, mcpserver.TypeMixArgs{})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var SearchCommand = &cli.Command{
	Name:      "search",
	Usage:     "Substring search across all stories. Emits scored hits as JSON.",
	ArgsUsage: "QUERY",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "limit", Value: 20, Usage: "max hits to return"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 1 {
			return fmt.Errorf("usage: am search QUERY")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		query := strings.Join(c.Args().Slice(), " ")
		res, err := mcpserver.Search(ctx, root, mcpserver.SearchArgs{Query: query, Limit: c.Int("limit")})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

var HistoryCommand = &cli.Command{
	Name:      "history",
	Usage:     "Emit the git commits that touched an item (newest first) as JSON",
	ArgsUsage: "ITEM_PATH",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "limit", Value: 50, Usage: "max commits to return"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am history ITEM_PATH")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		rel, err := relToRoot(root, c.Args().Get(0))
		if err != nil {
			return err
		}
		entries, err := git.HistoryFor(rel, c.Int("limit"))
		if err != nil {
			return err
		}
		return emitJSON(map[string]any{"entries": entries, "count": len(entries)})
	},
}

var WhoamiCommand = &cli.Command{
	Name:  "whoami",
	Usage: "Emit the current git user (name + email) as JSON. Used by clients filtering 'my work'.",
	Action: func(ctx context.Context, c *cli.Command) error {
		name, email, err := git.CurrentUser()
		if err != nil {
			return err
		}
		return emitJSON(map[string]string{"name": name, "email": email})
	},
}

var SetDescriptionCommand = &cli.Command{
	Name:      "set-description",
	Usage:     "Replace an item's body. Reads the new markdown body from stdin.",
	ArgsUsage: "ITEM_PATH",
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() != 1 {
			return fmt.Errorf("usage: am set-description ITEM_PATH < new-body.md")
		}
		root, err := findRootDirectory()
		if err != nil {
			return err
		}
		rel, err := relToRoot(root, c.Args().Get(0))
		if err != nil {
			return err
		}
		body, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		res, err := mcpserver.SetDescription(ctx, root, mcpserver.SetDescriptionArgs{Path: rel, Body: string(body)})
		if err != nil {
			return err
		}
		return emitJSON(res)
	},
}

// relToRoot turns an item path (absolute, relative, with or without
// `.md`) into a path relative to the project root, the shape the MCP
// tool args expect.
func relToRoot(root, arg string) (string, error) {
	p := arg
	if !strings.HasSuffix(p, ".md") {
		p += ".md"
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	return rel, nil
}
