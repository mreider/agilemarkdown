package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"strings"
	"time"
)

const (
	BacklogIdeaTitleMetadataKey  = "Title"
	BacklogIdeaAuthorMetadataKey = "Author"
	BacklogIdeaTagsMetadataKey   = "Tags"
)

type BacklogIdea struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogIdea(ideaPath string) (*BacklogIdea, error) {
	markdown, err := LoadMarkdown(ideaPath, []string{
		BacklogIdeaTitleMetadataKey, CreatedMetadataKey, ModifiedMetadataKey, BacklogIdeaAuthorMetadataKey, BacklogIdeaTagsMetadataKey}, false)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(ideaPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogIdea{name, markdown}, nil
}

func NewBacklogIdea(name string, markdownData string) *BacklogIdea {
	markdown := NewMarkdown(markdownData, "", []string{
		BacklogIdeaTitleMetadataKey, CreatedMetadataKey, ModifiedMetadataKey, BacklogIdeaAuthorMetadataKey, BacklogIdeaTagsMetadataKey}, false)
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
	return idea.markdown.MetadataValue(BacklogIdeaTitleMetadataKey)
}

func (idea *BacklogIdea) SetTitle(title string) {
	idea.markdown.SetMetadataValue(BacklogIdeaTitleMetadataKey, title)
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
