package markdown

import (
	"fmt"
	"regexp"
	"strings"
)

type Metadata struct {
	allowedTopKeys    []*regexp.Regexp
	allowedBottomKeys []*regexp.Regexp
	items             []*metadataItem
	bottomGroupLine   string
}

type metadataItem struct {
	key   string
	value string
}

func NewMetadata(allowedTopKeys, allowedBottomKeys []*regexp.Regexp) *Metadata {
	metadata := &Metadata{
		allowedTopKeys:    allowedTopKeys,
		allowedBottomKeys: allowedBottomKeys,
	}
	return metadata
}

func (m *Metadata) TopRawLines() []string {
	result := make([]string, 0, len(m.items))
	m.fillRawLines(&result, m.allowedTopKeys)
	return result
}

func (m *Metadata) BottomRawLines() []string {
	result := make([]string, 0, len(m.items))
	bottomGroupLine := m.bottomGroupLine
	if bottomGroupLine == "" {
		bottomGroupLine = "## Metadata"
	}
	result = append(result, bottomGroupLine, "")
	m.fillRawLines(&result, m.allowedBottomKeys)
	return result
}

func (m *Metadata) fillRawLines(lines *[]string, allowedKeys []*regexp.Regexp) {
	for _, item := range m.items {
		if m.isAllowedKey(item.key, allowedKeys) {
			*lines = append(*lines, fmt.Sprintf("%s: %s  ", item.key, item.value))
		}
	}
}

func (m *Metadata) IsAllowedKey(key string) bool {
	return m.isAllowedKey(key, m.allowedTopKeys) || m.isAllowedKey(key, m.allowedBottomKeys)
}

func (m *Metadata) isAllowedKey(key string, allowedKeys []*regexp.Regexp) bool {
	for _, allowedKey := range allowedKeys {
		if allowedKey.MatchString(key) {
			return true
		}
	}
	return false
}

func (m *Metadata) Value(key string) string {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			return item.value
		}
	}
	return ""
}

func (m *Metadata) SetValue(key, value string) bool {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			item.value = value
			return true
		}
	}

	if !m.IsAllowedKey(key) {
		return false
	}

	item := &metadataItem{key, value}
	m.items = append(m.items, item)
	return true
}

func (m *Metadata) ReplaceKey(oldKey, newKey string) bool {
	for _, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(oldKey) {
			item.key = newKey
			return true
		}
	}
	return false
}

func (m *Metadata) Remove(key string) bool {
	for i, item := range m.items {
		if strings.ToLower(item.key) == strings.ToLower(key) {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return true
		}
	}
	return false
}

func (m *Metadata) ParseLines(lines []string) int {
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

		var item *metadataItem
		if len(parts) == 1 {
			item = &metadataItem{strings.TrimSpace(parts[0]), ""}
		} else {
			item = &metadataItem{strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])}
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

func (m *Metadata) TopEmpty() bool {
	return m.empty(m.allowedTopKeys)
}

func (m *Metadata) BottomEmpty() bool {
	return m.bottomGroupLine == "" && m.empty(m.allowedBottomKeys)
}

func (m *Metadata) empty(allowedKeys []*regexp.Regexp) bool {
	for _, item := range m.items {
		if m.isAllowedKey(item.key, allowedKeys) {
			return false
		}
	}
	return true
}

func (m *Metadata) SetBottomGroupLine(line string) {
	m.bottomGroupLine = line
}

func AllowedKeyAsRegex(allowedKey string) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf("(?i)^%s$", strings.ToLower(strings.TrimSpace(allowedKey))))
}
