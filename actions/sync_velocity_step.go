package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type SyncVelocityStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncVelocityStep(root *backlog.BacklogsStructure) *SyncVelocityStep {
	return &SyncVelocityStep{root: root}
}

func (s *SyncVelocityStep) Execute() error {
	backlogDirs, err := s.root.BacklogDirs()
	if err != nil {
		return err
	}
	velocityPath := s.root.VelocityFile()
	velocity, err := backlog.LoadGlobalVelocity(velocityPath)
	if err != nil {
		return err
	}
	if velocity.Title() == "" {
		velocity.SetTitle("Velocity")
	}
	overviews := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	backlogs := make([]*backlog.Backlog, 0, len(backlogDirs))
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}

		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		overviews = append(overviews, overview)
		backlogs = append(backlogs, bck)
	}
	velocity.Update(backlogs, overviews, backlogDirs, s.root.Root())
	velocity.UpdateLinks(s.root.Root())

	return nil

}
