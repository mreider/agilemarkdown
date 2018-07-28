package backlog

import (
	"path/filepath"
	"sort"
	"strings"
)

type BacklogItemsSorter struct {
	sortedItemsByStatus map[string][]string
}

func NewBacklogItemsSorter(overviews ...*BacklogOverview) *BacklogItemsSorter {
	s := &BacklogItemsSorter{}
	sortedItemsByStatus := make(map[string][]string)
	for _, overview := range overviews {
		s.updateSortedItems(overview, sortedItemsByStatus)
	}
	s.sortedItemsByStatus = sortedItemsByStatus
	return s
}

func (s *BacklogItemsSorter) SortItemsByStatus(status *BacklogItemStatus, items []*BacklogItem) {
	itemsNames := s.sortedItemsByStatus[status.Name]
	itemsOrder := make(map[string]int, len(itemsNames))
	for i, itemName := range itemsNames {
		itemsOrder[itemName] = i
	}
	sort.Slice(items, func(i, j int) bool {
		iIndex1, iOk := itemsOrder[items[i].Name()]
		jIndex1, jOk := itemsOrder[items[j].Name()]
		if iOk && jOk {
			return iIndex1 < jIndex1
		}
		if iOk {
			return true
		}
		if jOk {
			return false
		}
		return i < j
	})
}

func (s *BacklogItemsSorter) SortedItemsByStatus() map[string][]string {
	return s.sortedItemsByStatus
}

func (s *BacklogItemsSorter) updateSortedItems(overview *BacklogOverview, sortedItemsByStatus map[string][]string) {
	for _, status := range AllStatuses {
		title := status.CapitalizedName()
		group := overview.markdown.Group(title)
		if group == nil {
			if _, ok := sortedItemsByStatus[status.Name]; !ok {
				sortedItemsByStatus[status.Name] = nil
			}
			continue
		}
		for _, line := range group.Lines() {
			matches := overviewItemRe.FindStringSubmatch(line)
			if len(matches) > 0 {
				itemPath := matches[1]
				itemName := filepath.Base(itemPath)
				itemName = strings.TrimSuffix(itemName, filepath.Ext(itemName))
				sortedItemsByStatus[status.Name] = append(sortedItemsByStatus[status.Name], itemName)
			}
		}
	}
}

func (s *BacklogItemsSorter) SortItemsByModifiedDesc(items []*BacklogItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Modified() != items[j].Modified() {
			return items[i].Modified().After(items[j].Modified())
		}
		return items[i].Name() < items[j].Name()
	})
}
