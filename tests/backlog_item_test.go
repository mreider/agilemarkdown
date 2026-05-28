package tests

import (
	"testing"

	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
)

const itemMarkdownData = `---
title: Support for clarification requests
project: Test job
status: started
assigned: falconandy
estimate: 3
tags: [docs, comments]
author: mreider
created: 2018-05-03T15:32:00Z
modified: 2018-05-08T21:18:00Z
---

## Problem statement

There is no way to show users what comments are waiting for their responses.

## Possible solution

We could add a comment section to each story, and in the overview page we could show the things that people need to respond to.

## Comments

  @falcon @bob Why so?

@falconandy, @peter I think the problem here is that things are hard.
 @falcon I think the problem here is comment 2.

	@falconandy. I think the problem here is comment 3.

@mreider What?
How to do?

## Attachments
`

func TestBacklogItem(t *testing.T) {
	item := backlog.NewBacklogItem("comments", itemMarkdownData)

	assert.Equal(t, "Support for clarification requests", item.Title())
	assert.Equal(t, "Test job", item.Project())
	assert.Equal(t, "2018-05-03 15:32", item.Created().Format("2006-01-02 15:04"))
	assert.Equal(t, "2018-05-08 21:18", item.Modified().Format("2006-01-02 15:04"))
	assert.Equal(t, "mreider", item.Author())
	assert.Equal(t, "started", item.Status())
	assert.Equal(t, "falconandy", item.Assigned())
	assert.Equal(t, "3", item.Estimate())
	assert.Equal(t, []string{"docs", "comments"}, item.Tags())

	comments := item.Comments()
	assert.Equal(t, 5, len(comments))

	assert.Equal(t, []string{"falcon", "bob"}, comments[0].Users)
	assert.True(t, comments[0].Closed)
	assert.Equal(t, 1, len(comments[0].Text))
	assert.Equal(t, "Why so?", comments[0].Text[0])

	assert.Equal(t, []string{"falconandy", "peter"}, comments[1].Users)
	assert.False(t, comments[1].Closed)
	assert.Equal(t, 1, len(comments[1].Text))
	assert.Equal(t, "I think the problem here is that things are hard.", comments[1].Text[0])

	assert.Equal(t, []string{"falcon"}, comments[2].Users)
	assert.True(t, comments[2].Closed)
	assert.Equal(t, 1, len(comments[2].Text))
	assert.Equal(t, "I think the problem here is comment 2.", comments[2].Text[0])

	assert.Equal(t, []string{"falconandy"}, comments[3].Users)
	assert.True(t, comments[3].Closed)
	assert.Equal(t, 1, len(comments[3].Text))
	assert.Equal(t, "I think the problem here is comment 3.", comments[3].Text[0])

	assert.Equal(t, []string{"mreider"}, comments[4].Users)
	assert.False(t, comments[4].Closed)
	assert.Equal(t, 2, len(comments[4].Text))
	assert.Equal(t, "What?", comments[4].Text[0])
	assert.Equal(t, "How to do?", comments[4].Text[1])
}
