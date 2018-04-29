package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

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
		itemName := strings.Join(c.Args(), " ")
		itemFileName := strings.Replace(itemName, " ", "_", -1)
		itemPath := filepath.Join(".", fmt.Sprintf("%s.md", itemFileName))
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
		item.SetTitle(itemName)
		item.SetCreated()
		item.SetModified()
		item.SetAuthor(currentUser)
		item.SetStatus(backlog.StatusNameByCode("h"))
		item.SetAssigned("")
		item.SetEstimate("")
		return item.Save()
	},
}
