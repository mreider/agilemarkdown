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

func DoneList(ctx context.Context, root string, args DoneListArgs) (DoneListResult, error) {
	_, r, err := doneListTool(wrapRoot(root))(ctx, nil, args)
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

func NextItem(ctx context.Context, root string, args NextItemArgs) (NextItemResult, error) {
	_, r, err := nextItemTool(wrapRoot(root))(ctx, nil, args)
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

// ---- write wrappers used by the laboratory (cmd/am-server) ----

func SetStatus(ctx context.Context, root string, args SetStatusArgs) (OkResult, error) {
	_, r, err := setStatus(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetEstimate(ctx context.Context, root string, args SetEstimateArgs) (OkResult, error) {
	_, r, err := setEstimate(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetAssigned(ctx context.Context, root string, args SetAssignedArgs) (OkResult, error) {
	_, r, err := setAssigned(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetTags(ctx context.Context, root string, args SetTagsArgs) (OkResult, error) {
	_, r, err := setTagsTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetEpic(ctx context.Context, root string, args SetEpicArgs) (OkResult, error) {
	_, r, err := setEpicTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func RankItem(ctx context.Context, root string, args RankItemArgs) (OkResult, error) {
	_, r, err := rankItemTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func MoveToIcebox(ctx context.Context, root string, args MoveToIceboxArgs) (OkResult, error) {
	_, r, err := moveToIceboxTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func MoveToPriority(ctx context.Context, root string, args MoveToPriorityArgs) (OkResult, error) {
	_, r, err := moveToPriorityTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func RejectItem(ctx context.Context, root string, args RejectItemArgs) (OkResult, error) {
	_, r, err := rejectItemTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func AddComment(ctx context.Context, root string, args AddCommentArgs) (OkResult, error) {
	_, r, err := addCommentTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func AddTask(ctx context.Context, root string, args AddTaskArgs) (OkResult, error) {
	_, r, err := addTaskTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func SetTaskDone(ctx context.Context, root string, args SetTaskDoneArgs) (OkResult, error) {
	_, r, err := setTaskDoneTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func CreateItem(ctx context.Context, root string, args CreateItemArgs) (CreateItemResult, error) {
	_, r, err := createItem(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func BlockItem(ctx context.Context, root string, args BlockItemArgs) (OkResult, error) {
	_, r, err := blockItemTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func UnblockItem(ctx context.Context, root string, args UnblockItemArgs) (OkResult, error) {
	_, r, err := unblockItemTool(wrapRoot(root))(ctx, nil, args)
	return r, err
}

func Sync(ctx context.Context, root string, args SyncArgs) (OkResult, error) {
	_, r, err := syncAction(wrapRoot(root))(ctx, nil, args)
	return r, err
}
