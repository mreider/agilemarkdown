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

const AttemptCount = 10

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

	_, err = NewSyncUsersCheck(a.root, userList).Check()
	if err != nil {
		return err
	}

	attempts := AttemptCount
	for attempts > 0 {
		if attempts < AttemptCount {
			fmt.Printf("Attempt %d\n", AttemptCount-attempts+1)
		}

		attempts--

		err := NewSyncDownCaseStep(a.root, userList).Execute()
		if err != nil {
			return err
		}

		err = NewSyncItemsStep(a.root).Execute()
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
			fmt.Println("OK")
			return nil
		}

		ok, err := a.syncToGit()
		if err != nil {
			return err
		}
		if ok {
			fmt.Println("OK")
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
	fmt.Println("git commit")
	err = git.Commit("sync", a.author) // TODO commit message
	if err != nil {
		return false, fmt.Errorf("can't commit: %v", err)
	}
	err = git.Fetch()
	if err != nil {
		return false, fmt.Errorf("can't fetch: %v", err)
	}
	fmt.Println("git merge")
	_, mergeErr := git.Merge()
	if mergeErr != nil {
		status, _ := git.Status()
		if !strings.Contains(status, "Your branch is based on 'origin/master', but the upstream is gone.") {
			fmt.Println("Auto-resolving merge conflicts")
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
				_ = git.AbortMerge()
				return false, fmt.Errorf("can't merge: %v", mergeErr)
			}
			for _, conflictFile := range conflictFiles {
				err = git.CheckoutOurVersion(conflictFile)
				if err != nil {
					return false, err
				}

				err := git.Add(conflictFile)
				if err != nil {
					return false, err
				}
				fmt.Printf("Remote changes to %s are ignored\n", conflictFile)
			}
			err = git.CommitNoEdit(a.author)
			return false, err
		}
	}
	fmt.Println("git push")
	err = git.Push()
	if err != nil {
		return false, fmt.Errorf("can't push: %v", err)
	}
	return true, nil
}
