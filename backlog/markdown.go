package backlog

import (
	"bytes"
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const (
	CreatedMetadataKey  = "Created"
	ModifiedMetadataKey = "Modified"
)

var (
	linksRe = regexp.MustCompile(`^(\[[^])]+]\([^])]+\)(\s*(•)?\s*)?)+$`)
)

type MarkdownContent struct {
	contentPath      string
	groupTitlePrefix string

	isDirty  bool
	title    string
	header   string
	links    string
	metadata *MarkdownMetadata
	groups   []*MarkdownGroup
	freeText []string
	footer   []string

	HideEmptyGroups bool
}

func LoadMarkdown(markdownPath string, metadataKeys []string, groupTitlePrefix string, footerRe *regexp.Regexp) (*MarkdownContent, error) {
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
	return NewMarkdown(string(data), markdownPath, metadataKeys, groupTitlePrefix, footerRe), nil
}

func NewMarkdown(data, markdownPath string, metadataKeys []string, groupTitlePrefix string, footerRe *regexp.Regexp) *MarkdownContent {
	content := &MarkdownContent{contentPath: markdownPath, groupTitlePrefix: groupTitlePrefix, metadata: NewMarkdownMetadata(metadataKeys)}
	if len(data) > 0 {
		lines := strings.Split(data, "\n")
		metadataIndex := 0
		if strings.HasPrefix(lines[0], "# ") {
			content.title = strings.TrimSpace(strings.TrimPrefix(lines[0], "# "))
			metadataIndex = 1
		}
	NextLine:
		for metadataIndex < len(lines) {
			line := strings.TrimSpace(lines[metadataIndex])
			if linksRe.MatchString(line) {
				content.links = line
				metadataIndex++
				break
			}
			if line != "" {
				if strings.HasPrefix(line, "#") {
					break
				}
				if content.header == "" {
					if !strings.Contains(line, ":") {
						content.header = line
					} else {
						parts := strings.SplitN(line, ":", 2)
						key := strings.ToLower(strings.TrimSpace(parts[0]))
						for _, mKey := range metadataKeys {
							if key == strings.ToLower(mKey) {
								break NextLine
							}
						}
						content.header = line
					}
				} else {
					break
				}
			}
			metadataIndex++
		}
		parsed := content.metadata.ParseLines(lines[metadataIndex:]) + metadataIndex

		if groupTitlePrefix != "" {
			var currentGroup *MarkdownGroup
			for _, line := range lines[parsed:] {
				if strings.HasPrefix(line, groupTitlePrefix) {
					if currentGroup != nil {
						content.addGroup(currentGroup)
					}
					currentGroup = &MarkdownGroup{content: content, title: strings.TrimSpace(strings.TrimPrefix(line, groupTitlePrefix))}
				} else if currentGroup != nil {
					if footerRe != nil && footerRe.MatchString(line) {
						content.addGroup(currentGroup)
						currentGroup = nil
						content.footer = append(content.footer, line)
					} else {
						if strings.TrimSpace(line) != "" {
							currentGroup.lines = append(currentGroup.lines, line)
						}
					}
				} else {
					if len(content.footer) > 0 {
						content.footer = append(content.footer, line)
					} else {
						content.freeText = append(content.freeText, line)
					}
				}
			}
			if currentGroup != nil {
				content.addGroup(currentGroup)
			}
		} else {
			content.freeText = lines[parsed:]
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
	data := content.Content(utils.GetCurrentTimestamp())
	err := ioutil.WriteFile(content.contentPath, data, 0644)
	if err != nil {
		return err
	}
	content.isDirty = false
	return nil
}

func (content *MarkdownContent) Content(timestamp string) []byte {
	emptyCreated := content.MetadataValue(CreatedMetadataKey) == ""
	if content.metadata.IsAllowedKey(CreatedMetadataKey) && emptyCreated {
		content.SetMetadataValue(CreatedMetadataKey, timestamp)
	}
	if content.metadata.IsAllowedKey(ModifiedMetadataKey) {
		if content.MetadataValue(ModifiedMetadataKey) != "" || emptyCreated {
			content.SetMetadataValue(ModifiedMetadataKey, timestamp)
		} else {
			content.SetMetadataValue(ModifiedMetadataKey, content.MetadataValue(CreatedMetadataKey))
		}
	}
	result := bytes.NewBuffer(nil)
	if content.title != "" {
		result.WriteString(fmt.Sprintf("# %s", content.title))
		result.WriteString("\n")
	}
	if content.header != "" {
		result.WriteString("\n")
		result.WriteString(content.header)
		result.WriteString("\n")
	}
	if content.links != "" {
		result.WriteString("\n")
		result.WriteString(content.links)
		result.WriteString("\n")
	}
	if !content.metadata.Empty() {
		result.WriteString("\n")
		result.WriteString(strings.Join(content.metadata.RawLines(), "\n"))
		result.WriteString("\n")
	}
	for i, line := range content.freeText {
		result.WriteString(line)
		if i < len(content.freeText)-1 {
			result.WriteString("\n")
		}
	}
	if len(content.groups) > 0 {
		for _, group := range content.groups {
			lines := group.RawLines()
			var nonEmptyLineCount int
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					nonEmptyLineCount++
				}
			}
			if nonEmptyLineCount > 2 || !content.HideEmptyGroups {
				result.WriteString("\n")
				result.WriteString(fmt.Sprintf("%s%s", content.groupTitlePrefix, group.title))
				result.WriteString("\n")
				result.WriteString(strings.Join(lines, "\n"))
				result.WriteString("\n")
			}
		}
	}
	if len(content.footer) > 0 {
		result.WriteString("\n")
		for i, line := range content.footer {
			result.WriteString(line)
			if i < len(content.footer)-1 {
				result.WriteString("\n")
			}
		}
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
	title = strings.ToLower(title)
	for _, group := range content.groups {
		if strings.ToLower(group.title) == title {
			return group
		}
	}
	return nil
}

func (content *MarkdownContent) addGroup(group *MarkdownGroup) {
	content.groups = append(content.groups, group)
	content.markDirty()
}

func (content *MarkdownContent) SetFreeText(freeText []string) {
	if utils.AreEqualStrings(content.freeText, freeText) {
		return
	}

	content.freeText = freeText
	content.markDirty()
}

func (content *MarkdownContent) Footer() []string {
	return content.footer
}

func (content *MarkdownContent) SetFooter(footer []string) {
	if utils.AreEqualStrings(content.footer, footer) {
		return
	}

	content.footer = footer
	content.markDirty()
}

func (content *MarkdownContent) markDirty() {
	content.isDirty = true
}

func (content *MarkdownContent) Title() string {
	return content.title
}

func (content *MarkdownContent) SetTitle(title string) {
	if content.title != title {
		content.title = title
		content.markDirty()
	}
}

func (content *MarkdownContent) Links() string {
	return content.links
}

func (content *MarkdownContent) SetLinks(links string) {
	if content.links != links {
		content.links = links
		content.markDirty()
	}
}

func (content *MarkdownContent) Header() string {
	return content.header
}

func (content *MarkdownContent) SetHeader(header string) {
	if content.header != header {
		content.header = header
		content.markDirty()
	}
}
