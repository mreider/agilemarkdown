package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
)

type GlobalIndex struct {
	markdown *MarkdownContent
}

func LoadGlobalIndex(indexPath string) (*GlobalIndex, error) {
	markdown, err := LoadMarkdown(indexPath, nil, "## ", nil)
	if err != nil {
		return nil, err
	}
	return NewGlobalIndex(markdown), nil
}

func NewGlobalIndex(markdown *MarkdownContent) *GlobalIndex {
	return &GlobalIndex{markdown}
}

func (index *GlobalIndex) Save() error {
	return index.markdown.Save()
}

func (index *GlobalIndex) FreeText() []string {
	return index.markdown.freeText
}

func (index *GlobalIndex) SetFreeText(freeText []string) {
	index.markdown.SetFreeText(freeText)
	index.Save()
}

func (index *GlobalIndex) UpdateBacklogs(overviews []*BacklogOverview, archives []*BacklogOverview, baseDir string) {
	lines := make([]string, 0, len(overviews)*5)
	for i := range overviews {
		lines = append(lines, "")
		lines = append(lines, utils.JoinMarkdownLinks(MakeOverviewLink(overviews[i], baseDir), MakeArchiveLink(archives[i], "archive", baseDir)))
		if i < len(overviews)-1 {
			lines = append(lines, "")
			lines = append(lines, "---")
		}
	}

	backlogsGroup := index.markdown.Group("Backlogs")
	if backlogsGroup == nil {
		backlogsGroup = &MarkdownGroup{title: "Backlogs", content: index.markdown}
		index.markdown.addGroup(backlogsGroup)
	}
	backlogsGroup.ReplaceLines(lines)
	index.Save()
}

func (index *GlobalIndex) UpdateLinks(rootDir string) {
	links := []string{
		MakeIndexLink(rootDir, filepath.Dir(index.markdown.contentPath)),
		MakeIdeasLink(rootDir, filepath.Dir(index.markdown.contentPath)),
		MakeTagsLink(rootDir, filepath.Dir(index.markdown.contentPath)),
	}
	index.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	index.Save()
}
