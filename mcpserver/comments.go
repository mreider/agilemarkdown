package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CommentRow struct {
	Author string   `json:"author,omitempty"`
	Users  []string `json:"users,omitempty"`
	When   string   `json:"when,omitempty"`
	Text   string   `json:"text"`
}

type GetCommentsArgs struct {
	Path string `json:"path"`
}

type GetCommentsResult struct {
	Comments []CommentRow `json:"comments"`
	Count    int          `json:"count"`
}

func getCommentsTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, GetCommentsArgs) (*mcp.CallToolResult, GetCommentsResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args GetCommentsArgs) (*mcp.CallToolResult, GetCommentsResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, GetCommentsResult{}, err
		}
		comments := item.Comments()
		out := make([]CommentRow, 0, len(comments))
		for _, c := range comments {
			row := CommentRow{Users: append([]string(nil), c.Users...), Text: strings.TrimSpace(strings.Join(c.Text, "\n"))}
			if len(c.Users) > 0 {
				row.Author = c.Users[0]
			}
			out = append(out, row)
		}
		return nil, GetCommentsResult{Comments: out, Count: len(out)}, nil
	}
}

type AddCommentArgs struct {
	Path   string `json:"path"`
	Text   string `json:"text"`
	Author string `json:"author,omitempty" jsonschema:"author handle (defaults to the item author or git user)"`
}

func addCommentTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, AddCommentArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args AddCommentArgs) (*mcp.CallToolResult, OkResult, error) {
		text := strings.TrimSpace(args.Text)
		if text == "" {
			return nil, OkResult{}, fmt.Errorf("text is required")
		}
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		author := strings.TrimSpace(args.Author)
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
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

