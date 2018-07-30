package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"strings"
)

type WorkAction struct {
	backlogDir string
	statusCode string
	user       string
	tags       string
}

func NewWorkAction(backlogDir, statusCode, user, tags string) *WorkAction {
	return &WorkAction{backlogDir: backlogDir, statusCode: statusCode, user: user, tags: tags}
}

func (a *WorkAction) Execute() error {
	if a.statusCode != "" && !backlog.IsValidStatusCode(a.statusCode) {
		fmt.Printf("illegal status: %s\n", a.statusCode)
		return nil
	}

	bck, err := backlog.LoadBacklog(a.backlogDir)
	if err != nil {
		return err
	}

	overviewPath, ok := backlog.FindOverviewFileInRootDirectory(a.backlogDir)
	if !ok {
		return fmt.Errorf("the overview file isn't found for %s", a.backlogDir)
	}
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return err
	}

	archivePath, _ := backlog.FindArchiveFileInDirectory(a.backlogDir)
	archive, err := backlog.LoadBacklogOverview(archivePath)
	if err != nil {
		return err
	}

	var statuses []*backlog.BacklogItemStatus
	if a.statusCode == "" {
		statuses = []*backlog.BacklogItemStatus{backlog.DoingStatus, backlog.PlannedStatus, backlog.UnplannedStatus}
	} else {
		statuses = []*backlog.BacklogItemStatus{backlog.StatusByCode(a.statusCode)}
	}

	sorter := backlog.NewBacklogItemsSorter(overview, archive)
	for _, status := range statuses {
		filter := &backlog.BacklogItemsAndFilter{}
		filter.And(backlog.NewBacklogItemsStatusCodeFilter(status.Code))
		filter.And(backlog.NewBacklogItemsAssignedFilter(a.user))
		filter.And(backlog.NewBacklogItemsTagsFilter(a.tags))
		items := bck.FilteredActiveItems(filter)

		sorter.SortItemsByStatus(status, items)
		lines := backlog.BacklogView{}.WriteAsciiItems(items, status, false, true)
		fmt.Println(strings.Join(lines, "\n"))
		fmt.Println("")
	}

	return nil
}
