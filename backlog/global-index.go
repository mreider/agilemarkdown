package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"regexp"
)

type GlobalIndex struct {
	markdown *markdown.Content
}

func LoadGlobalIndex(indexPath string) (*GlobalIndex, error) {
	content, err := markdown.LoadMarkdown(indexPath, nil, nil, "## ", regexp.MustCompile(`^\|.*`))
	if err != nil {
		return nil, err
	}
	return NewGlobalIndex(content), nil
}

func NewGlobalIndex(content *markdown.Content) *GlobalIndex {
	return &GlobalIndex{content}
}

func (index *GlobalIndex) Save() error {
	return index.markdown.Save()
}

func (index *GlobalIndex) FreeText() []string {
	return index.markdown.FreeText()
}

func (index *GlobalIndex) SetFreeText(freeText []string) {
	index.markdown.SetFreeText(freeText)
	index.Save()
}

func (index *GlobalIndex) SetFooter(footer []string) {
	index.markdown.SetFooter(footer)
	index.Save()
}

func (index *GlobalIndex) UpdateBacklogs(overviews []*BacklogOverview, archives []*BacklogOverview, baseDir string) {
	lines := make([]string, 0, len(overviews)*5)
	lines = append(lines, "| Backlogs |  |")
	lines = append(lines, "|---|---|")
	for i := range overviews {
		line := fmt.Sprintf("| %s | %s |", MakeOverviewLink(overviews[i], baseDir), MakeArchiveLink(archives[i], "archive", baseDir))
		lines = append(lines, line)
	}

	index.markdown.RemoveGroup("Backlogs")
	index.SetFooter(lines)
	index.Save()
}

func (index *GlobalIndex) UpdateLinks(rootDir string) {
	links := MakeStandardLinks(rootDir, filepath.Dir(index.markdown.ContentPath()))
	index.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	index.Save()
}
