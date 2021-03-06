package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
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

func (velocity *Velocity) Update(backlogs []*Backlog, overviews []*BacklogOverview, backlogDirs []string, baseDir string) error {
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
	return velocity.Save()
}

func (velocity *Velocity) generateVelocityImage(backlogDir string, bck *Backlog, overview *BacklogOverview) (string, error) {
	chart, err := BacklogView{}.VelocityImage(bck, 12)
	if err != nil {
		return "", nil
	}

	velocityDir := filepath.Join(filepath.Dir(backlogDir), velocityDirectoryName)
	err = os.MkdirAll(velocityDir, 0777)
	if err != nil {
		return "", err
	}
	velocityPngPath := filepath.Join(velocityDir, fmt.Sprintf("%s.png", filepath.Base(backlogDir)))
	err = ioutil.WriteFile(velocityPngPath, chart, 0644)
	return velocityPngPath, err
}

func (velocity *Velocity) UpdateLinks(rootDir string) error {
	links := MakeStandardLinks(rootDir, filepath.Dir(velocity.markdown.ContentPath()))
	velocity.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	return velocity.Save()
}
