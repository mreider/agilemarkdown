package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/git"
	"github.com/mreider/agilemarkdown/utils"
	"os"
	"path/filepath"
	"strings"
)

type SyncItemsStep struct {
	rootDir string
}

func NewSyncItemsStep(rootDir string) *SyncItemsStep {
	return &SyncItemsStep{rootDir: rootDir}
}

func (s *SyncItemsStep) Execute() error {
	err := s.updateItemsModifiedDate()
	if err != nil {
		return err
	}

	return s.updateItemsFileNames()
}

func (s *SyncItemsStep) updateItemsModifiedDate() error {
	backlogDirs, err := backlog.BacklogDirs(s.rootDir)
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		modifiedFiles, err := git.ModifiedFiles(backlogDir)
		if len(modifiedFiles) == 0 {
			continue
		}

		modifiedFilesSet := make(map[string]bool)
		for _, file := range modifiedFiles {
			modifiedFilesSet[file] = true
		}

		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}
		for _, item := range bck.AllItems() {
			if modifiedFilesSet[filepath.Base(item.Path())] {
				itemPath, _ := filepath.Rel(s.rootDir, item.Path())
				itemPath = fmt.Sprintf("./%s", itemPath)
				repoItemContent, err := git.RepoVersion(s.rootDir, itemPath)
				if err != nil {
					return err
				}
				repoItem := backlog.NewBacklogItem(filepath.Base(itemPath), repoItemContent)
				currentTimestamp := utils.GetCurrentTimestamp()
				if item.Assigned() != repoItem.Assigned() || item.Status() != repoItem.Status() || item.Estimate() != repoItem.Estimate() {
					if item.Modified() == repoItem.Modified() {
						item.SetModified(currentTimestamp)
						item.Save()
					}
				}

				oldStatus := backlog.StatusByName(repoItem.Status())
				newStatus := backlog.StatusByName(item.Status())
				if oldStatus != newStatus {
					if newStatus == backlog.FinishedStatus {
						item.SetFinished(currentTimestamp)
						item.Save()
					} else if oldStatus == backlog.FinishedStatus {
						item.SetFinished("")
						item.Save()
					} else {
						if !item.Finished().IsZero() {
							item.SetFinished("")
							item.Save()
						}
					}
				} else if oldStatus == backlog.FinishedStatus && newStatus == backlog.FinishedStatus {
					if item.Finished().IsZero() && !repoItem.Finished().IsZero() {
						item.SetFinished(utils.GetTimestamp(repoItem.Finished()))
						item.Save()
					}
				} else if oldStatus == newStatus {
					if !item.Finished().IsZero() {
						item.SetFinished("")
						item.Save()
					}
				}
			}
		}
	}
	return nil
}

func (s *SyncItemsStep) updateItemsFileNames() error {
	backlogDirs, err := backlog.BacklogDirs(s.rootDir)
	if err != nil {
		return err
	}
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}
		overview, err := backlog.LoadBacklogOverview(overviewPath)
		if err != nil {
			return err
		}

		bck, err := backlog.LoadBacklog(backlogDir)
		if err != nil {
			return err
		}
		for _, item := range bck.AllItems() {
			currentItemName := strings.ToLower(filepath.Base(item.Path()))
			expectedItemName := strings.ToLower(utils.GetValidFileName(item.Title()) + ".md")
			if currentItemName != expectedItemName {
				newItemPath := filepath.Join(filepath.Dir(item.Path()), expectedItemName)
				if _, err := os.Stat(newItemPath); os.IsNotExist(err) {
					err := os.Rename(item.Path(), newItemPath)
					if err == nil {
						git.Add(newItemPath)
						overview.UpdateItemLinkInOverviewFile(item.Path(), newItemPath)
					}
				}
			}
		}
	}
	return nil
}
