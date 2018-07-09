package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

const (
	markdownOverviewData = `# Test backlog  

### Doing
Story 1 [Story1](Story1) - -  
Story 2 [Story2](Story2) - -  

### Unplanned
Story 4 [Story4](Story4) - -  
Story 3 [Story3](Story3) - -  

### Planned
Story 5 [Story5](Story5) - -  
Story 6 [Story6](Story6) - -  
Story 7 [Story7](Story7) - -  

### Finished
Story 8 [Story8](Story8) - -  

[Archived stories](archive.md)

### Metadata

Data2: test2  
`
)

func TestOverviewCreate(t *testing.T) {
	markdown := backlog.NewMarkdown(markdownOverviewData, "",
		[]*regexp.Regexp{backlog.AllowedKeyAsRegex("Title"), backlog.AllowedKeyAsRegex("Data")},
		[]*regexp.Regexp{backlog.AllowedKeyAsRegex("Data2")},
		"### ", backlog.OverviewFooterRe)
	overview := backlog.NewBacklogOverview(markdown)
	sorter := backlog.NewBacklogItemsSorter(overview)
	itemsByStatus := sorter.SortedItemsByStatus()
	assert.Equal(t, 4, len(itemsByStatus))
	assert.Equal(t, []string{"Story1", "Story2"}, itemsByStatus["doing"])
	assert.Equal(t, []string{"Story5", "Story6", "Story7"}, itemsByStatus["planned"])
	assert.Equal(t, []string{"Story4", "Story3"}, itemsByStatus["unplanned"])
	assert.Equal(t, []string{"Story8"}, itemsByStatus["finished"])
}

func TestOverviewUpdate(t *testing.T) {
	updatedOverviewData := `# New backlog

### Doing
| User | Title | Points | Tags |
|---|---|:---:|---|
| mike | [First story](Story1) | 10 |  |
|  | [Story 5](Story5) | 20 |  |

### Planned
| User | Title | Points | Tags |
|---|---|:---:|---|
|  | [Story 7](Story7) |  |  |

### Unplanned
| User | Title | Points | Tags |
|---|---|:---:|---|
|  | [Story 4](Story4) | 30 |  |
|  | [Story Six](Story6) |  |  |

### Finished
| User | Title | Points | Tags |
|---|---|:---:|---|
|  | [Story 8](Story8) |  |  |
| robert | [Second story](Story2) | 15 |  |

[Archived stories](archive.md)

### Metadata

Data2: test2  
`

	markdown := backlog.NewMarkdown(markdownOverviewData, "",
		[]*regexp.Regexp{backlog.AllowedKeyAsRegex("Title"), backlog.AllowedKeyAsRegex("Data")},
		[]*regexp.Regexp{backlog.AllowedKeyAsRegex("Data2")},
		"### ", backlog.OverviewFooterRe)
	overview := backlog.NewBacklogOverview(markdown)
	sorter := backlog.NewBacklogItemsSorter(overview)
	overview.SetTitle("New backlog")

	story1 := createBacklogItem("Story1", "First story", "doing", "10", "mike")
	story2 := createBacklogItem("Story2", "Second story", "finished", "15", "robert")
	story5 := createBacklogItem("Story5", "Story 5", "doing", "20", "")
	story6 := createBacklogItem("Story6", "Story Six", "unplanned", "", "")
	story7 := createBacklogItem("Story7", "Story 7", "planned", "", "")
	story4 := createBacklogItem("Story4", "Story 4", "unplanned", "30", "")
	story8 := createBacklogItem("Story8", "Story 8", "finished", "", "")
	overview.Update([]*backlog.BacklogItem{
		story1, story2, story5, story6, story7, story4, story8,
	}, sorter)

	updatedData := string(overview.Content())
	assert.Equal(t, updatedOverviewData, updatedData)
}

func createBacklogItem(name, title, status, points, assigned string) *backlog.BacklogItem {
	item := backlog.NewBacklogItem(name, "")
	item.SetTitle(title)
	item.SetStatus(backlog.StatusByName(status))
	item.SetEstimate(points)
	item.SetAssigned(assigned)
	return item
}
