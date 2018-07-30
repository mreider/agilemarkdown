package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type SyncUsersStep struct {
	rootDir string
}

func NewSyncUsersStep(rootDir string) *SyncUsersStep {
	return &SyncUsersStep{rootDir: rootDir}
}

func (s *SyncUsersStep) Execute() error {
	userList := backlog.NewUserList(filepath.Join(s.rootDir, backlog.UsersDirectoryName))
	tagsDir := filepath.Join(s.rootDir, backlog.TagsDirectoryName)

	items, overviews, err := backlog.ActiveBacklogItems(s.rootDir)
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

		_, err := user.UpdateItems(s.rootDir, tagsDir, userItems, overviews)
		if err != nil {
			return err
		}
	}
	return s.updateUsersPage(userList)
}

func (s *SyncUsersStep) updateUsersPage(userList *backlog.UserList) error {
	lines := []string{"# Users", ""}
	lines = append(lines, fmt.Sprintf(utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.rootDir, s.rootDir)...)))
	lines = append(lines, "", "---", "")
	lines = append(lines, fmt.Sprintf("| Name | Nickname | Email |"))
	lines = append(lines, "|---|---|---|")
	for _, user := range userList.Users() {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s |", backlog.MakeUserLink(user, user.Name(), s.rootDir), user.Nickname(), strings.Join(user.Emails(), ", ")))
	}
	return ioutil.WriteFile(filepath.Join(s.rootDir, backlog.UsersFileName), []byte(strings.Join(lines, "  \n")), 0644)
}
