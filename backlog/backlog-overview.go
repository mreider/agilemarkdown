package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	overviewItemRe   = regexp.MustCompile(`\[[^]]*]\(([^)]+)\)`)
	OverviewFooterRe = regexp.MustCompile(`^\[Archived stories]\([^]]+\).*`)
)

type BacklogOverview struct {
	markdown *markdown.Content
}

func LoadBacklogOverview(overviewPath string) (*BacklogOverview, error) {
	content, err := markdown.LoadMarkdown(overviewPath, nil, nil, "### ", OverviewFooterRe)
	if err != nil {
		return nil, err
	}
	return NewBacklogOverview(content), nil
}

func NewBacklogOverview(content *markdown.Content) *BacklogOverview {
	return &BacklogOverview{content}
}

func (overview *BacklogOverview) Save() error {
	if overview.markdown.IsDirty() {
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

func (overview *BacklogOverview) Update(items []*BacklogItem, sorter *BacklogItemsSorter, userList *UserList) error {
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
			group = markdown.NewGroup(overview.markdown, title, nil)
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
				overview.markdown.AddGroup(group)
			}
			rootDir := filepath.Dir(overview.markdown.ContentPath())
			newLines := BacklogView{}.WriteMarkdownItems(items, status, rootDir, filepath.Join(rootDir, TagsDirectoryName), userList)
			group.ReplaceLines(newLines)
		}
	}
	return overview.Save()
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
	overview.markdown.SortGroups(func(group1, group2 *markdown.Group, i, j int) bool {
		iStatus, jStatus := StatusByName(group1.Title()), StatusByName(group2.Title())
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

func (overview *BacklogOverview) RemoveVelocity(bck *Backlog) error {
	overview.markdown.SetFreeText([]string{""})
	return overview.Save()
}

func (overview *BacklogOverview) SetHideEmptyGroups(value bool) {
	overview.markdown.HideEmptyGroups = value
}

func (overview *BacklogOverview) UpdateLinks(lastLinkTitle, lastLinkPath, rootDir, baseDir string) error {
	links := MakeStandardLinks(rootDir, baseDir)
	if _, err := os.Stat(lastLinkPath); err == nil {
		links = append(links, utils.MakeMarkdownLink(lastLinkTitle, lastLinkPath, baseDir))
	}
	overview.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	return overview.Save()
}

func (overview *BacklogOverview) UpdateItemLinkInOverviewFile(prevItemPath, newItemPath string) error {
	data, err := ioutil.ReadFile(overview.markdown.ContentPath())
	if err != nil {
		return err
	}
	info, err := os.Stat(overview.markdown.ContentPath())
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(overview.markdown.ContentPath())
	newData := strings.Replace(string(data), fmt.Sprintf("(%s)", utils.GetMarkdownLinkPath(prevItemPath, baseDir)), fmt.Sprintf("(%s)", utils.GetMarkdownLinkPath(newItemPath, baseDir)), -1)
	err = ioutil.WriteFile(overview.markdown.ContentPath(), []byte(newData), info.Mode())
	return err
}

func (overview *BacklogOverview) Path() string {
	return overview.markdown.ContentPath()
}
