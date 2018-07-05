package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	overviewItemRe   = regexp.MustCompile(`\[[^]]*]\(([^)]+)\)`)
	OverviewFooterRe = regexp.MustCompile(`^\[Archived stories]\([^]]+\).*`)
)

type BacklogOverview struct {
	markdown *MarkdownContent
}

func LoadBacklogOverview(overviewPath string) (*BacklogOverview, error) {
	markdown, err := LoadMarkdown(overviewPath, []string{CreatedMetadataKey, ModifiedMetadataKey}, "### ", OverviewFooterRe)
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

func (overview *BacklogOverview) Content() []byte {
	return overview.markdown.Content()
}

func (overview *BacklogOverview) Title() string {
	return overview.markdown.Title()
}

func (overview *BacklogOverview) SetTitle(title string) {
	overview.markdown.SetTitle(title)
}

func (overview *BacklogOverview) SetCreated(timestamp string) {
	overview.markdown.SetMetadataValue(CreatedMetadataKey, timestamp)
}

func (overview *BacklogOverview) Update(items []*BacklogItem, sorter *BacklogItemsSorter) {
	itemsByStatus := sorter.SortedItemsByStatus()
	itemsByName := make(map[string]*BacklogItem)
	for _, item := range items {
		itemsByName[item.Name()] = item
		overview.updateItem(item, itemsByStatus)
	}
	for _, status := range AllStatuses {
		title := status.CapitalizedName()
		statusItems := itemsByStatus[status.Name]
		group := overview.markdown.Group(title)
		isNewGroup := false
		if group == nil {
			group = &MarkdownGroup{content: overview.markdown, title: title}
			isNewGroup = true
		}
		items := make([]*BacklogItem, 0, len(statusItems))
		for _, itemName := range statusItems {
			item := itemsByName[itemName]
			if item != nil {
				items = append(items, item)
			}
		}
		if len(items) > 0 || !overview.markdown.HideEmptyGroups || !isNewGroup {
			if isNewGroup {
				overview.markdown.addGroup(group)
			}
			rootDir := filepath.Dir(overview.markdown.contentPath)
			newLines := BacklogView{}.WriteMarkdownItems(items, rootDir, filepath.Join(rootDir, TagsDirectoryName))
			group.ReplaceLines(newLines)
		}
	}
	overview.Save()
}

func (overview *BacklogOverview) updateItem(item *BacklogItem, itemsByStatus map[string][]string) {
	itemStatus := strings.ToLower(item.Status())
NextStatus:
	for _, status := range AllStatuses {
		items := itemsByStatus[status.Name]
		for i, it := range items {
			if it == item.Name() {
				if status.Name == itemStatus {
					continue NextStatus
				} else {
					copy(items[i:], items[i+1:])
					itemsByStatus[status.Name] = items[:len(items)-1]
					continue NextStatus
				}
			}
		}
		if status.Name == itemStatus {
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

func (overview *BacklogOverview) SendNewComments(items []*BacklogItem, onSend func(item *BacklogItem, to []string, comment []string) (me string, err error)) {
	for _, item := range items {
		comments := item.Comments()
		hasChanges := false
		for _, comment := range comments {
			if comment.Closed || comment.Unsent {
				continue
			}
			me, err := onSend(item, comment.Users, comment.Text)
			now := utils.GetCurrentTimestamp()
			hasChanges = true
			if err != nil {
				comment.AddLine(fmt.Sprintf("can't send by @%s at %s: %v", me, now, err))
			} else {
				comment.AddLine(fmt.Sprintf("sent by @%s at %s", me, now))
			}
		}
		if hasChanges {
			item.UpdateComments(comments)
		}
	}
}

func (overview *BacklogOverview) RemoveVelocity(bck *Backlog) {
	overview.markdown.SetFreeText([]string{""})
	overview.Save()
}

func (overview *BacklogOverview) SetHideEmptyGroups(value bool) {
	overview.markdown.HideEmptyGroups = value
}

func (overview *BacklogOverview) UpdateLinks(lastLinkTitle, lastLinkPath, rootDir, baseDir string) {
	links := MakeStandardLinks(rootDir, baseDir)
	if _, err := os.Stat(lastLinkPath); err == nil {
		links = append(links, utils.MakeMarkdownLink(lastLinkTitle, lastLinkPath, baseDir))
	}
	overview.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	overview.Save()
}
