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

func MakeOverviewLink(overview *BacklogOverview, baseDir string) string {
	overviewPath := overview.markdown.contentPath
	return utils.MakeMarkdownLink(overview.Title(), overviewPath, baseDir)
}

func MakeArchiveLink(archive *BacklogOverview, title string, baseDir string) string {
	archivePath := archive.markdown.contentPath
	return utils.MakeMarkdownLink(title, archivePath, baseDir)
}
