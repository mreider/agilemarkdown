package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
)

type CreateBacklogAction struct {
	rootDir     string
	backlogName string
}

func NewCreateBacklogAction(rootDir, backlogName string) *CreateBacklogAction {
	return &CreateBacklogAction{rootDir: rootDir, backlogName: backlogName}
}

func (a *CreateBacklogAction) Execute() error {
	if backlog.IsForbiddenBacklogName(a.backlogName) {
		fmt.Printf("'%s' can't be used as a backlog name\n", a.backlogName)
		return nil
	}

	backlogFileName := utils.GetValidFileName(a.backlogName)
	backlogDir := filepath.Join(a.rootDir, backlogFileName)
	if info, err := os.Stat(backlogDir); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		if info.IsDir() {
			fmt.Println("the backlog directory already exists")
		} else {
			fmt.Println("a file with the same name already exists")
		}
		return nil
	}

	git.SetUpstream()

	err := os.MkdirAll(backlogDir, 0777)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(filepath.Join(a.rootDir, backlog.IdeasDirectoryName)), 0777)
	if err != nil {
		return err
	}

	overviewFileName := fmt.Sprintf("%s.md", backlogFileName)
	overviewPath := filepath.Join(a.rootDir, overviewFileName)
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return err
	}
	overview.SetTitle(a.backlogName)
	overview.UpdateLinks("archive", filepath.Join(backlogDir, backlog.ArchiveFileName), a.rootDir, a.rootDir)
	overview.SetCreated(utils.GetCurrentTimestamp())
	return overview.Save()
}
