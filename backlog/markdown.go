package backlog

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
)

const (
	CreatedMetadataKey  = "Created"
	ModifiedMetadataKey = "Modified"
	GroupTitlePrefix    = "### "
)

type MarkdownContent struct {
	contentPath string
	isDirty     bool
	metadata    *MarkdownMetadata
	groups      []*MarkdownGroup
}

func LoadMarkdown(markdownPath string, metadataKeys []string) (*MarkdownContent, error) {
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
	return NewMarkdown(string(data), markdownPath, metadataKeys), nil
}

func NewMarkdown(data, markdownPath string, metadataKeys []string) *MarkdownContent {
	content := &MarkdownContent{contentPath: markdownPath, metadata: NewMarkdownMetadata(metadataKeys)}
	if len(data) > 0 {
		lines := strings.Split(data, "\n")
		parsed := content.metadata.ParseLines(lines)

		var currentGroup *MarkdownGroup
		for _, line := range lines[parsed:] {
			if strings.HasPrefix(line, GroupTitlePrefix) {
				if currentGroup != nil {
					content.addGroup(currentGroup)
				}
				currentGroup = &MarkdownGroup{content: content, title: strings.TrimSpace(strings.TrimPrefix(line, GroupTitlePrefix))}
			} else if currentGroup != nil {
				if strings.TrimSpace(line) != "" {
					currentGroup.lines = append(currentGroup.lines, strings.TrimRightFunc(line, unicode.IsSpace))
				}
			}
		}
		if currentGroup != nil {
			content.addGroup(currentGroup)
		}
	}
	content.isDirty = false
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
	if content.metadata.IsAllowedKey(CreatedMetadataKey) && content.MetadataValue(CreatedMetadataKey) == "" {
		content.SetMetadataValue(CreatedMetadataKey, timestamp)
	}
	if content.metadata.IsAllowedKey(ModifiedMetadataKey) {
		content.SetMetadataValue(ModifiedMetadataKey, timestamp)
	}
	result := bytes.NewBuffer(nil)
	result.WriteString(strings.Join(content.metadata.RawLines(), "\n"))
	result.WriteString("\n")
	for _, group := range content.groups {
		result.WriteString("\n")
		result.WriteString(strings.Join(group.RawLines(), "\n"))
		result.WriteString("\n")
	}
	return result.Bytes()
}

func (content *MarkdownContent) MetadataValue(key string) string {
	return content.metadata.Value(key)
}

func (content *MarkdownContent) SetMetadataValue(key, value string) {
	if content.metadata.SetValue(key, value) {
		content.markDirty()
	}
}

func (content *MarkdownContent) GroupCount() int {
	return len(content.groups)
}

func (content *MarkdownContent) Group(title string) *MarkdownGroup {
	for _, group := range content.groups {
		if group.title == title {
			return group
		}
	}
	return nil
}

func (content *MarkdownContent) addGroup(group *MarkdownGroup) {
	content.groups = append(content.groups, group)
	content.markDirty()
}

func (content *MarkdownContent) markDirty() {
	content.isDirty = true
}
