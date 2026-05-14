package mcpserver

// Public wrappers around the tool handlers, so non-server callers (the
// CLI, the VS Code extension shelling out per-verb) can run the same
// data path without spawning an MCP child. Each wrapper takes a root
// path string, builds the BacklogsStructure, and dispatches to the
// existing tool function. nil request is safe; handlers don't use it.

import (
	"context"

	"github.com/mreider/agilemarkdown/backlog"
)

func wrapRoot(root string) *backlog.BacklogsStructure {
	return backlog.NewBacklogsStructure(root)
}

func PriorityList(ctx context.Context, root string, args PriorityListArgs) (PriorityListResult, error) {
	_, r, err := priorityListTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func IceboxList(ctx context.Context, root string, args IceboxListArgs) (IceboxListResult, error) {
	_, r, err := iceboxListTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func EpicProgress(ctx context.Context, root string, args EpicProgressArgs) (EpicProgressResult, error) {
	_, r, err := epicProgressTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func Dashboard(ctx context.Context, root string, args DashboardArgs) (DashboardResult, error) {
	_, r, err := dashboardTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func TypeMix(ctx context.Context, root string, args TypeMixArgs) (TypeMixResult, error) {
	_, r, err := typeMixTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func VelocityHistory(ctx context.Context, root string, args VelocityHistoryArgs) (VelocityHistoryResult, error) {
	_, r, err := velocityHistoryTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func BurnupChart(ctx context.Context, root string, args BurnupArgs) (BurnupResult, error) {
	_, r, err := burnupChartTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func CumulativeFlow(ctx context.Context, root string, args CFDArgs) (CFDResult, error) {
	_, r, err := cfdTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func Search(ctx context.Context, root string, args SearchArgs) (SearchResult, error) {
	_, r, err := searchTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func ListBacklogs(ctx context.Context, root string) (ListBacklogsResult, error) {
	_, r, err := listBacklogs(wrapRoot(root))(ctx, nil, ListBacklogsArgs{})
	return r, err
}

func ListItems(ctx context.Context, root string, args ListItemsArgs) (ListItemsResult, error) {
	_, r, err := listItems(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func GetItem(ctx context.Context, root string, args GetItemArgs) (GetItemResult, error) {
	_, r, err := getItem(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func GetComments(ctx context.Context, root string, args GetCommentsArgs) (GetCommentsResult, error) {
	_, r, err := getCommentsTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func ListTasks(ctx context.Context, root string, args ListTasksArgs) (ListTasksResult, error) {
	_, r, err := listTasksTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SprintPlan(ctx context.Context, root string, args SprintPlanArgs) (SprintPlanResult, error) {
	_, r, err := sprintPlanTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetDescription(ctx context.Context, root string, args SetDescriptionArgs) (OkResult, error) {
	_, r, err := setDescriptionTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}
