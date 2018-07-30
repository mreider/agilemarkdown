package backlog

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

func BacklogDirs(rootDir string) ([]string, error) {
	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") || IsForbiddenBacklogName(info.Name()) {
			continue
		}
		result = append(result, filepath.Join(rootDir, info.Name()))
	}
	sort.Strings(result)
	return result, nil
}

func ItemsAndIdeasTags(rootDir string) (allTags map[string]struct{}, itemsTags map[string][]*BacklogItem, ideasTags map[string][]*BacklogIdea, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	backlogDirs, err := BacklogDirs(rootDir)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ideasDir := filepath.Join(rootDir, IdeasDirectoryName)
	ideas, err := LoadIdeas(ideasDir)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	allTags = make(map[string]struct{})
	itemsTags = make(map[string][]*BacklogItem)
	ideasTags = make(map[string][]*BacklogIdea)
	itemsOverviews = make(map[*BacklogItem]*BacklogOverview)

	for _, backlogDir := range backlogDirs {
		overviewPath, ok := FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return nil, nil, nil, nil, fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := LoadBacklogOverview(overviewPath)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		bck, err := LoadBacklog(backlogDir)
		if err != nil {
			return nil, nil, nil, nil, err
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

	for _, idea := range ideas {
		for _, tag := range idea.Tags() {
			tag = strings.ToLower(tag)
			allTags[tag] = struct{}{}
			ideasTags[tag] = append(ideasTags[tag], idea)
		}
	}

	return allTags, itemsTags, ideasTags, itemsOverviews, nil
}

func ActiveBacklogItems(rootDir string) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	return getBacklogItems(rootDir, func(backlog *Backlog) []*BacklogItem {
		return backlog.ActiveItems()
	})
}

func AllBacklogItems(rootDir string) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	return getBacklogItems(rootDir, func(backlog *Backlog) []*BacklogItem {
		return backlog.AllItems()
	})
}

func getBacklogItems(rootDir string, getItems func(backlog *Backlog) []*BacklogItem) (items []*BacklogItem, itemsOverviews map[*BacklogItem]*BacklogOverview, err error) {
	backlogDirs, err := BacklogDirs(rootDir)
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

	infos, err := ioutil.ReadDir(rootDir)
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
