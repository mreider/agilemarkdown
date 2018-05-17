package backlog

import (
	"github.com/mreider/agilemarkdown/utils"
)

func MakeItemLink(item *BacklogItem, baseDir string) string {
	return utils.MakeMarkdownLink(item.Title(), item.markdown.contentPath, baseDir)
}

func MakeIdeaLink(idea *BacklogIdea, baseDir string) string {
	return utils.MakeMarkdownLink(idea.Title(), idea.markdown.contentPath, baseDir)
}
