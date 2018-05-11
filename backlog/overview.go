package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	BacklogOverviewTitleMetadataKey = "Title"
	ExcerptMaxSize                  = 100
	ClarificationsTitle             = "Clarifications"
)

var (
	overviewItemRe   = regexp.MustCompile(`^.* \[.*]\(([^)]+)\).*$`)
	chartColorCodeRe = regexp.MustCompile(`.\[\d+m`)
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
		if overview.markdown.groups[i].title == ClarificationsTitle {
			return true
		}
		if overview.markdown.groups[j].title == ClarificationsTitle {
			return false
		}

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

func (overview *BacklogOverview) UpdateClarifications(items []*BacklogItem) {
	group := overview.markdown.Group(ClarificationsTitle)
	if group == nil {
		group = &MarkdownGroup{content: overview.markdown, title: ClarificationsTitle}
		overview.markdown.addGroup(group)
	}

	lines := []string{" User | Excerpt | Story ", "---|---|---"}
	for _, item := range items {
		for _, comment := range item.Comments() {
			if comment.Closed {
				continue
			}
			var text []string
			var textSize int
			for _, textLine := range comment.Text {
				if textSize+len(textLine) <= ExcerptMaxSize {
					text = append(text, textLine)
					textSize += len(textLine)
				} else {
					text = append(text, textLine[:ExcerptMaxSize-textSize]+"...")
					textSize += ExcerptMaxSize
				}
				if textSize >= ExcerptMaxSize {
					break
				}
			}

			lines = append(lines, fmt.Sprintf(" %s | %s | [%s](%s) ", comment.User, strings.Join(text, " "), item.Title(), item.Name()))
		}
	}
	group.ReplaceLines(lines)
	overview.Save()
}

func (overview *BacklogOverview) UpdateProgress(bck *Backlog) error {
	chart, err := BacklogView{}.Progress(bck, 12, 60)
	if err != nil {
		return err
	}

	chartStart, chartEnd := -1, -1
	for i, line := range overview.markdown.freeText {
		line = strings.TrimSpace(line)
		if line == "```" {
			if chartStart == -1 {
				chartStart = i
			} else {
				chartEnd = i
				break
			}
		}
	}

	chart = chartColorCodeRe.ReplaceAllString(chart, "")
	chartLines := utils.WrapLinesToMarkdownCodeBlock(strings.Split(chart, "\n"))
	var newFreeText []string
	if chartStart >= 0 && chartEnd >= 0 {
		newFreeText = append(newFreeText, overview.markdown.freeText[:chartStart]...)
		newFreeText = append(newFreeText, chartLines...)
		newFreeText = append(newFreeText, overview.markdown.freeText[chartEnd+1:]...)
	} else {
		newFreeText = make([]string, 0, len(overview.markdown.freeText)+len(chartLines))
		newFreeText = append(newFreeText, overview.markdown.freeText...)
		newFreeText = append(newFreeText, chartLines...)
	}

	overview.markdown.SetFreeText(newFreeText)
	overview.Save()
	return nil
}
