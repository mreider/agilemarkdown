package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
)

const (
	BacklogItemAuthorMetadataKey   = "Author"
	BacklogItemStatusMetadataKey   = "Status"
	BacklogItemAssignedMetadataKey = "Assigned"
	BacklogItemEstimateMetadataKey = "Estimate"
	BacklogItemTagsMetadataKey     = "Tags"
	BacklogItemArchiveMetadataKey  = "Archive"
)

var (
	commentsTitleRe = regexp.MustCompile(`^#{1,3}\s+Comments\s*$`)
	commentRe       = regexp.MustCompile(`^(\s*)@([\w.-_]+)(\s+.*)?$`)
)

type Comment struct {
	User   string
	Text   []string
	Closed bool
}

type BacklogItem struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := LoadMarkdown(itemPath, []string{
		CreatedMetadataKey, ModifiedMetadataKey, BacklogItemAuthorMetadataKey,
		BacklogItemStatusMetadataKey, BacklogItemAssignedMetadataKey, BacklogItemEstimateMetadataKey,
		BacklogItemTagsMetadataKey, BacklogItemArchiveMetadataKey}, "", nil)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogItem{name, markdown}, nil
}

func NewBacklogItem(name string, markdownData string) *BacklogItem {
	markdown := NewMarkdown(markdownData, "", []string{
		CreatedMetadataKey, ModifiedMetadataKey, BacklogItemAuthorMetadataKey,
		BacklogItemStatusMetadataKey, BacklogItemAssignedMetadataKey, BacklogItemEstimateMetadataKey,
		BacklogItemTagsMetadataKey, BacklogItemArchiveMetadataKey}, "", nil)
	return &BacklogItem{name, markdown}
}

func (item *BacklogItem) Save() error {
	return item.markdown.Save()
}

func (item *BacklogItem) Name() string {
	return item.name
}

func (item *BacklogItem) Title() string {
	return item.markdown.Title()
}

func (item *BacklogItem) SetTitle(title string) {
	item.markdown.SetTitle(title)
}

func (item *BacklogItem) SetCreated(timestamp string) {
	item.markdown.SetMetadataValue(CreatedMetadataKey, timestamp)
}

func (item *BacklogItem) Created() time.Time {
	value, _ := utils.ParseTimestamp(item.markdown.MetadataValue(CreatedMetadataKey))
	return value
}

func (item *BacklogItem) Modified() time.Time {
	value, _ := utils.ParseTimestamp(item.markdown.MetadataValue(ModifiedMetadataKey))
	return value
}

func (item *BacklogItem) SetModified() {
	item.markdown.SetMetadataValue(ModifiedMetadataKey, "")
}

func (item *BacklogItem) Author() string {
	return item.markdown.MetadataValue(BacklogItemAuthorMetadataKey)
}

func (item *BacklogItem) SetAuthor(author string) {
	item.markdown.SetMetadataValue(BacklogItemAuthorMetadataKey, author)
}

func (item *BacklogItem) Status() string {
	return item.markdown.MetadataValue(BacklogItemStatusMetadataKey)
}

func (item *BacklogItem) SetStatus(status *BacklogItemStatus) {
	item.markdown.SetMetadataValue(BacklogItemStatusMetadataKey, status.Name)
}

func (item *BacklogItem) Assigned() string {
	return item.markdown.MetadataValue(BacklogItemAssignedMetadataKey)
}

func (item *BacklogItem) SetAssigned(assigned string) {
	item.markdown.SetMetadataValue(BacklogItemAssignedMetadataKey, assigned)
}

func (item *BacklogItem) Estimate() string {
	return item.markdown.MetadataValue(BacklogItemEstimateMetadataKey)
}

func (item *BacklogItem) SetEstimate(estimate string) {
	item.markdown.SetMetadataValue(BacklogItemEstimateMetadataKey, estimate)
}

func (item *BacklogItem) SetDescription(description string) {
	if description != "" {
		description = "\n" + description
	}

	item.markdown.SetFreeText(strings.Split(description, "\n"))
}

