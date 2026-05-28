package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/actions"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SetTagsArgs struct {
	Path string   `json:"path" jsonschema:"item file path relative to project root"`
	Tags []string `json:"tags" jsonschema:"the new full tag list (replaces existing)"`
}

func setTagsTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetTagsArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetTagsArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		clean := make([]string, 0, len(args.Tags))
		for _, t := range args.Tags {
			t = strings.TrimSpace(t)
			if t != "" {
				clean = append(clean, t)
			}
		}
		item.SetTags(clean)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type SetEpicArgs struct {
	Path string `json:"path"`
	Slug string `json:"slug,omitempty" jsonschema:"epic slug; pass empty string to clear"`
}

func setEpicTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetEpicArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetEpicArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetEpic(strings.TrimSpace(args.Slug))
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type ChangeTagArgs struct {
	Old string `json:"old" jsonschema:"the existing tag to rename"`
	New string `json:"new" jsonschema:"the new tag name"`
}

func changeTagTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ChangeTagArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ChangeTagArgs) (*mcp.CallToolResult, OkResult, error) {
		oldT := strings.TrimSpace(args.Old)
		newT := strings.TrimSpace(args.New)
		if oldT == "" || newT == "" {
			return nil, OkResult{}, fmt.Errorf("old and new are required")
		}
		if oldT == newT {
			return nil, OkResult{OK: true, Message: "no-op: old and new are equal"}, nil
		}
		if err := actions.NewChangeTagAction(root.Root(), oldT, newT).Execute(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type DeleteTagArgs struct {
	Tag string `json:"tag" jsonschema:"the tag to remove from every item carrying it"`
}

func deleteTagTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, DeleteTagArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args DeleteTagArgs) (*mcp.CallToolResult, OkResult, error) {
		tag := strings.TrimSpace(args.Tag)
		if tag == "" {
			return nil, OkResult{}, fmt.Errorf("tag is required")
		}
		// DeleteTagAction prompts y/n. Bypass by inlining the safe path:
		// load every item, drop the tag, save.
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, OkResult{}, err
		}
		count := 0
		for _, dir := range dirs {
			bck, err := backlog.LoadBacklog(dir)
			if err != nil {
				return nil, OkResult{}, err
			}
			for _, item := range bck.AllItems() {
				before := item.Tags()
				next := make([]string, 0, len(before))
				dropped := false
				for _, t := range before {
					if strings.EqualFold(strings.TrimSpace(t), tag) {
						dropped = true
						continue
					}
					next = append(next, t)
				}
				if dropped {
					item.SetTags(next)
					item.ClearTimeline()
					item.SetModified(utils.GetCurrentTimestamp())
					if err := item.Save(); err != nil {
						return nil, OkResult{}, err
					}
					count++
				}
			}
		}
		return nil, OkResult{OK: true, Message: fmt.Sprintf("dropped tag from %d item(s)", count)}, nil
	}
}
