package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	markdownData = `Title: test backlog
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
`
)

func TestMarkdownLoad(t *testing.T) {
	content := backlog.NewMarkdown(markdownData, "", []string{"Title", "Data"}, true)
	assert.Equal(t, "test backlog", content.MetadataValue("title"))
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
}

func TestMarkdownSave(t *testing.T) {
	updatedData := `Title: new backlog  
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

`

	content := backlog.NewMarkdown(markdownData, "", []string{"Title", "Data"}, true)
	content.SetMetadataValue("title", "new backlog")
	content.Group("Doing").SetLine(0, "Story 1 [link1](link1.md) 12 Mike")
	content.Group("Planned").AddLine("Story 9 [link9](link9.md) 9 Robert")
	content.Group("Unplanned").DeleteLine(1)
	content.Group("Finished").DeleteLine(0)

	assert.Equal(t, updatedData, string(content.Content("")))
}
