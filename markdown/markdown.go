package markdown

import (
	"bytes"
	"fmt"
	"github.com/mreider/agilemarkdown/utils"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strings"
)

var (
	linksRe         = regexp.MustCompile(`^(\[[^])]+]\([^])]+\)(\s*(•|(\|\|))?\s*)?)+$`)
	metadataGroupRe = regexp.MustCompile(`^#+\s*metadata\s*$`)
)

type Content struct {
	contentPath      string
	groupTitlePrefix string

	isDirty  bool
	title    string
	header   string
	links    string
	metadata *Metadata
	groups   []*Group
	freeText []string
	footer   []string

	HideEmptyGroups bool
}

func LoadMarkdown(markdownPath string, topMetadataKeys, bottomMetadataKeys []*regexp.Regexp, groupTitlePrefix string, footerRe *regexp.Regexp) (*Content, error) {
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
	return NewMarkdown(string(data), markdownPath, topMetadataKeys, bottomMetadataKeys, groupTitlePrefix, footerRe), nil
}

func NewMarkdown(data, markdownPath string, topMetadataKeys, bottomMetadataKeys []*regexp.Regexp, groupTitlePrefix string, footerRe *regexp.Regexp) *Content {
	content := &Content{contentPath: markdownPath, groupTitlePrefix: groupTitlePrefix, metadata: NewMetadata(topMetadataKeys, bottomMetadataKeys)}
	if len(data) > 0 {
		lines := strings.Split(data, "\n")

		bottomMetadataGroupIndex := content.findBottomMetadataGroup(lines)
		if bottomMetadataGroupIndex >= 0 {
			content.metadata.SetBottomGroupLine(lines[bottomMetadataGroupIndex])
			if bottomMetadataGroupIndex < len(lines)-1 {
				content.metadata.ParseLines(lines[bottomMetadataGroupIndex+1:])
			}
			lines = lines[:bottomMetadataGroupIndex]
		}

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
						key := strings.TrimSpace(parts[0])
						if content.metadata.IsAllowedKey(key) {
							break NextLine
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
			var currentGroup *Group
			for _, line := range lines[parsed:] {
				if strings.HasPrefix(line, groupTitlePrefix) {
					if currentGroup != nil {
						content.AddGroup(currentGroup)
					}
					currentGroup = &Group{content: content, title: strings.TrimSpace(strings.TrimPrefix(line, groupTitlePrefix))}
				} else if currentGroup != nil {
					if footerRe != nil && footerRe.MatchString(line) {
						content.AddGroup(currentGroup)
						currentGroup = nil
						content.footer = append(content.footer, line)
					} else {
						if strings.TrimSpace(line) != "" {
							currentGroup.lines = append(currentGroup.lines, line)
						}
					}
				} else {
					if len(content.footer) == 0 && footerRe != nil && footerRe.MatchString(line) {
						content.footer = append(content.footer, line)
					} else if len(content.footer) > 0 {
						content.footer = append(content.footer, line)
					} else {
						content.freeText = append(content.freeText, line)
					}
				}
			}
			if currentGroup != nil {
				content.AddGroup(currentGroup)
			}
		} else {
			for _, line := range lines[parsed:] {
				if len(content.footer) == 0 && footerRe != nil && footerRe.MatchString(line) {
					content.footer = append(content.footer, line)
				} else if len(content.footer) > 0 {
					content.footer = append(content.footer, line)
				} else {
					content.freeText = append(content.freeText, line)
				}

			}
		}
	}
	content.isDirty = false
	return content
}

func (content *Content) Save() error {
	if content.contentPath == "" {
		return nil
	}
	if !content.isDirty {
		return nil
	}
	data := content.Content()
	err := ioutil.WriteFile(content.contentPath, data, 0644)
	if err != nil {
		return err
	}
	content.isDirty = false
	return nil
}

func (content *Content) Content() []byte {
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
	if !content.metadata.TopEmpty() {
		result.WriteString("\n")
		result.WriteString(strings.Join(content.metadata.TopRawLines(), "\n"))
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
	if !content.metadata.BottomEmpty() {
		result.WriteString("\n")
		result.WriteString(strings.Join(content.metadata.BottomRawLines(), "\n"))
		result.WriteString("\n")
	}
	return result.Bytes()
}

func (content *Content) MetadataValue(key string) string {
	return content.metadata.Value(key)
}

func (content *Content) SetMetadataValue(key, value string) {
	if content.metadata.SetValue(key, value) {
		content.markDirty()
	}
}

func (content *Content) ReplaceMetadataKey(oldKey, newKey string) {
	if content.metadata.ReplaceKey(oldKey, newKey) {
		content.markDirty()
	}
}

func (content *Content) RemoveMetadata(key string) {
	if content.metadata.Remove(key) {
		content.markDirty()
	}
}

func (content *Content) GroupCount() int {
	return len(content.groups)
}

func (content *Content) Group(title string) *Group {
	title = strings.ToLower(title)
	for _, group := range content.groups {
		if strings.ToLower(group.title) == title {
			return group
		}
	}
	return nil
}

func (content *Content) AddGroup(group *Group) {
	content.groups = append(content.groups, group)
	content.markDirty()
}

func (content *Content) RemoveGroup(title string) {
	title = strings.ToLower(title)
	for i, group := range content.groups {
		if strings.ToLower(group.title) == title {
			content.groups = append(content.groups[:i], content.groups[i+1:]...)
			content.markDirty()
			break
		}
	}
}

func (content *Content) SetFreeText(freeText []string) {
	if utils.AreEqualStrings(content.freeText, freeText) {
		return
	}

	content.freeText = freeText
	content.markDirty()
}

func (content *Content) Footer() []string {
	return content.footer
}

func (content *Content) SetFooter(footer []string) {
	if utils.AreEqualStrings(content.footer, footer) {
		return
	}

	content.footer = footer
	content.markDirty()
}

func (content *Content) markDirty() {
	content.isDirty = true
}

func (content *Content) Title() string {
	return content.title
}

func (content *Content) SetTitle(title string) {
	if content.title != title {
		content.title = title
		content.markDirty()
	}
}

func (content *Content) Links() string {
	return content.links
}

func (content *Content) SetLinks(links string) {
	if content.links != links {
		content.links = links
		content.markDirty()
	}
}

func (content *Content) Header() string {
	return content.header
}

func (content *Content) SetHeader(header string) {
	if content.header != header {
		content.header = header
		content.markDirty()
	}
}

func (content *Content) findBottomMetadataGroup(lines []string) int {
	metadataGroupIndex := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if metadataGroupRe.MatchString(strings.ToLower(lines[i])) {
			metadataGroupIndex = i
			break
		}
	}
	return metadataGroupIndex
}

func (content *Content) IsDirty() bool {
	return content.isDirty
}

func (content *Content) ContentPath() string {
	return content.contentPath
}

func (content *Content) SetContentPath(contentPath string) {
	content.contentPath = contentPath
}

func (content *Content) Metadata() *Metadata {
	return content.metadata
}

func (content *Content) FreeText() []string {
	return content.freeText
}

func (content *Content) SortGroups(less func(group1, group2 *Group, i, j int) bool) {
	sort.Slice(content.groups, func(i, j int) bool {
		return less(content.groups[i], content.groups[j], i, j)
	})
}
