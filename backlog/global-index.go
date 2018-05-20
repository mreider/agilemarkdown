package backlog

import "github.com/mreider/agilemarkdown/utils"

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
		lines = append(lines, MakeOverviewLink(overviews[i], baseDir))
		lines = append(lines, "")
		lines = append(lines, MakeArchiveLink(archives[i], "archive", baseDir))
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

func (index *GlobalIndex) UpdateIdeas(ideasPath, baseDir string) {
	lines := make([]string, 0, 2)
	lines = append(lines, "")
	lines = append(lines, utils.MakeMarkdownLink("idea board", ideasPath, baseDir))

	ideasGroup := index.markdown.Group("Idea board")
	if ideasGroup == nil {
		ideasGroup = &MarkdownGroup{title: "Idea board", content: index.markdown}
		index.markdown.addGroup(ideasGroup)
	}
	ideasGroup.ReplaceLines(lines)
	index.Save()
}

func (index *GlobalIndex) UpdateTags(tagsPath, baseDir string) {
	lines := make([]string, 0, 2)
	lines = append(lines, "")
	lines = append(lines, utils.MakeMarkdownLink("tags", tagsPath, baseDir))

	tagsGroup := index.markdown.Group("Tags")
	if tagsGroup == nil {
		tagsGroup = &MarkdownGroup{title: "Tags", content: index.markdown}
		index.markdown.addGroup(tagsGroup)
	}
	tagsGroup.ReplaceLines(lines)
	index.Save()
}
