package backlog

import (
	"fmt"
	"regexp"
	"strings"
)

type MarkdownMetadata struct {
	allowedTopKeys    []*regexp.Regexp
	allowedBottomKeys []*regexp.Regexp
	items             []*markdownMetadataItem
	bottomGroupLine   string
}

type markdownMetadataItem struct {
	key   string
	value string
}

func NewMarkdownMetadata(allowedTopKeys, allowedBottomKeys []*regexp.Regexp) *MarkdownMetadata {
	metadata := &MarkdownMetadata{
		allowedTopKeys:    allowedTopKeys,
		allowedBottomKeys: allowedBottomKeys,
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

func (m *MarkdownMetadata) fillRawLines(lines *[]string, allowedKeys []*regexp.Regexp) {
	for _, item := range m.items {
		if m.isAllowedKey(item.key, allowedKeys) {
			*lines = append(*lines, fmt.Sprintf("%s: %s  ", item.key, item.value))
		}
	}
}

func (m *MarkdownMetadata) IsAllowedKey(key string) bool {
	return m.isAllowedKey(key, m.allowedTopKeys) || m.isAllowedKey(key, m.allowedBottomKeys)
}

func (m *MarkdownMetadata) isAllowedKey(key string, allowedKeys []*regexp.Regexp) bool {
	for _, allowedKey := range allowedKeys {
		if allowedKey.MatchString(key) {
			return true
		}
	}
	return false
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

func (m *MarkdownMetadata) ReplaceKey(oldKey, newKey string) bool {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(oldKey) {
			item.key = newKey
			return true
		}
	}
	return false
}

func (m *MarkdownMetadata) Remove(key string) bool {
	for i, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return true
		}
	}
	return false
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

func (m *MarkdownMetadata) empty(allowedKeys []*regexp.Regexp) bool {
	for _, item := range m.items {
		if m.isAllowedKey(item.key, allowedKeys) {
			return false
		}
	}
	return true
}

func (m *MarkdownMetadata) SetBottomGroupLine(line string) {
	m.bottomGroupLine = line
}

func AllowedKeyAsRegex(allowedKey string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("(?i)^%s$", strings.ToLower(strings.TrimSpace(allowedKey))))
}
