package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"strings"
)

const newItemTemplate = `## Problem statement

## Possible solution

## Comments

## Attachments
`

type CreateItemAction struct {
	rootDir   string
	itemTitle string
	user      string
	simulate  bool
}

func NewCreateItemAction(rootDir, itemTitle, user string, simulate bool) *CreateItemAction {
	return &CreateItemAction{rootDir: rootDir, itemTitle: itemTitle, user: user, simulate: simulate}
}

func (a *CreateItemAction) Execute() error {
	itemName := utils.GetValidFileName(a.itemTitle)
	itemPath := filepath.Join(a.rootDir, fmt.Sprintf("%s.md", itemName))
	if existsFile(itemPath) {
		if !a.simulate {
			fmt.Println("file exists")
		} else {
			fmt.Println(itemPath)
		}
		return nil
	}

	if backlog.IsForbiddenItemName(itemName) {
		if !a.simulate {
			fmt.Printf("'%s' can't be used as an item name\n", itemName)
		}
		return nil
	}

	currentUser := a.user
	if currentUser == "" {
		var err error
		currentUser, _, err = git.CurrentUser()
		if err != nil {
			currentUser = "unknown"
		}
	}

	item, err := backlog.LoadBacklogItem(itemPath)
	if err != nil {
		return err
	}
	currentTimestamp := utils.GetCurrentTimestamp()
	item.SetTitle(utils.TitleFirstLetter(a.itemTitle))
	item.SetCreated(currentTimestamp)
	item.SetModified(currentTimestamp)
	item.SetTags(nil)
	item.SetAuthor(currentUser)
	item.SetStatus(backlog.UnplannedStatus)
	item.SetAssigned("")
	item.SetEstimate("")
	item.SetDescription(newItemTemplate)

	if !a.simulate {
		return item.Save()
	}

	itemPath, _ = filepath.Abs(itemPath)
	rootDir := filepath.Dir(filepath.Dir(itemPath))
	fmt.Println(strings.TrimPrefix(itemPath, rootDir))
	fmt.Print(string(item.Content()))
	return nil
}
