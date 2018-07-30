package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"path/filepath"
)

type SyncVelocityStep struct {
	rootDir string
}

func NewSyncVelocityStep(rootDir string) *SyncVelocityStep {
	return &SyncVelocityStep{rootDir: rootDir}
}

func (s *SyncVelocityStep) Execute() error {
	backlogDirs, err := backlog.BacklogDirs(s.rootDir)
	if err != nil {
		return err
	}
	velocityPath := filepath.Join(s.rootDir, backlog.VelocityFileName)
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
	velocity.Update(backlogs, overviews, backlogDirs, s.rootDir)
	velocity.UpdateLinks(s.rootDir)

	return nil

}
