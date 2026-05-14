package mcpserver

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type NextItemArgs struct {
	Backlog string `json:"backlog,omitempty" jsonschema:"optional backlog filter; default scans every backlog"`
}

type NextItemResult struct {
	Found   bool   `json:"found"`
	Backlog string `json:"backlog,omitempty"`
	Path    string `json:"path,omitempty"`
	Title   string `json:"title,omitempty"`
	Status  string `json:"status,omitempty"`
	Type    string `json:"type,omitempty"`
}

// nextItemTool returns the highest-ranked unstarted, unblocked item across
// the project (or one backlog when filtered). The "next pull" answer.
func nextItemTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, NextItemArgs) (*mcp.CallToolResult, NextItemResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args NextItemArgs) (*mcp.CallToolResult, NextItemResult, error) {
		dirs, err := root.BacklogDirs()
		if err != nil {
			return nil, NextItemResult{}, err
		}
		for _, d := range dirs {
			if args.Backlog != "" && filepath.Base(d) != args.Backlog {
				continue
			}
			pri, err := backlog.LoadPriority(d)
			if err != nil {
				return nil, NextItemResult{}, err
			}
			bck, err := backlog.LoadBacklog(d)
			if err != nil {
				return nil, NextItemResult{}, err
			}
			byBase := indexItems(bck)
			for _, e := range pri.Entries() {
				it, ok := byBase[e.Path]
				if !ok {
					continue
				}
				if it.Blocked() {
					continue
				}
				if !strings.EqualFold(it.Status(), backlog.UnstartedStatus.Name) {
					continue
				}
				if it.Type() == "release" {
					continue
				}
				rel, _ := filepath.Rel(root.Root(), it.Path())
				return nil, NextItemResult{
					Found:   true,
					Backlog: filepath.Base(d),
					Path:    rel,
					Title:   it.Title(),
					Status:  it.Status(),
					Type:    it.Type(),
				}, nil
			}
		}
		return nil, NextItemResult{Found: false}, nil
	}
}
