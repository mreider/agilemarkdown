package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
)

type DeleteUserAction struct {
	root        *backlog.BacklogsStructure
	nameOrEmail string
}

func NewDeleteUserAction(rootDir, nameOrEmail string) *DeleteUserAction {
	return &DeleteUserAction{root: backlog.NewBacklogsStructure(rootDir), nameOrEmail: nameOrEmail}
}

func (a *DeleteUserAction) Execute() error {
	userList := backlog.NewUserList(a.root.UsersDirectory())

	user := userList.User(a.nameOrEmail)
	if user == nil {
		fmt.Printf("User '%s' not found\n", a.nameOrEmail)
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
		if user == assigned {
			item.SetAssigned("")
			err := item.Save()
			if err != nil {
				return err
			}
		}
	}

	if !userList.DeleteUser(a.nameOrEmail) {
		fmt.Printf("Can't delete the user '%s'\n", a.nameOrEmail)
		return nil
	}
	return nil
}
