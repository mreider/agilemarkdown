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
	root        *backlog.BacklogsStructure
	backlogName string
}

func NewCreateBacklogAction(rootDir, backlogName string) *CreateBacklogAction {
	return &CreateBacklogAction{root: backlog.NewBacklogsStructure(rootDir), backlogName: backlogName}
}

func (a *CreateBacklogAction) Execute() error {
	if backlog.IsForbiddenBacklogName(a.backlogName) {
		fmt.Printf("'%s' can't be used as a backlog name\n", a.backlogName)
		return nil
	}

	backlogFileName := utils.GetValidFileName(a.backlogName)
	backlogDir := filepath.Join(a.root.Root(), backlogFileName)
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

	err = os.MkdirAll(a.root.IdeasDirectory(), 0777)
	if err != nil {
		return err
	}

	overviewFileName := fmt.Sprintf("%s.md", backlogFileName)
	overviewPath := filepath.Join(a.root.Root(), overviewFileName)
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return err
	}
	overview.SetTitle(a.backlogName)
	overview.UpdateLinks("archive", filepath.Join(backlogDir, backlog.ArchiveFileName), a.root.Root(), a.root.Root())
	overview.SetCreated(utils.GetCurrentTimestamp())
	return overview.Save()
}
