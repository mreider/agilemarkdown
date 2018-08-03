package backlog

import (
	"fmt"
	"github.com/mreider/agilemarkdown/markdown"
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
	BacklogItemTimelineMetadataKey = "Timeline"
)

var (
	commentsTitleRe            = regexp.MustCompile(`(?i)^#{1,3}\s+Comments\s*$`)
	commentRe                  = regexp.MustCompile(`^(\s*)((@[\w.-_]+[\s,;]+)+)(.*)$`)
	commentUserSeparatorRe     = regexp.MustCompile(`[\s,;]+`)
	topBacklogItemMetadataKeys = []string{
		BacklogItemFinishedMetadataKey, BacklogItemTagsMetadataKey, BacklogItemStatusMetadataKey, BacklogItemAssignedMetadataKey,
		BacklogItemEstimateMetadataKey, BacklogItemArchiveMetadataKey, BacklogItemTimelineMetadataKey}
	bottomBacklogItemMetadataKeys = []string{CreatedMetadataKey, ModifiedMetadataKey, BacklogItemAuthorMetadataKey}
)

type BacklogItem struct {
	name     string
	markdown *markdown.Content
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	content, err := markdown.LoadMarkdown(itemPath, topBacklogItemMetadataKeys, bottomBacklogItemMetadataKeys, "", nil)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogItem{name, content}, nil
}

func NewBacklogItem(name string, markdownData string) *BacklogItem {
	content := markdown.NewMarkdown(markdownData, "", topBacklogItemMetadataKeys, bottomBacklogItemMetadataKeys, "", nil)
	return &BacklogItem{name, content}
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

func (item *BacklogItem) TimelineStr() (startDate, endDate string) {
	value := item.markdown.MetadataValue(BacklogItemTimelineMetadataKey)
	parts := spacesRe.Split(value, 2)
	switch len(parts) {
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[1]
	}
}

func (item *BacklogItem) Timeline() (startDate, endDate time.Time) {
	startDateStr, endDateStr := item.TimelineStr()
	startDate, _ = time.Parse("2006-01-02", startDateStr)
	endDate, _ = time.Parse("2006-01-02", endDateStr)
	return startDate, endDate
}

func (item *BacklogItem) SetTimeline(startDate, endDate time.Time) {
	item.markdown.SetMetadataValue(BacklogItemTimelineMetadataKey, fmt.Sprintf("%s %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")))
}

func (item *BacklogItem) ClearTimeline() {
	item.markdown.RemoveMetadata(BacklogItemTimelineMetadataKey)
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

func (item *BacklogItem) UpdateComments(comments []*Comment) error {
	NewMarkdownComments(item.markdown).UpdateComments(comments)
	return item.Save()
}

func (item *BacklogItem) Tags() []string {
	rawTags := strings.TrimSpace(item.markdown.MetadataValue(BacklogItemTagsMetadataKey))
	return utils.SplitByRegexp(rawTags, tagSeparators)
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
	markdownDir := filepath.Dir(item.markdown.ContentPath())
	if filepath.Base(markdownDir) == archiveDirectoryName {
		newContentPath := filepath.Join(filepath.Dir(markdownDir), filepath.Base(item.markdown.ContentPath()))
		err := os.Rename(item.markdown.ContentPath(), newContentPath)
		if err != nil {
			return err
		}
		item.markdown.SetContentPath(newContentPath)
	}
	return nil
}

func (item *BacklogItem) MoveToBacklogArchiveDirectory() error {
	markdownDir := filepath.Dir(item.markdown.ContentPath())
	if filepath.Base(markdownDir) != archiveDirectoryName {
		newContentPath := filepath.Join(markdownDir, archiveDirectoryName, filepath.Base(item.markdown.ContentPath()))
		err := os.MkdirAll(filepath.Dir(newContentPath), 0777)
		if err != nil {
			return err
		}
		err = os.Rename(item.markdown.ContentPath(), newContentPath)
		if err != nil {
			return err
		}
		item.markdown.SetContentPath(newContentPath)
	}
	return nil
}

func (item *BacklogItem) UpdateLinks(rootDir string, overviewPath, archivePath string) error {
	links := MakeStandardLinks(rootDir, filepath.Dir(item.markdown.ContentPath()))
	links = append(links, utils.MakeMarkdownLink("project page", overviewPath, filepath.Dir(item.markdown.ContentPath())))
	if _, err := os.Stat(archivePath); err == nil {
		links = append(links, utils.MakeMarkdownLink("archive", archivePath, filepath.Dir(item.markdown.ContentPath())))
	}

	item.markdown.SetLinks(utils.JoinMarkdownLinks(links...))
	return item.Save()
}

func (item *BacklogItem) Content() []byte {
	return item.markdown.Content()
}

func (item *BacklogItem) Links() string {
	return item.markdown.Links()
}

func (item *BacklogItem) Header() string {
	return item.markdown.Header()
}

func (item *BacklogItem) SetHeader(header string) {
	item.markdown.SetHeader(header)
}

func (item *BacklogItem) Path() string {
	return item.markdown.ContentPath()
}
