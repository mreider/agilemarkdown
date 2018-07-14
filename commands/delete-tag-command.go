package commands

import (
	"bufio"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
	"strings"
)

var DeleteTagCommand = cli.Command{
	Name:      "delete-tag",
	Usage:     "Delete a tag",
	ArgsUsage: "TAG",
	Action: func(c *cli.Context) error {
		if c.NArg() != 1 {
			fmt.Println("a tag should be specified")
			return nil
		}

		rootDir, _ := filepath.Abs(".")
		if err := checkIsBacklogDirectory(); err == nil {
			rootDir = filepath.Dir(rootDir)
		} else if err := checkIsRootDirectory("."); err != nil {
			return err
		}

		tag := strings.ToLower(c.Args()[0])
		fmt.Println("This will delete links to ideas and timelines ok? (y or n)")

		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		text = strings.ToLower(strings.TrimSpace(text))
		if text != "y" {
			return nil
		}

		allTags, itemsTags, ideasTags, _, err := backlog.ItemsAndIdeasTags(rootDir)
		if err != nil {
			return err
		}

		if _, ok := allTags[tag]; !ok {
			fmt.Printf("Tag '%s' not found.\n", tag)
			return nil
		}

		tagItems := itemsTags[tag]
		for _, item := range tagItems {
			itemTags := item.Tags()
			itemTags = utils.RemoveItemIgnoreCase(itemTags, tag)
			item.SetTags(itemTags)
			item.ClearTimeline(tag)
			item.Save()
		}

		tagIdeas := ideasTags[tag]
		for _, idea := range tagIdeas {
			ideaTags := idea.Tags()
			ideaTags = utils.RemoveItemIgnoreCase(ideaTags, tag)
			idea.SetTags(ideaTags)
			idea.Save()
		}

		fmt.Printf("Tag '%s' deleted. Sync to regenerate files.\n", tag)

		return nil
	},
}
