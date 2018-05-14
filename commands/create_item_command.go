package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

const newItemTemplate = `# %s

## Problem statement

## Possible solution

## Comments

## Attachments
`

var CreateItemCommand = cli.Command{
	Name:      "create-item",
	Usage:     "Create a new item for the backlog",
	ArgsUsage: "ITEM_NAME",
	Action: func(c *cli.Context) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		if c.NArg() == 0 {
			fmt.Println("an item name should be specified")
			return nil
		}
		itemTitle := strings.Join(c.Args(), " ")
		itemName := strings.Replace(itemTitle, " ", "-", -1)
		itemPath := filepath.Join(".", fmt.Sprintf("%s.md", itemName))
		if existsFile(itemPath) {
			fmt.Println("file exists")
			return nil
		}

		currentUser, err := git.CurrentUser()
		if err != nil {
			currentUser = "unknown"
		}

		item, err := backlog.LoadBacklogItem(itemPath)
		if err != nil {
			return err
		}
		item.SetTitle(itemTitle)
		item.SetCreated("")
		item.SetModified()
		item.SetTags(nil)
		item.SetAuthor(currentUser)
		item.SetStatus(backlog.UnplannedStatus)
		item.SetAssigned("")
		item.SetEstimate("")
		item.SetDescription(fmt.Sprintf(newItemTemplate, utils.TitleFirstLetter(itemTitle)))
		return item.Save()
	},
}
