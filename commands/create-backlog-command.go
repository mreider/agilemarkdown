package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"path/filepath"
)

type CreateBacklogCommand struct {
	RootDir string
}

func (*CreateBacklogCommand) Name() string {
	return "create-backlog"
}

func (cmd *CreateBacklogCommand) Execute(args []string) error {
	if err := checkIsRootDirectory(cmd.RootDir); err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("A backlog name should be specified")
	}

	backlogDir := filepath.Join(cmd.RootDir, args[0])
	if info, err := os.Stat(backlogDir); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		if info.IsDir() {
			return fmt.Errorf("the backlog directory already exists")
		} else {
			return fmt.Errorf("a file with the same name already exists")
		}
	}

	err := os.MkdirAll(backlogDir, 0777)
	if err != nil {
		return err
	}

	// TODO: 0-landed.yaml, 0-flying.yaml, 0-gate.yaml, 0-hangar.yaml

	overviewPath := filepath.Join(backlogDir, backlog.OverviewFileName)
	overview, err := backlog.CreateBacklogOverview(overviewPath)
	if err != nil {
		return err
	}
	overview.SetTitle("")
	overview.SetCreated()
	return overview.Save()
}
