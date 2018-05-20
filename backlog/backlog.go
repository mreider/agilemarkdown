package backlog

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	IdeasDirectoryName    = "ideas"
	ArchiveDirectoryName  = "archive"
	TagsDirectoryName     = "tags"
	TagsPageFileName      = "tags.md"
	ForbiddenBacklogNames = []string{IdeasDirectoryName, ArchiveDirectoryName, TagsDirectoryName}
	ForbiddenItemNames    = []string{ArchiveDirectoryName}
)

type Backlog struct {
	items []*BacklogItem
}

func LoadBacklog(backlogDir string) (*Backlog, error) {
	var items []*BacklogItem
	activeItems, err := loadItems(backlogDir)
	if err != nil {
		return nil, err
	}
	items = append(items, activeItems...)

	archivedItems, err := loadItems(filepath.Join(backlogDir, ArchiveDirectoryName))
	if err != nil {
		return nil, err
	}
	items = append(items, archivedItems...)

	return &Backlog{items: items}, nil
}

func loadItems(dir string) ([]*BacklogItem, error) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var items []*BacklogItem
	for _, info := range infos {
		baseName := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") && !IsForbiddenItemName(baseName) {
			item, err := LoadBacklogItem(filepath.Join(dir, info.Name()))
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (bck *Backlog) AllItems() []*BacklogItem {
	return bck.items
}

func (bck *Backlog) ActiveItems() []*BacklogItem {
	filter := &BacklogItemsActiveFilter{}
	return bck.FilteredItems(filter)
}

func (bck *Backlog) ArchivedItems() []*BacklogItem {
	filter := &BacklogItemsArchivedFilter{}
	return bck.FilteredItems(filter)
}

func (bck *Backlog) AllItemsByStatus(statusCode string) []*BacklogItem {
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

func IsForbiddenBacklogName(backlogName string) bool {
	backlogName = strings.ToLower(backlogName)
	for _, name := range ForbiddenBacklogNames {
		if strings.ToLower(name) == backlogName {
			return true
		}
	}
	return false
}

func IsForbiddenItemName(itemName string) bool {
	itemName = strings.ToLower(itemName)
	for _, name := range ForbiddenItemNames {
		if strings.ToLower(name) == itemName {
			return true
		}
	}
	return false
}
