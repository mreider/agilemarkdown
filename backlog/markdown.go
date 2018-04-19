package backlog

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

const (
	CreatedHeader  = "created"
	ModifiedHeader = "modified"
)

type MarkdownContent struct {
	contentPath string

	isDirty       bool
	lines         []string
	headerIndexes map[string]int
}

func CreateMarkdown(contentPath string, headers map[string]struct{}) (*MarkdownContent, error) {
	var err error
	if _, err = os.Stat(contentPath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var data []byte
	if err == nil {
		itemFile, err := os.Open(contentPath)
		if err != nil {
			return nil, err
		}
		defer itemFile.Close()
		data, err = ioutil.ReadAll(itemFile)
		if err != nil {
			return nil, err
		}
	}

	content := &MarkdownContent{contentPath: contentPath, headerIndexes: make(map[string]int, len(headers))}
	if len(data) > 0 {
		content.lines = strings.Split(string(data), "\n")
		for i := range content.lines {
			header, _ := content.getHeaderAndValue(i)
			if _, ok := headers[header]; ok {
				if _, ok := content.headerIndexes[header]; !ok {
					content.headerIndexes[header] = i
				}
			}
		}
	}
	return content, nil
}

func (content *MarkdownContent) Save() error {
	if !content.isDirty {
		return nil
	}

	currentTime := getCurrentTimestamp()
	if _, ok := content.headerIndexes[CreatedHeader]; ok && content.Value(CreatedHeader) == "" {
		content.SetValue(CreatedHeader, currentTime)
	}
	if _, ok := content.headerIndexes[ModifiedHeader]; ok {
		content.SetValue(ModifiedHeader, currentTime)
	}
	contentFile, err := os.OpenFile(content.contentPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer contentFile.Close()
	for i, line := range content.lines {
		contentFile.WriteString(line)
		if i < len(content.lines)-1 {
			contentFile.WriteString("\n")
		}
	}
	content.isDirty = false
	return nil
}

func (content *MarkdownContent) Value(header string) string {
	if headerIndex, ok := content.headerIndexes[header]; !ok {
		return ""
	} else {
		_, value := content.getHeaderAndValue(headerIndex)
		return value
	}
}

func (content *MarkdownContent) SetValue(header, value string) {
	if headerIndex, ok := content.headerIndexes[header]; !ok {
		content.addValue(header, value)
	} else {
		content.setValue(headerIndex, value)
	}
}

func (content *MarkdownContent) getHeaderAndValue(headerIndex int) (string, string) {
	parts := strings.SplitN(content.lines[headerIndex], ":", 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func (content *MarkdownContent) setValue(headerIndex int, value string) {
	header, oldValue := content.getHeaderAndValue(headerIndex)
	if value != oldValue {
		content.lines[headerIndex] = fmt.Sprintf("%s: %s", header, value)
		content.isDirty = true
	}
}

func (content *MarkdownContent) addValue(header, value string) {
	if len(content.lines) > 0 {
		content.lines = append(content.lines, "")
	}
	content.lines = append(content.lines, fmt.Sprintf("%s: %s", header, value))
	content.isDirty = true
	content.headerIndexes[header] = len(content.lines) - 1
}
