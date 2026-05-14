package mcpserver

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type TaskRow struct {
	Index int    `json:"index"`
	Done  bool   `json:"done"`
	Text  string `json:"text"`
}

type ListTasksArgs struct {
	Path string `json:"path" jsonschema:"file path relative to project root"`
}

type ListTasksResult struct {
	Tasks []TaskRow `json:"tasks"`
	Count int       `json:"count"`
}

func listTasksTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, ListTasksArgs) (*mcp.CallToolResult, ListTasksResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ListTasksArgs) (*mcp.CallToolResult, ListTasksResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, ListTasksResult{}, err
		}
		tasks := backlog.ParseTasks(item.Body())
		out := make([]TaskRow, 0, len(tasks))
		for _, t := range tasks {
			out = append(out, TaskRow{Index: t.Index, Done: t.Done, Text: t.Text})
		}
		return nil, ListTasksResult{Tasks: out, Count: len(out)}, nil
	}
}

type AddTaskArgs struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

func addTaskTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, AddTaskArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args AddTaskArgs) (*mcp.CallToolResult, OkResult, error) {
		if args.Text == "" {
			return nil, OkResult{}, fmt.Errorf("text is required")
		}
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBody(backlog.AppendTask(item.Body(), args.Text))
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}

type SetTaskDoneArgs struct {
	Path  string `json:"path"`
	Index int    `json:"index" jsonschema:"1-based task index"`
	Done  bool   `json:"done"`
}

func setTaskDoneTool(root *backlog.BacklogsStructure) func(context.Context, *mcp.CallToolRequest, SetTaskDoneArgs) (*mcp.CallToolResult, OkResult, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args SetTaskDoneArgs) (*mcp.CallToolResult, OkResult, error) {
		path := filepath.Join(root.Root(), args.Path)
		item, err := backlog.LoadBacklogItem(path)
		if err != nil {
			return nil, OkResult{}, err
		}
		body, err := backlog.SetTaskDone(item.Body(), args.Index, args.Done)
		if err != nil {
			return nil, OkResult{}, err
		}
		item.SetBody(body)
		item.SetModified(utils.GetCurrentTimestamp())
		if err := item.Save(); err != nil {
			return nil, OkResult{}, err
		}
		return nil, OkResult{OK: true}, nil
	}
}
