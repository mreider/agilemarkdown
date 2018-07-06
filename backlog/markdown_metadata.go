package backlog

import (
	"fmt"
	"strings"
)

type MarkdownMetadata struct {
	allowedTopKeys    map[string]bool
	allowedBottomKeys map[string]bool
	items             []*markdownMetadataItem
	bottomGroupLine   string
}

type markdownMetadataItem struct {
	key   string
	value string
}

func NewMarkdownMetadata(allowedTopKeys, allowedBottomKeys []string) *MarkdownMetadata {
	metadata := &MarkdownMetadata{
		allowedTopKeys:    make(map[string]bool, len(allowedTopKeys)),
		allowedBottomKeys: make(map[string]bool, len(allowedBottomKeys)),
	}
	for _, key := range allowedTopKeys {
		metadata.allowedTopKeys[strings.ToLower(key)] = true
	}
	for _, key := range allowedBottomKeys {
		metadata.allowedBottomKeys[strings.ToLower(key)] = true
	}
	return metadata
}

func (m *MarkdownMetadata) TopRawLines() []string {
	result := make([]string, 0, len(m.items))
	m.fillRawLines(&result, m.allowedTopKeys)
	return result
}

func (m *MarkdownMetadata) BottomRawLines() []string {
	result := make([]string, 0, len(m.items))
	bottomGroupLine := m.bottomGroupLine
	if bottomGroupLine == "" {
		bottomGroupLine = "## Metadata"
	}
	result = append(result, bottomGroupLine, "")
	m.fillRawLines(&result, m.allowedBottomKeys)
	return result
}

func (m *MarkdownMetadata) fillRawLines(lines *[]string, allowedKeys map[string]bool) {
	for _, item := range m.items {
		if allowedKeys[strings.ToLower(item.key)] {
			*lines = append(*lines, fmt.Sprintf("%s: %s  ", item.key, item.value))
		}
	}
}

func (m *MarkdownMetadata) isAllowedKey(key string) bool {
	key = strings.ToLower(key)
	return m.allowedTopKeys[key] || m.allowedBottomKeys[key]
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

	if !m.isAllowedKey(key) {
		return false
	}

	item := &markdownMetadataItem{key, value}
	m.items = append(m.items, item)
	return true
}

func (m *MarkdownMetadata) ParseLines(lines []string) int {
	parsed := 0
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			parsed++
			continue
		}
		if !strings.Contains(trimmedLine, ":") { // TODO: multi-line metadata
			break
		}
		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(strings.Fields(parts[0])) > 5 {
			break
		}

		var item *markdownMetadataItem
		if len(parts) == 1 {
			item = &markdownMetadataItem{strings.TrimSpace(parts[0]), ""}
		} else {
			item = &markdownMetadataItem{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}
		}
		m.items = append(m.items, item)
		parsed++
	}
	for i := parsed - 1; i >= 0; i-- {
		trimmedLine := strings.TrimSpace(lines[i])
		if trimmedLine == "" {
			parsed--
		} else {
			break
		}
	}
	return parsed
}

func (m *MarkdownMetadata) TopEmpty() bool {
	return m.empty(m.allowedTopKeys)
}

func (m *MarkdownMetadata) BottomEmpty() bool {
	return m.bottomGroupLine == "" && m.empty(m.allowedBottomKeys)
}

func (m *MarkdownMetadata) empty(allowedKeys map[string]bool) bool {
	for _, item := range m.items {
		if allowedKeys[strings.ToLower(item.key)] {
			return false
		}
	}
	return true
}

func (m *MarkdownMetadata) SetBottomGroupLine(line string) {
	m.bottomGroupLine = line
}
