package backlog

import (
	"fmt"
	"strings"
)

type MarkdownMetadata struct {
	allowedKeys map[string]bool
	items       []*markdownMetadataItem
}

type markdownMetadataItem struct {
	key   string
	value string
}

func NewMarkdownMetadata(allowedKeys []string) *MarkdownMetadata {
	metadata := &MarkdownMetadata{
		allowedKeys: make(map[string]bool, len(allowedKeys)),
	}
	for _, key := range allowedKeys {
		metadata.allowedKeys[strings.ToLower(key)] = true
	}
	return metadata
}

func (m *MarkdownMetadata) RawLines() []string {
	result := make([]string, 0, len(m.items))
	for _, item := range m.items {
		result = append(result, fmt.Sprintf("%s: %s  ", item.key, item.value))
	}
	return result
}

func (m *MarkdownMetadata) IsAllowedKey(key string) bool {
	return m.allowedKeys[strings.ToLower(key)]
}

func (m *MarkdownMetadata) Value(key string) string {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			return item.value
		}
	}
	return ""
}

func (m *MarkdownMetadata) SetValue(key, value string) bool {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			item.value = value
			return true
		}
	}

	if !m.IsAllowedKey(key) {
		return false
	}

	item := &markdownMetadataItem{key, value}
	m.items = append(m.items, item)
	return true
}

func (m *MarkdownMetadata) ParseLines(lines []string) int {
	m.items = nil
	parsed := 0
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			parsed++
			continue
		}
		if !strings.Contains(trimmedLine, ":") { // TODO: multi-line metadata
			return parsed
		}
		parts := strings.SplitN(trimmedLine, ":", 2)
		var item *markdownMetadataItem
		if len(parts) == 1 {
			item = &markdownMetadataItem{strings.TrimSpace(parts[0]), ""}
		} else {
			item = &markdownMetadataItem{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}
		}
		m.items = append(m.items, item)
		parsed++
	}
	return parsed
}
