package commands

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"github.com/pkg/errors"
	"gopkg.in/urfave/cli.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

		err = a.updateIdeas(rootDir)
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
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}

		err = a.moveItemsToActiveAndArchiveDirectory(backlogDir)
		if err != nil {
			return err
		}

		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}
		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}

		archivePath, _ := findArchiveFileInDirectory(backlogDir)
		archive, err := backlog.LoadBacklogOverview(archivePath)
		if err != nil {
			return err
		}
		archive.SetHideEmptyGroups(true)

		sorter := backlog.NewBacklogItemsSorter(overview, archive)

		activeItems := bck.ActiveItems()
		overview.Update(activeItems, sorter)
		overview.UpdateClarifications(activeItems)

		archivedItems := bck.ArchivedItems()
		archive.Update(archivedItems, sorter)
		archive.UpdateClarifications(archivedItems)

		err = overview.UpdateProgress(bck)
		if err != nil {
			return err
		}

		overview.UpdateArchiveLink(len(archivedItems) > 0, archivePath)
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
				fileName = strings.TrimSuffix(fileName, string(os.PathSeparator) + ArchiveFileName)
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
		if !info.IsDir() || strings.HasPrefix(info.Name(), ".") || backlog.IsForbiddenBacklogName(info.Name()) {
			continue
		}
		result = append(result, filepath.Join(rootDir, info.Name()))
	}
	sort.Strings(result)
	return result, nil
}

func (a *SyncAction) updateIdeas(rootDir string) error {
	ideasDir := filepath.Join(rootDir, "ideas")
	infos, err := ioutil.ReadDir(ideasDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	ideasPaths := make([]string, 0, len(infos))
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		ideaPath := filepath.Join(ideasDir, info.Name())
		ideasPaths = append(ideasPaths, ideaPath)
	}

	sort.Strings(ideasPaths)

	ideas := make([]*backlog.BacklogIdea, 0, len(ideasPaths))
	for _, ideaPath := range ideasPaths {
		idea, err := a.updateIdea(ideaPath)
		if err != nil {
			fmt.Printf("can't update idea '%s'\n", err)
			continue
		}
		ideas = append(ideas, idea)
	}

	lines := backlog.BacklogView{}.WriteMarkdownIdeas(ideas)
	return ioutil.WriteFile(filepath.Join(rootDir, "ideas.md"), []byte(strings.Join(lines, "\n")), 0644)
}

func (a *SyncAction) updateIdea(ideaPath string) (*backlog.BacklogIdea, error) {
	idea, err := backlog.LoadBacklogIdea(ideaPath)
	if err != nil {
		return nil, err
	}
	if !idea.HasMetadata() {
		author, created, err := git.InitCommitInfo(ideaPath)
		if err != nil {
			return nil, err
		}
		if author == "" {
			author, _ = git.CurrentUser()
			created = time.Now()
		}

		ideaName := filepath.Base(ideaPath)
		ideaName = strings.TrimSuffix(ideaName, filepath.Ext(ideaName))
		ideaTitle := strings.Replace(ideaName, "-", " ", -1)
		ideaTitle = strings.Replace(ideaTitle, "_", " ", -1)
		ideaTitle = utils.TitleFirstLetter(ideaTitle)
		idea.SetTitle(ideaTitle)
		idea.SetCreated(utils.GetTimestamp(created))
		idea.SetModified(utils.GetTimestamp(created))
		idea.SetAuthor(author)
		idea.SetTags(nil)
		idea.SetText(idea.Text())
		idea.Save()
	}
	return idea, nil
}

func (a *SyncAction) moveItemsToActiveAndArchiveDirectory(backlogDir string) error {
	bck, err := backlog.LoadBacklog(backlogDir)
	if err != nil {
		return err
	}

	for _, item := range bck.ActiveItems() {
		err := item.MoveToBacklogDirectory()
		if err != nil {
			return err
		}
	}

	for _, item := range bck.ArchivedItems() {
		err := item.MoveToBacklogArchiveDirectory()
		if err != nil {
			return err
		}
	}

	return nil
}
