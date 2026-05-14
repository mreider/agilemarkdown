package mcpserver

import (
	"context"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type BlockItemArgs struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
}

func blockItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, BlockItemArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args BlockItemArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBlocked(true, args.Reason)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type UnblockItemArgs struct {
	Path string `json:"path"`
}

func unblockItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, UnblockItemArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args UnblockItemArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBlocked(false, "")
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type SetDescriptionArgs struct {
	Path string `json:"path"`
	Body string `json:"body" jsonschema:"new markdown body for the item; comments and tasks sections are preserved verbatim"`
}

func setDescriptionTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetDescriptionArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetDescriptionArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBody(args.Body)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}
