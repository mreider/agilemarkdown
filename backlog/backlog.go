package backlog

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	OverviewFileName = "0-overview.md"
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
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") && info.Name() != OverviewFileName {
			item, err := CreateBacklogItem(filepath.Join(backlogDir, info.Name()))
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}
	return &Backlog{items: items}, nil
}

func (bck *Backlog) ItemsByStatus(status string) []*BacklogItem {
	result := make([]*BacklogItem, 0, 10)
	for _, item := range bck.items {
		if item.Status() == status || status == "all" {
			result = append(result, item)
		}
	}
	return result
}
