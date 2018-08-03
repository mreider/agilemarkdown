package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"strings"
)

type SyncDownCaseStep struct {
	root     *backlog.BacklogsStructure
	userList *backlog.UserList
}

func NewSyncDownCaseStep(root *backlog.BacklogsStructure, userList *backlog.UserList) *SyncDownCaseStep {
	return &SyncDownCaseStep{root: root, userList: userList}
}

func (s *SyncDownCaseStep) Execute() error {
	fmt.Println("Downcasing story and idea metadata")

	err := s.updateItems()
	if err != nil {
		return err
	}

	return s.updateIdeas()
}

func (s *SyncDownCaseStep) updateItems() error {
	backlogDirs, err := s.root.BacklogDirs()
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}
		for _, item := range bck.AllItems() {
			statusName := item.Status()
			status := backlog.StatusByName(statusName)
			if status != nil {
				item.SetStatus(status)
			}

			tags := item.Tags()
			if len(tags) > 0 {
				for i, tag := range tags {
					tags[i] = strings.ToLower(tag)
				}
				item.SetTags(tags)
			}

			assigned := item.Assigned()
			if assigned != "" {
				user := s.userList.User(assigned)
				if user != nil {
					if strings.ToLower(user.Nickname()) == strings.ToLower(assigned) {
						item.SetAssigned(user.Nickname())
					} else if strings.ToLower(user.Name()) == strings.ToLower(assigned) {
						item.SetAssigned(user.Name())
					} else {
						item.SetAssigned(strings.ToLower(assigned))
					}
				}
			}

			err := item.Save()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SyncDownCaseStep) updateIdeas() error {
	ideasDir := s.root.IdeasDirectory()
	ideas, err := backlog.LoadIdeas(ideasDir)
	if err != nil {
		return err
	}

	for _, idea := range ideas {
		tags := idea.Tags()
		if len(tags) > 0 {
			for i, tag := range tags {
				tags[i] = strings.ToLower(tag)
			}
			idea.SetTags(tags)
		}
		err := idea.Save()
		if err != nil {
			return err
		}
	}
	return nil
}
