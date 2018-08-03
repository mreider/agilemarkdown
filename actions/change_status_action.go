package actions

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	itemStatusRe = regexp.MustCompile(`^(\d+)\s+([dpufa])$`)
)

type ChangeStatusAction struct {
	backlogDir string
	statusCode string
}

func NewChangeStatusAction(backlogDir, statusCode string) *ChangeStatusAction {
	return &ChangeStatusAction{backlogDir: backlogDir, statusCode: statusCode}
}

func (a *ChangeStatusAction) Execute() error {
	var items []*backlog.BacklogItem
	var err error
	if items, err = (backlog.BacklogView{}).ShowBacklogItems(a.backlogDir, a.statusCode); items == nil {
		return err
	}
	reader := bufio.NewReader(os.Stdin)
	for {
		hints := []string{backlog.UnplannedStatus.Hint(), backlog.PlannedStatus.Hint(), backlog.DoingStatus.Hint(), backlog.FinishedStatus.Hint(), "(a)rchive"}
		fmt.Printf("Enter story # number and status %s or e to exit (example: 1 f changes #1 to finished)\n", strings.Join(hints, ", "))
		text, _ := reader.ReadString('\n')
		text = strings.ToLower(strings.TrimSpace(text))
		if text == "e" {
			break
		}
		match := itemStatusRe.FindStringSubmatch(text)
		if match != nil {
			itemNo, _ := strconv.Atoi(match[1])
			statusCode := match[2]
			itemIndex := itemNo - 1
			if itemIndex < 0 || itemIndex >= len(items) {
				fmt.Println("illegal story number")
				continue
			}
			item := items[itemIndex]
			if statusCode != "a" {
				oldStatus := backlog.StatusByName(item.Status())
				newStatus := backlog.StatusByCode(statusCode)
				currentTimestamp := utils.GetCurrentTimestamp()

				item.SetStatus(newStatus)
				item.SetModified(currentTimestamp)
				if oldStatus != newStatus {
					if newStatus == backlog.FinishedStatus {
						item.SetFinished(currentTimestamp)
					} else if oldStatus == backlog.FinishedStatus {
						item.SetFinished("")
					}
				}
			} else {
				item.SetArchived(true)
			}
			err := item.Save()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
