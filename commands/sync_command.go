package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var chartColorCodeRe = regexp.MustCompile(`.\[\d+m`)

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

		err = a.updateHome(rootDir)
		if err != nil {
			return err
		}

		err = a.updateSidebar(rootDir)
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
	}
	return nil
}

func (a *SyncAction) updateHome(rootDir string) error {
	var lines []string
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
		lines = append(lines, fmt.Sprintf("### [%s](%s)", overview.Title(), strings.TrimSuffix(filepath.Base(overviewPath), ".md")))
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		progressAction := NewProgressAction(60)
		chart, err := progressAction.Execute(backlogDir, 12)
		if err != nil {
			return err
		}

		chart = chartColorCodeRe.ReplaceAllString(chart, "")
		lines = append(lines, utils.WrapLinesToMarkdownCodeBlock(strings.Split(chart, "\n"))...)

		doing := backlog.DoingStatus
		items := bck.ItemsByStatus(doing.Code)
		overview.SortItems(doing, items)
		itemsLines := backlog.BacklogView{}.WriteMarkdownTable(items)
		lines = append(lines, fmt.Sprintf("#### %s", doing.CapitalizedName()))
		lines = append(lines, itemsLines...)
	}
	err = ioutil.WriteFile(filepath.Join(rootDir, "Home.md"), []byte(strings.Join(lines, "  \n")), 0644)
	return err
}

func (a *SyncAction) updateSidebar(rootDir string) error {
	var lines []string
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
		lines = append(lines, fmt.Sprintf("[%s](%s)", overview.Title(), strings.TrimSuffix(filepath.Base(overviewPath), ".md")))
	}
	err = ioutil.WriteFile(filepath.Join(rootDir, "_Sidebar.md"), []byte(strings.Join(lines, "  \n")), 0644)
	return err
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
