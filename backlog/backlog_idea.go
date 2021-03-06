package backlog

import (
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	BacklogIdeaAuthorMetadataKey = "Author"
	BacklogIdeaTagsMetadataKey   = "Tags"
	BacklogIdeaRankMetadataKey   = "Rank"
)

var (
	RelatedItemsRegex = regexp.MustCompile(`(?i)^#+\s*stories\s*$`)
)

type BacklogIdea struct {
	name     string
	markdown *markdown.Content
}

func LoadBacklogIdea(ideaPath string) (*BacklogIdea, error) {
	content, err := markdown.LoadMarkdown(ideaPath,
		[]string{
			CreatedMetadataKey, ModifiedMetadataKey,
			BacklogIdeaAuthorMetadataKey, BacklogIdeaTagsMetadataKey,
			BacklogIdeaRankMetadataKey},
		nil,
		"", RelatedItemsRegex)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(ideaPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogIdea{name, content}, nil
}

func (idea *BacklogIdea) Save() error {
	return idea.markdown.Save()
}

func (idea *BacklogIdea) Name() string {
	return idea.name
}

func (idea *BacklogIdea) HasMetadata() bool {
	return !idea.markdown.Metadata().BottomEmpty() || !idea.markdown.Metadata().TopEmpty()
}

func (idea *BacklogIdea) Title() string {
	return idea.markdown.Title()
}

func (idea *BacklogIdea) SetTitle(title string) {
	idea.markdown.SetTitle(title)
}

func (idea *BacklogIdea) SetFooter(footer []string) {
	idea.markdown.SetFooter(footer)
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
	return utils.SplitByRegexp(rawTags, tagSeparators)
}

func (idea *BacklogIdea) SetTags(tags []string) {
	idea.markdown.SetMetadataValue(BacklogIdeaTagsMetadataKey, strings.Join(tags, ", "))
}

func (idea *BacklogIdea) SetRank(rank string) {
	idea.markdown.SetMetadataValue(BacklogIdeaRankMetadataKey, rank)
}

func (idea *BacklogIdea) SetText(text string) {
	if !strings.HasPrefix(text, "\n") {
		text = "\n" + text
	}

	idea.markdown.SetFreeText(strings.Split(text, "\n"))
}

func (idea *BacklogIdea) Text() string {
	return strings.Join(idea.markdown.FreeText(), "\n")
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
	return idea.markdown.ContentPath()
}

func (idea *BacklogIdea) UpdateLinks(rootDir string) error {
	links := MakeStandardLinks(rootDir, filepath.Dir(idea.markdown.ContentPath()))
	idea.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	return idea.Save()
}

func (idea *BacklogIdea) Rank() string {
	return utils.PadStringLeft(idea.markdown.MetadataValue(BacklogIdeaRankMetadataKey), 10)
}

func (idea *BacklogIdea) Content() []byte {
	return idea.markdown.Content()
}

func (idea *BacklogIdea) SetDescription(description string) {
	if description != "" {
		description = "\n" + description
	}

	idea.markdown.SetFreeText(strings.Split(description, "\n"))
}

func (idea *BacklogIdea) Comments() []*Comment {
	return NewMarkdownComments(idea.markdown).Comments()
}

func (idea *BacklogIdea) UpdateComments(comments []*Comment) error {
	NewMarkdownComments(idea.markdown).UpdateComments(comments)
	return idea.Save()
}
