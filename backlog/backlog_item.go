package backlog

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	BacklogItemTitleMetadataKey    = "Title"
	BacklogItemAuthorMetadataKey   = "Author"
	BacklogItemStatusMetadataKey   = "Status"
	BacklogItemAssignedMetadataKey = "Assigned"
	BacklogItemEstimateMetadataKey = "Estimate"
)

type BacklogItem struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := LoadMarkdown(itemPath, []string{
		BacklogItemTitleMetadataKey, CreatedMetadataKey, ModifiedMetadataKey, BacklogItemAuthorMetadataKey,
		BacklogItemStatusMetadataKey, BacklogItemAssignedMetadataKey, BacklogItemEstimateMetadataKey}, false)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.Replace(strings.TrimSuffix(name, filepath.Ext(name)), "_", " ", -1)
	return &BacklogItem{name, markdown}, nil
}

func NewBacklogItem(name string) *BacklogItem {
	markdown := NewMarkdown("", "", []string{
		BacklogItemTitleMetadataKey, CreatedMetadataKey, ModifiedMetadataKey, BacklogItemAuthorMetadataKey,
		BacklogItemStatusMetadataKey, BacklogItemAssignedMetadataKey, BacklogItemEstimateMetadataKey}, false)
	return &BacklogItem{name, markdown}
}

func (item *BacklogItem) Save() error {
	return item.markdown.Save()
}

func (item *BacklogItem) Name() string {
	return item.name
}

func (item *BacklogItem) FileName() string {
	return strings.Replace(item.name, " ", "_", -1) + ".md"
}

func (item *BacklogItem) Title() string {
	return item.markdown.MetadataValue(BacklogItemTitleMetadataKey)
}

func (item *BacklogItem) SetTitle(title string) {
	item.markdown.SetMetadataValue(BacklogItemTitleMetadataKey, title)
}

func (item *BacklogItem) SetCreated() {
	item.markdown.SetMetadataValue(CreatedMetadataKey, "")
}

func (item *BacklogItem) Modified() time.Time {
	value, _ := parseTimestamp(item.markdown.MetadataValue(ModifiedMetadataKey))
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

func (item *BacklogItem) SetStatus(status string) {
	item.markdown.SetMetadataValue(BacklogItemStatusMetadataKey, status)
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
