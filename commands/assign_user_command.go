package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/users"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	itemUserRe = regexp.MustCompile(`^(\d+)\s+(.*)$`)
)

var AssignUserCommand = cli.Command{
	Name:      "assign",
	Usage:     "Assign a story to a user",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "s",
			Usage: fmt.Sprintf("Status - %s", backlog.AllStatusesList()),
		},
	},
	Action: func(c *cli.Context) error {
		statusCode := c.String("s")

		if statusCode == "" {
			fmt.Println("-s option is required")
			return nil
		}
		if !backlog.IsValidStatusCode(statusCode) {
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

		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}

		archivePath, _ := findArchiveFileInDirectory(backlogDir)
		archive, err := backlog.LoadBacklogOverview(archivePath)
		if err != nil {
			return err
		}

		filter := backlog.NewBacklogItemsStatusCodeFilter(statusCode)
		items := bck.FilteredActiveItems(filter)
		status := backlog.StatusByCode(statusCode)
		if len(items) == 0 {
			fmt.Printf("No items with status '%s'\n", status.Name)
			return nil
		}

		sorter := backlog.NewBacklogItemsSorter(overview, archive)
		sorter.SortItemsByStatus(status, items)
		lines := backlog.BacklogView{}.WriteAsciiItems(items, status, true, false)
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Println("")

		userList := users.NewUserList(filepath.Join(backlogDir, "..", backlog.UsersDirectoryName))
		allUsers := userList.AllUsers()
		sort.Strings(allUsers)

		fmt.Printf("Users: %s\n", strings.Join(allUsers, ", "))
		fmt.Println()
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Println("Enter a number to a story number followed by a username, or e to exit")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			if strings.ToLower(text) == "e" {
				break
			}
			match := itemUserRe.FindStringSubmatch(text)
			if match != nil {
				itemNo, _ := strconv.Atoi(match[1])
				user := match[2]
				itemIndex := itemNo - 1
				if itemIndex < 0 || itemIndex >= len(items) {
					fmt.Println("illegal story number")
					continue
				}
				item := items[itemIndex]
				item.SetAssigned(user)
				item.SetModified(utils.GetCurrentTimestamp())
				item.Save()
			}
		}

		return nil
	},
}
