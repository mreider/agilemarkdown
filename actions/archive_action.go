package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"time"
)

type ArchiveAction struct {
	backlogDir string
	beforeDate time.Time
}

func NewArchiveAction(backlogDir string, beforeDate time.Time) *ArchiveAction {
	return &ArchiveAction{backlogDir: backlogDir, beforeDate: beforeDate}
}

func (a *ArchiveAction) Execute() error {
	bck, err := backlog.LoadBacklog(a.backlogDir)
	if err != nil {
		return err
	}

	var itemsToArchive []*backlog.BacklogItem
	for _, item := range bck.ActiveItems() {
		// beforeDate doesn't contain time part. So '<= beforeDate' means '< beforeDate+1day'
		if item.Modified().Before(a.beforeDate.Add(time.Hour * 24)) {
			itemsToArchive = append(itemsToArchive, item)
		}
	}

	for _, item := range itemsToArchive {
		item.SetArchived(true)
		err := item.Save()
		if err != nil {
			fmt.Printf("Can't archive the item '%s': %v\n", item.Title(), err)
		}
	}
	return nil

}
