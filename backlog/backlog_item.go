package backlog

import (
	"fmt"
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
	commentsTitleRe                = regexp.MustCompile(`^#{1,3}\s+Comments\s*$`)
	commentRe                      = regexp.MustCompile(`^(\s*)((@[\w.-_]+[\s,;]+)+)(.*)$`)
	commentUserSeparatorRe         = regexp.MustCompile(`[\s,;]+`)
	BacklogItemTimelineMetadataKey = regexp.MustCompile(`(?i)^Timeline\s+(\w+)$`)
)

type Comment struct {
	Users   []string
	Text    []string
	rawText []string
	Closed  bool
	Unsent  bool
}

type BacklogItem struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := LoadMarkdown(itemPath,
		[]*regexp.Regexp{
			AllowedKeyAsRegex(BacklogItemTagsMetadataKey), AllowedKeyAsRegex(BacklogItemStatusMetadataKey), AllowedKeyAsRegex(BacklogItemAssignedMetadataKey),
			AllowedKeyAsRegex(BacklogItemEstimateMetadataKey), AllowedKeyAsRegex(BacklogItemArchiveMetadataKey), BacklogItemTimelineMetadataKey},
		[]*regexp.Regexp{AllowedKeyAsRegex(CreatedMetadataKey), AllowedKeyAsRegex(ModifiedMetadataKey), AllowedKeyAsRegex(BacklogItemAuthorMetadataKey)},
		"", nil)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogItem{name, markdown}, nil
}

func NewBacklogItem(name string, markdownData string) *BacklogItem {
	markdown := NewMarkdown(markdownData, "",
		[]*regexp.Regexp{
			AllowedKeyAsRegex(BacklogItemTagsMetadataKey), AllowedKeyAsRegex(BacklogItemStatusMetadataKey), AllowedKeyAsRegex(BacklogItemAssignedMetadataKey),
			AllowedKeyAsRegex(BacklogItemEstimateMetadataKey), AllowedKeyAsRegex(BacklogItemArchiveMetadataKey), BacklogItemTimelineMetadataKey},
		[]*regexp.Regexp{AllowedKeyAsRegex(CreatedMetadataKey), AllowedKeyAsRegex(ModifiedMetadataKey), AllowedKeyAsRegex(BacklogItemAuthorMetadataKey)},
		"", nil)
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
			if comment != nil {
				comment.rawText = append(comment.rawText, item.markdown.freeText[i])
			}
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
			rawUsers := commentUserSeparatorRe.Split(matches[2], -1)
			allUsers := make(map[string]bool)
			users := make([]string, 0, len(rawUsers))
			for _, user := range rawUsers {
				user = strings.TrimPrefix(user, "@")
				user = strings.TrimSuffix(user, ".")
				if user == "" {
					continue
				}
				if !allUsers[user] {
					users = append(users, user)
					allUsers[user] = true
				}
			}
			comment = &Comment{Users: users}
			if len(matches[1]) > 0 {
				comment.Closed = true
			}
			text := strings.TrimSpace(matches[4])
			if text != "" {
				comment.Text = append(comment.Text, text)
			}
			comment.rawText = append(comment.rawText, item.markdown.freeText[i])
		} else {
			if comment != nil {
				line := strings.TrimSpace(line)
				if strings.HasPrefix(strings.ToLower(line), "sent by ") {
					comment.Closed = true
				} else if strings.HasPrefix(strings.ToLower(line), "can't send by ") {
					comment.Unsent = true
				}
				comment.Text = append(comment.Text, line)
				comment.rawText = append(comment.rawText, item.markdown.freeText[i])
			}
		}
	}
	if comment != nil {
		comments = append(comments, comment)
	}
	return comments
}

func (item *BacklogItem) UpdateComments(comments []*Comment) {
	commentsStartIndex := -1
	for i := len(item.markdown.freeText) - 1; i >= 0; i-- {
		if commentsTitleRe.MatchString(item.markdown.freeText[i]) {
			commentsStartIndex = i + 1
			break
		}
	}
	if commentsStartIndex == -1 {
		return
	}

	for commentsStartIndex < len(item.markdown.freeText) && strings.TrimSpace(item.markdown.freeText[commentsStartIndex]) == "" {
		commentsStartIndex++
	}

	commentsFinishIndex := commentsStartIndex
	for i := commentsStartIndex; i < len(item.markdown.freeText); i++ {
		line := strings.TrimRightFunc(item.markdown.freeText[i], unicode.IsSpace)
		commentsFinishIndex = i
		if strings.HasPrefix(line, "#") {
			break
		}
	}

	newFreeText := make([]string, 0, len(item.markdown.freeText))
	newFreeText = append(newFreeText, item.markdown.freeText[:commentsStartIndex]...)
	for _, comment := range comments {
		newFreeText = append(newFreeText, comment.rawText...)
	}
	newFreeText = append(newFreeText, item.markdown.freeText[commentsFinishIndex:]...)

	item.markdown.SetFreeText(newFreeText)
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

func (c *Comment) AddLine(line string) {
	c.Text = append(c.Text, line)

	c.rawText = append(c.rawText, line)
	i := len(c.rawText) - 1
	for i > 0 && strings.TrimSpace(c.rawText[i-1]) == "" {
		c.rawText[i-1], c.rawText[i] = c.rawText[i], c.rawText[i-1]
		i--
	}
}

func (item *BacklogItem) Path() string {
	return item.markdown.contentPath
}
