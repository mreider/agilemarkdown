package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"strings"
)

type Velocity struct {
	markdown *MarkdownContent
}

func LoadGlobalVelocity(velocityPath string) (*Velocity, error) {
	markdown, err := LoadMarkdown(velocityPath, nil, "", nil)
	if err != nil {
		return nil, err
	}
	return NewGlobalVelocity(markdown), nil
}

func NewGlobalVelocity(markdown *MarkdownContent) *Velocity {
	return &Velocity{markdown}
}

func (velocity *Velocity) Save() error {
	return velocity.markdown.Save()
}

func (velocity *Velocity) Title() string {
	return velocity.markdown.Title()
}

func (velocity *Velocity) SetTitle(title string) {
	velocity.markdown.SetTitle(title)
	velocity.Save()
}

func (velocity *Velocity) Update(backlogs []*Backlog, overviews []*BacklogOverview, baseDir string) {
	var lines []string
	for i, bck := range backlogs {
		overview := overviews[i]
		lines = append(lines, "")
		lines = append(lines, "---")
		lines = append(lines, "")
		lines = append(lines, utils.JoinMarkdownLinks(MakeOverviewLink(overview, baseDir)))
		lines = append(lines, "")
		lines = append(lines, velocity.backlogVelocity(bck, overview)...)
		lines = append(lines, "")
	}
	lines = append(lines, "")

	velocity.markdown.SetFreeText(lines)
	velocity.Save()
}

func (velocity *Velocity) backlogVelocity(bck *Backlog, overview *BacklogOverview) []string {
	chart, err := BacklogView{}.Velocity(bck, 12, 84)
	if err != nil {
		return nil
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

	return newFreeText
}

func (velocity *Velocity) UpdateLinks(rootDir string) {
	links := MakeStandardLinks(rootDir, filepath.Dir(velocity.markdown.contentPath))
	velocity.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	velocity.Save()
}
