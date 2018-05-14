package backlog

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Backlog struct {
	items []*BacklogItem
}

func LoadBacklog(backlogDir string) (*Backlog, error) {
	infos, err := ioutil.ReadDir(backlogDir)
	if err != nil {
		return nil, err
	}
	var items []*BacklogItem
	for _, info := range infos {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			item, err := LoadBacklogItem(filepath.Join(backlogDir, info.Name()))
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return &Backlog{items: items}, nil
}

func (bck *Backlog) Items() []*BacklogItem {
	return bck.items
}

func (bck *Backlog) ItemsByStatus(statusCode string) []*BacklogItem {
	filter := NewBacklogItemsStatusCodeFilter(statusCode)
	return bck.FilteredItems(filter)
}

func (bck *Backlog) KnownUsers() []string {
	users := make(map[string]bool)
	for _, item := range bck.items {
		if item.Assigned() != "" {
			users[item.Assigned()] = true
		}
		if item.Author() != "" {
			users[item.Author()] = true
		}
	}
	result := make([]string, 0, len(users))
	for user := range users {
		result = append(result, user)
	}
	return result
}

func (bck *Backlog) FilteredItems(filter BacklogItemsFilter) []*BacklogItem {
	result := make([]*BacklogItem, 0)
	for _, item := range bck.items {
		if filter.Match(item) {
			result = append(result, item)
		}
	}
	return result
}
