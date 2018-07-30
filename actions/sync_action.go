package actions

import (
	"errors"
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/git"
	"os"
	"path/filepath"
	"strings"
)

type SyncAction struct {
	rootDir    string
	configName string
	author     string
	testMode   bool
}

func NewSyncAction(rootDir, configName, author string, testMode bool) *SyncAction {
	return &SyncAction{rootDir: rootDir, configName: configName, author: author, testMode: testMode}
}

func (a *SyncAction) Execute() error {
	cfgPath := filepath.Join(a.rootDir, a.configName)
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("Can't load the config file %s: %v\n", cfgPath, err)
	}

	userList := backlog.NewUserList(filepath.Join(a.rootDir, backlog.UsersDirectoryName))

	attempts := 10
	for attempts > 0 {
		attempts--

		err := NewSyncItemsStep(a.rootDir).Execute()
		if err != nil {
			return err
		}

		err = NewSyncOverviewsAndIndexStep(a.rootDir, cfg, userList, a.author).Execute()
		if err != nil {
			return err
		}

		err = NewSyncVelocityStep(a.rootDir).Execute()
		if err != nil {
			return err
		}

		err = NewSyncIdeasStep(a.rootDir, cfg, userList, a.author).Execute()
		if err != nil {
			return err
		}

		err = NewSyncTagsStep(a.rootDir, userList).Execute()
		if err != nil {
			return err
		}

		err = NewSyncUsersStep(a.rootDir).Execute()
		if err != nil {
			return err
		}

		err = NewSyncTimelineStep(a.rootDir).Execute()
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