func (item *BacklogItem) Comments() []*Comment {
	commentsStartIndex := -1
	for i := len(item.markdown.freeText) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(item.markdown.freeText[i]) {
			commentsStartIndex = i + 1
			break
		}
	}
	if commentsStartIndex == -1 {
		return nil
	}

	comments := make([]*Comment, 0)
	var comment *Comment
	for i := commentsStartIndex; i < len(item.markdown.freeText); i++ {
		line := strings.TrimRightFunc(item.markdown.freeText[i], unicode.IsSpace)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			break
		}
		matches := commentRe.FindStringSubmatch(line)
		if len(matches) > 0 {
			if comment != nil {
				comments = append(comments, comment)
			}
			comment = &Comment{User: strings.TrimSuffix(matches[2], ".")}
			if len(matches[1]) > 0 {
				comment.Closed = true
			}
			text := strings.TrimSpace(matches[3])
			if text != "" {
				comment.Text = append(comment.Text, text)
			}
		} else {
			if comment != nil {
				comment.Text = append(comment.Text, strings.TrimSpace(line))
			}
		}
	}
	if comment != nil {
		comments = append(comments, comment)
	}
	return comments
}

func (item *BacklogItem) Tags() []string {
	rawTags := strings.TrimSpace(item.markdown.MetadataValue(BacklogItemTagsMetadataKey))
	return strings.Fields(rawTags)
}

func (item *BacklogItem) SetTags(tags []string) {
	item.markdown.SetMetadataValue(BacklogItemTagsMetadataKey, strings.Join(tags, " "))
}

func (item *BacklogItem) Archived() bool {
	archive := strings.ToLower(item.markdown.MetadataValue(BacklogItemArchiveMetadataKey))
	return archive == "1" || archive == "true" || archive == "yes"
}

func (item *BacklogItem) SetArchived(archived bool) {
	item.markdown.SetMetadataValue(BacklogItemArchiveMetadataKey, "true")
}

func (item *BacklogItem) MoveToBacklogDirectory() error {
	markdownDir := filepath.Dir(item.markdown.contentPath)
	if filepath.Base(markdownDir) == ArchiveDirectoryName {
		newContentPath := filepath.Join(filepath.Dir(markdownDir), filepath.Base(item.markdown.contentPath))
		err := os.Rename(item.markdown.contentPath, newContentPath)
		if err != nil {
			return err
		}
		item.markdown.contentPath = newContentPath
	}
	return nil
}

func (item *BacklogItem) MoveToBacklogArchiveDirectory() error {
	markdownDir := filepath.Dir(item.markdown.contentPath)
	if filepath.Base(markdownDir) != ArchiveDirectoryName {
		newContentPath := filepath.Join(markdownDir, ArchiveDirectoryName, filepath.Base(item.markdown.contentPath))
		os.MkdirAll(filepath.Dir(newContentPath), 0777)
		err := os.Rename(item.markdown.contentPath, newContentPath)
		if err != nil {
			return err
		}
		item.markdown.contentPath = newContentPath
	}
	return nil
}

func (item *BacklogItem) UpdateLinks(rootDir string, overviewPath, archivePath string) {
	links := []string{
		MakeIndexLink(rootDir, filepath.Dir(item.markdown.contentPath)),
		MakeIdeasLink(rootDir, filepath.Dir(item.markdown.contentPath)),
		MakeTagsLink(rootDir, filepath.Dir(item.markdown.contentPath)),
		utils.MakeMarkdownLink("project page", overviewPath, filepath.Dir(item.markdown.contentPath)),
	}
	if _, err := os.Stat(archivePath); err == nil {
		links = append(links, utils.MakeMarkdownLink("archive", archivePath, filepath.Dir(item.markdown.contentPath)))
	}

	item.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	item.Save()
}

func (item *BacklogItem) Content() []byte {
	return item.markdown.Content(utils.GetCurrentTimestamp())
}
