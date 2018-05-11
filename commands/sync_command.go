package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
)

var SyncCommand = cli.Command{
	Name:      "sync",
	Usage:     "Sync state",
	ArgsUsage: " ",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:   "test",
			Hidden: true,
		},
	},
	Action: func(c *cli.Context) error {
		action := &SyncAction{testMode: c.Bool("test")}
		return action.Execute()
	},
}

type SyncAction struct {
	testMode bool
}

func (a *SyncAction) Execute() error {
	rootDir, _ := filepath.Abs(".")
	if err := checkIsBacklogDirectory(); err == nil {
		rootDir = filepath.Dir(rootDir)
	} else if err := checkIsRootDirectory(); err != nil {
		return err
	}

	attempts := 10
	for attempts > 0 {
		attempts--

		err := a.updateOverviews(rootDir)
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

func (a *SyncAction) updateOverviews(rootDir string) error {
	backlogDirs, err := a.backlogDirs(rootDir)
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := findOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the index file isn't found for %s", backlogDir)
		}
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		items := bck.Items()
		overview.Update(items)
		overview.UpdateClarifications(items)
		err = overview.UpdateProgress(bck)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *SyncAction) syncToGit() (bool, error) {
	err := git.AddAll()
	if err != nil {
		return false, err
	}
	git.Commit("sync") // TODO commit message
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
			git.CommitNoEdit()
			return false, nil
		}
	}
	err = git.Push()
	if err != nil {
		return false, fmt.Errorf("can't push: %v", err)
	}
	return true, nil
}

func (a *SyncAction) backlogDirs(rootDir string) ([]string, error) {
	infos, err := ioutil.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			continue
		}
		result = append(result, filepath.Join(rootDir, info.Name()))
	}
	sort.Strings(result)
	return result, nil
}
