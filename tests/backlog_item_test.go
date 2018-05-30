package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	itemMarkdownData = `# Support for clarification requests

Project: Test job

[title1](link1) [title2](link2)

Created: 2018-05-03 03:32 PM  
Modified: 2018-05-08 09:18 PM  
Author: mreider  
Status: doing  
Assigned: falconandy  
Estimate: 3  

## Problem statement

There is no way to show users what comments are waiting for their responses.

## Possible solution

We could add a comment section to each story, and in the overview page we could show the things that people need to respond to.

## Comments

  @falcon Why so?

@falconandy I think the problem here is that things are hard.
 @falcon I think the problem here is comment 2.

	@falconandy. I think the problem here is comment 3.

@mreider What?
How to do?

## Attachments
`
)

func TestBacklogItem(t *testing.T) {
	item := backlog.NewBacklogItem("comments", itemMarkdownData)

	assert.Equal(t, "Support for clarification requests", item.Title())
	assert.Equal(t, "Project: Test job", item.Header())
	assert.Equal(t, "[title1](link1) [title2](link2)", item.Links())
	assert.Equal(t, "2018-05-03 15:32", item.Created().Format("2006-01-02 15:04"))
	assert.Equal(t, "2018-05-08 21:18", item.Modified().Format("2006-01-02 15:04"))
	assert.Equal(t, "mreider", item.Author())
	assert.Equal(t, "doing", item.Status())
	assert.Equal(t, "falconandy", item.Assigned())
	assert.Equal(t, "3", item.Estimate())

	comments := item.Comments()
	assert.Equal(t, 5, len(comments))

	assert.Equal(t, "falcon", comments[0].User)
	assert.True(t, comments[0].Closed)
	assert.Equal(t, 1, len(comments[0].Text))
	assert.Equal(t, "Why so?", comments[0].Text[0])

	assert.Equal(t, "falconandy", comments[1].User)
	assert.False(t, comments[1].Closed)
	assert.Equal(t, 1, len(comments[1].Text))
	assert.Equal(t, "I think the problem here is that things are hard.", comments[1].Text[0])

	assert.Equal(t, "falcon", comments[2].User)
	assert.True(t, comments[2].Closed)
	assert.Equal(t, 1, len(comments[2].Text))
	assert.Equal(t, "I think the problem here is comment 2.", comments[2].Text[0])

	assert.Equal(t, "falconandy", comments[3].User)
	assert.True(t, comments[3].Closed)
	assert.Equal(t, 1, len(comments[3].Text))
	assert.Equal(t, "I think the problem here is comment 3.", comments[3].Text[0])

	assert.Equal(t, "mreider", comments[4].User)
	assert.False(t, comments[4].Closed)
	assert.Equal(t, 2, len(comments[4].Text))
	assert.Equal(t, "What?", comments[4].Text[0])
	assert.Equal(t, "How to do?", comments[4].Text[1])
}
