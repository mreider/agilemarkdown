package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/urfave/cli/v3"
)

var CommentCommand = &cli.Command{
	Name:      "comment",
	Usage:     "Append a dated comment under '## Comments' on an item",
	ArgsUsage: "ITEM_PATH TEXT",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "author", Usage: "author handle (defaults to the item's author or 'user')"},
	},
	Action: func(ctx context.Context, c *cli.Command) error {
		if c.NArg() < 2 {
			fmt.Println("usage: am comment ITEM_PATH TEXT")
			return nil
		}
		path, err := filepath.Abs(c.Args().Get(0))
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			path += ".md"
		}
		text := strings.TrimSpace(strings.Join(c.Args().Tail(), " "))
		if text == "" {
			fmt.Println("comment text is required")
			return nil
		}
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return err
		}
		author := strings.TrimSpace(c.String("author"))
		if author == "" {
			author = strings.TrimSpace(item.Author())
		}
		if author == "" {
			author = "user"
		}
		body := backlog.AppendComment(item.Body(), author, text)
		item.SetBody(body)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return err
		}
		fmt.Printf("%s: comment by %s\n", filepath.Base(path), author)
		return nil
	},
}
