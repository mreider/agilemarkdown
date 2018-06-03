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

const newItemTemplate = `## Problem statement

## Possible solution

## Comments

## Attachments
`

var CreateItemCommand = cli.Command{
	Name:      "create-item",
	Usage:     "Create a new item for the backlog",
	ArgsUsage: "ITEM_NAME",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "simulate",
			Hidden: true,
		},
		cli.StringFlag{
			Name:   "user",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
		simulate := c.Bool("simulate")
		user := c.String("user")

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		if c.NArg() == 0 {
			if !simulate {
				fmt.Println("an item name should be specified")
			}
			return nil
		}
		itemTitle := strings.Join(c.Args(), " ")
		itemName := strings.Replace(itemTitle, " ", "-", -1)
		itemPath := filepath.Join(".", fmt.Sprintf("%s.md", itemName))
		if existsFile(itemPath) {
			if !simulate {
				fmt.Println("file exists")
			} else {
				fmt.Println(itemPath)
			}
			return nil
		}

		if backlog.IsForbiddenItemName(itemName) {
			if !simulate {
				fmt.Printf("'%s' can't be used as an item name\n", itemName)
			}
			return nil
		}

		currentUser := user
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
		item.SetTitle(utils.TitleFirstLetter(itemTitle))
		item.SetCreated("")
		item.SetModified()
		item.SetTags(nil)
		item.SetAuthor(currentUser)
		item.SetStatus(backlog.UnplannedStatus)
		item.SetAssigned("")
		item.SetEstimate("")
		item.SetDescription(newItemTemplate)

		if !simulate {
			return item.Save()
		} else {
			itemPath, _ := filepath.Abs(itemPath)
			rootDir := filepath.Dir(filepath.Dir(itemPath))
			fmt.Println(strings.TrimPrefix(itemPath, rootDir))
			fmt.Print(string(item.Content()))
			return nil
		}
	},
}
