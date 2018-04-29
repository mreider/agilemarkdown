package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	markdownOverviewData = `Title: test backlog  

### Flying
Story 1 [Story1](Story1.md) - -  
Story 2 [Story2](Story2.md) - -  

### Gate
Story 5 [Story5](Story5.md) - -  
Story 6 [Story6](Story6.md) - -  
Story 7 [Story7](Story7.md) - -  

### Hangar
Story 4 [Story4](Story4.md) - -  
Story 3 [Story3](Story3.md) - -  

### Landed
Story 8 [Story8](Story8.md) - -  
`
)

func TestOverviewCreate(t *testing.T) {
	markdown := backlog.NewMarkdown(markdownOverviewData, "", []string{"Title", "Data"})
	overview := backlog.NewBacklogOverview(markdown)
	itemsByStatus := overview.ItemsByStatus()
	assert.Equal(t, 4, len(itemsByStatus))
	assert.Equal(t, []string{"Story1", "Story2"}, itemsByStatus["flying"])
	assert.Equal(t, []string{"Story5", "Story6", "Story7"}, itemsByStatus["gate"])
	assert.Equal(t, []string{"Story4", "Story3"}, itemsByStatus["hangar"])
	assert.Equal(t, []string{"Story8"}, itemsByStatus["landed"])
}

func TestOverviewUpdate(t *testing.T) {
	updatedOverviewData := `Title: new backlog  

### Flying
First story [Story1](Story1.md) 10 mike  
Story 5 [Story5](Story5.md) 20 -  

### Gate
Story 7 [Story7](Story7.md) - -  

### Hangar
Story 4 [Story4](Story4.md) 30 -  
Story Six [Story6](Story6.md) - -  

### Landed
Story 8 [Story8](Story8.md) - -  
Second story [Story2](Story2.md) 15 robert  
`

	markdown := backlog.NewMarkdown(markdownOverviewData, "", []string{"Title", "Data"})
	overview := backlog.NewBacklogOverview(markdown)
	overview.SetTitle("new backlog")

	story1 := createBacklogItem("Story1", "First story", "flying", "10", "mike")
	story2 := createBacklogItem("Story2", "Second story", "landed", "15", "robert")
	story5 := createBacklogItem("Story5", "Story 5", "flying", "20", "")
	story6 := createBacklogItem("Story6", "Story Six", "hangar", "", "")
	story7 := createBacklogItem("Story7", "Story 7", "gate", "", "")
	story4 := createBacklogItem("Story4", "Story 4", "hangar", "30", "")
	story8 := createBacklogItem("Story8", "Story 8", "landed", "", "")
	overview.Update([]*backlog.BacklogItem{
		story1, story2, story5, story6, story7, story4, story8,
	})

	updatedData := string(overview.Content(""))
	assert.Equal(t, updatedOverviewData, updatedData)
}

func createBacklogItem(name, title, status, points, assigned string) *backlog.BacklogItem {
	item := backlog.NewBacklogItem(name)
	item.SetTitle(title)
	item.SetStatus(status)
	item.SetEstimate(points)
	item.SetAssigned(assigned)
	return item
}
