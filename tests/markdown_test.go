package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	markdownData = `title: test backlog

data: test

### Flying

Story 1 [link1] (points) (assigned)

Story 2 [link2] (points) (assigned)

### Gate

Story 5 [link5] (points) (assigned)

Story 6 [link6] (points) (assigned)

Story 7 [link7] (points) (assigned)

### Hangar

Story 4 [link4] (points) (assigned)

Story 3 [link3] (points) (assigned)

### Landed

Story 8 [link8] (points) (assigned)
`
)

func TestMarkdownLoad(t *testing.T) {
	content := backlog.NewMarkdown(markdownData, "", []string{"title", "data"})
	assert.Equal(t, "test backlog", content.FieldValue("title"))
	assert.Equal(t, 4, content.GroupCount())

	assert.Equal(t, "Flying", content.Group("Flying").Title())
	assert.Equal(t, 2, content.Group("Flying").Count())
	assert.Equal(t, "Gate", content.Group("Gate").Title())
	assert.Equal(t, 3, content.Group("Gate").Count())
	assert.Equal(t, "Hangar", content.Group("Hangar").Title())
	assert.Equal(t, 2, content.Group("Hangar").Count())
	assert.Equal(t, "Landed", content.Group("Landed").Title())
	assert.Equal(t, 1, content.Group("Landed").Count())

	assert.Equal(t, "Story 1 [link1] (points) (assigned)", content.Group("Flying").Line(0))
	assert.Equal(t, "Story 7 [link7] (points) (assigned)", content.Group("Gate").Line(2))
	assert.Equal(t, "Story 3 [link3] (points) (assigned)", content.Group("Hangar").Line(1))
	assert.Equal(t, "Story 8 [link8] (points) (assigned)", content.Group("Landed").Line(0))
}

func TestMarkdownSave(t *testing.T) {
	updatedData := `title: new backlog

data: test

### Flying

Story 1 [link1] 12 Mike

Story 2 [link2] (points) (assigned)

### Gate

Story 5 [link5] (points) (assigned)

Story 6 [link6] (points) (assigned)

Story 7 [link7] (points) (assigned)

Story 9 [link9] 9 Robert

### Hangar

Story 4 [link4] (points) (assigned)

### Landed

`

	content := backlog.NewMarkdown(markdownData, "", []string{"title", "data"})
	content.SetFieldValue("title", "new backlog")
	content.Group("Flying").SetLine(0, "Story 1 [link1] 12 Mike")
	content.Group("Gate").AddLine("Story 9 [link9] 9 Robert")
	content.Group("Hangar").DeleteLine(1)
	content.Group("Landed").DeleteLine(0)

	assert.Equal(t, updatedData, string(content.Content("")))
}
