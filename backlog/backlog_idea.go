package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	BacklogIdeaAuthorMetadataKey = "Author"
	BacklogIdeaTagsMetadataKey   = "Tags"
)

type BacklogIdea struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogIdea(ideaPath string) (*BacklogIdea, error) {
	markdown, err := LoadMarkdown(ideaPath, []string{
		CreatedMetadataKey, ModifiedMetadataKey, BacklogIdeaAuthorMetadataKey, BacklogIdeaTagsMetadataKey}, "", nil)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(ideaPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogIdea{name, markdown}, nil
}

func NewBacklogIdea(name string, markdownData string) *BacklogIdea {
	markdown := NewMarkdown(markdownData, "", []string{
		CreatedMetadataKey, ModifiedMetadataKey, BacklogIdeaAuthorMetadataKey, BacklogIdeaTagsMetadataKey}, "", nil)
	return &BacklogIdea{name, markdown}
}

func (idea *BacklogIdea) Save() error {
	return idea.markdown.Save()
}

func (idea *BacklogIdea) Name() string {
	return idea.name
}

func (idea *BacklogIdea) HasMetadata() bool {
	return !idea.markdown.metadata.Empty()
}

func (idea *BacklogIdea) Title() string {
	return idea.markdown.Title()
}

func (idea *BacklogIdea) SetTitle(title string) {
	idea.markdown.SetTitle(title)
}

func (idea *BacklogIdea) SetCreated(timestamp string) {
	idea.markdown.SetMetadataValue(CreatedMetadataKey, timestamp)
}

func (idea *BacklogIdea) Created() time.Time {
	value, _ := utils.ParseTimestamp(idea.markdown.MetadataValue(CreatedMetadataKey))
	return value
}

func (idea *BacklogIdea) Modified() time.Time {
	value, _ := utils.ParseTimestamp(idea.markdown.MetadataValue(ModifiedMetadataKey))
	return value
}

func (idea *BacklogIdea) SetModified(timestamp string) {
	idea.markdown.SetMetadataValue(ModifiedMetadataKey, timestamp)
}

func (idea *BacklogIdea) Author() string {
	return idea.markdown.MetadataValue(BacklogIdeaAuthorMetadataKey)
}

func (idea *BacklogIdea) SetAuthor(author string) {
	idea.markdown.SetMetadataValue(BacklogIdeaAuthorMetadataKey, author)
}

func (idea *BacklogIdea) Tags() []string {
	rawTags := strings.TrimSpace(idea.markdown.MetadataValue(BacklogIdeaTagsMetadataKey))
	return strings.Fields(rawTags)
}

func (idea *BacklogIdea) SetTags(tags []string) {
	idea.markdown.SetMetadataValue(BacklogIdeaTagsMetadataKey, strings.Join(tags, " "))
}

func (idea *BacklogIdea) SetText(text string) {
	if !strings.HasPrefix(text, "\n") {
		text = "\n" + text
	}

	idea.markdown.SetFreeText(strings.Split(text, "\n"))
}

func (idea *BacklogIdea) Text() string {
	return strings.Join(idea.markdown.freeText, "\n")
}

func LoadIdeas(ideasDir string) ([]*BacklogIdea, error) {
	infos, err := ioutil.ReadDir(ideasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	ideasPaths := make([]string, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		ideaPath := filepath.Join(ideasDir, info.Name())
		ideasPaths = append(ideasPaths, ideaPath)
	}

	sort.Strings(ideasPaths)

	ideas := make([]*BacklogIdea, 0, len(ideasPaths))
	for _, ideaPath := range ideasPaths {
		idea, err := LoadBacklogIdea(ideaPath)
		if err != nil {
			return nil, err
		}
		ideas = append(ideas, idea)
	}
	return ideas, nil
}

func (idea *BacklogIdea) Path() string {
	return idea.markdown.contentPath
}

func (idea *BacklogIdea) UpdateLinks(rootDir string) {
	links := []string{
		utils.MakeMarkdownLink("index", filepath.Join(rootDir, IndexFileName), filepath.Dir(idea.markdown.contentPath)),
		utils.MakeMarkdownLink("ideas", filepath.Join(rootDir, IdeasFileName), filepath.Dir(idea.markdown.contentPath)),
		utils.MakeMarkdownLink("tags", filepath.Join(rootDir, TagsFileName), filepath.Dir(idea.markdown.contentPath)),
	}
	idea.markdown.SetLinks(strings.Join(links, " "))
	idea.Save()
}
