package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"path/filepath"
	"strings"
)

const newIdeaTemplate = `

## Comments

`

type CreateIdeaAction struct {
	rootDir   string
	ideaTitle string
	user      string
	simulate  bool
}

func NewCreateIdeaAction(rootDir, ideaTitle, user string, simulate bool) *CreateIdeaAction {
	return &CreateIdeaAction{rootDir: rootDir, ideaTitle: ideaTitle, user: user, simulate: simulate}
}

func (a *CreateIdeaAction) Execute() error {
	ideaName := utils.GetValidFileName(a.ideaTitle)
	ideaPath := filepath.Join(a.rootDir, backlog.IdeasDirectoryName, fmt.Sprintf("%s.md", ideaName))
	if existsFile(ideaPath) {
		if !a.simulate {
			fmt.Println("file exists")
		} else {
			fmt.Println(ideaPath)
		}
		return nil
	}

	currentUser := a.user
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
	idea.SetTitle(utils.TitleFirstLetter(a.ideaTitle))
	idea.SetCreated(currentTimestamp)
	idea.SetModified(currentTimestamp)
	idea.SetAuthor(currentUser)
	idea.SetTags(nil)
	idea.SetRank("")
	idea.SetDescription(newIdeaTemplate)

	if !a.simulate {
		return idea.Save()
	} else {
		rootDir := filepath.Dir(filepath.Dir(ideaPath))
		fmt.Println(strings.TrimPrefix(ideaPath, rootDir))
		fmt.Print(string(idea.Content()))
		return nil
	}
}
