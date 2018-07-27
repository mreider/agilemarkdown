package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	BacklogItemAuthorMetadataKey   = "Author"
	BacklogItemStatusMetadataKey   = "Status"
	BacklogItemAssignedMetadataKey = "Assigned"
	BacklogItemEstimateMetadataKey = "Estimate"
	BacklogItemTagsMetadataKey     = "Tags"
	BacklogItemArchiveMetadataKey  = "Archive"
	BacklogItemFinishedMetadataKey = "Finished"
)

var (
	commentsTitleRe                = regexp.MustCompile(`^#{1,3}\s+Comments\s*$`)
	commentRe                      = regexp.MustCompile(`^(\s*)((@[\w.-_]+[\s,;]+)+)(.*)$`)
	commentUserSeparatorRe         = regexp.MustCompile(`[\s,;]+`)
	BacklogItemTimelineMetadataKey = regexp.MustCompile(`(?i)^Timeline\s+([-\w]+)$`)
	topBacklogItemMetadataKeys     = []*regexp.Regexp{
		AllowedKeyAsRegex(BacklogItemFinishedMetadataKey), AllowedKeyAsRegex(BacklogItemTagsMetadataKey),
		AllowedKeyAsRegex(BacklogItemStatusMetadataKey), AllowedKeyAsRegex(BacklogItemAssignedMetadataKey),
		AllowedKeyAsRegex(BacklogItemEstimateMetadataKey), AllowedKeyAsRegex(BacklogItemArchiveMetadataKey),
		BacklogItemTimelineMetadataKey}
	bottomBacklogItemMetadataKeys = []*regexp.Regexp{
		AllowedKeyAsRegex(CreatedMetadataKey), AllowedKeyAsRegex(ModifiedMetadataKey),
		AllowedKeyAsRegex(BacklogItemAuthorMetadataKey)}
)

type BacklogItem struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := LoadMarkdown(itemPath, topBacklogItemMetadataKeys, bottomBacklogItemMetadataKeys, "", nil)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogItem{name, markdown}, nil
}

func NewBacklogItem(name string, markdownData string) *BacklogItem {
	markdown := NewMarkdown(markdownData, "", topBacklogItemMetadataKeys, bottomBacklogItemMetadataKeys, "", nil)
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

func (item *BacklogItem) SetModified(timestamp string) {
	item.markdown.SetMetadataValue(ModifiedMetadataKey, timestamp)
}

func (item *BacklogItem) Finished() time.Time {
	value, _ := utils.ParseTimestamp(item.markdown.MetadataValue(BacklogItemFinishedMetadataKey))
	return value
}

func (item *BacklogItem) SetFinished(timestamp string) {
	item.markdown.SetMetadataValue(BacklogItemFinishedMetadataKey, timestamp)
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

func (item *BacklogItem) TimelineStr(tag string) (startDate, endDate string) {
	value := item.markdown.MetadataValue(fmt.Sprintf("Timeline %s", tag))
	parts := spacesRe.Split(value, 2)
	switch len(parts) {
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[1]
	}
}

func (item *BacklogItem) Timeline(tag string) (startDate, endDate time.Time) {
	startDateStr, endDateStr := item.TimelineStr(tag)
	startDate, _ = time.Parse("2006-01-02", startDateStr)
	endDate, _ = time.Parse("2006-01-02", endDateStr)
	return startDate, endDate
}

func (item *BacklogItem) SetTimeline(tag string, startDate, endDate time.Time) {
	item.markdown.SetMetadataValue(fmt.Sprintf("Timeline %s", tag), fmt.Sprintf("%s %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))
}

func (item *BacklogItem) ClearTimeline(tag string) {
	item.markdown.RemoveMetadata(fmt.Sprintf("Timeline %s", tag))
}

func (item *BacklogItem) ChangeTimelineTag(oldTag, newTag string) {
	oldKey := fmt.Sprintf("Timeline %s", oldTag)
	newKey := fmt.Sprintf("Timeline %s", newTag)

	if item.markdown.MetadataValue(newKey) != "" {
		item.ClearTimeline(oldTag)
	} else {
		item.markdown.RemoveMetadata(newKey)
		item.markdown.ReplaceMetadataKey(oldKey, newKey)
	}
}

func (item *BacklogItem) SetDescription(description string) {
	if description != "" {
		description = "\n" + description
	}

	item.markdown.SetFreeText(strings.Split(description, "\n"))
}

func (item *BacklogItem) Comments() []*Comment {
	return NewMarkdownComments(item.markdown).Comments()
}

func (item *BacklogItem) UpdateComments(comments []*Comment) {
	NewMarkdownComments(item.markdown).UpdateComments(comments)
	item.Save()
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
	links := MakeStandardLinks(rootDir, filepath.Dir(item.markdown.contentPath))
	links = append(links, utils.MakeMarkdownLink("project page", overviewPath, filepath.Dir(item.markdown.contentPath)))
	if _, err := os.Stat(archivePath); err == nil {
		links = append(links, utils.MakeMarkdownLink("archive", archivePath, filepath.Dir(item.markdown.contentPath)))
	}

	item.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	item.Save()
}

func (item *BacklogItem) Content() []byte {
	return item.markdown.Content()
}

func (item *BacklogItem) Links() string {
	return item.markdown.links
}

func (item *BacklogItem) Header() string {
	return item.markdown.header
}

func (item *BacklogItem) SetHeader(header string) {
	item.markdown.SetHeader(header)
	item.Save()
}

func (item *BacklogItem) Path() string {
	return item.markdown.contentPath
}
