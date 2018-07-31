package actions

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"os"
	"strings"
)

type SyncAction struct {
	root     *backlog.BacklogsStructure
	author   string
	testMode bool
}

func NewSyncAction(rootDir, author string, testMode bool) *SyncAction {
	return &SyncAction{root: backlog.NewBacklogsStructure(rootDir), author: author, testMode: testMode}
}

func (a *SyncAction) Execute() error {
	cfg, err := config.LoadConfig(a.root.ConfigFile())
	if err != nil {
		return fmt.Errorf("Can't load the config file %s: %v\n", a.root.ConfigFile(), err)
	}

	userList := backlog.NewUserList(a.root.UsersDirectory())

	ok, err := NewSyncUsersCheck(a.root, userList).Check()
	if err != nil {
		return err
	}

	if !ok {
		fmt.Println("Sync cancelled")
		return nil
	}

	attempts := 10
	for attempts > 0 {
		attempts--

		err := NewSyncItemsStep(a.root).Execute()
		if err != nil {
			return err
		}

		err = NewSyncOverviewsAndIndexStep(a.root, cfg, userList, a.author).Execute()
		if err != nil {
			return err
		}

		err = NewSyncVelocityStep(a.root).Execute()
		if err != nil {
			return err
		}

		err = NewSyncIdeasStep(a.root, cfg, userList, a.author).Execute()
		if err != nil {
			return err
		}

		err = NewSyncTagsStep(a.root, userList).Execute()
		if err != nil {
			return err
		}

		err = NewSyncUsersStep(a.root).Execute()
		if err != nil {
			return err
		}

		err = NewSyncTimelineStep(a.root).Execute()
		if err != nil {
			return err
		}

		if a.testMode {
			return nil
		}

		ok, err := a.syncToGit()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return errors.New("can't sync: too many failed attempts")
}

func (a *SyncAction) syncToGit() (bool, error) {
	err := git.AddAll()
	if err != nil {
		return false, err
	}
	git.Commit("sync", a.author) // TODO commit message
	err = git.Fetch()
	if err != nil {
		return false, fmt.Errorf("can't fetch: %v", err)
	}
	mergeOutput, mergeErr := git.Merge()
	if mergeErr != nil {
		status, _ := git.Status()
		if !strings.Contains(status, "Your branch is based on 'origin/master', but the upstream is gone.") {
			conflictFiles, conflictErr := git.ConflictFiles()
			hasConflictItems := false
			for _, fileName := range conflictFiles {
				if fileName == backlog.TagsFileName || strings.HasPrefix(fileName, backlog.TagsDirectoryName+string(os.PathSeparator)) {
					continue
				}

				fileName = strings.TrimSuffix(fileName, string(os.PathSeparator)+backlog.ArchiveFileName)
				if strings.Contains(fileName, "/") {
					hasConflictItems = true
					break
				}
			}
			if conflictErr != nil || hasConflictItems {
				fmt.Println(mergeOutput)
				git.AbortMerge()
				return false, fmt.Errorf("can't merge: %v", mergeErr)
			}
			for _, conflictFile := range conflictFiles {
				git.CheckoutOurVersion(conflictFile)
				git.Add(conflictFile)
				fmt.Printf("Remote changes to %s are ignored\n", conflictFile)
			}
			git.CommitNoEdit(a.author)
			return false, nil
		}
	}
	err = git.Push()
	if err != nil {
		return false, fmt.Errorf("can't push: %v", err)
	}
	return true, nil
}
