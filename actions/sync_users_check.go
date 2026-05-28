package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"sort"
	"strings"
)

type SyncUsersCheck struct {
	root     *backlog.BacklogsStructure
	userList *backlog.UserList
}

func NewSyncUsersCheck(root *backlog.BacklogsStructure, userList *backlog.UserList) *SyncUsersCheck {
	return &SyncUsersCheck{root: root, userList: userList}
}

func (ch *SyncUsersCheck) Check() (bool, error) {
	fmt.Println("Resolving unknown users")

	items, _, err := backlog.AllBacklogItems(ch.root)
	if err != nil {
		return false, err
	}

	unknownUsers := ch.getUnknownUsers(items)
	unknownUsers, err = ch.userList.ResolveGitUsers(unknownUsers)
	if err != nil {
		return false, err
	}

	if len(unknownUsers) != 0 {
		fmt.Printf("Note: %d user reference(s) without a users/ entry: %s. They will be auto-added on next sync if they appear in git log; or run `am create-user`.\n",
			len(unknownUsers), strings.Join(unknownUsers, ", "))
	}
	return true, nil
}

func (ch *SyncUsersCheck) getUnknownUsers(items []*backlog.BacklogItem) []string {
	unknownUsersSet := make(map[string]struct{})
	for _, item := range items {
		if item.Assigned() != "" {
			assigned := ch.userList.User(item.Assigned())
			if assigned == nil {
				unknownUsersSet[item.Assigned()] = struct{}{}
			}
		}

		for _, comment := range item.Comments() {
			for _, user := range comment.Users {
				if ch.userList.User(user) == nil {
					unknownUsersSet[user] = struct{}{}
				}
			}
		}
	}

	unknownUsers := make([]string, 0, len(unknownUsersSet))
	for user := range unknownUsersSet {
		unknownUsers = append(unknownUsers, user)
	}
	sort.Strings(unknownUsers)

	return unknownUsers
}
