package tests

import (
	"testing"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/markdown"
	"github.com/stretchr/testify/assert"
)

const markdownOverviewData = `# Test backlog

### Started
Story 1 [Story1](Story1) - -
Story 2 [Story2](Story2) - -

### Unstarted
Story 4 [Story4](Story4) - -
Story 3 [Story3](Story3) - -
Story 5 [Story5](Story5) - -
Story 6 [Story6](Story6) - -
Story 7 [Story7](Story7) - -

### Finished
Story 8 [Story8](Story8) - -

[Archived stories](archive.md)

### Metadata

Data2: test2
`

func TestOverviewCreate(t *testing.T) {
	content := markdown.NewMarkdown(markdownOverviewData, "",
		[]string{"Title", "Data"},
		[]string{"Data2"},
		"### ", backlog.OverviewFooterRe)
	overview := backlog.NewBacklogOverview(content)
	sorter := backlog.NewBacklogItemsSorter(overview)
	itemsByStatus := sorter.SortedItemsByStatus()
	assert.Equal(t, []string{"Story1", "Story2"}, itemsByStatus["started"])
	assert.Equal(t, []string{"Story4", "Story3", "Story5", "Story6", "Story7"}, itemsByStatus["unstarted"])
	assert.Equal(t, []string{"Story8"}, itemsByStatus["finished"])
}

func TestOverviewUpdateV2Statuses(t *testing.T) {
	updatedOverviewData := `# New backlog

### Unstarted
| User | Title | Points | Tags |
|---|---|:---:|---|
|  | [Story 4](Story4) | 30 |  |
|  | [Story Six](Story6) |  |  |
| Total Points | | 30 | |

### Started
| User | Title | Points | Tags |
|---|---|:---:|---|
| mike | [First story](Story1) | 10 |  |
|  | [Story 5](Story5) | 20 |  |
| Total Points | | 30 | |

### Finished
| User | Title | Points | Tags |
|---|---|:---:|---|

### Delivered
| User | Title | Points | Tags |
|---|---|:---:|---|
| robert | [Second story](Story2) | 15 |  |

### Accepted
| User | Title | Points | Tags |
|---|---|:---:|---|
|  | [Story 8](Story8) |  |  |

### Rejected
| User | Title | Points | Tags |
|---|---|:---:|---|

[Archived stories](archive.md)

### Metadata

Data2: test2  
`

	content := markdown.NewMarkdown(markdownOverviewData, "",
		[]string{"Title", "Data"},
		[]string{"Data2"},
		"### ", backlog.OverviewFooterRe)
	overview := backlog.NewBacklogOverview(content)
	sorter := backlog.NewBacklogItemsSorter(overview)
	userList := backlog.NewUserList("")
	overview.SetTitle("New backlog")

	story1 := createBacklogItem("Story1", "First story", "started", "10", "mike")
	story2 := createBacklogItem("Story2", "Second story", "delivered", "15", "robert")
	story5 := createBacklogItem("Story5", "Story 5", "started", "20", "")
	story6 := createBacklogItem("Story6", "Story Six", "unstarted", "", "")
	story4 := createBacklogItem("Story4", "Story 4", "unstarted", "30", "")
	story8 := createBacklogItem("Story8", "Story 8", "accepted", "", "")
	overview.Update([]*backlog.BacklogItem{
		story1, story2, story5, story6, story4, story8,
	}, sorter, userList)

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
