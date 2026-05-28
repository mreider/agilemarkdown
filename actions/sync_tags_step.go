package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SyncTagsStep struct {
	root     *backlog.BacklogsStructure
	userList *backlog.UserList
}

func NewSyncTagsStep(root *backlog.BacklogsStructure, userList *backlog.UserList) *SyncTagsStep {
	return &SyncTagsStep{root: root, userList: userList}
}

func (s *SyncTagsStep) Execute() error {
	fmt.Println("Generating tag pages")

	tagsDir := s.root.TagsDirectory()
	err := os.MkdirAll(tagsDir, 0777)
	if err != nil {
		return err
	}

	allTags, itemsTags, overviews, err := backlog.ItemsTags(s.root)
	if err != nil {
		return err
	}

	tagsFileNames := make(map[string]bool)
	for tag := range allTags {
		tagItems := itemsTags[tag]
		tagFileName, err := s.updateTagPage(tagsDir, tag, tagItems, overviews, s.userList)
		if err != nil {
			return err
		}
		tagsFileNames[tagFileName] = true
	}

	infos, _ := os.ReadDir(tagsDir)
	for _, info := range infos {
		if _, ok := tagsFileNames[info.Name()]; !ok {
			_ = os.Remove(filepath.Join(tagsDir, info.Name()))
		}
	}

	return s.updateTagsPage(tagsDir, itemsTags)
}

func (s *SyncTagsStep) updateTagPage(tagsDir, tag string, items []*backlog.BacklogItem, overviews map[*backlog.BacklogItem]*backlog.BacklogOverview, userList *backlog.UserList) (string, error) {
	itemsByStatus := make(map[string][]*backlog.BacklogItem)
	for _, item := range items {
		itemStatus := strings.ToLower(item.Status())
		itemsByStatus[itemStatus] = append(itemsByStatus[itemStatus], item)
	}
	for _, statusItems := range itemsByStatus {
		sorter := backlog.NewBacklogItemsSorter()
		sorter.SortItemsByModifiedDesc(statusItems)
	}

	lines := []string{
		fmt.Sprintf("# Tag: %s", tag),
		"",
		utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), tagsDir)...),
		"",
	}
	for _, status := range backlog.AllStatuses {
		statusItems := itemsByStatus[strings.ToLower(status.Name)]
		if len(statusItems) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("## %s", status.CapitalizedName()))
		itemsLines := backlog.BacklogView{}.WriteMarkdownItemsWithProject(overviews, statusItems, status, tagsDir, tagsDir, userList)
		lines = append(lines, itemsLines...)
		lines = append(lines, "")
	}
	tagFileName := fmt.Sprintf("%s.md", utils.GetValidFileName(tag))
	err := os.WriteFile(filepath.Join(tagsDir, tagFileName), []byte(strings.Join(lines, "\n")), 0644)
	return tagFileName, err
}

func (s *SyncTagsStep) updateTagsPage(tagsDir string, itemsTags map[string][]*backlog.BacklogItem) error {
	allTags := make([]string, 0, len(itemsTags))
	for tag := range itemsTags {
		allTags = append(allTags, tag)
	}
	sort.Strings(allTags)

	lines := []string{"# Tags", ""}
	lines = append(lines, utils.JoinMarkdownLinks(backlog.MakeStandardLinks(s.root.Root(), s.root.Root())...))
	lines = append(lines, "", "---", "")
	for _, tag := range allTags {
		lines = append(lines, backlog.MakeTagLink(tag, tagsDir, s.root.Root()))
	}
	return os.WriteFile(s.root.TagsFile(), []byte(strings.Join(lines, "  \n")), 0644)
}
