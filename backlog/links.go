package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
)

func MakeItemLink(item *BacklogItem, baseDir string) string {
	itemPath := item.markdown.contentPath
	if itemPath == "" {
		itemPath = item.Name()
	}
	return utils.MakeMarkdownLink(item.Title(), itemPath, baseDir)
}

func MakeIdeaLink(idea *BacklogIdea, baseDir string) string {
	ideaPath := idea.markdown.contentPath
	if ideaPath == "" {
		ideaPath = idea.Name()
	}
	return utils.MakeMarkdownLink(idea.Title(), ideaPath, baseDir)
}
