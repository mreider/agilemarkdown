package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ArchiveFileName = "archive.md"

// ItemsTags walks every backlog and returns: the set of tags in use, the
// items carrying each tag, and a map from item to its backlog overview.
// Tags are lowercased.
func ItemsTags(root *BacklogsStructure) (allTags map[string]struct{}, itemsTags map[string][]*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	backlogDirs, err := root.BacklogDirs()
	if err != nil {
		return nil, nil, nil, err
	}

	allTags = make(map[string]struct{})
	itemsTags = make(map[string][]*BacklogItem)
	itemsOverviews = make(map[*BacklogItem]*BacklogOverview)

	for _, backlogDir := range backlogDirs {
		overviewPath, ok := FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return nil, nil, nil, fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := LoadBacklogOverview(overviewPath)
		if err != nil {
			return nil, nil, nil, err
		}

		bck, err := LoadBacklog(backlogDir)
		if err != nil {
			return nil, nil, nil, err
		}

		items := bck.ActiveItems()
		for _, item := range items {
			for _, tag := range item.Tags() {
				tag = strings.ToLower(tag)
				allTags[tag] = struct{}{}
				itemsTags[tag] = append(itemsTags[tag], item)
				itemsOverviews[item] = overview
			}
		}
	}

	return allTags, itemsTags, itemsOverviews, nil
}

func ActiveBacklogItems(root *BacklogsStructure) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	return getBacklogItems(root, func(backlog *Backlog) []*BacklogItem {
		return backlog.ActiveItems()
	})
}

func AllBacklogItems(root *BacklogsStructure) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	return getBacklogItems(root, func(backlog *Backlog) []*BacklogItem {
		return backlog.AllItems()
	})
}

func getBacklogItems(root *BacklogsStructure, getItems func(backlog *Backlog) []*BacklogItem) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	backlogDirs, err := root.BacklogDirs()
	if err != nil {
		return nil, nil, err
	}

	itemsOverviews = make(map[*BacklogItem]*BacklogOverview)

	for _, backlogDir := range backlogDirs {
		overviewPath, ok := FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return nil, nil, fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := LoadBacklogOverview(overviewPath)
		if err != nil {
			return nil, nil, err
		}

		bck, err := LoadBacklog(backlogDir)
		if err != nil {
			return nil, nil, err
		}

		bckItems := getItems(bck)
		for _, item := range bckItems {
			items = append(items, item)
			itemsOverviews[item] = overview
		}
	}

	return items, itemsOverviews, nil
}

func FindOverviewFileInRootDirectory(backlogDir string) (string, bool) {
	backlogDir, _ = filepath.Abs(backlogDir)
	rootDir := filepath.Dir(backlogDir)
	overviewName := filepath.Base(backlogDir)
	if IsForbiddenBacklogName(overviewName) {
		return "", false
	}
	overviewFileName := fmt.Sprintf("%s.md", overviewName)

	infos, err := os.ReadDir(rootDir)
	if err != nil {
		return "", false
	}
	for _, info := range infos {
		if info.Name() == overviewFileName {
			return filepath.Join(rootDir, info.Name()), true
		}
	}
	return "", false
}

func FindArchiveFileInDirectory(dir string) (string, bool) {
	dir, _ = filepath.Abs(dir)
	infos, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	for _, info := range infos {
		if info.Name() == ArchiveFileName {
			return filepath.Join(dir, info.Name()), true
		}
	}
	return filepath.Join(dir, ArchiveFileName), false
}
