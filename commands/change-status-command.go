package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	itemStatusRe = regexp.MustCompile(`^(\d+)\s+([flgh])$`)
)

type ChangeStatusCommand struct {
	Status  string `short:"s" required:"true" description:"status (f, l, g or h)"`
	RootDir string
}

func (*ChangeStatusCommand) Name() string {
	return "change-status"
}

func (cmd *ChangeStatusCommand) Execute(args []string) error {
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

	items := bck.ItemsByStatus(cmd.Status)
	if len(items) == 0 {
		fmt.Printf("No items with status '%s'\n", backlog.GetStatusByCode(cmd.Status))
		return nil
	}

	printBacklogItems(items, fmt.Sprintf("Stories %s", backlog.GetStatusDescriptionByCode(cmd.Status)))
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
}
