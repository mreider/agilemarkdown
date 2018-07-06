package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"gopkg.in/urfave/cli.v1"
	"path/filepath"
	"strings"
)

var CreateIdeaCommand = cli.Command{
	Name:      "create-idea",
	Usage:     "Create a new idea",
	ArgsUsage: "IDEA_TITLE",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "simulate",
			Hidden: true,
		},
		cli.StringFlag{
			Name:   "user",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
		simulate := c.Bool("simulate")
		user := c.String("user")

		rootDir, _ := filepath.Abs(".")
		if err := checkIsBacklogDirectory(); err == nil {
			rootDir = filepath.Dir(rootDir)
		} else if filepath.Base(rootDir) == backlog.IdeasDirectoryName {
			rootDir = filepath.Dir(rootDir)
		} else if err := checkIsRootDirectory("."); err != nil {
			return err
		}

		if c.NArg() == 0 {
			if !simulate {
				fmt.Println("an idea name should be specified")
			}
			return nil
		}
		ideaTitle := strings.Join(c.Args(), " ")
		ideaName := strings.Replace(ideaTitle, " ", "-", -1)
		ideaPath := filepath.Join(rootDir, backlog.IdeasDirectoryName, fmt.Sprintf("%s.md", ideaName))
		if existsFile(ideaPath) {
			if !simulate {
				fmt.Println("file exists")
			} else {
				fmt.Println(ideaPath)
			}
			return nil
		}

		currentUser := user
		if currentUser == "" {
			var err error
			currentUser, _, err = git.CurrentUser()
			if err != nil {
				currentUser = "unknown"
			}
		}

		idea, err := backlog.LoadBacklogIdea(ideaPath)
		if err != nil {
			return err
		}
		currentTimestamp := utils.GetCurrentTimestamp()
		idea.SetTitle(utils.TitleFirstLetter(ideaTitle))
		idea.SetCreated(currentTimestamp)
		idea.SetModified(currentTimestamp)
		idea.SetAuthor(currentUser)
		idea.SetTags(nil)
		idea.SetRank("")

		if !simulate {
			return idea.Save()
		} else {
			rootDir := filepath.Dir(filepath.Dir(ideaPath))
			fmt.Println(strings.TrimPrefix(ideaPath, rootDir))
			fmt.Print(string(idea.Content()))
			return nil
		}
	},
}
