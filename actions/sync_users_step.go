package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"strings"
)

type SyncUsersStep struct {
	root *backlog.BacklogsStructure
}

func NewSyncUsersStep(root *backlog.BacklogsStructure) *SyncUsersStep {
	return &SyncUsersStep{root: root}
}

func (s *SyncUsersStep) Execute() error {
	fmt.Println("Generating user pages")

	userList := backlog.NewUserList(s.root.UsersDirectory())
	tagsDir := s.root.TagsDirectory()

	items, overviews, err := backlog.ActiveBacklogItems(s.root)
	if err != nil {
		return err
	}

	for _, user := range userList.Users() {
		userName, userNick := strings.ToLower(user.Name()), strings.ToLower(user.Nickname())
		var userItems []*backlog.BacklogItem
		for _, item := range items {
			assigned := strings.ToLower(utils.CollapseWhiteSpaces(item.Assigned()))
			if assigned == userNick || assigned == userName {
				userItems = append(userItems, item)
			}
		}

		_, err := user.UpdateItems(s.root.Root(), tagsDir, userItems, overviews)
		if err != nil {
			return err
		}
	}
	return s.updateUsersPage(userList)
}

func (s *SyncUsersStep) updateUsersPage(userList *backlog.UserList) error {
	lines := []string{"# Users", ""}
	lines = append(lines, utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), s.root.Root())...))
	lines = append(lines, "", "---", "")
	lines = append(lines, fmt.Sprintf("| Name | Nickname | Email |"))
	lines = append(lines, "|---|---|---|")
	for _, user := range userList.Users() {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s |", backlog.MakeUserLink(user, user.Name(), s.root.Root()), user.Nickname(), strings.Join(user.Emails(), ", ")))
	}
	return ioutil.WriteFile(s.root.UsersFile(), []byte(strings.Join(lines, "  \n")), 0644)
}
