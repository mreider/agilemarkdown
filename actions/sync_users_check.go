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
	items, _, err := backlog.AllBacklogItems(ch.root)
	if err != nil {
		return false, err
	}

	ideasDir := ch.root.IdeasDirectory()
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return false, err
	}

	unknownUsers := ch.getUnknownUsers(items, ideas)
	unknownUsers = ch.userList.ResolveGitUsers(unknownUsers)

	if len(unknownUsers) != 0 {
		fmt.Printf("You should resolve unknown users: %s\n", strings.Join(unknownUsers, ", "))
		return false, nil
	}

	return true, nil
}

func (ch *SyncUsersCheck) getUnknownUsers(items []*backlog.BacklogItem, ideas []*backlog.BacklogIdea) []string {
	unknownUsersSet := make(map[string]struct{})
	for _, item := range items {
		// TODO check git status

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

	for _, idea := range ideas {
		// TODO check git status

		for _, comment := range idea.Comments() {
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
