package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"path/filepath"
)

type CreateItemCommand struct {
	RootDir string
}

func (*CreateItemCommand) Name() string {
	return "create-item"
}

func (cmd *CreateItemCommand) Execute(args []string) error {
	if err := checkIsBacklogDirectory(cmd.RootDir); err != nil {
		return err
	}
	if len(args) != 1 {
		return fmt.Errorf("an item name should be specified")
	}
	itemName := args[0]
	itemPath := filepath.Join(cmd.RootDir, fmt.Sprintf("%s.md", itemName))
	if existsFile(itemPath) {
		return fmt.Errorf("file exists")
	}

	// TODO: user list
	// TODO: initial status
	currentUser, err := git.CurrentUser()
	if err != nil {
		currentUser = "unknown"
	}

	item, err := backlog.CreateBacklogItem(itemPath)
	if err != nil {
		return err
	}
	item.SetTitle("")
	item.SetCreated()
	item.SetModified()
	item.SetAuthor(currentUser)
	item.SetStatus("")
	item.SetAssigned("")
	item.SetEstimate("")
	return item.Save()
}
