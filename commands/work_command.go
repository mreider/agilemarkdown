package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var WorkCommand = cli.Command{
	Name:      "work",
	Usage:     "Show user work by status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "u",
			Usage: "User Name",
		},
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
		cli.StringFlag{
			Name:  "t",
			Usage: "List of Tags",
		},
	},
	Action: func(c *cli.Context) error {
		user := c.String("u")
		statusCode := c.String("s")
		tags := c.String("t")

		if c.NArg() > 0 {
			fmt.Printf("illegal arguments: %s\n", strings.Join(c.Args(), " "))
			return nil
		}

		if statusCode != "" && !backlog.IsValidStatusCode(statusCode) {
			fmt.Printf("illegal status: %s\n", statusCode)
			return nil
		}

		if err := checkIsBacklogDirectory(); err != nil {
			fmt.Println(err)
			return nil
		}
		backlogDir, _ := filepath.Abs(".")
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the index file isn't found for %s", backlogDir)
		}
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}

		var statuses []*backlog.BacklogItemStatus
		if statusCode == "" {
			statuses = []*backlog.BacklogItemStatus{backlog.DoingStatus, backlog.PlannedStatus, backlog.UnplannedStatus}
		} else {
			statuses = []*backlog.BacklogItemStatus{backlog.StatusByCode(statusCode)}
		}

		for _, status := range statuses {
			filter := &backlog.BacklogItemsAndFilter{}
			filter.And(backlog.NewBacklogItemsStatusCodeFilter(status.Code))
			filter.And(backlog.NewBacklogItemsAssignedFilter(user))
			filter.And(backlog.NewBacklogItemsTagsFilter(tags))
			items := bck.FilteredItems(filter)

			overview.SortItems(status, items)
			lines := backlog.BacklogView{}.WriteAsciiTable(items, fmt.Sprintf("Status: %s", status.Name), false)
			fmt.Println(strings.Join(lines, "\n"))
			fmt.Println("")
		}

		return nil
	},
}
