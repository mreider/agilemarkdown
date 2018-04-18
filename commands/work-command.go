package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"sort"
	"strconv"
	"strings"
)

type WorkCommand struct {
	User    string `short:"u" required:"false" description:"user"`
	Status  string `short:"s" required:"false" description:"status (f, l, g or h)" default:"f"`
	RootDir string
}

func (*WorkCommand) Name() string {
	return "work"
}

func (cmd *WorkCommand) Execute(args []string) error {
	if cmd.Status != "f" && cmd.Status != "l" && cmd.Status != "g" && cmd.Status != "h" {
		return fmt.Errorf("illegal status: %s", cmd.Status)
	}
	if err := checkIsBacklogDirectory(cmd.RootDir); err != nil {
		return err
	}
	bck, err := backlog.LoadBacklog(cmd.RootDir)
	if err != nil {
		return err
	}

	items := bck.ItemsByStatusAndUser(cmd.Status, cmd.User)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Assigned() < items[j].Assigned() {
			return true
		}
		if items[i].Assigned() > items[j].Assigned() {
			return false
		}
		return items[i].Title() < items[j].Title()
	})

	userHeader, titleHeader, pointsHeader := "User", "Title", "Points"
	maxAssignedLen, maxTitleLen := len(userHeader), len(titleHeader)
	for _, item := range items {
		if len(item.Assigned()) > maxAssignedLen {
			maxAssignedLen = len(item.Assigned())
		}
		if len(item.Title()) > maxTitleLen {
			maxTitleLen = len(item.Title())
		}
	}

	fmt.Printf("Status: %s\n", backlog.GetStatusByCode(cmd.Status))
	fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
	fmt.Printf(" %s | %s | %s\n", padStringRight(userHeader, maxAssignedLen), padStringRight(titleHeader, maxTitleLen), pointsHeader)
	fmt.Printf("-%s---%s---%s\n", strings.Repeat("-", maxAssignedLen), strings.Repeat("-", maxTitleLen), strings.Repeat("-", len(pointsHeader)))
	for _, item := range items {
		estimate, _ := strconv.ParseFloat(item.Estimate(), 64)
		estimateStr := padIntLeft(int(estimate), len(pointsHeader))
		if estimate == 0 {
			estimateStr = ""
		}
		fmt.Printf(" %s | %s | %s\n", padStringRight(item.Assigned(), maxAssignedLen), padStringRight(item.Title(), maxTitleLen), estimateStr)
	}

	return nil
}
