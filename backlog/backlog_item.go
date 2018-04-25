package backlog

import (
	"path/filepath"
	"strings"
	"time"
)

const (
	BacklogItemTitleField    = "title"
	BacklogItemAuthorField   = "author"
	BacklogItemStatusField   = "status"
	BacklogItemAssignedField = "assigned"
	BacklogItemEstimateField = "estimate"
)

type BacklogItem struct {
	name     string
	markdown *MarkdownContent
}

func LoadBacklogItem(itemPath string) (*BacklogItem, error) {
	markdown, err := LoadMarkdown(itemPath, []string{
		BacklogItemTitleField, CreatedField, ModifiedField, BacklogItemAuthorField,
		BacklogItemStatusField, BacklogItemAssignedField, BacklogItemEstimateField})
	if err != nil {
		return nil, err
	}
	name := filepath.Base(itemPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return &BacklogItem{name, markdown}, nil
}

func NewBacklogItem(name string) *BacklogItem {
	markdown := NewMarkdown("", "", []string{
		BacklogItemTitleField, CreatedField, ModifiedField, BacklogItemAuthorField,
		BacklogItemStatusField, BacklogItemAssignedField, BacklogItemEstimateField})
	return &BacklogItem{name, markdown}
}

func (item *BacklogItem) Save() error {
	return item.markdown.Save()
}

func (item *BacklogItem) Name() string {
	return item.name
}

func (item *BacklogItem) Title() string {
	return item.markdown.FieldValue(BacklogItemTitleField)
}

func (item *BacklogItem) SetTitle(title string) {
	item.markdown.SetFieldValue(BacklogItemTitleField, title)
}

func (item *BacklogItem) SetCreated() {
	item.markdown.SetFieldValue(CreatedField, "")
}

func (item *BacklogItem) Modified() time.Time {
	value, _ := parseTimestamp(item.markdown.FieldValue(ModifiedField))
	return value
}

func (item *BacklogItem) SetModified() {
	item.markdown.SetFieldValue(ModifiedField, "")
}

func (item *BacklogItem) Author() string {
	return item.markdown.FieldValue(BacklogItemAuthorField)
}

func (item *BacklogItem) SetAuthor(author string) {
	item.markdown.SetFieldValue(BacklogItemAuthorField, author)
}

func (item *BacklogItem) Status() string {
	return item.markdown.FieldValue(BacklogItemStatusField)
}

func (item *BacklogItem) SetStatus(status string) {
	item.markdown.SetFieldValue(BacklogItemStatusField, status)
}

func (item *BacklogItem) Assigned() string {
	return item.markdown.FieldValue(BacklogItemAssignedField)
}

func (item *BacklogItem) SetAssigned(assigned string) {
	item.markdown.SetFieldValue(BacklogItemAssignedField, assigned)
}

func (item *BacklogItem) Estimate() string {
	return item.markdown.FieldValue(BacklogItemEstimateField)
}

func (item *BacklogItem) SetEstimate(estimate string) {
	item.markdown.SetFieldValue(BacklogItemEstimateField, estimate)
}
