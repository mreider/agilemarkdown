package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

const (
	markdownData = `# Test backlog

Root: qwerty

[title1](link1) [title2](link2)

Data: test

### Doing
Story 1 [link1](link1.md) (points) (assigned)  
Story 2 [link2](link2.md) (points) (assigned)  

### Planned
Story 5 [link5](link5.md) (points) (assigned)  
Story 6 [link6](link6.md) (points) (assigned)  
Story 7 [link7](link7.md) (points) (assigned)  

### Unplanned
Story 4 [link4](link4.md) (points) (assigned)  
Story 3 [link3](link3.md) (points) (assigned)  

### Finished
Story 8 [link8](link8.md) (points) (assigned)  

[Archived stories](archive.md)

### Metadata

Data2: test2  
`
)

func TestMarkdownLoad(t *testing.T) {
	content := markdown.NewMarkdown(markdownData, "", []*regexp.Regexp{markdown.AllowedKeyAsRegex("Data")}, []*regexp.Regexp{markdown.AllowedKeyAsRegex("Data2")}, "### ", backlog.OverviewFooterRe)
	assert.Equal(t, "Test backlog", content.Title())
	assert.Equal(t, "Root: qwerty", content.Header())
	assert.Equal(t, "[title1](link1) [title2](link2)", content.Links())
	assert.Equal(t, 4, content.GroupCount())

	assert.Equal(t, "Doing", content.Group("Doing").Title())
	assert.Equal(t, 2, content.Group("Doing").Count())
	assert.Equal(t, "Planned", content.Group("Planned").Title())
	assert.Equal(t, 3, content.Group("Planned").Count())
	assert.Equal(t, "Unplanned", content.Group("Unplanned").Title())
	assert.Equal(t, 2, content.Group("Unplanned").Count())
	assert.Equal(t, "Finished", content.Group("Finished").Title())
	assert.Equal(t, 1, content.Group("Finished").Count())

	assert.Equal(t, "Story 1 [link1](link1.md) (points) (assigned)  ", content.Group("Doing").Line(0))
	assert.Equal(t, "Story 7 [link7](link7.md) (points) (assigned)  ", content.Group("Planned").Line(2))
	assert.Equal(t, "Story 3 [link3](link3.md) (points) (assigned)  ", content.Group("Unplanned").Line(1))
	assert.Equal(t, "Story 8 [link8](link8.md) (points) (assigned)  ", content.Group("Finished").Line(0))

	assert.Equal(t, 2, len(content.Footer()))
	assert.Equal(t, "[Archived stories](archive.md)", content.Footer()[0])
	assert.Equal(t, "", content.Footer()[1])
}

func TestMarkdownSave(t *testing.T) {
	updatedData := `# New backlog

Test: header

[title1](link1) [title22](link22) [title3](link3)

Data: test  

### Doing
Story 1 [link1](link1.md) 12 Mike
Story 2 [link2](link2.md) (points) (assigned)  

### Planned
Story 5 [link5](link5.md) (points) (assigned)  
Story 6 [link6](link6.md) (points) (assigned)  
Story 7 [link7](link7.md) (points) (assigned)  
Story 9 [link9](link9.md) 9 Robert

### Unplanned
Story 4 [link4](link4.md) (points) (assigned)  

### Finished


footer1
footer2

### Metadata

Data2: test2  
`

	content := markdown.NewMarkdown(markdownData, "", []*regexp.Regexp{markdown.AllowedKeyAsRegex("Data")}, []*regexp.Regexp{markdown.AllowedKeyAsRegex("Data2")}, "### ", backlog.OverviewFooterRe)
	content.SetTitle("New backlog")
	content.SetHeader("Test: header")
	content.SetLinks("[title1](link1) [title22](link22) [title3](link3)")
	content.Group("Doing").SetLine(0, "Story 1 [link1](link1.md) 12 Mike")
	content.Group("Planned").AddLine("Story 9 [link9](link9.md) 9 Robert")
	content.Group("Unplanned").DeleteLine(1)
	content.Group("Finished").DeleteLine(0)
	content.SetFooter([]string{"footer1", "footer2", ""})

	assert.Equal(t, updatedData, string(content.Content()))
}
