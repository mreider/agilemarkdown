package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type ChangeUserAction struct {
	root            *backlog.BacklogsStructure
	fromNameOrEmail string
	toNameOrEmail   string
}

func NewChangeUserAction(rootDir, fromNameOrEmail, toNameOrEmail string) *ChangeUserAction {
	return &ChangeUserAction{root: backlog.NewBacklogsStructure(rootDir), fromNameOrEmail: fromNameOrEmail, toNameOrEmail: toNameOrEmail}
}

func (a *ChangeUserAction) Execute() error {
	userList := backlog.NewUserList(a.root.UsersDirectory())

	fromUser := userList.User(a.fromNameOrEmail)
	if fromUser == nil {
		fmt.Printf("User '%s' not found\n", a.fromNameOrEmail)
		return nil
	}

	toUser := userList.User(a.toNameOrEmail)
	if toUser == nil {
		fmt.Printf("User '%s' not found\n", a.toNameOrEmail)
		return nil
	}

	if fromUser == toUser {
		fmt.Printf("Users '%s' and '%s' are the same\n", a.fromNameOrEmail, a.toNameOrEmail)
		return nil
	}

	if !confirmAction("Are you sure? (y or n)") {
		return nil
	}

	items, _, err := backlog.AllBacklogItems(a.root)
	if err != nil {
		return err
	}
	for _, item := range items {
		assigned := userList.User(item.Assigned())
		if fromUser == assigned {
			item.SetAssigned(toUser.Nickname())
			err := item.Save()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
