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
	if !confirmAction("This will delete the tag from all items and remove its timeline ok? (y or n)") {
		return nil
	}

	allTags, itemsTags, _, err := backlog.ItemsTags(a.root)
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
		item.ClearTimeline()
		err := item.Save()
		if err != nil {
			return err
		}
	}

	fmt.Printf("Tag '%s' deleted. Sync to regenerate files.\n", a.tag)

	return nil
}
