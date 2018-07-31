package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
)

type ChangeTagAction struct {
	root   *backlog.BacklogsStructure
	oldTag string
	newTag string
}

func NewChangeTagAction(rootDir, oldTag, newTag string) *ChangeTagAction {
	return &ChangeTagAction{root: backlog.NewBacklogsStructure(rootDir), oldTag: oldTag, newTag: newTag}
}

func (a *ChangeTagAction) Execute() error {
	if a.oldTag == a.newTag {
		fmt.Println("Old and new tags are equal")
		return nil
	}

	allTags, itemsTags, ideasTags, _, err := backlog.ItemsAndIdeasTags(a.root)
	if err != nil {
		return err
	}

	if _, ok := allTags[a.oldTag]; !ok {
		fmt.Printf("Tag '%s' not found.\n", a.oldTag)
		return nil
	}

	tagItems := itemsTags[a.oldTag]
	for _, item := range tagItems {
		itemTags := item.Tags()
		itemTags = utils.RenameItemIgnoreCase(itemTags, a.oldTag, a.newTag)
		item.SetTags(itemTags)
		item.ChangeTimelineTag(a.oldTag, a.newTag)
		item.Save()
	}

	tagIdeas := ideasTags[a.oldTag]
	for _, idea := range tagIdeas {
		ideaTags := idea.Tags()
		ideaTags = utils.RenameItemIgnoreCase(ideaTags, a.oldTag, a.newTag)
		idea.SetTags(ideaTags)
		idea.Save()
	}

	backlog.NewTimelineGenerator(a.root).RenameTimeline(a.oldTag, a.newTag)

	fmt.Printf("Tag '%s' changed to '%s'. Sync to regenerate files.\n", a.oldTag, a.newTag)

	return nil
}
