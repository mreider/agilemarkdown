package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
)

type DeleteTagAction struct {
	root *backlog.BacklogsStructure
	tag  string
}

func NewDeleteTagAction(rootDir, tag string) *DeleteTagAction {
	return &DeleteTagAction{root: backlog.NewBacklogsStructure(rootDir), tag: tag}
}

func (a *DeleteTagAction) Execute() error {
	if !confirmAction("This will delete links to ideas and timelines ok? (y or n)") {
		return nil
	}

	allTags, itemsTags, ideasTags, _, err := backlog.ItemsAndIdeasTags(a.root)
	if err != nil {
		return err
	}

	if _, ok := allTags[a.tag]; !ok {
		fmt.Printf("Tag '%s' not found.\n", a.tag)
		return nil
	}

	tagItems := itemsTags[a.tag]
	for _, item := range tagItems {
		itemTags := item.Tags()
		itemTags = utils.RemoveItemIgnoreCase(itemTags, a.tag)
		item.SetTags(itemTags)
		item.ClearTimeline(a.tag)
		item.Save()
	}

	tagIdeas := ideasTags[a.tag]
	for _, idea := range tagIdeas {
		ideaTags := idea.Tags()
		ideaTags = utils.RemoveItemIgnoreCase(ideaTags, a.tag)
		idea.SetTags(ideaTags)
		idea.Save()
	}

	fmt.Printf("Tag '%s' deleted. Sync to regenerate files.\n", a.tag)

	return nil
}
