package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"path/filepath"
	"strings"
)

type CreateUserAction struct {
	rootDir string
	name    string
	email   string
	parts   []string
}

func NewCreateUserAction(rootDir, name, email string, parts []string) *CreateUserAction {
	return &CreateUserAction{rootDir: rootDir, name: name, email: email, parts: parts}
}

func (a *CreateUserAction) Execute() error {
	name, email, parts := a.email, a.name, a.parts

	if len(parts) == 0 && name == "" {
		return nil
	}

	if a.email == "" && len(a.parts) > 0 && strings.Contains(a.parts[len(parts)-1], "@") {
		email = parts[len(parts)-1]
		parts = parts[:len(parts)-1]
	}

	if name == "" && len(parts) > 0 {
		name = strings.Join(parts, " ")
	}

	if name == "" || email == "" {
		fmt.Println("Both name and email should be specified")
		return nil
	}

	userList := backlog.NewUserList(filepath.Join(a.rootDir, backlog.UsersDirectoryName))

	existingUser := userList.User(email)
	if existingUser != nil {
		if existingUser.Name() != name {
			fmt.Printf("User '%s' with email %s already exists\n", existingUser.Name(), email)
		} else {
			fmt.Printf("User '%s' already exists\n", name)
		}
		return nil
	}

	existingUser = userList.User(name)
	if existingUser != nil {
		fmt.Printf("User '%s' already exists\n", name)
		return nil
	}

	if userList.AddUser(name, email) {
		return userList.Save()
	} else {
		fmt.Printf("Can't add the user '%s'\n", name)
		return nil
	}
	return nil

}
