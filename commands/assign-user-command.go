package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"gopkg.in/urfave/cli.v1"
	"os"
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
		bck, err := backlog.LoadBacklog(".")
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
	},
}
