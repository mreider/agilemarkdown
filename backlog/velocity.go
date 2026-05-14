package backlog

import (
	"path/filepath"
	"strings"

	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
)

type Velocity struct {
	markdown *markdown.Content
}

func LoadGlobalVelocity(velocityPath string) (*Velocity, error) {
	content, err := markdown.LoadMarkdown(velocityPath, nil, nil, "", nil)
	if err != nil {
		return nil, err
	}
	return NewGlobalVelocity(content), nil
}

func NewGlobalVelocity(content *markdown.Content) *Velocity {
	return &Velocity{content}
}

func (velocity *Velocity) Save() error {
	return velocity.markdown.Save()
}

func (velocity *Velocity) Title() string {
	return velocity.markdown.Title()
}

func (velocity *Velocity) SetTitle(title string) {
	velocity.markdown.SetTitle(title)
}

// Update regenerates velocity.md with one ASCII chart per backlog.
func (velocity *Velocity) Update(backlogs []*Backlog, overviews []*BacklogOverview, backlogDirs []string, baseDir string, cfg *config.Config) error {
	overrides, _ := LoadIterationOverrides(baseDir)
	var lines []string
	for i, bck := range backlogs {
		overview := overviews[i]
		lines = append(lines, "")
		lines = append(lines, "---")
		lines = append(lines, "")
		lines = append(lines, utils.JoinMarkdownLinks(MakeOverviewLink(overview, baseDir)))
		lines = append(lines, "")
		lines = append(lines, "```")
		ascii := strings.TrimRight(VelocityASCII(bck, 12, cfg, overrides), "\n")
		lines = append(lines, strings.Split(ascii, "\n")...)
		lines = append(lines, "```")
		lines = append(lines, "")
	}
	lines = append(lines, "")

	velocity.markdown.SetFreeText(lines)
	return velocity.Save()
}

func (velocity *Velocity) UpdateLinks(rootDir string) error {
	links := MakeStandardLinks(rootDir, filepath.Dir(velocity.markdown.ContentPath()))
	velocity.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	return velocity.Save()
}
