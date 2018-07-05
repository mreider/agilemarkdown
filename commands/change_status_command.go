package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	itemStatusRe = regexp.MustCompile(`^(\d+)\s+([dpufa])$`)
)

var ChangeStatusCommand = cli.Command{
	Name:      "change-status",
	Usage:     "Change story status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
	},
	Action: func(c *cli.Context) error {
		var items []*backlog.BacklogItem
		var err error
		if items, err = showBacklogItems(c); items == nil {
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
					item.SetStatus(backlog.StatusByCode(statusCode))
					item.SetModified(utils.GetCurrentTimestamp())
				} else {
					item.SetArchived(true)
				}
				item.Save()
			}
		}

		return nil
	},
}
