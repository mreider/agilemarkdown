package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	itemUserRe = regexp.MustCompile(`^(\d+)\s+(.*)$`)
)

type AssignUserCommand struct {
	Status  string `short:"s" required:"true" description:"status (f, l, g or h)"`
	RootDir string
}

func (*AssignUserCommand) Name() string {
	return "assign"
}

func (cmd *AssignUserCommand) Execute(args []string) error {
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

	gitUsers, _ := git.KnownUsers()
	knownUsers := bck.KnownUsers()
	usersSet := make(map[string]bool)
	for _, user := range gitUsers {
		usersSet[user] = true
	}
	for _, user := range knownUsers {
		usersSet[user] = true
	}
	users := make([]string, 0, len(usersSet))
	for user := range usersSet {
		users = append(users, user)
	}
	sort.Strings(users)

	// TODO: need users check?
	//usersSet := make(map[string]bool)
	//for _, user := range users {
	//	usersSet[strings.ToLower(user)] = true
	//}

	fmt.Printf("Users: %s\n", strings.Join(users, ", "))
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
			item.Save()
		}
	}

	return nil
}
