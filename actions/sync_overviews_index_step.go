package actions

import (
	"fmt"
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/mreider/agilemarkdown/config"
	"github.com/mreider/agilemarkdown/utils"
)

type SyncOverviewsAndIndexStep struct {
	root     *backlog.BacklogsStructure
	cfg      *config.Config
	userList *backlog.UserList
	author   string
}

func NewSyncOverviewsAndIndexStep(root *backlog.BacklogsStructure, cfg *config.Config, userList *backlog.UserList, author string) *SyncOverviewsAndIndexStep {
	return &SyncOverviewsAndIndexStep{root: root, cfg: cfg, userList: userList, author: author}
}

func (s *SyncOverviewsAndIndexStep) Execute() error {
	backlogDirs, err := s.root.BacklogDirs()
	if err != nil {
		return err
	}
	indexPath := s.root.IndexFile()
	index, err := backlog.LoadGlobalIndex(indexPath)
	if err != nil {
		return err
	}
	if len(index.FreeText()) == 0 {
		index.SetFreeText([]string{
			"# Agile Markdown",
			"",
			"Welcome aboard",
			"",
		})
	}
	overviews := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	archives := make([]*backlog.BacklogOverview, 0, len(backlogDirs))
	for _, backlogDir := range backlogDirs {
		overviewPath, ok := backlog.FindOverviewFileInRootDirectory(backlogDir)
		if !ok {
			return fmt.Errorf("the overview file isn't found for %s", backlogDir)
		}

		err = s.moveItemsToActiveAndArchiveDirectory(backlogDir)
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

		archivePath, _ := backlog.FindArchiveFileInDirectory(backlogDir)
		archive, err := backlog.LoadBacklogOverview(archivePath)
		if err != nil {
			return err
		}
		archive.SetHideEmptyGroups(true)

		overviews = append(overviews, overview)
		archives = append(archives, archive)

		sorter := backlog.NewBacklogItemsSorter(overview, archive)

		activeItems := bck.ActiveItems()
		overview.UpdateLinks("archive", archivePath, s.root.Root(), s.root.Root())
		overview.Update(activeItems, sorter, s.userList)
		s.sendNewCommentsForItems(overview, activeItems)
		overview.Save()

		archivedItems := bck.ArchivedItems()
		archive.SetTitle(fmt.Sprintf("Archive: %s", overview.Title()))
		archive.UpdateLinks("project page", overviewPath, s.root.Root(), backlogDir)
		archive.Update(archivedItems, sorter, s.userList)
		archive.Save()

		overview.RemoveVelocity(bck)

		for _, item := range bck.AllItems() {
			item.SetHeader(fmt.Sprintf("Project: %s", overview.Title()))
			item.UpdateLinks(s.root.Root(), overviewPath, archivePath)
		}
	}
	index.UpdateBacklogs(overviews, archives, s.root.Root())
	index.UpdateLinks(s.root.Root())

	return nil
}

func (s *SyncOverviewsAndIndexStep) moveItemsToActiveAndArchiveDirectory(backlogDir string) error {
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

func (s *SyncOverviewsAndIndexStep) sendNewCommentsForItems(overview *backlog.BacklogOverview, items []*backlog.BacklogItem) {
	userList := backlog.NewUserList(s.root.UsersDirectory())
	var mailSender *utils.MailSender
	if s.cfg.SmtpServer != "" {
		mailSender = utils.NewMailSender(s.cfg.SmtpServer, s.cfg.SmtpUser, s.cfg.SmtpPassword, s.cfg.EmailFrom)
	}

	commented := make([]backlog.Commented, len(items))
	for i := range items {
		commented[i] = items[i]
	}
	sendNewComments(commented, func(item backlog.Commented, to []string, comment []string) (me string, err error) {
		return sendComment(userList, comment, overview.Title(), s.author, to, mailSender, s.cfg, s.root.Root(), item.Path())
	})
}