package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"gopkg.in/urfave/cli.v1"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	itemStatusRe = regexp.MustCompile(`^(\d+)\s+([flgh])$`)
)

var ChangeStatusCommand = cli.Command{
	Name:      "change-status",
	Usage:     "Change story status",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "s",
			Usage: "Status - either l (landed), f (flying), g (gate), or h (hangar).",
		},
	},
	Action: func(c *cli.Context) error {
		status := c.String("s")

		if status == "" {
			fmt.Println("-s option is required")
			return nil
		}
		if status != "f" && status != "l" && status != "g" && status != "h" {
			fmt.Printf("illegal status: %s\n", status)
			return nil
		}
		if err := checkIsBacklogDirectory(); err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog("")
		if err != nil {
			return err
		}

		items := bck.ItemsByStatus(status)
		if len(items) == 0 {
			fmt.Printf("No items with status '%s'\n", backlog.GetStatusByCode(status))
			return nil
		}

		printBacklogItems(items, fmt.Sprintf("Stories %s", backlog.GetStatusDescriptionByCode(status)))
		fmt.Println("")
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Enter a number to a story number followed by a status, or e to exit")
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
				item.SetStatus(backlog.GetStatusByCode(statusCode))
				item.Save()
			}
		}

		return nil
	},
}
