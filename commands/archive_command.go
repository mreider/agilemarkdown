package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"time"
)

var ArchiveCommand = cli.Command{
	Name:      "archive",
	Usage:     "Archive stories before a certain date",
	ArgsUsage: "YYYY-MM-DD",
	Action: func(c *cli.Context) error {
		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		if c.NArg() != 1 {
			fmt.Println("A date should be specified")
			return nil
		}

		beforeDate, err := time.Parse("2006-1-2", c.Args()[0])
		if err != nil {
			fmt.Println("Invalid date. Should be in YYYY-MM-DD format.")
			return nil
		}

		backlogDir, _ := filepath.Abs(".")
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		var itemsToArchive []*backlog.BacklogItem
		for _, item := range bck.ActiveItems() {
			// beforeDate doesn't contain time part. So '<= beforeDate' means '< beforeDate+1day'
			if item.Modified().Before(beforeDate.Add(time.Hour * 24)) {
				itemsToArchive = append(itemsToArchive, item)
			}
		}

		for _, item := range itemsToArchive {
			item.SetArchived(true)
			err := item.Save()
			if err != nil {
				fmt.Printf("Can't archive the item '%s': %v\n", item.Title(), err)
			}
		}
		return nil
	},
}
