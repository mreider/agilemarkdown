package backlog

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
)

const (
	CreatedField     = "created"
	ModifiedField    = "modified"
	GroupTitlePrefix = "### "
)

type MarkdownItem interface {
	Lines() []string
}

type MarkdownContent struct {
	contentPath string
	fieldsSet   map[string]bool
	isDirty     bool
	items       []MarkdownItem
}

func LoadMarkdown(markdownPath string, fields []string) (*MarkdownContent, error) {
	var err error
	if _, err = os.Stat(markdownPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var data []byte
	if err == nil {
		itemFile, err := os.Open(markdownPath)
		if err != nil {
			return nil, err
		}
		defer itemFile.Close()
		data, err = ioutil.ReadAll(itemFile)
		if err != nil {
			return nil, err
		}
	}
	return NewMarkdown(string(data), markdownPath, fields), nil
}

func NewMarkdown(data, markdownPath string, fields []string) *MarkdownContent {
	content := &MarkdownContent{contentPath: markdownPath, fieldsSet: make(map[string]bool)}
	for _, field := range fields {
		content.fieldsSet[field] = true
	}

	if len(data) > 0 {
		lines := strings.Split(data, "\n")

		var currentGroup *MarkdownGroup
		for _, line := range lines {
			field, value := content.getFieldAndValue(line)
			if content.fieldsSet[field] {
				content.addItem(&MarkdownField{field, value})
			}
			if strings.HasPrefix(line, GroupTitlePrefix) {
				if currentGroup != nil {
					content.addItem(currentGroup)
				}
				currentGroup = &MarkdownGroup{content: content, title: strings.TrimSpace(strings.TrimPrefix(line, GroupTitlePrefix))}
			} else if currentGroup != nil {
				if strings.TrimSpace(line) != "" {
					currentGroup.lines = append(currentGroup.lines, line)
				}
			}
		}
		if currentGroup != nil {
			content.addItem(currentGroup)
		}
	}
	return content
}

func (content *MarkdownContent) Save() error {
	if content.contentPath == "" {
		return nil
	}
	if !content.isDirty {
		return nil
	}
	data := content.Content(getCurrentTimestamp())
	err := ioutil.WriteFile(content.contentPath, data, 0644)
	if err != nil {
		return err
	}
	content.isDirty = false
	return nil
}

func (content *MarkdownContent) Content(timestamp string) []byte {
	if content.fieldsSet[CreatedField] && content.FieldValue(CreatedField) == "" {
		content.SetFieldValue(CreatedField, timestamp)
	}
	if content.fieldsSet[ModifiedField] {
		content.SetFieldValue(ModifiedField, timestamp)
	}
	result := bytes.NewBuffer(nil)
	for _, item := range content.items {
		result.WriteString(strings.Join(item.Lines(), "\n"))
	}
	return result.Bytes()
}

func (content *MarkdownContent) FieldValue(field string) string {
	for _, item := range content.items {
		if f, ok := item.(*MarkdownField); ok {
			if f.field == field {
				return f.value
			}
		}
	}
	return ""
}

func (content *MarkdownContent) SetFieldValue(field, value string) {
	for _, item := range content.items {
		if f, ok := item.(*MarkdownField); ok {
			if f.field == field {
				f.value = value
				content.markDirty()
				return
			}
		}
	}

	if !content.fieldsSet[field] {
		return
	}

	f := &MarkdownField{field, value}
	content.addItem(f)
}

func (content *MarkdownContent) GroupCount() int {
	result := 0
	for _, item := range content.items {
		if _, ok := item.(*MarkdownGroup); ok {
			result++
		}
	}
	return result
}

func (content *MarkdownContent) Group(title string) *MarkdownGroup {
	for _, item := range content.items {
		if g, ok := item.(*MarkdownGroup); ok {
			if g.title == title {
				return g
			}
		}
	}
	return nil
}

func (content *MarkdownContent) getFieldAndValue(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func (content *MarkdownContent) addItem(item MarkdownItem) {
	content.items = append(content.items, item)
	content.markDirty()
}

func (content *MarkdownContent) markDirty() {
	content.isDirty = true
}
