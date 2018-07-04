package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (velocity *Velocity) Update(backlogs []*Backlog, overviews []*BacklogOverview, backlogDirs []string, baseDir string) {
	var lines []string
	for i, bck := range backlogs {
		overview := overviews[i]
		velocityImagePath, err := velocity.generateVelocityImage(backlogDirs[i], bck, overview)
		if err != nil {
			continue
		}
		lines = append(lines, "")
		lines = append(lines, "---")
		lines = append(lines, "")
		lines = append(lines, utils.JoinMarkdownLinks(MakeOverviewLink(overview, baseDir)))
		lines = append(lines, "")
		lines = append(lines, utils.MakeMarkdownImageLink("velocity", velocityImagePath, baseDir))
		lines = append(lines, "")
	}
	lines = append(lines, "")

	velocity.markdown.SetFreeText(lines)
	velocity.Save()
}

func (velocity *Velocity) generateVelocityImage(backlogDir string, bck *Backlog, overview *BacklogOverview) (string, error) {
	chart, err := BacklogView{}.VelocityImage(bck, 12)
	if err != nil {
		return "", nil
	}

	velocityDir := filepath.Join(filepath.Dir(backlogDir), "velocity")
	os.MkdirAll(velocityDir, 0777)
	velocityPngPath := filepath.Join(velocityDir, fmt.Sprintf("%s.png", filepath.Base(backlogDir)))
	err = ioutil.WriteFile(velocityPngPath, chart, 0644)
	return velocityPngPath, err
}

func (velocity *Velocity) UpdateLinks(rootDir string) {
	links := MakeStandardLinks(rootDir, filepath.Dir(velocity.markdown.contentPath))
	velocity.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	velocity.Save()
}
