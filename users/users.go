package users

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

type User struct {
	userFile string
	name     string
	email    string
}

func (u *User) Name() string {
	return u.name
}

func (u *User) Email() string {
	return u.email
}

func (u *User) Nick() string {
	parts := strings.SplitN(u.email, "@", 2)
	return parts[0]
}

func NewUserList(usersDir string) *UserList {
	userList := &UserList{usersDir: usersDir}
	userList.init()
	userList.load()
	return userList
}

func (ul *UserList) User(nameOrNick string) *User {
	nameOrNick = strings.ToLower(utils.CollapseWhiteSpaces(nameOrNick))
	for _, user := range ul.users {
		if strings.ToLower(user.Name()) == nameOrNick || strings.ToLower(user.Nick()) == "" {
			return user
		}
	}
	return nil
}

func (ul *UserList) AddUser(name, email string) bool {
	name = utils.CollapseWhiteSpaces(name)
	email = utils.CollapseWhiteSpaces(email)

	currentUser := ul.User(name)
	if currentUser != nil {
		if currentUser.Email() == email {
			return false
		}
		currentUser.email = email
		return true
	}

	user := &User{name: name, email: email}
	ul.users = append(ul.users, user)
	return true
}

func (ul *UserList) Save() error {
	for _, user := range ul.users {
		userFile := user.userFile
		if userFile == "" {
			userFile = filepath.Join(ul.usersDir, user.name)
		}
		err := ioutil.WriteFile(userFile, []byte(user.Email()), 0644)
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
			if !item.IsDir() {
				userFile := filepath.Join(ul.usersDir, item.Name())
				userName := utils.CollapseWhiteSpaces(item.Name())
				userEmail := ""
				content, err := ioutil.ReadFile(userFile)
				if err != nil {
					return err
				}
				userEmail = utils.CollapseWhiteSpaces(string(content))
				user := &User{userFile: userFile, name: userName, email: userEmail}
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
		if user.Name() != user.Nick() {
			userInfo = fmt.Sprintf("%s (%s)", user.Name(), user.Nick())
		}

		users = append(users, userInfo)
	}
	return users
}
