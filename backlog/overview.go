package backlog

import (
	"math"
	"regexp"
	"sort"
)

const (
	BacklogOverviewTitleMetadataKey = "Title"
)

var (
	overviewItemRe = regexp.MustCompile(`^.* \[.*]\(([^)]+)\).*$`)
)

type BacklogOverview struct {
	markdown *MarkdownContent
}

func LoadBacklogOverview(overviewPath string) (*BacklogOverview, error) {
	markdown, err := LoadMarkdown(overviewPath, []string{BacklogOverviewTitleMetadataKey, CreatedMetadataKey, ModifiedMetadataKey}, true)
	if err != nil {
		return nil, err
	}
	return NewBacklogOverview(markdown), nil
}

func NewBacklogOverview(markdown *MarkdownContent) *BacklogOverview {
	return &BacklogOverview{markdown}
}

func (overview *BacklogOverview) Save() error {
	if overview.markdown.isDirty {
		overview.sortGroupsByStatus()
	}
	return overview.markdown.Save()
}

func (overview *BacklogOverview) Content(timestamp string) []byte {
	return overview.markdown.Content(timestamp)
}

func (overview *BacklogOverview) Title() string {
	return overview.markdown.MetadataValue(BacklogOverviewTitleMetadataKey)
}

func (overview *BacklogOverview) SetTitle(title string) {
	overview.markdown.SetMetadataValue(BacklogOverviewTitleMetadataKey, title)
}

func (overview *BacklogOverview) SetCreated() {
	overview.markdown.SetMetadataValue(CreatedMetadataKey, "")
}

func (overview *BacklogOverview) SortedItemsByStatus() map[string][]string {
	result := make(map[string][]string)
	for _, status := range AllStatuses {
		title := status.CapitalizedName()
		group := overview.markdown.Group(title)
		if group == nil {
			result[status.Name] = nil
			continue
		}
		for _, line := range group.lines {
			matches := overviewItemRe.FindStringSubmatch(line)
			if len(matches) > 0 {
				result[status.Name] = append(result[status.Name], matches[1])
			}
		}
	}
	return result
}

func (overview *BacklogOverview) SortItems(status *BacklogItemStatus, items []*BacklogItem) {
	itemsNames := overview.SortedItemsByStatus()[status.Name]
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

func (overview *BacklogOverview) Update(items []*BacklogItem) {
	itemsByStatus := overview.SortedItemsByStatus()
	itemsByName := make(map[string]*BacklogItem)
	for _, item := range items {
		itemsByName[item.Name()] = item
		overview.updateItem(item, itemsByStatus)
	}
	for _, status := range AllStatuses {
		title := status.CapitalizedName()
		statusItems := itemsByStatus[status.Name]
		group := overview.markdown.Group(title)
		if group == nil {
			group = &MarkdownGroup{content: overview.markdown, title: title}
			overview.markdown.addGroup(group)
		}
		items := make([]*BacklogItem, 0, len(statusItems))
		for _, itemName := range statusItems {
			item := itemsByName[itemName]
			if item != nil {
				items = append(items, item)
			}
		}
		newLines := BacklogView{}.WriteMarkdownTable(items)
		group.ReplaceLines(newLines)
	}
	overview.Save()
}

func (overview *BacklogOverview) updateItem(item *BacklogItem, itemsByStatus map[string][]string) {
NextStatus:
	for _, status := range AllStatuses {
		items := itemsByStatus[status.Name]
		for i, it := range items {
			if it == item.Name() {
				if status.Name == item.Status() {
					continue NextStatus
				} else {
					copy(items[i:], items[i+1:])
					itemsByStatus[status.Name] = items[:len(items)-1]
					continue NextStatus
				}
			}
		}
		if status.Name == item.Status() {
			itemsByStatus[status.Name] = append(itemsByStatus[status.Name], item.Name())
		}
	}
}

func (overview *BacklogOverview) sortGroupsByStatus() {
	sort.Slice(overview.markdown.groups, func(i, j int) bool {
		iStatus, jStatus := StatusByName(overview.markdown.groups[i].title), StatusByName(overview.markdown.groups[j].title)
		iIndex, jIndex := int(math.MaxInt32), int(math.MaxInt32)
		if iStatus != nil {
			iIndex = StatusIndex(iStatus)
		}
		if jStatus != nil {
			jIndex = StatusIndex(jStatus)
		}

		if iIndex != jIndex {
			return iIndex < jIndex
		}

		return i < j
	})
}
