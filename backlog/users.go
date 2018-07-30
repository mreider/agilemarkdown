package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type UserList struct {
	usersDir string
	users    []*User
}

func NewUserList(usersDir string) *UserList {
	userList := &UserList{usersDir: usersDir}
	if usersDir != "" {
		userList.fixObsoleteUserFiles()
		userList.init()
		userList.load()
		userList.initGitUsers()
		userList.load()
	}
	return userList
}

func (ul *UserList) User(nameOrNickOrEmail string) *User {
	nameOrNickOrEmail = strings.ToLower(utils.CollapseWhiteSpaces(nameOrNickOrEmail))
	for _, user := range ul.users {
		if strings.ToLower(user.PrimaryEmail()) == nameOrNickOrEmail {
			return user
		}
	}
	for _, user := range ul.users {
		if strings.ToLower(user.Name()) == nameOrNickOrEmail || strings.ToLower(user.Nickname()) == nameOrNickOrEmail {
			return user
		}
	}
	return nil
}

func (ul *UserList) AddUser(name, email string) bool {
	name = utils.CollapseWhiteSpaces(name)
	email = utils.CollapseWhiteSpaces(email)

	currentUser := ul.User(email)
	if currentUser != nil {
		return true
	}

	currentUser = ul.User(name)
	if currentUser != nil {
		return currentUser.AddEmailIfNotExist(email)
	}

	userFile := filepath.Join(ul.usersDir, name)
	if _, err := os.Stat(userFile); os.IsNotExist(err) {
		userFile += ".md"
	}

	user, err := LoadUser(userFile)
	if err != nil {
		return false
	}
	user.SetName(name)
	user.AddEmailIfNotExist(email)
	ul.users = append(ul.users, user)
	return true
}

func (ul *UserList) Save() error {
	for _, user := range ul.users {
		err := user.Save()
		if err != nil {
			return err
		}
	}
	return nil
}

func (ul *UserList) init() error {
	_, err := os.Stat(ul.usersDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(ul.usersDir, 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ul *UserList) initGitUsers() error {
	names, emails, err := git.KnownUsers()
	if err == nil {
		for i := range names {
			ul.AddUser(names[i], emails[i])
		}
	}
	name, email, err := git.CurrentUser()
	if err == nil {
		ul.AddUser(name, email)
	}
	return ul.Save()
}

func (ul *UserList) load() error {
	items, err := ioutil.ReadDir(ul.usersDir)
	users := make([]*User, 0, len(items))
	if err == nil {
		for _, item := range items {
			if !item.IsDir() && strings.HasSuffix(item.Name(), ".md") {
				userFile := filepath.Join(ul.usersDir, item.Name())
				user, err := LoadUser(userFile)
				if err != nil {
					return err
				}
				users = append(users, user)
			}
		}
	}
	ul.users = users
	return nil
}

func (ul *UserList) AllUsers() []string {
	users := make([]string, 0, len(ul.users))
	for _, user := range ul.users {
		userInfo := user.Name()
		if user.Name() != user.Nickname() {
			userInfo = fmt.Sprintf("%s (%s)", user.Name(), user.Nickname())
		}

		users = append(users, userInfo)
	}
	return users
}

func (ul *UserList) Users() []*User {
	return ul.users
}

func (ul *UserList) fixObsoleteUserFiles() error {
	items, err := ioutil.ReadDir(ul.usersDir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if strings.HasSuffix(item.Name(), ".md") {
			continue
		}

		itemPath := filepath.Join(ul.usersDir, item.Name())
		userPath := itemPath + ".md"
		if _, err := os.Stat(userPath); err == nil {
			os.Remove(itemPath)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		user, _ := LoadUser(itemPath + ".md")
		user.SetName(item.Name())
		content, err := ioutil.ReadFile(itemPath)
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, "@") {
				user.AddEmailIfNotExist(line)
			}
		}
		err = user.Save()
		if err != nil {
			return err
		}
		os.Remove(itemPath)
	}
	return nil
}
