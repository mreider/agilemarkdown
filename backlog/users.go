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
		userList.init()
		userList.fixObsoleteUserFiles()
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

	var currentUser *User
	if email != "" {
		currentUser = ul.User(email)
		if currentUser != nil {
			return true
		}
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

func (ul *UserList) DeleteUser(nameOrEmailOrNick string) bool {
	user := ul.User(nameOrEmailOrNick)
	if user == nil {
		return true
	}
	err := os.Remove(user.Path())
	if err != nil {
		fmt.Println(err)
		return false
	}
	users := make([]*User, 0, len(ul.users))
	for _, u := range ul.users {
		if u != user {
			users = append(users, u)
		}
	}
	ul.users = users
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

func (ul *UserList) ResolveGitUsers(unknownUsers []string) (unresolvedUsers []string) {
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
		if ul.User(user) != nil {
			continue
		}

		normalizedUser := utils.CollapseWhiteSpaces(strings.ToLower(user))
		for i, email := range normalizedEmails {
			if email == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ul.AddUser(names[i], emails[i]) {
					ul.Save()
				}
				continue NextUser
			}
		}

		for i, email := range normalizedEmails {
			nickname := strings.SplitN(email, "@", 2)[0]
			if nickname == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ul.AddUser(names[i], emails[i]) {
					ul.Save()
				}
				continue NextUser
			}
		}

		for i, name := range normalizedNames {
			if name == normalizedUser {
				fmt.Printf("User '%s' is associated with a git user '%s <%s>'\n", user, names[i], emails[i])
				if ul.AddUser(names[i], emails[i]) {
					ul.Save()
				}
				continue NextUser
			}
		}

		unresolvedUsers = append(unresolvedUsers, user)
	}

	return unresolvedUsers
}
