package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	itemTimelineRe = regexp.MustCompile(`^(\d+)\s+(.+)$`)
)

var TimelineCommand = cli.Command{
	Name:      "timeline",
	Usage:     "Build a new timeline",
	ArgsUsage: "TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			fmt.Println("a tag should be specified")
			return nil
		}

		tag := strings.ToLower(c.Args()[0])
		action := &TimelineAction{tag: tag}
		return action.Execute()
	},
}

type TimelineAction struct {
	tag string
}

func (a *TimelineAction) Execute() error {
	rootDir, _ := filepath.Abs(".")
	if err := checkIsBacklogDirectory(); err == nil {
		rootDir = filepath.Dir(rootDir)
	} else if err := checkIsRootDirectory("."); err != nil {
		return err
	}
	_, itemsTags, _, overviews, err := backlog.ItemsAndIdeasTags(rootDir)
	if err != nil {
		return err
	}

	items := itemsTags[a.tag]
	if len(items) == 0 {
		fmt.Printf("No items with the tag '%s'\n", a.tag)
		return nil
	}

	a.sortItems(items)

	reader := bufio.NewReader(os.Stdin)
	needOutput := true
	for {
		if needOutput {
			lines := backlog.BacklogView{}.WriteAsciiItemsWithProjectAndStatus(items, overviews, "", true, a.tag)
			fmt.Println(strings.Join(lines, "\n"))
			needOutput = false
		}

		fmt.Println("Enter story # number and start/end dates (as YYYY-MM-DD) or e to exit (example: 1 2018-07-08 2018-07-15)")
		fmt.Println("Enter story # number and clear to clear the dates (example: 3 clear)")
		text, _ := reader.ReadString('\n')
		text = strings.ToLower(strings.TrimSpace(text))
		if text == "e" {
			break
		}

		match := itemTimelineRe.FindStringSubmatch(text)
		if match != nil {
			itemNo, _ := strconv.Atoi(match[1])
			itemIndex := itemNo - 1
			if itemIndex < 0 || itemIndex >= len(items) {
				fmt.Println("illegal story number")
				continue
			}

			item := items[itemIndex]
			if match[2] == "clear" {
				item.ClearTimeline(a.tag)
				item.Save()
				needOutput = true
			} else {
				parts := strings.Fields(match[2])
				if len(parts) != 2 {
					fmt.Println("illegal format")
					continue
				}
				startDateStr, endDateStr := parts[0], parts[1]
				startDate, err := time.Parse("06-1-2", startDateStr)
				if err != nil {
					fmt.Println("illegal start date")
					continue
				}
				endDate, err := time.Parse("06-1-2", endDateStr)
				if err != nil {
					fmt.Println("illegal end date")
					continue
				}
				if startDate.After(endDate) {
					fmt.Println("The start date shouldn't be after the end date")
					continue
				}

				item.SetTimeline(a.tag, startDate, endDate)
				item.Save()
				needOutput = true
			}
		}
	}
	return nil
}

func (a *TimelineAction) sortItems(items []*backlog.BacklogItem) {
	statuses := map[string]int{
		strings.ToLower(backlog.FinishedStatus.Name):  0,
		strings.ToLower(backlog.DoingStatus.Name):     1,
		strings.ToLower(backlog.PlannedStatus.Name):   2,
		strings.ToLower(backlog.UnplannedStatus.Name): 3,
	}

	sort.Slice(items, func(i, j int) bool {
		var iStatus, jStatus int
		var ok bool
		if iStatus, ok = statuses[strings.ToLower(items[i].Status())]; !ok {
			iStatus = 10
		}
		if jStatus, ok = statuses[strings.ToLower(items[j].Status())]; !ok {
			jStatus = 10
		}

		if iStatus != jStatus {
			return iStatus < jStatus
		}

		if items[i].Modified() != items[j].Modified() {
			return items[i].Modified().Before(items[j].Modified())
		}
		return items[i].Name() < items[j].Name()
	})
}
