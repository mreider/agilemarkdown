package actions

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
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

type AssignUserAction struct {
	backlogDir string
	statusCode string
}

func NewAssignUserAction(backlogDir, statusCode string) *AssignUserAction {
	return &AssignUserAction{backlogDir: backlogDir, statusCode: statusCode}
}

func (a *AssignUserAction) Execute() error {
	bck, err := backlog.LoadBacklog(a.backlogDir)
	if err != nil {
		return err
	}

	overviewPath, ok := backlog.FindOverviewFileInRootDirectory(a.backlogDir)
	if !ok {
		return fmt.Errorf("the overview file isn't found for %s", a.backlogDir)
	}
	overview, err := backlog.LoadBacklogOverview(overviewPath)
	if err != nil {
		return err
	}

	archivePath, _ := backlog.FindArchiveFileInDirectory(a.backlogDir)
	archive, err := backlog.LoadBacklogOverview(archivePath)
	if err != nil {
		return err
	}

	filter := backlog.NewBacklogItemsStatusCodeFilter(a.statusCode)
	items := bck.FilteredActiveItems(filter)
	status := backlog.StatusByCode(a.statusCode)
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

	userList := backlog.NewUserList(filepath.Join(a.backlogDir, "..", backlog.UsersDirectoryName))
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
}
