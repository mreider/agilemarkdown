package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
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
	unknownUsers = ch.resolveGitUsers(unknownUsers)

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

func (ch *SyncUsersCheck) resolveGitUsers(unknownUsers []string) (unresolvedUsers []string) {
	names, emails, _ := git.KnownUsers()
	currentUserName, currentUserEmail, _ := git.CurrentUser()
	names = append(names, currentUserName)
	emails = append(emails, currentUserEmail)

	normalizedNames := make([]string, len(names))
	for i, name := range names {
		normalizedNames[i] = utils.CollapseWhiteSpaces(strings.ToLower(name))
	}

	normalizedEmails := make([]string, len(emails))
	for i, email := range emails {
		normalizedEmails[i] = utils.CollapseWhiteSpaces(strings.ToLower(email))
	}

NextUser:
	for _, user := range unknownUsers {
		if ch.userList.User(user) != nil {
			continue
		}

		normalizedUser := utils.CollapseWhiteSpaces(strings.ToLower(user))
		for i, email := range normalizedEmails {
			if email == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ch.userList.AddUser(names[i], emails[i]) {
					ch.userList.Save()
				}
				continue NextUser
			}
		}

		for i, email := range normalizedEmails {
			nickname := strings.SplitN(email, "@", 2)[0]
			if nickname == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ch.userList.AddUser(names[i], emails[i]) {
					ch.userList.Save()
				}
				continue NextUser
			}
		}

		for i, name := range normalizedNames {
			if name == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ch.userList.AddUser(names[i], emails[i]) {
					ch.userList.Save()
				}
				continue NextUser
			}
		}

		unresolvedUsers = append(unresolvedUsers, user)
	}

	return unresolvedUsers
}
