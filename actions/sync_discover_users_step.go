package actions

import (
	"fmt"
	"strings"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
)

// SyncDiscoverUsersStep auto-creates user files for everyone git knows about:
// the running user (from `git config`), every committer in `git log`, and
// optionally the GitHub repo collaborators. This makes `am create-user` an
// optional override rather than a required onboarding step.
type SyncDiscoverUsersStep struct {
	root     *backlog.BacklogsStructure
	userList *backlog.UserList
}

func NewSyncDiscoverUsersStep(root *backlog.BacklogsStructure, userList *backlog.UserList) *SyncDiscoverUsersStep {
	return &SyncDiscoverUsersStep{root: root, userList: userList}
}

func (s *SyncDiscoverUsersStep) Execute() error {
	added := 0

	// Layer 1: running user from git config. Works even on empty repos.
	if name, email, err := git.CurrentUser(); err == nil && (name != "" || email != "") {
		if s.addIfNew(name, email) {
			added++
		}
	}

	// Layer 2: every committer ever. Works as soon as anyone has committed.
	if names, emails, err := git.KnownUsers(); err == nil {
		for i := range names {
			n := names[i]
			var e string
			if i < len(emails) {
				e = emails[i]
			}
			if s.addIfNew(n, e) {
				added++
			}
		}
	}

	if added > 0 {
		if err := s.userList.Save(); err != nil {
			return err
		}
		fmt.Printf("Discovered %d user(s) from git\n", added)
	}
	return nil
}

func (s *SyncDiscoverUsersStep) addIfNew(name, email string) bool {
	name = utils.CollapseWhiteSpaces(name)
	email = strings.TrimSpace(email)
	if name == "" && email == "" {
		return false
	}
	// AddUser is idempotent on existing names and emails: it returns true
	// if it created or extended a user, false otherwise. Detect creation
	// by checking before/after.
	existed := s.userList.User(email) != nil || s.userList.User(name) != nil
	if !s.userList.AddUser(name, email) {
		return false
	}
	return !existed
}
