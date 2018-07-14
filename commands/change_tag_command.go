package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var ChangeTagCommand = cli.Command{
	Name:      "change-tag",
	Usage:     "Change a tag",
	ArgsUsage: "OLD_TAG NEW_TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() != 2 {
			fmt.Println("old and new tags should be specified")
			return nil
		}

		rootDir, _ := filepath.Abs(".")
		if err := checkIsBacklogDirectory(); err == nil {
			rootDir = filepath.Dir(rootDir)
		} else if err := checkIsRootDirectory("."); err != nil {
			return err
		}

		oldTag := strings.ToLower(c.Args()[0])
		newTag := strings.ToLower(c.Args()[1])

		if oldTag == newTag {
			fmt.Println("Old and new tags are equal")
			return nil
		}

		allTags, itemsTags, ideasTags, _, err := backlog.ItemsAndIdeasTags(rootDir)
		if err != nil {
			return err
		}

		if _, ok := allTags[oldTag]; !ok {
			fmt.Printf("Tag '%s' not found.\n", oldTag)
			return nil
		}

		tagItems := itemsTags[oldTag]
		for _, item := range tagItems {
			itemTags := item.Tags()
			itemTags = utils.RenameItemIgnoreCase(itemTags, oldTag, newTag)
			item.SetTags(itemTags)
			item.ChangeTimelineTag(oldTag, newTag)
			item.Save()
		}

		tagIdeas := ideasTags[oldTag]
		for _, idea := range tagIdeas {
			ideaTags := idea.Tags()
			ideaTags = utils.RenameItemIgnoreCase(ideaTags, oldTag, newTag)
			idea.SetTags(ideaTags)
			idea.Save()
		}

		backlog.NewTimelineGenerator(rootDir).RenameTimeline(oldTag, newTag)

		fmt.Printf("Tag '%s' changed to '%s'. Sync to regenerate files.\n", oldTag, newTag)

		return nil
	},
}
